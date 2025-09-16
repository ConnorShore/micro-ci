package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ConnorShore/micro-ci/internal/common"
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
	runners          map[common.JobType]Runner
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
		runners:          make(map[common.JobType]Runner),
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

func (m *Machine) RegisterRunner(t common.JobType) error {
	v, ok := m.runners[t]
	if ok {
		return fmt.Errorf("already registered a runner of type [%v] with machine [%v]: %+v", t, m.Name, v)
	}

	runner, err := NewRunner(t, m)
	if err != nil {
		return err
	}

	m.runners[t] = runner
	log.Printf("Registered runner of type [%v] with machine [%v]\n", t, m.Name)
	return nil
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

	log.Printf("Machine [%v] found job [%v]...\n", m.Name, job.GetName())

	// Job was found, run it
	if err := m.runJob(job); err != nil {
		log.Printf("Failed to execute job [%v]: %v\n", job.GetName(), err)
	}
}

func (m *Machine) runJob(j common.Job) error {
	fmt.Printf("Running job: %v\n", j.GetName())

	m.State = common.StateBusy
	defer func() {
		m.State = common.StateIdle
	}()

	// Set job to running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.mciClient.UpdateJobStatus(ctx, j.GetRunId(), common.StatusRunning)
	if err != nil {
		return fmt.Errorf("failed to update job status to %v: %v", common.StatusRunning, err)
	}

	// Run job with the proper runner
	err = m.runJobWithRunner(j)
	if err != nil {
		return fmt.Errorf("failed to run job [%v] with runner", j.GetName())
	}

	// Update job status based on completion
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var status common.JobStatus
	success := true
	if success {
		err = m.mciClient.UpdateJobStatus(ctx, j.GetRunId(), common.StatusSuccess)
		status = common.StatusSuccess
	} else {
		err = m.mciClient.UpdateJobStatus(ctx, j.GetRunId(), common.StatusFailed)
		status = common.StatusFailed
	}
	if err != nil {
		return fmt.Errorf("failed to update job status to %v: %v", status, err)
	}

	return nil
}

// spins up a container and runs the job's steps. will stop/remove container once finished
func (m *Machine) runJobWithRunner(j common.Job) error {
	log.Printf("Running job [%v] with runner\n", j.GetName())
	runner, ok := m.runners[j.GetType()]
	if !ok {
		return fmt.Errorf("no runner registered for type: %v", j.GetType())
	}

	return runner.Run(j)
}
