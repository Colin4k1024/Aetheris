package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"rag-platform/internal/agent/job"
	agentexec "rag-platform/internal/agent/runtime/executor"
	"rag-platform/internal/runtime/jobstore"
	"rag-platform/pkg/log"
)

func TestExecuteJob_ReturnsAfterRunJob(t *testing.T) {
	logger, err := log.NewLogger(&log.Config{Level: "error"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	meta := job.NewJobStoreMem()
	ev := jobstore.NewMemoryStore()
	r := NewAgentJobRunner(
		"worker-test",
		ev,
		meta,
		func(ctx context.Context, j *job.Job) error {
			return nil
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	jid, err := meta.Create(context.Background(), &job.Job{
		AgentID: "a1",
		Goal:    "g1",
		Status:  job.StatusPending,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	done := make(chan struct{})
	go func() {
		r.executeJob(context.Background(), jid, "attempt-test")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("executeJob blocked after runJob returned")
	}
}

func TestNewAgentJobRunner(t *testing.T) {
	logger, _ := log.NewLogger(&log.Config{Level: "error"})
	meta := job.NewJobStoreMem()
	ev := jobstore.NewMemoryStore()

	r := NewAgentJobRunner(
		"worker-1",
		ev,
		meta,
		func(ctx context.Context, j *job.Job) error {
			return nil
		},
		10*time.Millisecond,
		100*time.Millisecond,
		0, // test default concurrency
		nil,
		logger,
	)

	if r == nil {
		t.Fatal("expected non-nil runner")
	}
	// Default maxConcurrency should be 2
	if r.maxConcurrency != 2 {
		t.Errorf("expected maxConcurrency 2, got %d", r.maxConcurrency)
	}
}

func TestNewAgentJobRunner_WithCapabilities(t *testing.T) {
	logger, _ := log.NewLogger(&log.Config{Level: "error"})
	meta := job.NewJobStoreMem()
	ev := jobstore.NewMemoryStore()

	r := NewAgentJobRunner(
		"worker-1",
		ev,
		meta,
		func(ctx context.Context, j *job.Job) error {
			return nil
		},
		10*time.Millisecond,
		100*time.Millisecond,
		5,
		[]string{"capability1", "capability2"},
		logger,
	)

	if r == nil {
		t.Fatal("expected non-nil runner")
	}
	if len(r.capabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(r.capabilities))
	}
}

func TestDefaultWorkerID(t *testing.T) {
	id := DefaultWorkerID()
	if id == "" {
		t.Error("expected non-empty worker ID")
	}
}

func TestAgentJobRunner_SetWakeupQueue(t *testing.T) {
	logger, _ := log.NewLogger(&log.Config{Level: "error"})
	meta := job.NewJobStoreMem()
	ev := jobstore.NewMemoryStore()

	r := NewAgentJobRunner(
		"worker-1",
		ev,
		meta,
		func(ctx context.Context, j *job.Job) error {
			return nil
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	// Test that SetWakeupQueue doesn't panic
	r.SetWakeupQueue(nil)
}

func TestAgentJobRunner_SetInboxReader(t *testing.T) {
	logger, _ := log.NewLogger(&log.Config{Level: "error"})
	meta := job.NewJobStoreMem()
	ev := jobstore.NewMemoryStore()

	r := NewAgentJobRunner(
		"worker-1",
		ev,
		meta,
		func(ctx context.Context, j *job.Job) error {
			return nil
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	// Test that SetInboxReader doesn't panic
	r.SetInboxReader(nil)
}

func TestAgentJobRunner_SetInstanceStore(t *testing.T) {
	logger, _ := log.NewLogger(&log.Config{Level: "error"})
	meta := job.NewJobStoreMem()
	ev := jobstore.NewMemoryStore()

	r := NewAgentJobRunner(
		"worker-1",
		ev,
		meta,
		func(ctx context.Context, j *job.Job) error {
			return nil
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	// Test that SetInstanceStore doesn't panic
	r.SetInstanceStore(nil)
}

func TestAppendTerminalEventAndUpdateStatus_Success(t *testing.T) {
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	eventStore := jobstore.NewMemoryStore()

	jobID, err := meta.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := meta.UpdateStatus(ctx, jobID, job.StatusRunning); err != nil {
		t.Fatalf("UpdateStatus running: %v", err)
	}
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{"goal": "g1"})
	if err := appendTerminalEventAndUpdateStatus(ctx, eventStore, meta, jobID, payload, jobstore.JobCompleted, job.StatusCompleted); err != nil {
		t.Fatalf("appendTerminalEventAndUpdateStatus: %v", err)
	}

	stored, err := meta.Get(ctx, jobID)
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
		t.Fatalf("expected terminal job_completed event, got %+v", events)
	}
}

func TestAppendTerminalEventAndUpdateStatus_StaleAttemptDoesNotUpdateMetadata(t *testing.T) {
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	eventStore := jobstore.NewMemoryStore()

	jobID, err := meta.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := meta.UpdateStatus(ctx, jobID, job.StatusRunning); err != nil {
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

	staleCtx := jobstore.WithAttemptID(ctx, "attempt-stale")
	payload, _ := json.Marshal(map[string]any{"goal": "g1"})
	err = appendTerminalEventAndUpdateStatus(staleCtx, eventStore, meta, jobID, payload, jobstore.JobCompleted, job.StatusCompleted)
	if err != jobstore.ErrStaleAttempt {
		t.Fatalf("expected ErrStaleAttempt, got %v", err)
	}

	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored.Status != job.StatusRunning {
		t.Fatalf("stale attempt should not update metadata status, got %v want %v", stored.Status, job.StatusRunning)
	}
	events, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 || events[0].Type != jobstore.JobCreated {
		t.Fatalf("stale attempt should not append terminal event, got %+v", events)
	}
}

func TestExecuteJob_CancelPersistsTerminalEventWithActiveAttemptContext(t *testing.T) {
	logger, err := log.NewLogger(&log.Config{Level: "error"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	baseEventStore := jobstore.NewMemoryStore()
	eventStore := &cancelAwareEventStore{JobStore: baseEventStore}
	r := NewAgentJobRunner(
		"worker-test",
		eventStore,
		meta,
		func(ctx context.Context, j *job.Job) error {
			<-ctx.Done()
			return ctx.Err()
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	jobID, err := meta.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := baseEventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}
	if err := meta.RequestCancel(ctx, jobID); err != nil {
		t.Fatalf("RequestCancel: %v", err)
	}
	_, attemptID, err := baseEventStore.ClaimJob(ctx, "worker-test", jobID)
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}

	r.executeJob(ctx, jobID, attemptID)

	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored.Status != job.StatusCancelled {
		t.Fatalf("job status = %v, want %v", stored.Status, job.StatusCancelled)
	}
	if eventStore.sawCanceledCtx {
		t.Fatal("terminal append should not use a canceled context")
	}
	events, _, err := baseEventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 || events[1].Type != jobstore.JobCancelled {
		t.Fatalf("expected job_cancelled event, got %+v", events)
	}
}

func TestExecuteJob_StaleCancelDoesNotPersistTerminalState(t *testing.T) {
	logger, err := log.NewLogger(&log.Config{Level: "error"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	baseEventStore := jobstore.NewMemoryStore()
	r := NewAgentJobRunner(
		"worker-test",
		baseEventStore,
		meta,
		func(ctx context.Context, j *job.Job) error {
			<-ctx.Done()
			return ctx.Err()
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	jobID, err := meta.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := baseEventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}
	if err := meta.RequestCancel(ctx, jobID); err != nil {
		t.Fatalf("RequestCancel: %v", err)
	}
	_, currentAttemptID, err := baseEventStore.ClaimJob(ctx, "worker-current", jobID)
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}
	if currentAttemptID == "" {
		t.Fatal("expected non-empty current attempt ID")
	}

	r.executeJob(ctx, jobID, "attempt-stale")

	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored.Status != job.StatusRunning {
		t.Fatalf("stale cancel should not update metadata status, got %v want %v", stored.Status, job.StatusRunning)
	}
	events, _, err := baseEventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 || events[0].Type != jobstore.JobCreated {
		t.Fatalf("stale cancel should not append terminal event, got %+v", events)
	}
}

func TestExecuteJob_WaitingDoesNotAppendJobFailed(t *testing.T) {
	logger, err := log.NewLogger(&log.Config{Level: "error"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	baseEventStore := jobstore.NewMemoryStore()
	r := NewAgentJobRunner(
		"worker-test",
		baseEventStore,
		meta,
		func(ctx context.Context, j *job.Job) error {
			if err := meta.UpdateStatus(ctx, j.ID, job.StatusWaiting); err != nil {
				return err
			}
			_, ver, _ := baseEventStore.ListEvents(ctx, j.ID)
			payload := []byte(`{"node_id":"wait1","wait_kind":"signal","reason":"need-approval"}`)
			if _, err := baseEventStore.Append(ctx, j.ID, ver, jobstore.JobEvent{JobID: j.ID, Type: jobstore.JobWaiting, Payload: payload}); err != nil {
				return err
			}
			return agentexec.ErrJobWaiting
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	jobID, err := meta.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := baseEventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}
	_, attemptID, err := baseEventStore.ClaimJob(ctx, "worker-test", jobID)
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}

	r.executeJob(ctx, jobID, attemptID)

	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored.Status != job.StatusWaiting {
		t.Fatalf("waiting job status = %v, want %v", stored.Status, job.StatusWaiting)
	}
	events, _, err := baseEventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 || events[1].Type != jobstore.JobWaiting {
		t.Fatalf("waiting job should keep only job_waiting terminal state, got %+v", events)
	}
}

func TestExecuteJob_RunJobFailureDoesNotDuplicateTerminalFailureEvent(t *testing.T) {
	logger, err := log.NewLogger(&log.Config{Level: "error"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	baseEventStore := jobstore.NewMemoryStore()
	r := NewAgentJobRunner(
		"worker-test",
		baseEventStore,
		meta,
		func(ctx context.Context, j *job.Job) error {
			payload := []byte(`{"goal":"g1","error":"boom"}`)
			if err := appendTerminalEventAndUpdateStatus(ctx, baseEventStore, meta, j.ID, payload, jobstore.JobFailed, job.StatusFailed); err != nil {
				return err
			}
			return assertiveFailure("boom")
		},
		10*time.Millisecond,
		100*time.Millisecond,
		1,
		nil,
		logger,
	)

	jobID, err := meta.Create(ctx, &job.Job{AgentID: "a1", Goal: "g1", Status: job.StatusPending})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := baseEventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
		t.Fatalf("append job_created: %v", err)
	}
	_, attemptID, err := baseEventStore.ClaimJob(ctx, "worker-test", jobID)
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}

	r.executeJob(ctx, jobID, attemptID)

	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if stored.Status != job.StatusFailed {
		t.Fatalf("failed job status = %v, want %v", stored.Status, job.StatusFailed)
	}
	events, _, err := baseEventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 || events[1].Type != jobstore.JobFailed {
		t.Fatalf("expected exactly one job_failed event, got %+v", events)
	}
}

type cancelAwareEventStore struct {
	jobstore.JobStore
	sawCanceledCtx bool
}

func (s *cancelAwareEventStore) Append(ctx context.Context, jobID string, expectedVersion int, event jobstore.JobEvent) (int, error) {
	if ctx.Err() != nil {
		s.sawCanceledCtx = true
		return 0, ctx.Err()
	}
	return s.JobStore.Append(ctx, jobID, expectedVersion, event)
}

type assertiveFailure string

func (e assertiveFailure) Error() string { return string(e) }
