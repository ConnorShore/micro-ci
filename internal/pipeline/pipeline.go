package pipeline

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Script string

type Step struct {
	Name            string            `yaml:"name"`
	Condition       string            `yaml:"condition"`
	Variables       map[string]string `yaml:"variables"`
	ContinueOnError bool              `yaml:"continueOnError"`
	Script          Script            `yaml:"script"`
}

type Pipeline struct {
	Name      string            `yaml:"name"`
	Variables map[string]string `yaml:"variables"`
	Steps     []Step            `yaml:"steps"`
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
