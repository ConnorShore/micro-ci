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
	// TODO: See if the incoming request and headers match valid event handler
	// if so call ParseEvent() for that handlers
	// otherwise return nil

	eventType := WebhookEventType(r.Header.Get(GitHubHeader))
	if eventType == "" {
		return nil, fmt.Errorf("event type header X-GITHUB-EVENT not present")
	}

	handler, ok := gh.eventHandlers[eventType]
	if !ok {
		return nil, fmt.Errorf("no handler present for event type: %s", eventType)
	}

	return handler.ParseEvent(r.Body)
	// jobs := []*pipeline.Job{}
	// for _, j := range p.Jobs {
	// 	j.RunId = uuid.NewString()
	// 	j.Variables = common.MergeVariables(p.Variables, j.Variables)
	// 	jobs = append(jobs, &j)
	// }

	// return jobs, nil
}

func (gh *GithubWebhookHandler) AddEventHandler(t WebhookEventType, h WebhookEventHandler) {
	gh.eventHandlers[t] = h
}
