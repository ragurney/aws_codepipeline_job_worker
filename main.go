package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ragurney/aws_codepipeline_job_worker/pkg/actions/travis"
)

func main() {
	zerolog.TimeFieldFormat = ""

	log.Debug().Msg("Starting worker service...")
	travis.NewDispatcher().Initialize()

	select {} // block to keep polling
}
