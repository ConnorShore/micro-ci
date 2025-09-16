package runner

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/internal/runner/executor"
	"github.com/google/uuid"
)

type PipelineRunner struct {
	BaseRunner
}

func NewPipelineRunner(m *Machine) *PipelineRunner {
	return &PipelineRunner{
		BaseRunner: BaseRunner{
			ID:           uuid.NewString(),
			WorkingDir:   m.WorkingDirectory,
			executor:     m.executor,
			mciClient:    m.mciClient,
			dockerClient: m.dockerClient,
		},
	}
}

func (r *PipelineRunner) Run(j common.Job) error {
	t, ok := j.(*pipeline.Job)
	if !ok {
		return fmt.Errorf("pipeline runner cannot run job of type: %v", t)
	}

	pJob := j.(*pipeline.Job)
	log.Printf("Runnign pipeline job: %v\n", pJob.Name)

	options := DockerContainerOptions{
		Image:      pJob.Image,
		Port:       "8080",
		WorkingDir: r.WorkingDir,
		Env:        pJob.Variables,
	}

	container, err := r.createAndStartDockerContainer(options)
	if err != nil {
		return err
	}

	// Remove clone directory after finishing method
	defer func() {
		log.Printf("Stopping and removing docker container: %s\n", container.ID)
		r.stopAndRemoveDockerContainer(container)
	}()

	log.Println("Starting to run all steps")
	if err := r.runAllSteps(container.ctx, pJob, container.ID); err != nil {
		return err
	}

	return nil
}

// Runs all the steps for a job in the specified container
func (r *PipelineRunner) runAllSteps(ctx context.Context, j *pipeline.Job, id string) error {
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

		_, err = r.runStep(ctx, s, common.MergeVariables(j.Variables, s.Variables), id, j.RunId)
		if err != nil && !s.ContinueOnError {
			log.Printf("Exiting due to error on step [%v+]\n", s)
			return err
		}
		fmt.Println(strings.Repeat("-", 60))
	}

	return nil
}

// runs an individual step in a job
func (r *PipelineRunner) runStep(ctx context.Context, s pipeline.Step, vars common.VariableMap, containerID, jobRunId string) (bool, error) {
	if err := r.executor.Execute(executor.ExecutorOpts{
		Ctx:           ctx,
		Script:        s.Script,
		Vars:          vars,
		EnvironmentId: containerID,
	}, func(line string) {
		fmt.Printf("[Runner: %v] Execute pipeline job log: %v\n", r.ID, line)
		r.mciClient.StreamLogs(ctx, jobRunId, line)
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
