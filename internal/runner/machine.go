package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	DefaultCloneDir   = "working-repo"
	DefaultImage      = "golang:1.21-alpine"
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

	// spin up container and run job
	m.spinUpContainerAndRunSteps(j)

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
func (m *Machine) spinUpContainerAndRunSteps(j common.Job) error {
	switch j.GetType() {
	case common.TypeBootstrap:
		return m.runBootstrapJob(j.(*common.BootstrapJob))
	case common.TypePipeline:
		return m.runPipelineJob(j.(*pipeline.Job))
	default:
		return fmt.Errorf("unknown job type to run on machine. Job type = %v", j.GetType())
	}
}

func (m *Machine) runBootstrapJob(j *common.BootstrapJob) error {
	log.Printf("Running bootstrap job: %+v\n", *j)
	options := DockerContainerOptions{
		Image:      DefaultImage,
		Port:       "8080",
		WorkingDir: m.WorkingDirectory,
	}

	container, err := m.createAndStartDockerContainer(options)
	if err != nil {
		return err
	}

	// Remove clone directory after finishing method
	defer func() {
		log.Printf("Stopping and removing docker container: %s\n", container.ID)
		m.stopAndRemoveDockerContainer(container)
	}()

	// Test command
	log.Printf("Installing git")
	script := "apk update && apk add git"
	ctx := context.Background()
	if err := m.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        pipeline.Script(script),
		EnvironmentId: container.ID,
	}, func(line string) {
		fmt.Printf("[Machine: %v] Execute bootstrap job log: %v\n", m.Name, line)
		m.mciClient.StreamLogs(ctx, j.RunId, line)
	}); err != nil {
		return fmt.Errorf("failed to run step [%v]: %v", j.Name, err)
	}

	// Clone repository for specific sha or branch
	log.Printf("Cloning repo: %s\n", j.RepoURL)
	script = fmt.Sprintf("git clone %s %s", j.RepoURL, DefaultCloneDir)
	ctx = context.Background()
	if err := m.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        pipeline.Script(script),
		EnvironmentId: container.ID,
	}, func(line string) {
		fmt.Printf("[Machine: %v] Execute bootstrap job log: %v\n", m.Name, line)
		m.mciClient.StreamLogs(ctx, j.RunId, line)
	}); err != nil {
		return fmt.Errorf("failed to run step [%v]: %v", j.Name, err)
	}

	// TODO: Verify the triggers for the pipeline match the bootstrap job event
	log.Printf("Extracting pipelineFiles")
	var pipelineFiles []string
	script = fmt.Sprintf("cd %s/%s && ls", DefaultCloneDir, pipeline.PipelineDir)
	ctx = context.Background()
	if err := m.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        pipeline.Script(script),
		EnvironmentId: container.ID,
	}, func(line string) {
		fmt.Printf("[Machine: %v] Execute bootstrap job log: %v\n", m.Name, line)
		m.mciClient.StreamLogs(ctx, j.RunId, line)
		pipelineFiles = append(pipelineFiles, line)
	}); err != nil {
		return fmt.Errorf("failed to run step [%v]: %v", j.Name, err)
	}

	log.Printf("Pipeline files: %+v\nTesting ls\n", pipelineFiles)

	script = "ls"
	ctx = context.Background()
	if err := m.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        pipeline.Script(script),
		EnvironmentId: container.ID,
	}, func(line string) {
		fmt.Printf("[Machine: %v] Execute bootstrap job log: %v\n", m.Name, line)
	}); err != nil {
		return fmt.Errorf("failed to run step [%v]: %v", j.Name, err)
	}

	// Parse the pipeline for jobs
	log.Println("Parsing pipelines for jobs")

	// Get content of bytes in the pipeline files
	var pipelineData [][]byte
	for _, f := range pipelineFiles {
		script = fmt.Sprintf("cat %s/%s/%s", DefaultCloneDir, pipeline.PipelineDir, f)
		ctx = context.Background()

		var data []byte
		if err := m.executor.Execute(executor.ExecutorOpts{
			Ctx:           ctx,
			Script:        pipeline.Script(script),
			EnvironmentId: container.ID,
		}, func(line string) {
			data = append(data, []byte(line)...)
			data = append(data, '\n')
		}); err != nil {
			return fmt.Errorf("failed to run step [%v]: %v", j.Name, err)
		}

		pipelineData = append(pipelineData, data)
	}

	// Parse pipeline files to pipeline objects
	log.Printf("Adding %v vals to wait group", len(pipelineData))

	var wg sync.WaitGroup
	jobCh := make(chan pipeline.Job)

	for _, d := range pipelineData {
		wg.Add(1)
		go func(data []byte, jobCh chan<- pipeline.Job) {
			defer wg.Done()

			p, err := pipeline.ParsePipeline(data)
			if err != nil {
				fmt.Printf("error parsing pipeline: %v\n", err)
				return
			}

			log.Printf("Parsed pipeline successfully: %+v\n", p.Name)
			for _, j := range p.Jobs {
				jobCh <- j
			}

			log.Printf("End of method")
		}(d, jobCh)
	}

	// Wait for waitgroup to finish to close pipelineCh
	go func() {
		wg.Wait()
		close(jobCh)
	}()

	for j := range jobCh {
		log.Printf("Recieved job from channel: %+v\n", j)
		if err := m.mciClient.AddJob(ctx, &j); err != nil {
			log.Printf("Error sending add job request to server %v\n", err)
			return err
		}
	}

	return nil
}

