package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConnorShore/micro-ci/internal/server/api"
)

func main() {
	testScript := "./cmd/micro-ci-server/.micro-ci/micro-pipeline-test-docker.yaml"
	mciServer, err := api.NewMicroCIServer(testScript)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// start the server
	go func() {
		log.Fatal(mciServer.Start())
	}()

	// handle safe shutdown
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)
	<-osSignals // block until interuption is called

	fmt.Println("MicroCI Server is shutting down...")

	// any shutdown/post-shutdown logic

	fmt.Println("MicroCI Serveris shutdown.")
}
