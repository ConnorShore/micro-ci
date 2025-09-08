package main

import (
	"log"

	"github.com/ConnorShore/micro-ci/internal/runner"
	"github.com/ConnorShore/micro-ci/internal/runner/client"
	"github.com/ConnorShore/micro-ci/internal/runner/executor"
)

func main() {
	executor := executor.NewDockerShellExecutor()

	client, err := client.NewGrpcClient("localhost:3001")
	if err != nil {
		log.Fatal("Failed to start grpc client: %v\n", err)
	}

	machine1, err := runner.NewMachine("test-runner-1", client, executor)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(machine1.Run())

	// testScript := "./cmd/micro-ci/.micro-ci/micro-pipeline-test-docker.yaml"
	// pipeline, err := pipeline.ParsePipeline(testScript)
	// if err != nil {
	// 	fmt.Println("Error: ", err)
	// 	return
	// }

	// // fmt.Printf("%v+\n", pipeline)

	// runner, err := runner.NewDockerRunner()
	// if err != nil {
	// 	log.Fatalf("error creating docker runner: %v\n", err)
	// }

	// err = runner.Run(*pipeline)

	// if err != nil {
	// 	log.Fatalf("error running pipeline: %v\n", err)
	// }
}
