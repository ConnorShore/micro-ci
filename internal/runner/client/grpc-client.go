package client

import (
	"context"
	"fmt"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/pkg/rpc/micro_ci"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcClient struct {
	conn   *grpc.ClientConn
	client micro_ci.MicroCIClient
}

func NewGrpcClient(serverAddr string) (MicroCIClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(serverAddr, opts...)
	if err != nil {
		return nil, err
	}

	client := micro_ci.NewMicroCIClient(conn)

	return &grpcClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *grpcClient) Register(ctx context.Context, machineId string) error {
	_, err := c.client.Register(ctx, &micro_ci.RegisterRequest{MachineId: machineId})
	return err
}

func (c *grpcClient) Unregister(ctx context.Context, machineId string) error {
	_, err := c.client.Unregister(ctx, &micro_ci.UnregisterRequest{MachineId: machineId})
	return err
}

func (c *grpcClient) FetchJob(ctx context.Context, machineId string) (common.Job, error) {
	res, err := c.client.FetchJob(ctx, &micro_ci.FetchJobRequest{MachineId: machineId})
	if err != nil {
		return nil, err
	}

	if res.Job == nil {
		return nil, nil
	}

	switch t := res.Job.JobType.(type) {
	case *micro_ci.Job_BootstrapJob_:
		return c.convertProtoJobToBootstrapJob(res.Job)
	case *micro_ci.Job_PipelineJob_:
		return c.convertProtoJobToPipelineJob(res.Job)
	default:
		return nil, fmt.Errorf("unknown type when fetching job [%+v]", t)
	}
}

func (c *grpcClient) UpdateJobStatus(ctx context.Context, jobRunId string, status common.JobStatus) error {
	_, err := c.client.UpdateJobStatus(ctx, &micro_ci.UpdateJobStatusRequest{
		JobRunId: jobRunId,
		Status:   string(status),
	})
	return err
}

func (c *grpcClient) StreamLogs(ctx context.Context, jobRunId, line string) error {
	_, err := c.client.StreamLogs(ctx, &micro_ci.StreamLogsRequest{
		JobRunId: jobRunId,
		LogData:  line,
	})
	return err
}

func (c *grpcClient) Close() error {
	return c.conn.Close()
}

func (c *grpcClient) convertProtoJobToBootstrapJob(j *micro_ci.Job) (*common.BootstrapJob, error) {
	bj := j.GetBootstrapJob()
	return &common.BootstrapJob{
		BaseJob: common.BaseJob{
			Name:  j.Name,
			RunId: j.RunId,
		},
		RepoURL:   bj.RepoUrl,
		CommitSha: bj.CommitSha,
		Branch:    bj.Branch,
	}, nil
}

func (c *grpcClient) convertProtoJobToPipelineJob(j *micro_ci.Job) (*pipeline.Job, error) {
	var steps []pipeline.Step

	pj := j.GetPipelineJob()
	for _, s := range pj.Steps {
		step := pipeline.Step{
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
