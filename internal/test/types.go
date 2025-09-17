package test

import (
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
)
