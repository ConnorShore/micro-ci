package mappings

// whitebox test

import (
	"reflect"
	"testing"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/test"
	"github.com/ConnorShore/micro-ci/pkg/rpc/micro_ci"
)

var (
	MockInvalidProtoJob micro_ci.Job = micro_ci.Job{
		RunId:   test.DefaultGuid,
		Name:    test.DefaultJobName,
		JobType: nil, // Represents an unknown or unsupported job type
	}
)

type mockInvalidJob struct {
	common.Job
}

func (m *mockInvalidJob) GetType() common.JobType {
	return common.JobType(-1)
}

func (m *mockInvalidJob) GetName() string {
	return "invalid_job_type"
}

func (m *mockInvalidJob) GetRunId() string {
	return "-1"
}

func TestConvertBootstrapJobToProtoJob(t *testing.T) {
	input, expected := test.DefaultBootstrapJob, test.DefaultProtoBootstrapJob
	actual := convertBootstrapJobToProtoJob(input)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertPipelineJobToProtoJob(t *testing.T) {
	input, expected := test.DefaultPipelineJob, test.DefaultProtoPipelineJob
	actual := convertPipelineJobToProtoJob(input)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertProtoJobToBootstrapJob(t *testing.T) {
	input, expected := test.DefaultProtoBootstrapJob, test.DefaultBootstrapJob
	actual, err := convertProtoJobToBootstrapJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to bootstrap job:%v\n", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertProtoJobToPipelineJob(t *testing.T) {
	input, expected := test.DefaultProtoPipelineJob, test.DefaultPipelineJob
	actual, err := convertProtoJobToPipelineJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to pipeline job:%v\n", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertProtoJobToBootstrapJobGenericMethodProducesSameResults(t *testing.T) {
	input := test.DefaultProtoBootstrapJob
	actual1, err := convertProtoJobToBootstrapJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to bootstrap job:%v\n", err)
	}
	actual2, err := ConvertProtoJobToJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to job:%v\n", err)
	}

	if !reflect.DeepEqual(actual1, actual2) {
		t.Errorf("Expected same value but got different values: [%v | %v]\n", actual1, actual2)
	}
}

func TestConvertProtoJobToPipelineJobGenericMethodProducesSameResults(t *testing.T) {
	input := test.DefaultProtoPipelineJob
	actual1, err := convertProtoJobToPipelineJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to pipeline job:%v\n", err)
	}
	actual2, err := ConvertProtoJobToJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to job:%v\n", err)
	}

	if !reflect.DeepEqual(actual1, actual2) {
		t.Errorf("Expected same value but got different values: [%v | %v]\n", actual1, actual2)
	}
}

func TestConvertBootstrapJobToProtoJobGenericMethodProducesSameResults(t *testing.T) {
	input := test.DefaultBootstrapJob
	actual1 := convertBootstrapJobToProtoJob(input)
	actual2, err := ConvertJobToProtoJob(input)
	if err != nil {
		t.Errorf("Error converting job to proto job:%v\n", err)
	}

	if !reflect.DeepEqual(actual1, actual2) {
		t.Errorf("Expected same value but got different values: [%v | %v]\n", actual1, actual2)
	}
}

func TestConvertPipelineobToProtoJobGenericMethodProducesSameResults(t *testing.T) {
	input := test.DefaultPipelineJob
	actual1 := convertPipelineJobToProtoJob(input)
	actual2, err := ConvertJobToProtoJob(input)
	if err != nil {
		t.Errorf("Error converting job to proto job:%v\n", err)
	}

	if !reflect.DeepEqual(actual1, actual2) {
		t.Errorf("Expected same value but got different values: [%v | %v]\n", actual1, actual2)
	}
}

func TestTryConvertProtoBootstrapJobToPipelineJobFails(t *testing.T) {
	// Attempt to convert and expect an error
	_, err := convertProtoJobToPipelineJob(test.DefaultProtoBootstrapJob)
	if err == nil {
		t.Error("Expected error when converting bootstrap proto job to pipeline job, but got none")
	}
}

func TestTryConvertProtoPipelineJobToBootstrapJobFails(t *testing.T) {
	// Attempt to convert and expect an error
	_, err := convertProtoJobToBootstrapJob(test.DefaultProtoPipelineJob)
	if err == nil {
		t.Error("Expected error when converting pipeline proto job to bootstrap job, but got none")
	}
}

func TestConvertInvalidTypeFailsProtoJobToJob(t *testing.T) {
	// Attempt to convert and expect an error
	_, err := ConvertProtoJobToJob(&MockInvalidProtoJob)
	if err == nil {
		t.Error("Expected error when converting job of unknown type, but got none")
	}
}

func TestConvertInvalidTypeFailsJobToProtoJob(t *testing.T) {
	// Intentionally change the type to an unsupported one
	invalidJob := &mockInvalidJob{}
	// Attempt to convert and expect an error
	_, err := ConvertJobToProtoJob(invalidJob)
	if err == nil {
		t.Error("Expected error when converting job of unknown type, but got none")
	}
}
