package pipeline

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Script string

type Step struct {
	Name      string `yaml:"name"`
	Condition string `yaml:"condition"`
	Script    Script `yaml:"script"`
}

type Pipeline struct {
	Name  string `yaml:"name"`
	Steps []Step `yaml:"steps"`
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
