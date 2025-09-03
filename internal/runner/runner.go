package runner

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

type Runner struct{}

func NewRunner() (*Runner, error) {
	return &Runner{}, nil
}

// Runs the passed in pipeline
func (r *Runner) Run(p pipeline.Pipeline) error {

	// get environment variables from pipeline
	envVars := parseVariables(p.Variables)

	// Run each step in the pipeline
	for _, step := range p.Steps {
		run, err := canRunStep(step)
		if err != nil {
			fmt.Printf("Step [%v] condition failed to parse. Skipping step.\n", step.Name)
			continue
		}

		if !run {
			fmt.Printf("Skipping step [%v].\n", step.Name)
			continue
		}

		localVars := parseVariables(step.Variables)

		fmt.Printf("===== Executing Step [%v] =====\n", step.Name)
		err = executeScript(step.Script, append(envVars, localVars...))
		if err != nil {
			return err
		}
	}

	return nil
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
func executeScript(script pipeline.Script, variables []string) error {
	cmd := exec.Command("sh", "-c", string(script))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = variables

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}
