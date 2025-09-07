package pipeline

import (
	"github.com/ConnorShore/micro-ci/internal/common"
)

type Script string

type Step struct {
	Name            string             `yaml:"name"`
	Condition       string             `yaml:"condition"`
	Variables       common.VariableMap `yaml:"variables"`
	ContinueOnError bool               `yaml:"continueOnError"`
	Script          Script             `yaml:"script"`
}

// TODO: Add depends on for jobs (job2 depends on job1, etc)
type Job struct {
	Name      string             `yaml:"job"`
	Condition string             `yaml:"condition"`
	Variables common.VariableMap `yaml:"variables"`
	Image     string             `yaml:"image"`
	Steps     []Step             `yaml:"steps"`
	// DependsOn
}

type Pipeline struct {
	Name      string             `yaml:"name"`
	Variables common.VariableMap `yaml:"variables"`
	Jobs      []Job
}
