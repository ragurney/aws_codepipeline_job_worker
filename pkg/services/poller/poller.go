package poller

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/ragurney/aws_codepipeline_job_worker/pkg/actions"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"time"
)

// Poller configuration for polling AWS CodePipeline for a job
type Poller struct {
	pollInterval int
	jobBatchSize *int64
	queryParam   map[string]*string
	dispatcher   actions.Dispatcher `required:"true"`
}

// New initializes Poller
func New(d actions.Dispatcher, options ...Option) *Poller {
	zerolog.TimeFieldFormat = ""

	p := Poller{
		dispatcher:   d,
		jobBatchSize: aws.Int64(1), // Currently only support 1 job at a time
		pollInterval: 30,
		queryParam:   nil,
	}

	for i := range options {
		options[i](&p)
	}

	return &p
}

// Start kicks off polling of AWS CodePipeline, checking it with a frequency of `pollInterval`
// once a job is found it is passed to the action's dispatcher via `dispatcher.Dispatch()`
func (p *Poller) Start() {
	sess := session.Must(session.NewSession())
	svc := codepipeline.New(sess)
	provider := aws.StringValue(p.dispatcher.ActionTypeId().Provider)

	ticker := time.NewTicker(time.Duration(p.pollInterval) * time.Second)
	go func() {
		for range ticker.C {
			log.Debug().Msgf("POLLER: Checking for a job for %s...", provider)

			jobsOutput, err := svc.PollForJobs(
				&codepipeline.PollForJobsInput{
					ActionTypeId: p.dispatcher.ActionTypeId(),
					MaxBatchSize: p.jobBatchSize,
					QueryParam:   p.queryParam,
				},
			)

			if err != nil {
				log.Error().Msg(err.Error())
				continue
			}

			if len(jobsOutput.Jobs) > 0 {
				log.Debug().Msgf("POLLER: Job found for %s.", provider)

				j := jobsOutput.Jobs[0] // TODO: support multiple jobs
				log.Debug().Msgf("POLLER: Acknowledging job %s", aws.StringValue(j.Id))
				_, err = svc.AcknowledgeJob(&codepipeline.AcknowledgeJobInput{JobId: j.Id, Nonce: j.Nonce})
				if err != nil {
					log.Error().Msg(err.Error())
					continue
				}

				log.Debug().Msgf("POLLER: Passing job %s to dispatcher...", aws.StringValue(j.Id))
				p.dispatcher.Dispatch(j)
			} else {
				log.Debug().Msgf("POLLER: No jobs found for %s. Trying again in %d second(s).", provider, p.pollInterval)
			}
		}
	}()
}
