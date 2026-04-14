package worker

import (
	"context"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/job"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
	"github.com/Colin4k1024/Aetheris/v2/pkg/log"
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
