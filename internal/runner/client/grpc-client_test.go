package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/test"
	"github.com/ConnorShore/micro-ci/pkg/rpc/micro_ci"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	InvalidID = "-1"
)

type mockMicroCIServer struct {
	micro_ci.UnimplementedMicroCIServer

	// mutex for data
	mu sync.Mutex

	// Data for fetch job
	expectedJob *micro_ci.Job
	fetchError  error

	// channel for add job (to inspect added jobs)
	addJobCh chan *micro_ci.Job

	// registered machine ids
	machines map[string]bool

	// job status
	jobStatus map[string]string

	// log steam channel
	logStream chan string
}

func (s *mockMicroCIServer) FetchJob(ctx context.Context, req *micro_ci.FetchJobRequest) (*micro_ci.FetchJobResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fetchError != nil {
		return nil, s.fetchError
	}

	return &micro_ci.FetchJobResponse{Job: s.expectedJob}, nil
}

func (s *mockMicroCIServer) AddJob(ctx context.Context, j *micro_ci.Job) (*micro_ci.AddJobResponse, error) {
	s.addJobCh <- j
	return &micro_ci.AddJobResponse{Success: true}, nil
}

func (s *mockMicroCIServer) Register(ctx context.Context, req *micro_ci.RegisterRequest) (*micro_ci.RegisterResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.MachineId == InvalidID {
		return &micro_ci.RegisterResponse{Success: false}, fmt.Errorf("Failed to register machine")
	}

	s.machines[req.MachineId] = true
	return &micro_ci.RegisterResponse{Success: true}, nil
}

func (s *mockMicroCIServer) Unregister(ctx context.Context, req *micro_ci.UnregisterRequest) (*micro_ci.UnregisterResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.MachineId == InvalidID {
		return &micro_ci.UnregisterResponse{Success: false}, fmt.Errorf("Failed to unregister machine")
	}

	s.machines[req.MachineId] = false
	return &micro_ci.UnregisterResponse{Success: true}, nil
}

func (s *mockMicroCIServer) UpdateJobStatus(ctx context.Context, req *micro_ci.UpdateJobStatusRequest) (*micro_ci.UpdateJobStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.JobRunId == InvalidID {
		return &micro_ci.UpdateJobStatusResponse{Success: false}, fmt.Errorf("Failed to update job status")
	}

	s.jobStatus[req.JobRunId] = req.Status
	return &micro_ci.UpdateJobStatusResponse{Success: true}, nil
}

func (s *mockMicroCIServer) StreamLogs(ctx context.Context, req *micro_ci.StreamLogsRequest) (*micro_ci.StreamLogsResponse, error) {
	s.logStream <- req.LogData
	return &micro_ci.StreamLogsResponse{Success: true}, nil
}

