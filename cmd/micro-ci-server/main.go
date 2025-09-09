package main

import (
	"log"

	"github.com/ConnorShore/micro-ci/internal/server/api"
)

func main() {
	testScript := "./cmd/micro-ci-server/.micro-ci/micro-pipeline-test-docker.yaml"
	mciServer, err := api.NewMicroCIServer(testScript)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Fatal(mciServer.Start())
}
