package main

import (
	"fmt"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/internal/runner"
)

func main() {
	testScript := "./cmd/micro-ci/.micro-ci/micro-pipeline-test.yaml"
	pipeline, err := pipeline.ParsePipeline(testScript)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// fmt.Printf("%v+\n", pipeline)

	runner, err := runner.NewRunner()
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	err = runner.Run(*pipeline)
	if err != nil {
		fmt.Println("Error Running Pipeline: ", err)
		return
	}
}
