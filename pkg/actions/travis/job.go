package travis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"time"
)

// Job a Travis action's job configuration
type Job struct {
	awsJobID          string
	branch            string `required:"true"`
	client            *http.Client
	continuationToken string
	repoOwner         string `required:"true"`
	repoName          string `required:"true"`
	travisToken       string `required:"true"`
}

type triggerBuildResponse struct {
	Request struct {
		ID json.Number `json:"id"`
	} `json:"request"`
}

type build struct {
	ID            json.Number `json:"id"`
	PreviousState string      `json:"previous_state"`
	State         string      `json:"state"`
}

type buildStatusResponse struct {
	Builds []build `json:"builds"`
}

// For conditional statement to stop polling job.
type pollBuildEndCondition func(build) bool

var travisSuccessTermSet = map[string]struct{}{
	"created":  {},
	"received": {},
	"started":  {},
	"passed":   {},
}

var travisFailureTermSet = map[string]struct{}{
	"failed":   {},
	"errored":  {},
	"canceled": {},
}

var travisDoneTermSet = map[string]struct{}{
	"passed":   {},
	"failed":   {},
	"errored":  {},
	"canceled": {},
}

var sess = session.Must(session.NewSession())
var svc = codepipeline.New(sess)

// NewJob initializes a Travis action's job
func NewJob(cj *codepipeline.Job) *Job {
	zerolog.TimeFieldFormat = ""
	config := cj.Data.ActionConfiguration.Configuration

	j := Job{
		awsJobID:          aws.StringValue(cj.Id),
		client:            &http.Client{Timeout: 5 * time.Second},
		continuationToken: aws.StringValue(cj.Data.ContinuationToken),
		branch:            aws.StringValue(config["Branch"]),
		repoOwner:         aws.StringValue(config["Owner"]),
		repoName:          aws.StringValue(config["ProjectName"]),
		travisToken:       aws.StringValue(config["APIToken"]),
	}

	return &j
}

func (j *Job) triggerBuild() (requestID string, err error) {
	// TODO: make travis action url configurable, e.g. .org vs .com
	url := fmt.Sprintf("https://api.travis-ci.org/repo/%s%%2F%s/requests", j.repoOwner, j.repoName)
	data := []byte(fmt.Sprintf(`{"request": {"branch": %q}}`, j.branch))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Travis-API-Version", "3")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", j.travisToken))

	resp, err := j.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	res := triggerBuildResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return "", err
	}

	return string(res.Request.ID), nil
}

func (j *Job) getBuildStatus(requestID string) (b build, err error) {
	log.Debug().Msgf("JOB - TRAVIS: Fetching build status for request '%s'", requestID)

	url := fmt.Sprintf("https://api.travis-ci.org/repo/%s%%2F%s/request/%s", j.repoOwner, j.repoName, requestID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return build{}, errors.New("JOB - TRAVIS: Error trying to fetch build status")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Travis-API-Version", "3")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", j.travisToken))

	resp, err := j.client.Do(req)
	if err != nil {
		return build{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return build{}, err
	}

	res := buildStatusResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return build{}, err
	}

	if len(res.Builds) > 0 {
		return res.Builds[0], nil // Only expect one build for branch
	}
	return build{}, errors.New("no builds found") // TODO: maybe shouldn't be an error
}

func (j *Job) pollForResult(requestID string, f pollBuildEndCondition) (build, error) {
	c := make(chan build, 1)

	ticker := time.NewTicker(30 * time.Second) // TODO: make configurable
	go func() {
		for range ticker.C {
			log.Debug().Msg("JOB - TRAVIS: Polling for build result...")
			if b, err := j.getBuildStatus(requestID); err != nil {
				log.Error().Msgf("JOB - TRAVIS: %s", err.Error())
			} else {
				// TODO: add retry logic
				if f(b) {
					c <- b
				}
			}
		}
	}()

	select {
	case b := <-c:
		ticker.Stop()
		return b, nil
	case <-time.After(40 * time.Minute): // TODO: make this configurable.
		ticker.Stop()
		return build{}, errors.New("timed out waiting for build result")
	}
}

func (j *Job) reportSuccess(requestID string, buildID string, inProgress bool) error {
	log.Debug().Msgf("JOB - TRAVIS: Reporting success for build '%s', continuing='%t'.", buildID, inProgress)

	input := codepipeline.PutJobSuccessResultInput{}
	if inProgress {
		input.SetContinuationToken(requestID)
		input.SetExecutionDetails(&codepipeline.ExecutionDetails{ExternalExecutionId: aws.String(buildID)})
	}

	input.SetJobId(j.awsJobID)
	_, err := svc.PutJobSuccessResult(&input)
	return err
}

func (j *Job) reportFailure(buildID string) error {
	log.Debug().Msgf("JOB - TRAVIS: Reporting failure for build '%s'.", buildID)

	failureDetails := codepipeline.FailureDetails{}
	failureDetails.SetExternalExecutionId(buildID)
	failureDetails.SetMessage("Travis build failed.")
	failureDetails.SetType(codepipeline.FailureTypeJobFailed)

	input := codepipeline.PutJobFailureResultInput{}
	input.SetFailureDetails(&failureDetails)
	input.SetJobId(j.awsJobID)

	_, err := svc.PutJobFailureResult(&input)
	return err
}

func (j *Job) reportStatus(requestID string, buildID string, status string) error {
	if contains(travisSuccessTermSet, status) {
		inProgress := !contains(travisDoneTermSet, status)
		return j.reportSuccess(requestID, buildID, inProgress)
	}
	return j.reportFailure(buildID)
}

// Execute starts Travis job. If it is a new job (no continuation token present), it first submits a new
// travis build, then reports the build id to CodePipeline. If it is a continuing job, it polls Travis for
// build progress and reports the result back to CodePipeline once it is complete.
func (j *Job) Execute() {
	var err error

	if requestID := j.continuationToken; requestID == "" {
		if requestID, err := j.triggerBuild(); err == nil {
			f := func(b build) bool { return true } // just need a build to report back status
			if b, err := j.pollForResult(requestID, f); err == nil {
				j.reportSuccess(requestID, string(b.ID), true)
			}
		}
	} else {
		f := func(b build) bool { return contains(travisDoneTermSet, b.State) } // wait for travis build to complete
		if b, err := j.pollForResult(requestID, f); err == nil {
			if contains(travisSuccessTermSet, b.State) {
				err = j.reportSuccess(requestID, string(b.ID), false)
			} else {
				err = j.reportFailure(string(b.ID))
			}
		}
	}
	if err != nil {
		log.Fatal().Msgf("JOB - TRAVIS: %s", err.Error())
	}
}

func contains(set map[string]struct{}, item string) bool {
	_, ok := set[item]
	return ok
}
