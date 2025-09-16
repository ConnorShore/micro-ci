package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	mciClient "github.com/ConnorShore/micro-ci/internal/runner/client"
	"github.com/ConnorShore/micro-ci/internal/runner/executor"
	dockerClient "github.com/docker/docker/client"
)

const (
	DefaultCloneDir = "working-repo"
	DefaultImage    = "golang:1.21-alpine"
)

// TODO: Implement this. Should have the run job logic (for both bootstrap and pipeline jobs) from machine.go
// That way machine.go is only responsible for polling for jobs (or adding jobs) to the server
type Runner interface {
	Run(j common.Job) error
}

type BaseRunner struct {
	ID           string
	WorkingDir   string
	mciClient    mciClient.MicroCIClient
	executor     executor.Executor
	dockerClient *dockerClient.Client
}

func (r *BaseRunner) createAndStartDockerContainer(opts DockerContainerOptions) (*DockerContainer, error) {
	container, err := NewDockerContainer(r.dockerClient, opts)
	if err != nil {
		return nil, err
	}

	err = container.Start()
	if err != nil {
		return nil, err
	}
	return container, nil
}

func (r *BaseRunner) stopAndRemoveDockerContainer(container *DockerContainer) {
	if err := container.Stop(); err != nil {
		// Log the cleanup error, but don't return it,
		// as the original error from the steps is more important.
		fmt.Fprintf(os.Stderr, "error stopping container on cleanup: %v\n", err)
	}

	if err := container.Remove(); err != nil {
		fmt.Fprintf(os.Stderr, "error removing container on cleanup: %v\n", err)
	}
}

func (r *BaseRunner) executeCommand(script, containerId, runId string) error {
	ctx := context.Background()
	if err := r.executeCommandWithOut(ctx, script, containerId, func(line string) {
		fmt.Printf("[Runner: %v] Execute bootstrap job log: %v\n", r.ID, line)
		r.mciClient.StreamLogs(ctx, runId, line)
	}); err != nil {
		return err
	}

	return nil
}

func (r *BaseRunner) executeCommandWithOut(ctx context.Context, script, containerId string, onStdOut func(line string)) error {
	err := r.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        pipeline.Script(script),
		EnvironmentId: containerId,
	}, onStdOut)

	return err
}
