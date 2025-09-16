package webhook

import (
	"fmt"
	"net/http"

	"github.com/ConnorShore/micro-ci/internal/common"
)

const GitHubHeader = "X-GITHUB-EVENT"

type GithubWebhookHandler struct {
	eventHandlers map[WebhookEventType]WebhookEventHandler
}

func NewGithubWebhookHandler() *GithubWebhookHandler {
	return &GithubWebhookHandler{
		eventHandlers: make(map[WebhookEventType]WebhookEventHandler),
	}
}

func (gh *GithubWebhookHandler) Parse(r *http.Request) (*common.BootstrapJob, error) {
	eventType := WebhookEventType(r.Header.Get(GitHubHeader))
	if eventType == "" {
		return nil, fmt.Errorf("event type header X-GITHUB-EVENT not present")
	}

	handler, ok := gh.eventHandlers[eventType]
	if !ok {
		return nil, fmt.Errorf("no handler present for event type: %s", eventType)
	}

	return handler.ParseEvent(r.Body)
}

func (gh *GithubWebhookHandler) AddEventHandler(t WebhookEventType, h WebhookEventHandler) {
	gh.eventHandlers[t] = h
}
