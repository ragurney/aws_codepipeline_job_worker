package actions

import (
	"github.com/aws/aws-sdk-go/service/codepipeline"
)

// Dispatcher interface used by `poller.go` to dispatch jobs.
type Dispatcher interface {
	Initialize()
	Dispatch(job *codepipeline.Job)
	ActionTypeId() *codepipeline.ActionTypeId
}