func createTestServer(t *testing.T) (MicroCIClient, *mockMicroCIServer) {
	mockMicroCIServer := &mockMicroCIServer{
		addJobCh:  make(chan *micro_ci.Job, 1),
		machines:  make(map[string]bool),
		jobStatus: make(map[string]string),
		logStream: make(chan string, 1),
	}

	lis := bufconn.Listen(1024 * 1024)

	s := grpc.NewServer()
	micro_ci.RegisterMicroCIServer(s, mockMicroCIServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Mock server exited with error: %v", err)
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	client := &grpcClient{
		conn:   conn,
		client: micro_ci.NewMicroCIClient(conn),
	}

	t.Cleanup(func() {
		s.Stop()
		client.Close()
	})

	return client, mockMicroCIServer
}

func TestFetchJob(t *testing.T) {
	client, s := createTestServer(t)
	ctx := context.Background()

	t.Run("should return expected job", func(t *testing.T) {
		s.mu.Lock()
		s.expectedJob = test.DefaultProtoPipelineJob
		s.mu.Unlock()

		job, err := client.FetchJob(ctx, "machine-1")
		if err != nil {
			t.Fatal("failed to fetch job:", err)
		}

		if job.GetName() != test.DefaultJobName {
			t.Errorf("expected name %v but got %v\n", test.DefaultProtoPipelineJob.Name, job.GetName())
		}
		if job.GetRunId() != test.DefaultGuid {
			t.Errorf("expected runId %v but got %v\n", test.DefaultProtoPipelineJob.RunId, job.GetRunId())
		}
	})

	t.Run("should return nil when no jobs available", func(t *testing.T) {
		s.mu.Lock()
		s.expectedJob = nil
		s.fetchError = nil
		s.mu.Unlock()

		job, err := client.FetchJob(ctx, "machine-1")
		if err != nil {
			t.Fatal("failed to fetch job:", err)
		}

		if job != nil {
			t.Errorf("Expected nil job but got: %v\n", job)
		}
	})

	t.Run("should return nil when no jobs available", func(t *testing.T) {
		s.mu.Lock()
		s.fetchError = fmt.Errorf("test fetch error")
		s.mu.Unlock()

		_, err := client.FetchJob(ctx, "machine-1")
		if err == nil {
			t.Fatal("should have thrown error")
		}
	})
}

func TestAddJob(t *testing.T) {
	client, s := createTestServer(t)
	ctx := context.Background()

	err := client.AddJob(ctx, test.DefaultPipelineJob)
	if err != nil {
		t.Fatal("failed to add job:", err)
	}

	addJobResult := <-s.addJobCh
	if addJobResult.GetName() != test.DefaultPipelineJob.Name {
		t.Errorf("expected result name %v but got %v\n", test.DefaultPipelineJob.Name, addJobResult.GetName())
	}
	if addJobResult.GetRunId() != test.DefaultPipelineJob.RunId {
		t.Errorf("expected result runId %v but got %v\n", test.DefaultPipelineJob.RunId, addJobResult.GetRunId())
	}
}

func TestMachineRegistration(t *testing.T) {
	client, _ := createTestServer(t)
	ctx := context.Background()

	t.Run("registering and unregistering machine should work", func(t *testing.T) {
		machineId := test.DefaultGuid
		err := client.Register(ctx, machineId)
		if err != nil {
			t.Errorf("Failed to register machine: %v\n", err)
		}

		err = client.Unregister(ctx, machineId)
		if err != nil {
			t.Errorf("Failed to unregister machine: %v\n", err)
		}
	})

	t.Run("registering something invalid should return error", func(t *testing.T) {
		err := client.Register(ctx, InvalidID)
		if err == nil {
			t.Error("Invalid registration should fail")
		}
	})

	t.Run("unregistering something invalid should return error", func(t *testing.T) {
		err := client.Unregister(ctx, InvalidID)
		if err == nil {
			t.Error("Unregistering an unknown machine id should fail")
		}
	})
}

func TestUpdateStatus(t *testing.T) {
	client, _ := createTestServer(t)
	ctx := context.Background()

	t.Run("updating valid status should succeed", func(t *testing.T) {
		err := client.UpdateJobStatus(ctx, test.DefaultGuid, common.StatusRunning)
		if err != nil {
			t.Errorf("should not fail when updating with valid status: %v\n", err)
		}
	})

	t.Run("updating invalid status should fail", func(t *testing.T) {
		err := client.UpdateJobStatus(ctx, InvalidID, common.StatusRunning)
		if err == nil {
			t.Errorf("should not succeed when updating with invalid status: %v\n", err)
		}
	})
}

func TestLogStreamSingleLine(t *testing.T) {
	client, s := createTestServer(t)
	ctx := context.Background()

	message := "This is a test streaming message"
	err := client.StreamLogs(ctx, test.DefaultGuid, message)
	if err != nil {
		t.Fatalf("failed to stream logs: %v\n", err)
	}

	result := <-s.logStream
	if result != message {
		t.Errorf("Log stream expected %v but got %v\n", message, result)
	}

}

func TestLogStreamMultiLine(t *testing.T) {
	client, s := createTestServer(t)
	ctx := context.Background()

	numMessages := 100

	errCh := make(chan error, 1)
	go func() {
		defer close(s.logStream)

		for i := range numMessages {
			message := fmt.Sprintf("This is a test streaming message %d", i)
			err := client.StreamLogs(ctx, test.DefaultGuid, message)
			if err != nil {
				errCh <- err
			}
		}
	}()

	set := make(map[string]bool)
	for {
		select {
		case line, ok := <-s.logStream:
			if !ok {
				goto endLoop
			}

			set[line] = true

		case err := <-errCh:
			t.Fatalf("failed to stream logs: %v\n", err)
		}
	}

endLoop:
	if len(set) != numMessages {
		t.Errorf("Expected %v messages but got %v\n", numMessages, len(set))
	}
}

func TestNewGrpcClient(t *testing.T) {
	t.Run("creating grpc client with valid address should succeed", func(t *testing.T) {
		client, err := NewGrpcClient("http://localhost:8080")
		if err != nil {
			t.Errorf("Failed to create new grpc client with valid address: %v", err)
		}
		if client == nil {
			t.Errorf("The client is nil")
		}
	})

	t.Run("creating grpc client with invalid address should fail", func(t *testing.T) {
		client, err := NewGrpcClient(":")
		if err == nil {
			t.Errorf("Creating a new grpc client with invalid address should fail")
		}
		if client != nil {
			t.Errorf("The client should be nil")
		}
	})
}
