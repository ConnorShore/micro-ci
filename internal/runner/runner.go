package runner

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

type Runner struct {
	executionDir    string
	environmentVars []string
}

func NewRunner() (*Runner, error) {
	return &Runner{}, nil
}

// Runs the passed in pipeline
func (r *Runner) Run(p pipeline.Pipeline) error {
	dir, err := os.MkdirTemp("", "microci-runner-env")
	fmt.Println("Runner dir: ", dir)
	if err != nil {
		return fmt.Errorf("failed to create micro-ci runner environment: %v", err)
	}
	defer os.RemoveAll(dir)

	r.executionDir = dir

	// get environment variables from pipeline
	r.environmentVars = parseVariables(p.Variables)

	fmt.Println("Running pipeline in directory: ", dir)
	// Run each step in the pipeline
	for _, step := range p.Steps {
		_, err := r.RunStep(step)
		if err != nil && !step.ContinueOnError {
			fmt.Printf("Exiting due to error on step [%v+]\n", step)
			return err
		}
	}

	return nil
}

func (r *Runner) RunStep(s pipeline.Step) (bool, error) {
	run, err := canRunStep(s)
	if err != nil {
		fmt.Printf("Step [%v] condition failed to parse. Skipping step.\n", s.Name)
		return false, err
	}

	if !run {
		fmt.Printf("Skipping step [%v].\n", s.Name)
		return false, nil
	}

	// Get the local variables to the step to append to the environment vars for this step
	localVars := parseVariables(s.Variables)

	fmt.Printf("===== Executing Step [%v] =====\n", s.Name)
	err = executeScript(s.Script, r.executionDir, append(r.environmentVars, localVars...))
	if err != nil {
		return false, fmt.Errorf("failed to execute script for step [%v]: %v", s.Name, err)
	}

	return true, nil
}

// Evaluates Step's condition to determine if step should be ran
func canRunStep(s pipeline.Step) (bool, error) {
	// TODO: Need to create an expression evaluator
	return true, nil
}

// Parse variables from pipeline in a form to pass to the exec.Cmd Env property
func parseVariables(vars map[string]string) []string {
	var ret []string = make([]string, 0)
	for key, val := range vars {
		ret = append(ret, string(key+"="+val))
	}
	return ret
}

// Executes the script for a step
func executeScript(script pipeline.Script, directory string, variables []string) error {
	cmd := exec.Command("sh", "-c", string(script))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = directory
	cmd.Env = variables

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}
