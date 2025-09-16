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
}
