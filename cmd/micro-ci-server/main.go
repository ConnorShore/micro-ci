package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConnorShore/micro-ci/internal/server/api"
	"github.com/ConnorShore/micro-ci/internal/server/scheduler"
	"github.com/ConnorShore/micro-ci/internal/server/webhook"
	"github.com/ConnorShore/micro-ci/internal/server/webhook/event"
)

func main() {

	var jobQ scheduler.JobQueue = scheduler.NewInMemoryJobQueue(100)

	var githubWebhookHandler webhook.WebhookHandler = webhook.NewGithubWebhookHandler()
	githubWebhookHandler.AddEventHandler(webhook.EventPush, event.NewGithubPushEventHandler())

	webhookServer := api.NewWebhookServer(":4000", jobQ)
	webhookServer.AddWebhook("/webhook/github", githubWebhookHandler)

	mciServer, err := api.NewMicroCIServer(jobQ)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// start the mci server
	go func() {
		log.Fatal(mciServer.Start())
	}()

	// start the webhook server
	go func() {
		log.Fatal(webhookServer.Start())
	}()

	// handle safe shutdown
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)
	<-osSignals // block until interuption is called

	fmt.Println("MicroCI Server is shutting down...")

	// any shutdown/post-shutdown logic
	log.Fatal(webhookServer.Shutdown())

	fmt.Println("MicroCI Serveris shutdown.")
}
