package webhook

import (
	"io"
	"net/http"

	"github.com/ConnorShore/micro-ci/internal/common"
)

type WebhookEventType string

const (
	EventPush WebhookEventType = "push"
)

type WebhookEvent struct {
	RepoUrl string           `json:"repo_url"`
	Type    WebhookEventType `json:"_"`
}

// An event handler for a webhook
// i.e. code push, pr request, etc
type WebhookEventHandler interface {
	ParseEvent(r io.ReadCloser) (*common.BootstrapJob, error)
}

// A webhook handler for a platform
// i.e. github, gitlab, etc
type WebhookHandler interface {
	// Parse request and create a bootstrap job
	Parse(r *http.Request) (*common.BootstrapJob, error)
	AddEventHandler(t WebhookEventType, h WebhookEventHandler)
}
