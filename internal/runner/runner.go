package runner

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

type Runner interface {
	Run(p pipeline.Pipeline) error
	RunJob(j pipeline.Job, variables common.VariableMap) (bool, error)
	RunStep(ctx context.Context, s pipeline.Step, variables common.VariableMap) (bool, error)
}

type BaseRunner struct {
	WorkingDirectory string
}

func (br *BaseRunner) runAllJobs(r Runner, p pipeline.Pipeline) (bool, error) {
	pipelineVars := mergeVariables(variablesSliceToMap(os.Environ()), p.Variables)

	for _, j := range p.Jobs {
		fmt.Printf("\n\n======= Running Job [%v] ======\n\n", j.Name)
		run, err := canRun(j.Condition)
		if err != nil {
			fmt.Printf("Job [%v] condition failed to parse. Skipping job.\n", j.Name)
			fmt.Println(strings.Repeat("=", 60))
			continue
		}

		if !run {
			fmt.Printf("Skipping job [%v].\n", j.Name)
			fmt.Println(strings.Repeat("=", 60))
			continue
		}

		_, err = r.RunJob(j, pipelineVars)
		if err != nil {
			return false, fmt.Errorf("job [%v] failed to complete: %v", j.Name, err)
		}
		fmt.Printf("\n%v\n", strings.Repeat("=", 60))
	}

	return true, nil
}

func (br *BaseRunner) runAllSteps(ctx context.Context, r Runner, j pipeline.Job, variables common.VariableMap) error {
	// Add job variables
	jobVars := mergeVariables(variables, j.Variables)

	for _, s := range j.Steps {
		fmt.Printf("\n---- Running Step [%v] ----\n", s.Name)
		run, err := canRun(s.Condition)
		if err != nil {
			fmt.Printf("Step [%v] condition failed to parse. Skipping step.\n", s.Name)
			fmt.Println(strings.Repeat("-", 60))
			continue
		}

		if !run {
			fmt.Printf("Skipping step [%v].\n", s.Name)
			fmt.Println(strings.Repeat("-", 60))
			continue
		}

		_, err = r.RunStep(ctx, s, jobVars)
		if err != nil && !s.ContinueOnError {
			fmt.Printf("Exiting due to error on step [%v+]\n", s)
			return err
		}
		fmt.Println(strings.Repeat("-", 60))
	}

	return nil
}

// Evaluates Job or Step's condition to determine if step should be ran
func canRun(condition string) (bool, error) {
	// TODO: Need to create an expression evaluator and move this there
	return true, nil
}

// Converts a VariableMap to a VariableSlice
func variablesMapToSlice(variables common.VariableMap) common.VariableSlice {
	var ret common.VariableSlice = make(common.VariableSlice, 0)
	for key, val := range variables {
		ret = append(ret, string(key+"="+val))
	}
	return ret
}

// Converts a VariableSlice to a VariableMap
func variablesSliceToMap(variables common.VariableSlice) common.VariableMap {
	var ret common.VariableMap = make(common.VariableMap)
	for _, val := range variables {
		keyValSplit := strings.Split(val, "=")
		ret[strings.TrimSpace(keyValSplit[0])] = strings.TrimSpace(keyValSplit[1])
	}

	return ret
}

func mergeVariables(vars ...common.VariableMap) common.VariableMap {
	var ret common.VariableMap = make(common.VariableMap)
	for _, v := range vars {
		maps.Copy(ret, v)
	}
	return ret
}
