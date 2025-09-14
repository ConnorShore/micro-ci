package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"slices"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
	"github.com/ConnorShore/micro-ci/internal/server/scheduler"
	"github.com/ConnorShore/micro-ci/pkg/rpc/micro_ci"
	"google.golang.org/grpc"
)

type MicroCIServer struct {
	micro_ci.UnimplementedMicroCIServer

	// TODO: Better approach (db or message queue)
	jobQ scheduler.JobQueue

	// TODO: Move items to db
	machines []string

	// TODO: Make thread safe (sync.Map)
	jobStatus     map[string]common.JobStatus // (jobId, jobStatus)
	jobMachineMap map[string]string           // (jobRunId, machineId)

	// TODO: Need to track when a pipeline is fully complete (all jobs complete/fail)
}

func NewMicroCIServer(jq scheduler.JobQueue) (*MicroCIServer, error) {
	// TODO: Pass in config props for making the server

	return &MicroCIServer{
		jobQ:          jq,
		jobStatus:     make(map[string]common.JobStatus),
		jobMachineMap: make(map[string]string),
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
	log.Printf("Register request from machine: %v\n", req.MachineId)
	s.machines = append(s.machines, req.MachineId)

	return &micro_ci.RegisterResponse{
		Success: true,
	}, nil
}

// Register a new runner with the CI server
func (s *MicroCIServer) Unregister(ctx context.Context, req *micro_ci.UnregisterRequest) (*micro_ci.UnregisterResponse, error) {
	log.Printf("Unregister request from machine: %v\n", req.MachineId)

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
	j := s.jobQ.Dequeue()
	if j == nil {
		return &micro_ci.FetchJobResponse{
			Job: nil,
		}, nil
	}

	log.Printf("Fetched Job [%v] for runner [%v]\n", j.GetName(), req.MachineId)
	s.jobMachineMap[j.GetRunId()] = req.MachineId
	s.jobStatus[j.GetRunId()] = common.StatusPending

	job, err := s.convertJobToProtoJob(j)
	return &micro_ci.FetchJobResponse{Job: job}, err
}

// Update the status of a job
func (s *MicroCIServer) UpdateJobStatus(ctx context.Context, req *micro_ci.UpdateJobStatusRequest) (*micro_ci.UpdateJobStatusResponse, error) {
	log.Printf("Update job status request for job [%v]. Status: %v\n", req.JobRunId, req.Status)
	if _, exists := s.jobStatus[req.JobRunId]; !exists {
		return &micro_ci.UpdateJobStatusResponse{
			Success: false,
		}, fmt.Errorf("job with run id [%v] does not exist", req.JobRunId)
	}

	s.jobStatus[req.JobRunId] = common.JobStatus(req.Status)
	return &micro_ci.UpdateJobStatusResponse{
		Success: true,
	}, nil
}

// Stream logs from the runner to the CI server
func (s *MicroCIServer) StreamLogs(ctx context.Context, req *micro_ci.StreamLogsRequest) (*micro_ci.StreamLogsResponse, error) {
	log.Printf("[Server] [JobID: %v]: %v\n", req.JobRunId, req.LogData)
	return &micro_ci.StreamLogsResponse{
		Success: true,
	}, nil
}

func (s *MicroCIServer) convertJobToProtoJob(j common.Job) (*micro_ci.Job, error) {
	switch t := j.(type) {
	case *pipeline.Job:
		return s.convertPipelineJobToProtoJob(j.(*pipeline.Job)), nil
	case *common.BootstrapJob:
		return s.convertBootstrapJobToProtoJob(j.(*common.BootstrapJob)), nil
	default:
		return nil, fmt.Errorf("failed to convert job of type [%+v] to proto job", t)
	}
}

func (s *MicroCIServer) convertPipelineJobToProtoJob(j *pipeline.Job) *micro_ci.Job {
	fmt.Printf("Converting pipeline job [%+v] to proto job\n", *j)
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

func (s *MicroCIServer) convertBootstrapJobToProtoJob(j *common.BootstrapJob) *micro_ci.Job {
	fmt.Printf("Converting bootstrap job [%+v] to proto job\n", *j)
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
