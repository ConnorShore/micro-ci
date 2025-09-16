package runner

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/google/uuid"
)

type BootstrapRunner struct {
	BaseRunner
}

func NewBootstrapRunner(m *Machine) *BootstrapRunner {
	return &BootstrapRunner{
		BaseRunner: BaseRunner{
			ID:           uuid.NewString(),
			WorkingDir:   m.WorkingDirectory,
			executor:     m.executor,
			mciClient:    m.mciClient,
			dockerClient: m.dockerClient,
		},
	}
}

func (r *BootstrapRunner) Run(j common.Job) error {
	t, ok := j.(*common.BootstrapJob)
	if !ok {
		return fmt.Errorf("pipeline runner cannot run job of type: %v", t)
	}

	bJob := j.(*common.BootstrapJob)
	log.Printf("Running bootstrap job for repo: %s\n", bJob.RepoURL)

	container, err := r.setupContainerForBootstrapJob()
	if err != nil {
		return fmt.Errorf("failed to create and start container for bootstrap job: %v", err)
	}

	// Remove clone directory after finishing method
	defer func() {
		log.Printf("Stopping and removing docker container: %s\n", container.ID)
		r.stopAndRemoveDockerContainer(container)
	}()

	if err := r.prepareRepoInContainer(container, bJob); err != nil {
		return fmt.Errorf("failed to prepare container: %v", err)
	}

	piplineData, err := r.extractPipelineData(container, bJob)
	if err != nil {
		return fmt.Errorf("failed to extract pipeline data: %v", err)
	}

	if err := r.parseAndSubmitPipelineJobs(piplineData); err != nil {
		return fmt.Errorf("failed to parse and submit all pipleine jobs: %v", err)
	}

	return nil
}

// Setup a container to run bootstrap job
func (r *BootstrapRunner) setupContainerForBootstrapJob() (*DockerContainer, error) {
	options := DockerContainerOptions{
		Image:      DefaultImage,
		Port:       "8080",
		WorkingDir: r.WorkingDir,
	}

	return r.createAndStartDockerContainer(options)
}

// Prepares the repo to be cloned in the container
func (r *BootstrapRunner) prepareRepoInContainer(container *DockerContainer, j *common.BootstrapJob) error {
	// Install git in container
	log.Printf("Installing git")

	script := "apk update && apk add git"
	if err := r.executeCommand(script, container.ID, j.RunId); err != nil {
		return fmt.Errorf("failed to initialize git: %v", err)
	}

	// Clone repository for specific sha or branch
	log.Printf("Cloning repo: %s\n", j.RepoURL)

	script = fmt.Sprintf("git clone %s %s", j.RepoURL, DefaultCloneDir)
	if err := r.executeCommand(script, container.ID, j.RunId); err != nil {
		return fmt.Errorf("failed to clone repository: %v", err)
	}

	return nil
}

// Retrieves the pipeline file data from the container
func (r *BootstrapRunner) extractPipelineData(container *DockerContainer, j *common.BootstrapJob) ([][]byte, error) {
	// Extract pipeline file data so we can parse it
	log.Printf("Extracting pipelineFiles")

	var pipelineFiles []string
	script := fmt.Sprintf("cd %s/%s && ls", DefaultCloneDir, pipeline.PipelineDir)
	if err := r.executeCommandWithOut(context.Background(), script, container.ID, func(line string) {
		pipelineFiles = append(pipelineFiles, line)
	}); err != nil {
		return nil, fmt.Errorf("failed to run step [%v]: %v", j.Name, err)
	}

	// Get content of bytes in the pipeline files
	log.Println("Parsing pipeline files")

	var pipelineData [][]byte
	for _, f := range pipelineFiles {
		script = fmt.Sprintf("cat %s/%s/%s", DefaultCloneDir, pipeline.PipelineDir, f)
		var data []byte
		if err := r.executeCommandWithOut(context.Background(), script, container.ID, func(line string) {
			data = append(data, []byte(line)...)
			data = append(data, '\n')
		}); err != nil {
			return nil, fmt.Errorf("failed to run step [%v]: %v", j.Name, err)
		}

		pipelineData = append(pipelineData, data)
	}

	return pipelineData, nil
}

// Parse pipelines and submit jobs to queue on server
func (r *BootstrapRunner) parseAndSubmitPipelineJobs(pipelineData [][]byte) error {
	// Parse pipeline files to pipeline objects
	var wg sync.WaitGroup
	jobCh := make(chan pipeline.Job)

	for _, d := range pipelineData {
		wg.Add(1)
		go func(data []byte, jobCh chan<- pipeline.Job) {
			defer wg.Done()

			// Parse the pipeline
			p, err := pipeline.ParsePipeline(data)
			if err != nil {
				fmt.Printf("error parsing pipeline: %v\n", err)
				return
			}

			// Validate the pipeline
			valid, errors := pipeline.ValidatePipeline(p)
			if !valid {
				fmt.Printf("Pipeline validation errors: %+v\n", errors)
				return
			}

			log.Printf("Parsed and validated pipeline successfully: %+v\n", p.Name)

			// Extract jobs from pipleine
			for _, j := range p.Jobs {
				jobCh <- j
			}
		}(d, jobCh)
	}

	// Wait for waitgroup to finish to close pipelineCh
	go func() {
		wg.Wait()
		close(jobCh)
	}()

	// Add all jobs from the job channel to the server queue
	for j := range jobCh {
		if err := r.mciClient.AddJob(context.Background(), &j); err != nil {
			log.Printf("Error sending add job request to server %v\n", err)
			return err
		}
	}

	return nil
}
