package runner

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	mciClient "github.com/ConnorShore/micro-ci/internal/runner/client"
	"github.com/ConnorShore/micro-ci/internal/runner/executor"
	dockerClient "github.com/docker/docker/client"
	"github.com/google/uuid"
)

type Machine struct {
	ID           string
	Name         string
	State        common.MachineState
	executor     executor.Executor
	mciClient    mciClient.MicroCIClient
	dockerClient *dockerClient.Client
	shutdown     chan struct{} // shutdown signal
}

func NewMachine(name string, mciClient mciClient.MicroCIClient, executor executor.Executor) (*Machine, error) {
	log.Println("Creating Docker client...")
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to start docker client: %v", err)
	}

	return &Machine{
		ID:           uuid.NewString(),
		Name:         name,
		State:        common.StateOffline,
		mciClient:    mciClient,
		dockerClient: cli,
		executor:     executor,
		shutdown:     make(chan struct{}),
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

	// Spawns container and runs each step in the job
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.mciClient.UpdateJobStatus(ctx, j.Id, common.StatusRunning)
	if err != nil {
		return fmt.Errorf("failed to update job status to %v: %v", common.StatusRunning, err)
	}

	// Run the job
	log.Printf("Running job: %+v...\n", j)

	// Update job status based on completion
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

func (m *Machine) Shutdown() {
	close(m.shutdown)
}

// TODO: Implement run job stuff once server/runner communication is working

// func (r *DockerRunner) Run(p pipeline.Pipeline) error {
// 	// Implementation for running the pipeline in a Docker container
// 	dir, err := os.MkdirTemp("", "microci-runner-env")
// 	if err != nil {
// 		return fmt.Errorf("failed to create micro-ci runner environment: %v", err)
// 	}
// 	defer os.RemoveAll(dir)

// 	absDir, err := filepath.Abs(dir)
// 	if err != nil {
// 		return fmt.Errorf("failed to create absolute path for dir: %v", dir)
// 	}

// 	r.WorkingDirectory = absDir

// 	_, err = r.runAllJobs(r, p)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (r *DockerRunner) RunJob(j pipeline.Job, variables common.VariableMap) (bool, error) {
// 	// base context to use throughout job
// 	ctx := context.Background()

// 	options := DockerContainerOptions{
// 		Image:      j.Image,
// 		Name:       createContainerName(),
// 		Port:       "8080",
// 		WorkingDir: r.WorkingDirectory,
// 		Env:        variables,
// 	}

// 	container, err := r.createAndStartDockerContainer(ctx, options)
// 	if err != nil {
// 		return false, err
// 	}
// 	defer func() {
// 		if err := r.stopAndRemoveDockerContainer(ctx); err != nil {
// 			// Log the cleanup error, but don't return it,
// 			// as the original error from the steps is more important.
// 			log.Fprintf(os.Stderr, "error during container cleanup: %v\n", err)
// 		}
// 	}()

// 	r.Container = container

// 	// Wait for container to be ready before running steps
// 	if err := r.waitForDockerContainerInitialization(ctx); err != nil {
// 		return false, err
// 	}

// 	log.Println("Starting to run all steps")
// 	if err := r.runAllSteps(ctx, r, j, variables); err != nil {
// 		return false, err
// 	}

// 	return true, nil
// }

// func (r *DockerRunner) RunStep(ctx context.Context, s pipeline.Step, variables common.VariableMap) (bool, error) {
// 	stepVariables := mergeVariables(variables, s.Variables)
// 	if err := r.executeScript(ctx, s, stepVariables); err != nil {
// 		return false, fmt.Errorf("failed to run step [%v]: %v", s.Name, err)
// 	}

// 	return true, nil
// }

// func (br *BaseRunner) runAllJobs(r Runner, p pipeline.Pipeline) (bool, error) {
// 	pipelineVars := mergeVariables(variablesSliceToMap(os.Environ()), p.Variables)

// 	for _, j := range p.Jobs {
// 		log.Printf("\n\n======= Running Job [%v] ======\n\n", j.Name)
// 		run, err := canRun(j.Condition)
// 		if err != nil {
// 			log.Printf("Job [%v] condition failed to parse. Skipping job.\n", j.Name)
// 			log.Println(strings.Repeat("=", 60))
// 			continue
// 		}

// 		if !run {
// 			log.Printf("Skipping job [%v].\n", j.Name)
// 			log.Println(strings.Repeat("=", 60))
// 			continue
// 		}

// 		_, err = r.RunJob(j, pipelineVars)
// 		if err != nil {
// 			return false, fmt.Errorf("job [%v] failed to complete: %v", j.Name, err)
// 		}
// 		log.Printf("\n%v\n", strings.Repeat("=", 60))
// 	}

// 	return true, nil
// }

// func (br *BaseRunner) runAllSteps(ctx context.Context, r Runner, j pipeline.Job, variables common.VariableMap) error {
// 	// Add job variables
// 	jobVars := mergeVariables(variables, j.Variables)

// 	for _, s := range j.Steps {
// 		log.Printf("\n---- Running Step [%v] ----\n", s.Name)
// 		run, err := canRun(s.Condition)
// 		if err != nil {
// 			log.Printf("Step [%v] condition failed to parse. Skipping step.\n", s.Name)
// 			log.Println(strings.Repeat("-", 60))
// 			continue
// 		}

// 		if !run {
// 			log.Printf("Skipping step [%v].\n", s.Name)
// 			log.Println(strings.Repeat("-", 60))
// 			continue
// 		}

// 		_, err = r.RunStep(ctx, s, jobVars)
// 		if err != nil && !s.ContinueOnError {
// 			log.Printf("Exiting due to error on step [%v+]\n", s)
// 			return err
// 		}
// 		log.Println(strings.Repeat("-", 60))
// 	}

// 	return nil
// }

// // Evaluates Job or Step's condition to determine if step should be ran
// func canRun(condition string) (bool, error) {
// 	// TODO: Need to create an expression evaluator and move this there
// 	return true, nil
// }
