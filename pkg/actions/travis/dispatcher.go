package travis

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/ragurney/aws_codepipeline_job_worker/pkg/services/poller"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Dispatcher dispatcher type for a Travis action
type Dispatcher struct {
	actionTypeId *codepipeline.ActionTypeId
}

// NewDispatcher initializes dispatcher for Travis action
func NewDispatcher() *Dispatcher {
	zerolog.TimeFieldFormat = ""

	d := Dispatcher{
		actionTypeId: &codepipeline.ActionTypeId{
			Category: aws.String(codepipeline.ActionCategoryTest),
			Owner:    aws.String(codepipeline.ActionOwnerCustom),
			Provider: aws.String("Travis"),
			Version:  aws.String("1"),
		},
	}
	return &d
}

// Initialize starts poller for Travis actions
func (d *Dispatcher) Initialize() {
	poller.New(d).Start()
}

// ActionTypeId returns the ActionTypeId, which should match this action's configuration (see travis.json)
func (d *Dispatcher) ActionTypeId() *codepipeline.ActionTypeId {
	return d.actionTypeId
}

// Dispatch kicks off a job for the Travis action
func (d *Dispatcher) Dispatch(j *codepipeline.Job) {
	log.Debug().Msgf(
		"DISPATCHER: Kicking off job for Travis action in the '%s' pipeline.",
		aws.StringValue(j.Data.PipelineContext.PipelineName),
	)

	go NewJob(j).Execute()
}
