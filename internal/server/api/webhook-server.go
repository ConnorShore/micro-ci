package api

import (
	"context"
	"log"
	"net/http"

	"github.com/ConnorShore/micro-ci/internal/server/scheduler"
	"github.com/ConnorShore/micro-ci/internal/server/webhook"
	"github.com/google/uuid"
)

type WebhookServer struct {
	Addr     string
	JobQueue scheduler.JobQueue
	handlers map[string]webhook.WebhookHandler
	server   *http.Server
}

func NewWebhookServer(addr string, jobQueue scheduler.JobQueue) *WebhookServer {
	return &WebhookServer{
		Addr:     addr,
		JobQueue: jobQueue,
		handlers: make(map[string]webhook.WebhookHandler),
		server: &http.Server{
			Addr: addr,
		},
	}
}

// Starts the webhooks erver
func (s *WebhookServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Default path handler"))
	})

	for path := range s.handlers {
		if path[0] != '/' {
			path = "/" + path
		}

		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			s.handleWebhook(w, r, path)
		})
	}
	s.server.Handler = mux

	log.Printf("Starting webhook server at address [%v]\n", s.Addr)
	return s.server.ListenAndServe()
}

// Shutdown the webhook server
func (s *WebhookServer) Shutdown() error {
	return s.server.Shutdown(context.Background())
}

// Handler method for webhooks that
// 1. handles the webhook request
// 2. parses pipeline with appropriate webhook handler
// 3. adds the parsed jobs to the jobQ
func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request, path string) {
	log.Printf("Handling webhook at path: %s\n", path)
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	handler, ok := s.handlers[path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	bootstrapJob, err := handler.Parse(r)
	if err != nil {
		log.Printf("Failed to handle webhook for path %v: %v\n", path, err)
		http.Error(w, "could not handle webhook", http.StatusBadRequest)
		return
	}

	bootstrapJob.RunId = uuid.NewString()
	bootstrapJob.Name = "Bootstrap Job " + bootstrapJob.RunId
	s.JobQueue.Enqueue(bootstrapJob)

	log.Printf("Successfully handled webhook at path %s\n", path)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("successfully handled webhook and jobs enqueued"))
}

func (s *WebhookServer) AddWebhook(path string, h webhook.WebhookHandler) {
	s.handlers[path] = h
}
