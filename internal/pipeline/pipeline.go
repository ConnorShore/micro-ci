package pipeline

import (
	"os"

	"github.com/ConnorShore/micro-ci/internal/common"
	"gopkg.in/yaml.v3"
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
}

type Pipeline struct {
	Name      string             `yaml:"name"`
	Variables common.VariableMap `yaml:"variables"`
	Jobs      []Job
}

// Given a filepath, parse the file into a Pipeline object
func ParsePipeline(file string) (*Pipeline, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	pipeline := &Pipeline{}
	err = yaml.Unmarshal(data, pipeline)
	if err != nil {
		return nil, err
	}

	return pipeline, nil
}
