package main

import (
	"fmt"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

func main() {
	testScript := "./cmd/micro-ci/.micro-ci/micro-pipeline-test.yaml"
	pipeline, err := pipeline.ParsePipeline(testScript)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	fmt.Printf("%v+\n", pipeline)
}
