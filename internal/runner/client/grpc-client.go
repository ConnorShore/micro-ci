package client

import (
	"context"

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

func (c *grpcClient) FetchJob(ctx context.Context, machineId string) (*pipeline.Job, error) {
	res, err := c.client.FetchJob(ctx, &micro_ci.FetchJobRequest{MachineId: machineId})
	if err != nil {
		return nil, err
	}

	if res.Job == nil {
		return nil, nil
	}

	var steps []pipeline.Step
	for _, s := range res.Job.Steps {
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
		RunId:     res.Job.RunId,
		Name:      res.Job.Name,
		Condition: res.Job.Condition,
		Variables: res.Job.Variables,
		Image:     res.Job.Image,
		Steps:     steps,
	}, nil
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
