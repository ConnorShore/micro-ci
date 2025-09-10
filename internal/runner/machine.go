package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	mciClient "github.com/ConnorShore/micro-ci/internal/runner/client"
	"github.com/ConnorShore/micro-ci/internal/runner/executor"
	dockerClient "github.com/docker/docker/client"
	"github.com/google/uuid"
)

const (
	DefaultWorkingDir = "microci-runner-env"
)

type Machine struct {
	ID               string
	Name             string
	WorkingDirectory string
	State            common.MachineState
	executor         executor.Executor
	mciClient        mciClient.MicroCIClient
	dockerClient     *dockerClient.Client
	shutdown         chan struct{} // shutdown signal
}

func NewMachine(name string, mciClient mciClient.MicroCIClient, executor executor.Executor) (*Machine, error) {
	log.Println("Creating Docker client...")
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to start docker client: %v", err)
	}

	dir, err := os.MkdirTemp("", DefaultWorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create micro-ci runner environment: %v", err)
	}
	defer os.RemoveAll(dir)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to create absolute path for dir: %v", dir)
	}

	return &Machine{
		ID:               uuid.NewString(),
		Name:             name,
		State:            common.StateOffline,
		WorkingDirectory: absDir,
		mciClient:        mciClient,
		dockerClient:     cli,
		executor:         executor,
		shutdown:         make(chan struct{}),
	}, nil
}

func (m *Machine) Run() error {
	parentCtx := context.Background()
	defer func(ctx context.Context) error {
		err := m.mciClient.Unregister(ctx, m.ID)
		if err != nil {
			// TODO: Need to probably do better handling if fail to unregister, just log for now
			log.Printf("Failed to unregister machine [%v] from server\n. Proceeding to close client connection...", m.ID)
		}

		err = m.mciClient.Close()
		return err
	}(parentCtx)

	m.State = common.StateIdle

	// Register machine with the server
	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	err := m.mciClient.Register(ctx, m.ID)
	if err != nil {
		return err
	}
	log.Println("Successfully registered machine with server")

	pollTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-m.shutdown:
			m.State = common.StateOffline
			log.Printf("Machine [%v] has been shutdown.\n", m.Name)
			return nil
		case <-pollTicker.C:
			if m.State == common.StateIdle {
				m.pollForJobs()
			}
		}
	}
}

func (m *Machine) Shutdown() {
	close(m.shutdown)
}

func (m *Machine) pollForJobs() {
	log.Printf("Machine [%v] polling for job...\n", m.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ask the server to see if any jobs are available
	job, err := m.mciClient.FetchJob(ctx, m.ID)
	if err != nil {
		log.Printf("Failed to fetch job with error: %v\n", err)
		return
	}

	if job == nil {
		log.Println("No jobs found. Will continue polling...")
		return
	}

	log.Printf("Machine [%v] found job [%v]...\n", m.Name, job.Name)

	// Job was found, run it
	if err := m.runJob(job); err != nil {
		log.Printf("Failed to execute job [%v]: %v\n", job.Name, err)
	}
}

func (m *Machine) runJob(j *pipeline.Job) error {
	m.State = common.StateBusy
	defer func() {
		m.State = common.StateIdle
	}()

	// Set job to running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.mciClient.UpdateJobStatus(ctx, j.Id, common.StatusRunning)
	if err != nil {
		return fmt.Errorf("failed to update job status to %v: %v", common.StatusRunning, err)
	}

	// spin up container and run job
	m.spinUpContainerAndRunSteps(j)

	// Update job status based on completion
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var status common.JobStatus
	success := true
	if success {
		err = m.mciClient.UpdateJobStatus(ctx, j.Id, common.StatusSuccess)
		status = common.StatusSuccess
	} else {
		err = m.mciClient.UpdateJobStatus(ctx, j.Id, common.StatusFailed)
		status = common.StatusFailed
	}
	if err != nil {
		return fmt.Errorf("failed to update job status to %v: %v", status, err)
	}

	return nil
}

// spins up a container and runs the job's steps. will stop/remove container once finished
func (m *Machine) spinUpContainerAndRunSteps(j *pipeline.Job) error {
	options := DockerContainerOptions{
		Image:      j.Image,
		Port:       "8080",
		WorkingDir: m.WorkingDirectory,
		Env:        j.Variables,
	}

	container, err := NewDockerContainer(m.dockerClient, options)
	if err != nil {
		return err
	}

	container.Start()
	defer func() {
		if err := container.Stop(); err != nil {
			// Log the cleanup error, but don't return it,
			// as the original error from the steps is more important.
			fmt.Fprintf(os.Stderr, "error stopping container on cleanup: %v\n", err)
		}

		if err := container.Remove(); err != nil {
			fmt.Fprintf(os.Stderr, "error removing container on cleanup: %v\n", err)
		}
	}()

	log.Println("Starting to run all steps")
	if err := m.runAllSteps(container.ctx, j, container.ID); err != nil {
		return err
	}

	return nil
}

// Runs all the steps for a job in the specified container
func (m *Machine) runAllSteps(ctx context.Context, j *pipeline.Job, id string) error {
	for _, s := range j.Steps {
		log.Printf("\n---- Running Step [%v] ----\n", s.Name)
		run, err := canRun(s.Condition)
		if err != nil {
			log.Printf("Step [%v] condition failed to parse. Skipping step.\n", s.Name)
			log.Println(strings.Repeat("-", 60))
			continue
		}

		if !run {
			log.Printf("Skipping step [%v].\n", s.Name)
			log.Println(strings.Repeat("-", 60))
			continue
		}

		_, err = m.runStep(ctx, s, common.MergeVariables(j.Variables, s.Variables), id)
		if err != nil && !s.ContinueOnError {
			log.Printf("Exiting due to error on step [%v+]\n", s)
			return err
		}
		log.Println(strings.Repeat("-", 60))
	}

	return nil
}

// runs an individual step in a job
func (m *Machine) runStep(ctx context.Context, s pipeline.Step, vars common.VariableMap, id string) (bool, error) {
	if err := m.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        s.Script,
		Vars:          vars,
		EnvironmentId: id,
	}, func(line string) {
		fmt.Println("Execute job log: ", line)
	}); err != nil {
		return false, fmt.Errorf("failed to run step [%v]: %v", s.Name, err)
	}
	return true, nil
}

// Evaluates Job or Step's condition to determine if step should be ran
func canRun(condition string) (bool, error) {
	// TODO: Need to create an expression evaluator and move this there
	return true, nil
}
