package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/runner"
	"github.com/ConnorShore/micro-ci/internal/runner/client"
	"github.com/ConnorShore/micro-ci/internal/runner/executor"
)

// TODO:
//		1. Add additional pipelines in test repo to see if multiple pipelines work
//		2. Look into adding unit tests for stuff

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

	// Register job types with machine
	if err := machine1.RegisterJobType(common.TypeBootstrap); err != nil {
		log.Fatal(err)
	}
	if err := machine1.RegisterJobType(common.TypePipeline); err != nil {
		log.Fatal(err)
	}

	// Start the machine
	go func() {
		log.Fatal(machine1.Run())
	}()

	// handle safe shutdown
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)
	<-osSignals // block until interuption is called

	fmt.Printf("MicroCI Runner [%v] is shutting down...\n", machine1.Name)

	machine1.Shutdown()

	fmt.Printf("MicroCI Runner [%v] shutdown.\n", machine1.Name)
}
