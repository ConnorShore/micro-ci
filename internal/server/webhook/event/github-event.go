package event

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/server/webhook"
)

type githhubPushEvent struct {
	webhook.WebhookEvent
	Branch string `json:"branch"`
}

type GithubPushEventHandler struct{}

func NewGithubPushEventHandler() *GithubPushEventHandler {
	return &GithubPushEventHandler{}
}

func (h *GithubPushEventHandler) ParseEvent(r io.ReadCloser) (*common.BootstrapJob, error) {
	defer r.Close()

	var evt githhubPushEvent
	if err := json.NewDecoder(r).Decode(&evt); err != nil {
		return nil, fmt.Errorf("failed to decode githubPushEvent: %v", err)
	}

	bootstrapJob := &common.BootstrapJob{
		RepoURL: evt.RepoUrl,
		Branch:  evt.Branch,
	}

	return bootstrapJob, nil

	// New approach
	// Create a bootstrap job for the runner
	// This job will tell the runner to clone the repo and then parse the pipelines
	// to get all the jobs to run
	// The runner then returns the jobs back to the server to enqueue them

	// Old thought
	// Clone the repo (or is there a better way / can just get contents of .micro_ci folder)

	// Parse the .micro_ci folder for pipeline files

	// Figure out which pipelines should run based on this event
	// i.e does the pipeline contain push trigger for the specified branch

	// Return all jobs to run (may need to track which job is for which pipeline)
	// in the Job struct
}
