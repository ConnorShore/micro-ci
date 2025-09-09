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
		log.Fatalf("Failed to start grpc client: %v\n", err)
	}

	machine1, err := runner.NewMachine("test-runner-1", client, executor)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(machine1.Run())
}