func (m *Machine) runPipelineJob(j *pipeline.Job) error {
	options := DockerContainerOptions{
		Image:      j.Image,
		Port:       "8080",
		WorkingDir: m.WorkingDirectory,
		Env:        j.Variables,
	}

	container, err := m.createAndStartDockerContainer(options)
	if err != nil {
		return err
	}

	log.Println("Starting to run all steps")
	if err := m.runAllSteps(container.ctx, j, container.ID); err != nil {
		return err
	}

	return nil
}

func (m *Machine) createAndStartDockerContainer(opts DockerContainerOptions) (*DockerContainer, error) {
	container, err := NewDockerContainer(m.dockerClient, opts)
	if err != nil {
		return nil, err
	}

	err = container.Start()
	if err != nil {
		return nil, err
	}
	return container, nil
}

func (m *Machine) stopAndRemoveDockerContainer(container *DockerContainer) {
	if err := container.Stop(); err != nil {
		// Log the cleanup error, but don't return it,
		// as the original error from the steps is more important.
		fmt.Fprintf(os.Stderr, "error stopping container on cleanup: %v\n", err)
	}

	if err := container.Remove(); err != nil {
		fmt.Fprintf(os.Stderr, "error removing container on cleanup: %v\n", err)
	}
}

// Runs all the steps for a job in the specified container
func (m *Machine) runAllSteps(ctx context.Context, j *pipeline.Job, id string) error {
	for _, s := range j.Steps {
		fmt.Printf("\n---- Running Step [%v] ----\n", s.Name)
		run, err := canRun(s.Condition)
		if err != nil {
			log.Printf("Step [%v] condition failed to parse. Skipping step.\n", s.Name)
			fmt.Println(strings.Repeat("-", 60))
			continue
		}

		if !run {
			log.Printf("Skipping step [%v].\n", s.Name)
			fmt.Println(strings.Repeat("-", 60))
			continue
		}

		_, err = m.runStep(ctx, s, common.MergeVariables(j.Variables, s.Variables), id, j.RunId)
		if err != nil && !s.ContinueOnError {
			log.Printf("Exiting due to error on step [%v+]\n", s)
			return err
		}
		fmt.Println(strings.Repeat("-", 60))
	}

	return nil
}

// runs an individual step in a job
func (m *Machine) runStep(ctx context.Context, s pipeline.Step, vars common.VariableMap, containerID, jobRunId string) (bool, error) {
	if err := m.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        s.Script,
		Vars:          vars,
		EnvironmentId: containerID,
	}, func(line string) {
		fmt.Printf("[Machine: %v] Execute pipeline job log: %v\n", m.Name, line)
		m.mciClient.StreamLogs(ctx, jobRunId, line)
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
