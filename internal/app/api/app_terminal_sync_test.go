package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"rag-platform/internal/agent/job"
	"rag-platform/internal/runtime/jobstore"
)

func TestAppendTerminalEventAndUpdateStatus_Success(t *testing.T) {
	ctx := context.Background()
	eventStore := jobstore.NewMemoryStore()
	metaStore := job.NewJobStoreMem()

	jobID, err := metaStore.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}
	if err := metaStore.UpdateStatus(ctx, jobID, job.StatusRunning); err != nil {
		t.Fatalf("UpdateStatus running: %v", err)
	}
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}

	if err := appendTerminalEventAndUpdateStatus(ctx, eventStore, metaStore, jobID, []byte(`{"goal":"g1"}`), jobstore.JobCompleted, job.StatusCompleted); err != nil {
		t.Fatalf("appendTerminalEventAndUpdateStatus: %v", err)
	}

	stored, err := metaStore.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored.Status != job.StatusCompleted {
		t.Fatalf("job status = %v, want %v", stored.Status, job.StatusCompleted)
	}
	events, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 || events[1].Type != jobstore.JobCompleted {
		t.Fatalf("expected job_completed event, got %+v", events)
	}
}

func TestAppendTerminalEventAndUpdateStatus_EventPhaseReturnsTerminalStateSyncError(t *testing.T) {
	ctx := context.Background()
	eventStore := jobstore.NewMemoryStore()
	metaStore := job.NewJobStoreMem()

	jobID, err := metaStore.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}
	if err := metaStore.UpdateStatus(ctx, jobID, job.StatusRunning); err != nil {
		t.Fatalf("UpdateStatus running: %v", err)
	}
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}
	_, attemptID, err := eventStore.ClaimJob(ctx, "worker-current", jobID)
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}
	if attemptID == "" {
		t.Fatal("expected non-empty attempt ID")
	}

	err = appendTerminalEventAndUpdateStatus(jobstore.WithAttemptID(ctx, "attempt-stale"), eventStore, metaStore, jobID, []byte(`{"goal":"g1"}`), jobstore.JobCompleted, job.StatusCompleted)
	var syncErr *job.TerminalStateSyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected TerminalStateSyncError, got %v", err)
	}
	if syncErr.Phase != "event" {
		t.Fatalf("sync error phase = %q, want event", syncErr.Phase)
	}
	if syncErr.Status != job.StatusCompleted {
		t.Fatalf("sync error status = %v, want %v", syncErr.Status, job.StatusCompleted)
	}

	stored, getErr := metaStore.Get(ctx, jobID)
	if getErr != nil {
		t.Fatalf("Get job: %v", getErr)
	}
	if stored.Status != job.StatusRunning {
		t.Fatalf("event-phase failure should not update metadata status, got %v want %v", stored.Status, job.StatusRunning)
	}
	events, _, listErr := eventStore.ListEvents(ctx, jobID)
	if listErr != nil {
		t.Fatalf("ListEvents: %v", listErr)
	}
	if len(events) != 1 || events[0].Type != jobstore.JobCreated {
		t.Fatalf("event-phase failure should not append terminal event, got %+v", events)
	}
}

func TestAppendTerminalEventAndUpdateStatus_StatusPhaseReturnsTerminalStateSyncError(t *testing.T) {
	ctx := context.Background()
	eventStore := jobstore.NewMemoryStore()
	metaStore := &statusFailingJobStore{JobStore: job.NewJobStoreMem(), failStatus: true}

	jobID, err := metaStore.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}
	metaStore.failStatus = false
	if err := metaStore.UpdateStatus(ctx, jobID, job.StatusRunning); err != nil {
		t.Fatalf("UpdateStatus running: %v", err)
	}
	metaStore.failStatus = true
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}

	err = appendTerminalEventAndUpdateStatus(ctx, eventStore, metaStore, jobID, []byte(`{"goal":"g1"}`), jobstore.JobCompleted, job.StatusCompleted)
	var syncErr *job.TerminalStateSyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected TerminalStateSyncError, got %v", err)
	}
	if syncErr.Phase != "status" {
		t.Fatalf("sync error phase = %q, want status", syncErr.Phase)
	}

	stored, getErr := metaStore.Get(ctx, jobID)
	if getErr != nil {
		t.Fatalf("Get job: %v", getErr)
	}
	if stored.Status != job.StatusRunning {
		t.Fatalf("status-phase failure should leave metadata at running, got %v want %v", stored.Status, job.StatusRunning)
	}
	events, _, listErr := eventStore.ListEvents(ctx, jobID)
	if listErr != nil {
		t.Fatalf("ListEvents: %v", listErr)
	}
	if len(events) != 2 || events[1].Type != jobstore.JobCompleted {
		t.Fatalf("status-phase failure should still append terminal event, got %+v", events)
	}
}

