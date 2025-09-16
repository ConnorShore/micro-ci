package pipeline

import (
	"gopkg.in/yaml.v3"
)

// Given a filepath, parse the file into a Pipeline object
func ParsePipeline(data []byte) (*Pipeline, error) {
	pipeline := &Pipeline{}
	err := yaml.Unmarshal(data, pipeline)
	if err != nil {
		return nil, err
	}

	return pipeline, nil
}

// Validates if a pipline is valid
// Returns (valid, list of errors)
func ValidatePipeline(p *Pipeline) (bool, []string) {
	// TODO: Implement validation
	return true, []string{}
}
