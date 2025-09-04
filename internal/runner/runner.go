package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

type Runner interface {
	Run(p pipeline.Pipeline) error
	RunStep(ctx context.Context, s pipeline.Step) (bool, error)
}

type BaseRunner struct {
	EnvironmentVars []string
}

func (br *BaseRunner) runAllSteps(ctx context.Context, r Runner, p pipeline.Pipeline) error {
	br.EnvironmentVars = append(os.Environ(), parseVariables(p.Variables)...)

	for _, s := range p.Steps {
		run, err := canRunStep(s)
		if err != nil {
			fmt.Printf("Step [%v] condition failed to parse. Skipping step.\n", s.Name)
			return err
		}

		if !run {
			fmt.Printf("Skipping step [%v].\n", s.Name)
			return nil
		}

		_, err = r.RunStep(ctx, s)
		if err != nil && !s.ContinueOnError {
			fmt.Printf("Exiting due to error on step [%v+]\n", s)
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
