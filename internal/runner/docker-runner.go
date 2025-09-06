package runner

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	containerNameBase = "microci-runner"
)

// Options for creating a new Docker container for running the pipeline steps
type DockerContainerOptions struct {
	Image      string
	Name       string
	Port       string
	WorkingDir string
	Env        common.VariableMap
}

// Represents a Docker container that is created to run the pipeline steps
type DockerContainer struct {
	ID   string
	Name string
}

// DockerRunner is responsible for running the pipeline steps inside a Docker container
type DockerRunner struct {
	BaseRunner
	Image     string
	Container *DockerContainer
	client    *client.Client
}

func NewDockerRunner() (*DockerRunner, error) {
	fmt.Println("Creating Docker client...")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to start docker client: %v", err)
	}

	return &DockerRunner{client: cli}, nil
}

func (r *DockerRunner) Run(p pipeline.Pipeline) error {
	// Implementation for running the pipeline in a Docker container
	dir, err := os.MkdirTemp("", "microci-runner-env")
	if err != nil {
		return fmt.Errorf("failed to create micro-ci runner environment: %v", err)
	}
	defer os.RemoveAll(dir)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to create absolute path for dir: %v", dir)
	}

	r.WorkingDirectory = absDir

	_, err = r.runAllJobs(r, p)
	if err != nil {
		return err
	}

	return nil
}

func (r *DockerRunner) RunJob(j pipeline.Job, variables common.VariableMap) (bool, error) {
	// base context to use throughout job
	ctx := context.Background()

	options := DockerContainerOptions{
		Image:      j.Image,
		Name:       createContainerName(),
		Port:       "8080",
		WorkingDir: r.WorkingDirectory,
		Env:        variables,
	}

	container, err := r.createAndStartDockerContainer(ctx, options)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := r.stopAndRemoveDockerContainer(ctx); err != nil {
			// Log the cleanup error, but don't return it,
			// as the original error from the steps is more important.
			fmt.Fprintf(os.Stderr, "error during container cleanup: %v\n", err)
		}
	}()

	r.Container = container

	// Wait for container to be ready before running steps
	if err := r.waitForDockerContainerInitialization(ctx); err != nil {
		return false, err
	}

	fmt.Println("Starting to run all steps")
	if err := r.runAllSteps(ctx, r, j, variables); err != nil {
		return false, err
	}

	return true, nil
}

func (r *DockerRunner) RunStep(ctx context.Context, s pipeline.Step, variables common.VariableMap) (bool, error) {
	stepVariables := mergeVariables(variables, s.Variables)
	if err := r.executeScript(ctx, s, stepVariables); err != nil {
		return false, fmt.Errorf("failed to run step [%v]: %v", s.Name, err)
	}

	return true, nil
}

func (r *DockerRunner) executeScript(ctx context.Context, s pipeline.Step, variables common.VariableMap) error {
	cmd := append([]string{"sh", "-c"}, makeSingleLineScript(s.Script))
	execOpts := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          variablesMapToSlice(variables),
		Cmd:          cmd,
	}

	execCreateResp, err := r.client.ContainerExecCreate(ctx, r.Container.ID, execOpts)
	if err != nil {
		return fmt.Errorf("failed to execute script [%v] in container: %v", s.Script, err)
	}

	execResp, err := r.client.ContainerExecAttach(ctx, execCreateResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("error attaching to exec instance: %v", err)
	}
	defer execResp.Close()

	// Copy output to os.Stdout and os.Stderr
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, execResp.Reader)
	if err != nil {
		return fmt.Errorf("error reading from exec output: %v", err)
	}

	// Inspect exit code to verify successful exit
	inspectResp, err := r.client.ContainerExecInspect(ctx, execCreateResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec instance: %v", err)
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("step exited with non-zero status code: %d", inspectResp.ExitCode)
	}

	fmt.Printf("Step [%s] completed successfully.\n", s.Name)
	return nil
}

func (r *DockerRunner) createAndStartDockerContainer(ctx context.Context, opts DockerContainerOptions) (*DockerContainer, error) {
	fmt.Println("Pulling Docker image: ", opts.Image)
	imageName := opts.Image

	// Check if image exists locally
	args := filters.NewArgs()
	args.Add("reference", imageName)
	images, err := r.client.ImageList(ctx, image.ListOptions{Filters: args})
	if err != nil {
		return nil, fmt.Errorf("failed to list local images: %w", err)
	}

	// If image doesn't exist locally, pull it in
	if len(images) == 0 {
		fmt.Println("Image not found locally. Pulling Docker image:", imageName)
		out, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to pull docker image [%v]: %v", opts.Image, err)
		}
		defer out.Close()
		io.Copy(os.Stdout, out)
	} else {
		fmt.Println("Found Docker image locally:", imageName)
	}

	fmt.Println("Creating Docker container with image: ", imageName)
	resp, err := r.client.ContainerCreate(ctx, &container.Config{
		Image: opts.Image,
		// Keep STDIN open and run a command that never exits
		OpenStdin:  true,
		Cmd:        []string{"tail", "-f", "/dev/null"},
		Tty:        false,
		WorkingDir: "/workspace",
		Env:        variablesMapToSlice(opts.Env),
	}, &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:/workspace", opts.WorkingDir)},
	}, nil, nil, opts.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker container: %v", err)
	}

	fmt.Println("Starting Docker container with ID: ", resp.ID)
	if err := r.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start docker container: %v", err)
	}

	return &DockerContainer{
		ID:   resp.ID,
		Name: opts.Name,
	}, nil
}

func (r *DockerRunner) waitForDockerContainerInitialization(ctx context.Context) error {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(30*time.Second))
	defer cancel()

	for {
		resp, err := r.client.ContainerInspect(deadlineCtx, r.Container.ID)

		if err != nil {
			return fmt.Errorf("error waiting for container initialization: %v", err)
		}

		if resp.State.Running {
			fmt.Println("Container is running.")
			return nil
		}

		fmt.Println("Waiting for container to become healthy...")
		select {
		case <-time.After(1 * time.Second):
			continue
		case <-deadlineCtx.Done():
			// The context was canceled (e.g., timeout), return immediately.
			return deadlineCtx.Err()
		}
	}
}

func (r *DockerRunner) stopAndRemoveDockerContainer(ctx context.Context) error {
	fmt.Println("Stopping Docker container with ID: ", r.Container.ID)
	if err := r.client.ContainerStop(ctx, r.Container.ID, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop Docker container: %w", err)
	}

	fmt.Println("Removing Docker container with ID: ", r.Container.ID)
	if err := r.client.ContainerRemove(ctx, r.Container.ID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove Docker container: %w", err)
	}
	fmt.Println("Successfully stopped and removed Docker container with ID: ", r.Container.ID)
	r.Container = nil

	return nil
}

func makeSingleLineScript(s pipeline.Script) string {
	return strings.Join(strings.Split(strings.TrimSpace(string(s)), "\n"), " && ")
}

func createContainerName() string {
	return containerNameBase + "-" + fmt.Sprint(rand.IntN(90000)+10000)
}
