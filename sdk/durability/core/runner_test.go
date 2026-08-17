package core_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Colin4k1024/Aetheris/durability/core"
)

func TestRunner_Start(t *testing.T) {
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)
	ctx := context.Background()

	job, err := runner.Start(ctx, "test-job", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if job.ID == "" {
		t.Fatal("expected non-empty job ID")
	}
	if job.Name != "test-job" {
		t.Fatalf("expected name 'test-job', got %q", job.Name)
	}
	if job.State != core.StateCreated {
		t.Fatalf("expected state 'created', got %q", job.State)
	}
}

func TestRunner_Execute_Simple(t *testing.T) {
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)
	ctx := context.Background()

	job, _ := runner.Start(ctx, "simple", map[string]any{"count": float64(0)})

	steps := []core.Step{
		{
			ID:   "add",
			Name: "Add 10",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				count := state["count"].(float64)
				return map[string]any{"count": count + 10}, nil
			},
		},
		{
			ID:   "multiply",
			Name: "Multiply by 2",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				count := state["count"].(float64)
				return map[string]any{"count": count * 2}, nil
			},
		},
	}

	result, err := runner.Execute(ctx, job.ID, steps)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result["count"] != float64(20) {
		t.Fatalf("expected count=20, got %v", result["count"])
	}

	// Verify job state
	updatedJob, _ := runner.GetJob(ctx, job.ID)
	if updatedJob.State != core.StateCompleted {
		t.Fatalf("expected state 'completed', got %q", updatedJob.State)
	}
}

func TestRunner_Execute_CrashRecovery(t *testing.T) {
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)
	ctx := context.Background()

	job, _ := runner.Start(ctx, "crash-test", map[string]any{"records": float64(0)})

	var step1Called atomic.Int32
	var step2Called atomic.Int32

	steps := []core.Step{
		{
			ID: "process-batch-1",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				step1Called.Add(1)
				return map[string]any{"records": float64(100)}, nil
			},
		},
		{
			ID: "process-batch-2",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				step2Called.Add(1)
				if step2Called.Load() == 1 {
					// Simulate crash on first attempt
					return nil, errors.New("process killed")
				}
				return map[string]any{"records": float64(200)}, nil
			},
		},
		{
			ID: "process-batch-3",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				return map[string]any{"records": float64(300)}, nil
			},
		},
	}

	// First attempt — crashes at step 2
	_, err := runner.Execute(ctx, job.ID, steps)
	if err == nil {
		t.Fatal("expected error from crash")
	}
	if step1Called.Load() != 1 {
		t.Fatalf("step1 should be called once, called %d times", step1Called.Load())
	}

	// Resume — step 1 should NOT be re-executed, step 2 should retry
	result, err := runner.Execute(ctx, job.ID, steps)
	if err != nil {
		t.Fatalf("Resume error: %v", err)
	}
	if step1Called.Load() != 1 {
		t.Fatalf("step1 should still be called only once (checkpoint), called %d times", step1Called.Load())
	}
	if step2Called.Load() != 2 {
		t.Fatalf("step2 should be called twice (crash + resume), called %d times", step2Called.Load())
	}
	if result["records"] != float64(300) {
		t.Fatalf("expected records=300, got %v", result["records"])
	}
}

func TestRunner_Execute_StepRetry(t *testing.T) {
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)
	ctx := context.Background()

	job, _ := runner.Start(ctx, "retry-test", nil)

	var attempts atomic.Int32

	steps := []core.Step{
		{
			ID:         "flaky",
			MaxRetries: 3,
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				n := attempts.Add(1)
				if n < 3 {
					return nil, errors.New("temporary failure")
				}
				return map[string]any{"ok": true}, nil
			},
		},
	}

	result, err := runner.Execute(ctx, job.ID, steps)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %v", result["ok"])
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRunner_Execute_AllRetriesExhausted(t *testing.T) {
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)
	ctx := context.Background()

	job, _ := runner.Start(ctx, "fail-test", nil)

	steps := []core.Step{
		{
			ID:         "always-fail",
			MaxRetries: 2,
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				return nil, errors.New("permanent failure")
			},
		},
	}

	_, err := runner.Execute(ctx, job.ID, steps)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "permanent failure") {
		t.Fatalf("expected 'permanent failure' in error, got: %v", err)
	}

	// Job should be in failed state
	job, _ = runner.GetJob(ctx, job.ID)
	if job.State != core.StateFailed {
		t.Fatalf("expected state 'failed', got %q", job.State)
	}
}

func TestRunner_Execute_AlreadyCompleted(t *testing.T) {
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)
	ctx := context.Background()

	job, _ := runner.Start(ctx, "done-test", map[string]any{"x": float64(1)})

	steps := []core.Step{
		{ID: "noop", Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
			return state, nil
		}},
	}

	// First run
	result1, err := runner.Execute(ctx, job.ID, steps)
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}

	// Second run — should return cached result, not re-execute
	result2, err := runner.Execute(ctx, job.ID, steps)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}
	if result1["x"] != result2["x"] {
		t.Fatalf("results differ: %v vs %v", result1, result2)
	}
}

func TestRunner_GetEvents(t *testing.T) {
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)
	ctx := context.Background()

	job, _ := runner.Start(ctx, "event-test", nil)

	steps := []core.Step{
		{ID: "s1", Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
			return map[string]any{"done": true}, nil
		}},
	}

	runner.Execute(ctx, job.ID, steps)

	events, err := runner.GetEvents(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetEvents() error: %v", err)
	}

	// Expected events: job_created, step_started, step_finished, checkpoint_saved, step_started, step_finished, checkpoint_saved, job_completed
	// At minimum: job_created + some step events + job_completed
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}
	if events[0].Type != core.EventJobCreated {
		t.Fatalf("first event should be job_created, got %q", events[0].Type)
	}

	// Find job_completed
	found := false
	for _, e := range events {
		if e.Type == core.EventJobCompleted {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected job_completed event")
	}
}

func TestMemoryStore_VersionMismatch(t *testing.T) {
	store := core.NewMemoryStore()
	ctx := context.Background()

	jobID := "test-job"
	// Append first event at version 0
	_, err := store.AppendEvent(ctx, jobID, 0, Event(core.Event{Type: core.EventJobCreated}))
	if err != nil {
		t.Fatalf("first append error: %v", err)
	}

	// Try to append at wrong version (0 again, should be 1)
	_, err = store.AppendEvent(ctx, jobID, 0, Event(core.Event{Type: core.EventJobStarted}))
	if err != core.ErrVersionMismatch {
		t.Fatalf("expected ErrVersionMismatch, got: %v", err)
	}
}

// Event is a type alias for testing
type Event = core.Event
