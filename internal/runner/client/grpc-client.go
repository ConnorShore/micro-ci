package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/mappings"
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
	if _, err := url.Parse(serverAddr); err != nil {
		return nil, fmt.Errorf("invalid server address: %v", err)
	}

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

	return mappings.ConvertProtoJobToJob(res.Job)
}

func (c *grpcClient) AddJob(ctx context.Context, j *pipeline.Job) error {
	var steps []*micro_ci.Step
	for _, s := range j.Steps {
		steps = append(steps, &micro_ci.Step{
			Name:            s.Name,
			Condition:       s.Condition,
			ContinueOnError: s.ContinueOnError,
			Variables:       s.Variables,
			Script:          string(s.Script),
		})
	}

	job := &micro_ci.Job{
		RunId: j.RunId,
		Name:  j.Name,
		JobType: &micro_ci.Job_PipelineJob_{
			PipelineJob: &micro_ci.Job_PipelineJob{
				Condition: j.Condition,
				Image:     j.Image,
				Variables: j.Variables,
				Steps:     steps,
			},
		},
	}

	_, err := c.client.AddJob(ctx, job)
	return err
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
