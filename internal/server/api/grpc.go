package api

import (
	"context"
	"fmt"
	"net"
	"slices"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/pkg/rpc/micro_ci"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type MicroCIServer struct {
	micro_ci.UnimplementedMicroCIServer

	// TODO: Better approach (db or message queue)
	jobCh chan (pipeline.Job)

	// TODO: Move items to db
	machines      []string
	jobStatus     map[string]string // (jobId, jobStatus)
	jobMachineMap map[string]string // (jobRunId, machineId)

	// TODO: Need to track when a pipeline is fully complete (all jobs complete/fail)
}

func NewMicroCIServer(testPipelineFile string) (*MicroCIServer, error) {

	// TODO: Pass in config props for making the server

	// Temporary
	p, err := pipeline.ParsePipeline(testPipelineFile)
	if err != nil {
		return nil, err
	}

	var jobStatus = make(map[string]common.JobStatus, 0)

	jobQ := make(chan pipeline.Job, 100)
	for _, j := range p.Jobs {
		j.Id = uuid.NewString()
		jobStatus[j.Id] = common.StatusPending
		jobQ <- j
	}

	return &MicroCIServer{
		jobCh: jobQ,
	}, nil
}

// Start the micro ci server
func (s *MicroCIServer) Start() error {
	// TODO: Have config for specifying start parameters
	lis, err := net.Listen("tcp", "localhost:3001")
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	micro_ci.RegisterMicroCIServer(grpcServer, s)
	return grpcServer.Serve(lis)
}

// Register a new runner with the CI server
func (s *MicroCIServer) Register(ctx context.Context, req *micro_ci.RegisterRequest) (*micro_ci.RegisterResponse, error) {
	s.machines = append(s.machines, req.MachineId)

	return &micro_ci.RegisterResponse{
		Success: true,
	}, nil
}

// Register a new runner with the CI server
func (s *MicroCIServer) Unregister(ctx context.Context, req *micro_ci.UnregisterRequest) (*micro_ci.UnregisterResponse, error) {
	if !slices.Contains(s.machines, req.MachineId) {
		return &micro_ci.UnregisterResponse{
			Success: false,
		}, fmt.Errorf("server does not contain machine with id [%v]", req.MachineId)
	}
	return &micro_ci.UnregisterResponse{
		Success: true,
	}, nil
}

// Fetch a job for the registered runner to execute
func (s *MicroCIServer) FetchJob(ctx context.Context, req *micro_ci.FetchJobRequest) (*micro_ci.FetchJobResponse, error) {
	select {
	case j := <-s.jobCh:
		s.jobMachineMap[j.Id] = req.MachineId

		return &micro_ci.FetchJobResponse{
			Job: s.convertJobToProtoJob(j),
		}, nil
	default:
		return &micro_ci.FetchJobResponse{
			Job: nil,
		}, nil
	}

}

// Update the status of a job
func (s *MicroCIServer) UpdateJobStatus(ctx context.Context, req *micro_ci.UpdateJobStatusRequest) (*micro_ci.UpdateJobStatusResponse, error) {
	if _, exists := s.jobStatus[req.JobRunId]; !exists {
		return &micro_ci.UpdateJobStatusResponse{
			Success: false,
		}, fmt.Errorf("job with run id [%v] does not exist", req.JobRunId)
	}

	s.jobStatus[req.JobRunId] = req.Status
	return &micro_ci.UpdateJobStatusResponse{
		Success: true,
	}, nil
}

// // Stream logs from the runner to the CI server
// func (s *MicroCIServer) StreamLogs(ctx context.Context, req *micro_ci.StreamLogsRequest) (*micro_ci.StreamLogsResponse, error) {

// }

func (s *MicroCIServer) convertJobToProtoJob(j pipeline.Job) *micro_ci.Job {
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

	return &micro_ci.Job{
		Id:        j.Id,
		Name:      j.Name,
		Condition: j.Condition,
		Variables: j.Variables,
		Image:     j.Image,
		Steps:     steps,
	}
}