type statusFailingJobStore struct {
	job.JobStore
	failStatus bool
}

func (s *statusFailingJobStore) UpdateStatus(ctx context.Context, jobID string, status job.JobStatus) error {
	if s.failStatus {
		s.failStatus = false
		return apiTestError("status update failed")
	}
	return s.JobStore.UpdateStatus(ctx, jobID, status)
}

type apiTestError string

func (e apiTestError) Error() string { return string(e) }

func TestAPITerminalSync_StatusPhaseIsRepairedByScheduler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventStore := jobstore.NewMemoryStore()
	metaStore := &statusFailingJobStore{JobStore: job.NewJobStoreMem()}

	jobID, err := metaStore.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}

	var failStatusOnce = true
	metaStore.failStatus = false
	runJob := func(runCtx context.Context, j *job.Job) error {
		payload := []byte(`{"goal":"g1"}`)
		metaStore.failStatus = failStatusOnce
		failStatusOnce = false
		return appendTerminalEventAndUpdateStatus(runCtx, eventStore, metaStore, j.ID, payload, jobstore.JobCompleted, job.StatusCompleted)
	}
	sched := job.NewScheduler(metaStore, runJob, job.SchedulerConfig{MaxConcurrency: 1, RetryMax: 1, Backoff: 10 * time.Millisecond})
	sched.Start(ctx)
	defer sched.Stop()

	for i := 0; i < 30; i++ {
		time.Sleep(30 * time.Millisecond)
		stored, _ := metaStore.Get(ctx, jobID)
		if stored != nil && stored.Status == job.StatusCompleted {
			break
		}
	}
	stored, err := metaStore.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored == nil || stored.Status != job.StatusCompleted {
		t.Fatalf("scheduler should repair status-phase failure to completed, got %+v", stored)
	}
	events, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 || events[1].Type != jobstore.JobCompleted {
		t.Fatalf("expected one job_completed event after repair flow, got %+v", events)
	}
}

func TestAPITerminalSync_EventPhaseIsNotRepairedByScheduler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventStore := jobstore.NewMemoryStore()
	metaStore := job.NewJobStoreMem()

	jobID, err := metaStore.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}
	_, attemptID, err := eventStore.ClaimJob(ctx, "worker-current", jobID)
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}
	if attemptID == "" {
		t.Fatal("expected non-empty attempt ID")
	}

	runJob := func(runCtx context.Context, j *job.Job) error {
		payload := []byte(`{"goal":"g1"}`)
		return appendTerminalEventAndUpdateStatus(jobstore.WithAttemptID(runCtx, "attempt-stale"), eventStore, metaStore, j.ID, payload, jobstore.JobCompleted, job.StatusCompleted)
	}
	sched := job.NewScheduler(metaStore, runJob, job.SchedulerConfig{MaxConcurrency: 1, RetryMax: 1, Backoff: 10 * time.Millisecond})
	sched.Start(ctx)
	defer sched.Stop()

	time.Sleep(150 * time.Millisecond)
	stored, err := metaStore.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored == nil || stored.Status != job.StatusRunning {
		t.Fatalf("event-phase failure should remain running and not be repaired, got %+v", stored)
	}
	events, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 || events[0].Type != jobstore.JobCreated {
		t.Fatalf("event-phase failure should not append terminal event, got %+v", events)
	}
}
