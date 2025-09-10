package client

import (
	"context"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

type MicroCIClient interface {
	Register(ctx context.Context, machineId string) error
	Unregister(ctx context.Context, machineId string) error
	FetchJob(ctx context.Context, machineId string) (*pipeline.Job, error)
	UpdateJobStatus(ctx context.Context, jobRunId string, status common.JobStatus) error
	StreamLogs(ctx context.Context, jobRunId, line string) error
	Close() error
}
