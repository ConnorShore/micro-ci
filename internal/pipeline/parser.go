package pipeline

import (
	"os"

	"gopkg.in/yaml.v3"
)

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

// Validates if a pipline is valid
// Returns (valid, list of errors)
func ValidatePipeline(p Pipeline) (bool, []string) {
	// TODO: Implement validation
	return true, []string{}
}
