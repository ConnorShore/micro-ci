package runner

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"time"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

const (
	DefaultWorkspace string = "/workspace"
)

// Options for creating a new Docker container for running the job
type DockerContainerOptions struct {
	Image      string
	Name       string
	Port       string
	WorkingDir string
	Env        common.VariableMap
}

// Represents a Docker container that is created to run the job
type DockerContainer struct {
	ID         string
	Name       string
	Port       string
	WorkingDir string
	ctx        context.Context
	client     *client.Client
}

func NewDockerContainer(client *client.Client, opts DockerContainerOptions) (*DockerContainer, error) {
	ctx := context.Background()
	if err := pullImage(ctx, client, opts); err != nil {
		return nil, err
	}

	return createContainer(ctx, client, opts)
}

func (c *DockerContainer) Start() error {
	fmt.Println("Starting Docker container with ID: ", c.ID)
	if err := c.client.ContainerStart(c.ctx, c.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start docker container: %v", err)
	}

	if err := c.waitForDockerContainerInitialization(); err != nil {
		return err
	}

	return nil
}

func (c *DockerContainer) Stop() error {
	fmt.Println("Stopping Docker container with ID: ", c.ID)
	if err := c.client.ContainerStop(c.ctx, c.ID, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop Docker container: %w", err)
	}
	fmt.Println("Successfully stopped Docker container with ID: ", c.ID)
	return nil
}

func (c *DockerContainer) Remove() error {
	fmt.Println("Removing Docker container with ID: ", r.Container.ID)
	if err := c.client.ContainerRemove(c.ctx, c.ID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove Docker container: %w", err)
	}
	fmt.Println("Successfully removed Docker container with ID: ", c.ID)

	return nil
}

func (c *DockerContainer) waitForDockerContainerInitialization() error {
	deadlineCtx, cancel := context.WithDeadline(c.ctx, time.Now().Add(30*time.Second))
	defer cancel()

	for {
		resp, err := c.client.ContainerInspect(deadlineCtx, c.ID)

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

func pullImage(ctx context.Context, client *client.Client, opts DockerContainerOptions) error {
	fmt.Println("Pulling Docker image: ", opts.Image)

	// Check if image exists locally
	args := filters.NewArgs()
	args.Add("reference", opts.Image)
	images, err := client.ImageList(ctx, image.ListOptions{Filters: args})
	if err != nil {
		return fmt.Errorf("failed to list local images: %w", err)
	}

	// If image doesn't exist locally, pull it in
	if len(images) == 0 {
		fmt.Println("Image not found locally. Pulling Docker image:", opts.Image)
		out, err := client.ImagePull(ctx, opts.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull docker image [%v]: %v", opts.Image, err)
		}
		defer out.Close()
		io.Copy(os.Stdout, out)
	} else {
		fmt.Println("Found Docker image locally:", opts.Image)
	}

	return nil
}

func createContainer(ctx context.Context, client *client.Client, opts DockerContainerOptions) (*DockerContainer, error) {
	fmt.Println("Creating Docker container with image: ", opts.Image)

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: opts.Image,
		// Keep STDIN open and run a command that never exits
		OpenStdin:  true,
		Cmd:        []string{"tail", "-f", "/dev/null"},
		Tty:        false,
		WorkingDir: DefaultWorkspace,
		Env:        common.VariablesMapToSlice(opts.Env),
	}, &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:%s", opts.WorkingDir, DefaultWorkspace)},
	}, nil, nil, opts.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker container: %v", err)
	}

	return &DockerContainer{
		ID:         resp.ID,
		Name:       opts.Name,
		Port:       opts.Port,
		WorkingDir: DefaultWorkspace,
	}, nil
}

func createContainerName() string {
	return containerNameBase + "-" + fmt.Sprint(rand.IntN(90000)+10000)
}
