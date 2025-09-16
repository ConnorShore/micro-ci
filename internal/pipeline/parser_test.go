package pipeline_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

const (
	TestFileLocation = "../../.micro-ci/test-pipelines/"
)

var (
	DefaultValidSimplePipeline = &pipeline.Pipeline{
		Name: "Simple Valid pipeline 1",
		Jobs: []pipeline.Job{
			{
				Name:  "First Valid Job",
				Image: "golang:1.21-alpine",
				Steps: []pipeline.Step{
					{
						Name:   "Display Message",
						Script: pipeline.Script(`echo "Successfully displayed message from pipeline script!"`),
					},
				},
			},
		},
	}

	DefaultValidFullPipeline = &pipeline.Pipeline{
		Name: "Advanced Pipeline Test 1",
		Variables: common.VariableMap(map[string]string{
			"GO_ENV":      "testing",
			"APP_VERSION": "1.0.0",
		}),
		Jobs: []pipeline.Job{
			{
				Name:  "Micro CI Pipeline 1 Test Job 1",
				Image: "golang:1.21-alpine",
				Variables: common.VariableMap(map[string]string{
					"JOB_1_VAR": "job1-tester",
				}),
				Steps: []pipeline.Step{
					{
						Name:      "Hello World Step",
						Condition: "1=1",
						Script:    pipeline.Script("echo \"Hello World\"\n"),
					},
					{
						Name:            "Failure Step Example",
						ContinueOnError: true,
						Script:          pipeline.Script("echo \"This pipeline should not exit due to a bash script failure\"\nabcd\n"),
					},
				},
			},
			{
				Name:  "Micro CI Pipeline 1 Test Job 2",
				Image: "alpine:latest",
				Variables: common.VariableMap(map[string]string{
					"JOB_2_VAR": "job2-tester",
				}),
				Steps: []pipeline.Step{
					{
						Name: "Show Local Variales and Environment Variables",
						Variables: common.VariableMap(map[string]string{
							"TEST_VAR": "tester",
						}),
						Script: pipeline.Script(`echo "This step is showing local variables and environment variables"`),
					},
				},
			},
		},
	}
)

func TestParsePipelineSimpleValid(t *testing.T) {
	input, expected := getPipelineFileData(t, "valid-basic-1.yaml"), DefaultValidSimplePipeline
	actual, err := pipeline.ParsePipeline([]byte(input))
	if err != nil {
		t.Errorf("Failed to parse pipeline: %v\n", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v from parsing the pipeline\n", actual, expected)
	}
}

func TestParsePipelineFullValid(t *testing.T) {
	input, expected := getPipelineFileData(t, "valid-full-1.yaml"), DefaultValidFullPipeline
	actual, err := pipeline.ParsePipeline([]byte(input))
	if err != nil {
		t.Errorf("Failed to parse pipeline: %v\n", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v from parsing the pipeline\n", actual, expected)
	}
}

func TestValidateSimplePipeline(t *testing.T) {
	if ok, errs := pipeline.ValidatePipeline(DefaultValidSimplePipeline); !ok {
		t.Errorf("Failed to validate valid pipeline. Errors: %v\n", errs)
	}
}

func TestValidateFullPipeline(t *testing.T) {
	if ok, errs := pipeline.ValidatePipeline(DefaultValidFullPipeline); !ok {
		t.Errorf("Failed to validate valid pipeline. Errors: %v\n", errs)
	}
}

func TestParsePipelineInvalid(t *testing.T) {
	input := getPipelineFileData(t, "invalid-basic-1.yaml")
	_, err := pipeline.ParsePipeline([]byte(input))
	if err == nil {
		t.Error("Expected to fail parsing pipeline")
	}
}

func TestParsePipelineWithNoNameFails(t *testing.T) {
	// TODO: Implement when we have validate
}

func TestParsePipelineWithNoJobsFails(t *testing.T) {
	// TODO: Implement when we have validate
}

func TestParsePipelineWithNoStepsFails(t *testing.T) {
	// TODO: Implement when we have validate
}

func getPipelineFileData(t *testing.T, filename string) []byte {
	fileName := filepath.Join(TestFileLocation, filename)
	input, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Failed to read pipeline file: %v\n", fileName)
	}

	return input
}
