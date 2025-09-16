package scheduler

import (
	"github.com/ConnorShore/micro-ci/internal/common"
)

type JobQueue interface {
	Enqueue(j common.Job)
	Dequeue() common.Job
}

// temporary in-memory job queue
type InMemJobQueue struct {
	jobCh chan (common.Job)
}

func NewInMemoryJobQueue(capacity int) *InMemJobQueue {
	return &InMemJobQueue{
		jobCh: make(chan common.Job, capacity),
	}
}

func (q *InMemJobQueue) Enqueue(j common.Job) {
	q.jobCh <- j
}

func (q *InMemJobQueue) Dequeue() common.Job {
	select {
	case j := <-q.jobCh:
		return j
	default:
		return nil
	}
}
