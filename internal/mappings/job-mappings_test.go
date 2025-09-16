package mappings

// whitebox test

import (
	"reflect"
	"testing"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/pkg/rpc/micro_ci"
)

const (
	DefaultGuid string = "45d6ea26-bf35-460c-9bde-eabaf41ea916"

	DefaultRepo      string = "http://gittest.com/TestRepo.git"
	DefaultBranch    string = "master"
	DefaultCommitSha string = "346ca091076783c70623aba03fb7139d3d27134f"

	DefaultJobName   string = "Test Job"
	DefaultImage     string = "test-image"
	DefaultStepName  string = "Test Step"
	DefaultScript    string = `echo "Hello World"`
	DefaultCondition string = "1=1"
)

var (
	DefaultBootstrapJob *common.BootstrapJob = &common.BootstrapJob{
		BaseJob: common.BaseJob{
			RunId: DefaultGuid,
			Name:  DefaultJobName,
		},
		RepoURL:   DefaultRepo,
		Branch:    DefaultBranch,
		CommitSha: DefaultCommitSha,
	}

	DefaultProtoBootstrapJob *micro_ci.Job = &micro_ci.Job{
		RunId: DefaultGuid,
		Name:  DefaultJobName,
		JobType: &micro_ci.Job_BootstrapJob_{
			BootstrapJob: &micro_ci.Job_BootstrapJob{
				RepoUrl:   DefaultRepo,
				Branch:    DefaultBranch,
				CommitSha: DefaultCommitSha,
			},
		},
	}

	DefaultPipelineJob *pipeline.Job = &pipeline.Job{
		RunId:     DefaultGuid,
		Name:      DefaultJobName,
		Condition: DefaultCondition,
		Variables: make(map[string]string),
		Image:     DefaultImage,
		Steps: []pipeline.Step{
			{
				Name:            DefaultStepName,
				Script:          pipeline.Script(DefaultScript),
				Condition:       DefaultCondition,
				Variables:       make(map[string]string),
				ContinueOnError: false,
			},
		},
	}

	DefaultProtoPipelineJob *micro_ci.Job = &micro_ci.Job{
		RunId: DefaultGuid,
		Name:  DefaultJobName,
		JobType: &micro_ci.Job_PipelineJob_{
			PipelineJob: &micro_ci.Job_PipelineJob{
				Condition: DefaultCondition,
				Variables: make(map[string]string),
				Image:     DefaultImage,
				Steps: []*micro_ci.Step{
					{
						Name:            DefaultStepName,
						Script:          DefaultScript,
						Condition:       DefaultCondition,
						Variables:       make(map[string]string),
						ContinueOnError: false,
					},
				},
			},
		},
	}

	MockInvalidProtoJob micro_ci.Job = micro_ci.Job{
		RunId:   DefaultGuid,
		Name:    DefaultJobName,
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
	input, expected := DefaultBootstrapJob, DefaultProtoBootstrapJob
	actual := convertBootstrapJobToProtoJob(input)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertPipelineJobToProtoJob(t *testing.T) {
	input, expected := DefaultPipelineJob, DefaultProtoPipelineJob
	actual := convertPipelineJobToProtoJob(input)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertProtoJobToBootstrapJob(t *testing.T) {
	input, expected := DefaultProtoBootstrapJob, DefaultBootstrapJob
	actual, err := convertProtoJobToBootstrapJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to bootstrap job:%v\n", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertProtoJobToPipelineJob(t *testing.T) {
	input, expected := DefaultProtoPipelineJob, DefaultPipelineJob
	actual, err := convertProtoJobToPipelineJob(input)
	if err != nil {
		t.Errorf("Error converting proto job to pipeline job:%v\n", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestConvertProtoJobToBootstrapJobGenericMethodProducesSameResults(t *testing.T) {
	input := DefaultProtoBootstrapJob
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
	input := DefaultProtoPipelineJob
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
	input := DefaultBootstrapJob
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
	input := DefaultPipelineJob
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
	_, err := convertProtoJobToPipelineJob(DefaultProtoBootstrapJob)
	if err == nil {
		t.Error("Expected error when converting bootstrap proto job to pipeline job, but got none")
	}
}

func TestTryConvertProtoPipelineJobToBootstrapJobFails(t *testing.T) {
	// Attempt to convert and expect an error
	_, err := convertProtoJobToBootstrapJob(DefaultProtoPipelineJob)
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
