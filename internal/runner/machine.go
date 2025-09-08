package runner

import (
	"fmt"
	"time"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/internal/runner/executor"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

type MachineState string
type JobStatus string

const (
	StateOffline MachineState = "offline"
	StateIdle    MachineState = "idle"
	StateBusy    MachineState = "busy"

	StatePending JobStatus = "pending"
	StateRunning JobStatus = "running"
	StateFailed  JobStatus = "failed"
	StateSuccess JobStatus = "success"
)

type Machine struct {
	ID       string
	Name     string
	Address  string
	State    MachineState
	client   *client.Client
	executor *executor.Executor
	shutdown chan struct{} // shutdown signal
}

func NewMachine(name, address string, executor *executor.Executor) (*Machine, error) {
	fmt.Println("Creating Docker client...")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to start docker client: %v", err)
	}

	return &Machine{
		ID:       uuid.New().String(),
		Name:     name,
		Address:  address,
		State:    StateOffline,
		client:   cli,
		executor: executor,
	}, nil
}

func (m *Machine) Run() error {
	m.State = StateIdle

	// TODO: Register machine with server
	// m.RunnerID = {responseID}

	pollTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-m.shutdown:
			m.State = StateOffline
			fmt.Printf("Machine [%v] has been shutdown.\n", m.Name)
			return nil
		case <-pollTicker.C:
			if m.State == StateIdle {
				m.pollForJobs()
			}
		}
	}
}

func (m *Machine) pollForJobs() {
	// Ask the server to see if any jobs are available
}

func (m *Machine) runJob(j pipeline.Job) {
	// Spawns container and runs each step in the job
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
// 			fmt.Fprintf(os.Stderr, "error during container cleanup: %v\n", err)
// 		}
// 	}()

// 	r.Container = container

// 	// Wait for container to be ready before running steps
// 	if err := r.waitForDockerContainerInitialization(ctx); err != nil {
// 		return false, err
// 	}

// 	fmt.Println("Starting to run all steps")
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
// 		fmt.Printf("\n\n======= Running Job [%v] ======\n\n", j.Name)
// 		run, err := canRun(j.Condition)
// 		if err != nil {
// 			fmt.Printf("Job [%v] condition failed to parse. Skipping job.\n", j.Name)
// 			fmt.Println(strings.Repeat("=", 60))
// 			continue
// 		}

// 		if !run {
// 			fmt.Printf("Skipping job [%v].\n", j.Name)
// 			fmt.Println(strings.Repeat("=", 60))
// 			continue
// 		}

// 		_, err = r.RunJob(j, pipelineVars)
// 		if err != nil {
// 			return false, fmt.Errorf("job [%v] failed to complete: %v", j.Name, err)
// 		}
// 		fmt.Printf("\n%v\n", strings.Repeat("=", 60))
// 	}

// 	return true, nil
// }

// func (br *BaseRunner) runAllSteps(ctx context.Context, r Runner, j pipeline.Job, variables common.VariableMap) error {
// 	// Add job variables
// 	jobVars := mergeVariables(variables, j.Variables)

// 	for _, s := range j.Steps {
// 		fmt.Printf("\n---- Running Step [%v] ----\n", s.Name)
// 		run, err := canRun(s.Condition)
// 		if err != nil {
// 			fmt.Printf("Step [%v] condition failed to parse. Skipping step.\n", s.Name)
// 			fmt.Println(strings.Repeat("-", 60))
// 			continue
// 		}

// 		if !run {
// 			fmt.Printf("Skipping step [%v].\n", s.Name)
// 			fmt.Println(strings.Repeat("-", 60))
// 			continue
// 		}

// 		_, err = r.RunStep(ctx, s, jobVars)
// 		if err != nil && !s.ContinueOnError {
// 			fmt.Printf("Exiting due to error on step [%v+]\n", s)
// 			return err
// 		}
// 		fmt.Println(strings.Repeat("-", 60))
// 	}

// 	return nil
// }

// // Evaluates Job or Step's condition to determine if step should be ran
// func canRun(condition string) (bool, error) {
// 	// TODO: Need to create an expression evaluator and move this there
// 	return true, nil
// }
