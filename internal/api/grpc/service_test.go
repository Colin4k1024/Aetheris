package grpc

import (
	"context"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/job"
	"github.com/Colin4k1024/Aetheris/v2/internal/api/grpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestServerWithJobStore() *Server {
	js := job.NewJobStoreMem()
	return NewServerWithJobStore(nil, nil, js)
}

func newTestServerNoJobStore() *Server {
	return NewServer(nil, nil)
}

func TestSubmitJob_Success(t *testing.T) {
	srv := newTestServerWithJobStore()
	resp, err := srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId:  "test-agent",
		Input:    "do something",
		TenantId: "tenant-A",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success=true")
	}
	if resp.GetJobId() == "" {
		t.Fatalf("expected non-empty job_id")
	}
}

func TestSubmitJob_NoJobStore(t *testing.T) {
	srv := newTestServerNoJobStore()
	_, err := srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId: "test-agent",
		Input:   "do something",
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got: %v", err)
	}
	if st.Code() != codes.Unimplemented {
		t.Fatalf("expected Unimplemented, got %v", st.Code())
	}
}

func TestSubmitJob_EmptyAgentID(t *testing.T) {
	srv := newTestServerWithJobStore()
	_, err := srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId: "",
		Input:   "do something",
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestGetJob_Success(t *testing.T) {
	srv := newTestServerWithJobStore()
	submitResp, _ := srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId:  "test-agent",
		Input:    "do something",
		TenantId: "tenant-A",
	})
	resp, err := srv.GetJob(context.Background(), &pb.GetJobRequest{
		JobId: submitResp.GetJobId(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetJob() == nil {
		t.Fatalf("expected non-nil job")
	}
	if resp.GetJob().GetJobId() != submitResp.GetJobId() {
		t.Fatalf("job_id mismatch: %s vs %s", resp.GetJob().GetJobId(), submitResp.GetJobId())
	}
	if resp.GetJob().GetAgentId() != "test-agent" {
		t.Fatalf("agent_id mismatch: %s", resp.GetJob().GetAgentId())
	}
}

func TestGetJob_NotFound(t *testing.T) {
	srv := newTestServerWithJobStore()
	_, err := srv.GetJob(context.Background(), &pb.GetJobRequest{
		JobId: "nonexistent",
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
}

func TestGetJob_EmptyJobID(t *testing.T) {
	srv := newTestServerWithJobStore()
	_, err := srv.GetJob(context.Background(), &pb.GetJobRequest{
		JobId: "",
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestCancelJob_Success(t *testing.T) {
	srv := newTestServerWithJobStore()
	submitResp, _ := srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId:  "test-agent",
		Input:    "do something",
		TenantId: "tenant-A",
	})
	resp, err := srv.CancelJob(context.Background(), &pb.CancelJobRequest{
		JobId: submitResp.GetJobId(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success=true")
	}
}

func TestCancelJob_NoJobStore(t *testing.T) {
	srv := newTestServerNoJobStore()
	_, err := srv.CancelJob(context.Background(), &pb.CancelJobRequest{
		JobId: "some-id",
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unimplemented {
		t.Fatalf("expected Unimplemented, got %v", st.Code())
	}
}

func TestHeartbeat_Success(t *testing.T) {
	srv := newTestServerWithJobStore()
	submitResp, _ := srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId:  "test-agent",
		Input:    "do something",
		TenantId: "tenant-A",
	})
	resp, err := srv.Heartbeat(context.Background(), &pb.HeartbeatRequest{
		WorkerId: "worker-1",
		JobId:    submitResp.GetJobId(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success=true, message: %s", resp.GetMessage())
	}
}

func TestHeartbeat_NotFound(t *testing.T) {
	srv := newTestServerWithJobStore()
	resp, err := srv.Heartbeat(context.Background(), &pb.HeartbeatRequest{
		WorkerId: "worker-1",
		JobId:    "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSuccess() {
		t.Fatalf("expected success=false for nonexistent job")
	}
}

func TestListJobs_Success(t *testing.T) {
	srv := newTestServerWithJobStore()
	srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId:  "agent-list",
		Input:    "task 1",
		TenantId: "tenant-A",
	})
	srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId:  "agent-list",
		Input:    "task 2",
		TenantId: "tenant-A",
	})
	resp, err := srv.ListJobs(context.Background(), &pb.ListJobsRequest{
		AgentId:  "agent-list",
		TenantId: "tenant-A",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetJobs()) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(resp.GetJobs()))
	}
	if resp.GetTotal() != 2 {
		t.Fatalf("expected total=2, got %d", resp.GetTotal())
	}
}

func TestListJobs_EmptyAgentID(t *testing.T) {
	srv := newTestServerWithJobStore()
	_, err := srv.ListJobs(context.Background(), &pb.ListJobsRequest{
		AgentId: "",
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestListJobs_NoJobStore(t *testing.T) {
	srv := newTestServerNoJobStore()
	_, err := srv.ListJobs(context.Background(), &pb.ListJobsRequest{
		AgentId: "test-agent",
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unimplemented {
		t.Fatalf("expected Unimplemented, got %v", st.Code())
	}
}

func TestListJobs_StatusFilter(t *testing.T) {
	srv := newTestServerWithJobStore()
	srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
		AgentId:  "agent-filter",
		Input:    "task",
		TenantId: "tenant-A",
	})
	resp, err := srv.ListJobs(context.Background(), &pb.ListJobsRequest{
		AgentId:  "agent-filter",
		TenantId: "tenant-A",
		Status:   job.StatusPending.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetJobs()) != 1 {
		t.Fatalf("expected 1 pending job, got %d", len(resp.GetJobs()))
	}
}

func TestListJobs_Pagination(t *testing.T) {
	srv := newTestServerWithJobStore()
	for i := 0; i < 5; i++ {
		srv.SubmitJob(context.Background(), &pb.SubmitJobRequest{
			AgentId:  "agent-page",
			Input:    "task",
			TenantId: "tenant-A",
		})
	}
	resp, err := srv.ListJobs(context.Background(), &pb.ListJobsRequest{
		AgentId:  "agent-page",
		TenantId: "tenant-A",
		Limit:    2,
		Offset:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetJobs()) != 2 {
		t.Fatalf("expected 2 jobs after offset=1 limit=2, got %d", len(resp.GetJobs()))
	}
}

func TestJobToPB(t *testing.T) {
	j := &job.Job{
		ID:         "job-123",
		AgentID:    "agent-1",
		TenantID:   "tenant-A",
		Goal:       "do something",
		Status:     job.StatusRunning,
		RetryCount: 2,
	}
	info := jobToPB(j)
	if info.GetJobId() != "job-123" {
		t.Fatalf("job_id mismatch: %s", info.GetJobId())
	}
	if info.GetAgentId() != "agent-1" {
		t.Fatalf("agent_id mismatch: %s", info.GetAgentId())
	}
	if info.GetStatus() != job.StatusRunning.String() {
		t.Fatalf("status mismatch: %s", info.GetStatus())
	}
	if info.GetInput() != "do something" {
		t.Fatalf("input mismatch: %s", info.GetInput())
	}
	if info.GetAttempt() != 2 {
		t.Fatalf("attempt mismatch: %d", info.GetAttempt())
	}
}
