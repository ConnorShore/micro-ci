package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

// TODO: Deprecate local runner, Docker runner is the default, and will have sub classes based on architecture
//  	 i.e. Windows, Linux, Mac (x64, ARM, etc)

type LocalRunner struct {
	BaseRunner
	ExecutionDir string
}

func NewLocalRunner() *LocalRunner {
	return &LocalRunner{}
}

// Runs the passed in pipeline
func (r *LocalRunner) Run(p pipeline.Pipeline) error {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "microci-runner-env")
	if err != nil {
		return fmt.Errorf("failed to create micro-ci runner environment: %v", err)
	}
	defer os.RemoveAll(dir)

	r.ExecutionDir = dir

	fmt.Println("Running pipeline in directory: ", dir)
	// Run each step in the pipeline
	if err := r.runAllSteps(ctx, r, p); err != nil {
		panic(err)
	}

	return nil
}

func (r *LocalRunner) RunStep(ctx context.Context, s pipeline.Step) (bool, error) {
	fmt.Printf("===== Executing Step [%v] =====\n", s.Name)
	if err := r.executeScript(s.Script, append(r.EnvironmentVars, parseVariables(s.Variables)...)); err != nil {
		return false, fmt.Errorf("failed to execute script for step [%v]: %v", s.Name, err)
	}

	return true, nil
}

// Executes the script for a step
func (r *LocalRunner) executeScript(script pipeline.Script, variables []string) error {
	cmd := exec.Command("sh", "-c", string(script))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = r.ExecutionDir
	cmd.Env = variables

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}
