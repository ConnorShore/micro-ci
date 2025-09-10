package runner

import (
	"context"
	"fmt"
	"io"
	"log"
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
	DefaultWorkspace  string = "/workspace"
	ContainerNameBase string = "microci-runner"
)

// Options for creating a new Docker container for running the job
type DockerContainerOptions struct {
	Image      string
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
	log.Println("Starting Docker container with ID: ", c.ID)
	if err := c.client.ContainerStart(c.ctx, c.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start docker container: %v", err)
	}

	log.Println("Begin waiting for container initialization")
	if err := c.waitForDockerContainerInitialization(); err != nil {
		return err
	}

	return nil
}

func (c *DockerContainer) Stop() error {
	log.Println("Stopping Docker container with ID: ", c.ID)
	if err := c.client.ContainerStop(c.ctx, c.ID, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop Docker container: %w", err)
	}
	log.Println("Successfully stopped Docker container with ID: ", c.ID)
	return nil
}

func (c *DockerContainer) Remove() error {
	log.Println("Removing Docker container with ID: ", c.ID)
	if err := c.client.ContainerRemove(c.ctx, c.ID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove Docker container: %w", err)
	}
	log.Println("Successfully removed Docker container with ID: ", c.ID)

	return nil
}

func (c *DockerContainer) waitForDockerContainerInitialization() error {
	deadlineCtx, cancel := context.WithDeadline(c.ctx, time.Now().Add(30*time.Second))
	defer cancel()

	for {
		log.Println("Waiting for container...")
		resp, err := c.client.ContainerInspect(deadlineCtx, c.ID)

		if err != nil {
			return fmt.Errorf("error waiting for container initialization: %v", err)
		}

		if resp.State.Running {
			log.Println("Container is running.")
			return nil
		}

		log.Println("Waiting for container to become healthy...")
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
	log.Println("Pulling Docker image: ", opts.Image)

	// Check if image exists locally
	args := filters.NewArgs()
	args.Add("reference", opts.Image)
	images, err := client.ImageList(ctx, image.ListOptions{Filters: args})
	if err != nil {
		return fmt.Errorf("failed to list local images: %w", err)
	}

	// If image doesn't exist locally, pull it in
	if len(images) == 0 {
		log.Println("Image not found locally. Pulling Docker image:", opts.Image)
		out, err := client.ImagePull(ctx, opts.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull docker image [%v]: %v", opts.Image, err)
		}
		defer out.Close()
		io.Copy(os.Stdout, out)
	} else {
		log.Println("Found Docker image locally:", opts.Image)
	}

	return nil
}

func createContainer(ctx context.Context, client *client.Client, opts DockerContainerOptions) (*DockerContainer, error) {
	log.Println("Creating Docker container with image: ", opts.Image)

	containerName := createContainerName()
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
	}, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker container: %v", err)
	}

	return &DockerContainer{
		ID:         resp.ID,
		Name:       containerName,
		Port:       opts.Port,
		WorkingDir: DefaultWorkspace,
		ctx:        ctx,
		client:     client,
	}, nil
}

func createContainerName() string {
	return ContainerNameBase + "-" + fmt.Sprint(rand.IntN(90000)+10000)
}
