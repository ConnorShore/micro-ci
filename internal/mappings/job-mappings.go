package mappings

import (
	"fmt"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/pkg/rpc/micro_ci"
)

// Generic method to convert a job into a proto (rpc) job
func ConvertJobToProtoJob(j common.Job) (*micro_ci.Job, error) {
	switch j.GetType() {
	case common.TypePipeline:
		return convertPipelineJobToProtoJob(j.(*pipeline.Job)), nil
	case common.TypeBootstrap:
		return convertBootstrapJobToProtoJob(j.(*common.BootstrapJob)), nil
	default:
		return nil, fmt.Errorf("failed to convert job of type [%+v] to proto job", j.GetType())
	}
}

// Generic method to convert a proto (rpc) job into a job
func ConvertProtoJobToJob(j *micro_ci.Job) (common.Job, error) {
	switch t := j.JobType.(type) {
	case *micro_ci.Job_PipelineJob_:
		return convertProtoJobToPipelineJob(j)
	case *micro_ci.Job_BootstrapJob_:
		return convertProtoJobToBootstrapJob(j)
	default:
		return nil, fmt.Errorf("cannot add job of unknown type: %v", t)
	}
}

// Converts a proto (rpc) job to a bootstrap job
func convertProtoJobToBootstrapJob(j *micro_ci.Job) (*common.BootstrapJob, error) {
	if t, ok := j.JobType.(*micro_ci.Job_BootstrapJob_); !ok {
		return nil, fmt.Errorf("cannot convert proto job of type [%+v] to bootstrap job", t)
	}

	return &common.BootstrapJob{
		BaseJob: common.BaseJob{
			Name:  j.Name,
			RunId: j.RunId,
		},
		RepoURL:   j.GetBootstrapJob().GetRepoUrl(),
		CommitSha: j.GetBootstrapJob().GetCommitSha(),
		Branch:    j.GetBootstrapJob().GetBranch(),
	}, nil
}

// converts a proto (rpc) job to a pipeline job
func convertProtoJobToPipelineJob(j *micro_ci.Job) (*pipeline.Job, error) {
	if t, ok := j.JobType.(*micro_ci.Job_PipelineJob_); !ok {
		return nil, fmt.Errorf("cannot convert proto job of type [%+v] to pipeline job", t)
	}

	var steps []pipeline.Step

	pj := j.GetPipelineJob()
	for _, s := range pj.Steps {
		step := pipeline.Step{
			Name:            s.Name,
			Condition:       s.Condition,
			Variables:       s.Variables,
			ContinueOnError: s.ContinueOnError,
			Script:          pipeline.Script(s.Script),
		}
		steps = append(steps, step)
	}

	return &pipeline.Job{
		Name:      j.Name,
		RunId:     j.RunId,
		Condition: pj.Condition,
		Variables: pj.Variables,
		Image:     pj.Image,
		Steps:     steps,
	}, nil
}

// Converts a pipeline job to a proto (rpc) job
func convertPipelineJobToProtoJob(j *pipeline.Job) *micro_ci.Job {
	var steps []*micro_ci.Step
	for _, s := range j.Steps {
		ps := &micro_ci.Step{
			Name:            s.Name,
			Condition:       s.Condition,
			Variables:       s.Variables,
			ContinueOnError: s.ContinueOnError,
			Script:          string(s.Script),
		}

		steps = append(steps, ps)
	}

	var pipelineJob = &micro_ci.Job_PipelineJob{
		Condition: j.Condition,
		Variables: j.Variables,
		Image:     j.Image,
		Steps:     steps,
	}

	return &micro_ci.Job{
		RunId: j.GetRunId(),
		Name:  j.Name,
		JobType: &micro_ci.Job_PipelineJob_{
			PipelineJob: pipelineJob,
		},
	}
}

// Converts a bootstrap job to a proto (rpc) job
func convertBootstrapJobToProtoJob(j *common.BootstrapJob) *micro_ci.Job {
	var bootstrapJob = &micro_ci.Job_BootstrapJob{
		RepoUrl:   j.RepoURL,
		CommitSha: j.CommitSha,
		Branch:    j.Branch,
	}

	return &micro_ci.Job{
		RunId: j.GetRunId(),
		Name:  j.Name,
		JobType: &micro_ci.Job_BootstrapJob_{
			BootstrapJob: bootstrapJob,
		},
	}
}
