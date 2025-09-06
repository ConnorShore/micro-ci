package main

import (
	"fmt"
	"log"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/internal/runner"
)

func main() {
	testScript := "./cmd/micro-ci/.micro-ci/micro-pipeline-test-docker.yaml"
	pipeline, err := pipeline.ParsePipeline(testScript)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// fmt.Printf("%v+\n", pipeline)

	runner, err := runner.NewDockerRunner()
	if err != nil {
		log.Fatalf("error creating docker runner: %v\n", err)
	}

	err = runner.Run(*pipeline)

	if err != nil {
		log.Fatalf("error running pipeline: %v\n", err)
	}
}
