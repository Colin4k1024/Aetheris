package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Runner executes durable jobs with automatic checkpointing and crash recovery.
//
// Usage:
//
//	runner := NewRunner(NewMemoryStore())
//	job, _ := runner.Start(ctx, "process-order", map[string]any{"order_id": "123"})
//	steps := []Step{
//		{ID: "validate", Fn: validateOrder},
//		{ID: "charge", Fn: chargePayment},
//		{ID: "ship", Fn: createShipment},
//	}
//	result, _ := runner.Execute(ctx, job.ID, steps)
//
// If the process crashes during "charge", calling Execute again resumes from "charge"
// — "validate" is not re-executed because its result is checkpointed.
type Runner struct {
	store Store
}

// NewRunner creates a new durable runner with the given store.
func NewRunner(store Store) *Runner {
	return &Runner{store: store}
}

// Start creates a new durable job and returns it.
// The job starts in "created" state; use Execute to run steps.
func (r *Runner) Start(ctx context.Context, name string, input map[string]any) (*Job, error) {
	jobID := fmt.Sprintf("job-%s-%d", name, time.Now().UnixNano())

	var inputBytes []byte
	if input != nil {
		var err error
		inputBytes, err = json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("durability: marshal input: %w", err)
		}
	}

	job := &Job{
		ID:             jobID,
		Name:           name,
		State:          StateCreated,
		Version:        0,
		Input:          inputBytes,
		CompletedSteps: make(map[string]json.RawMessage),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := r.store.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("durability: save job: %w", err)
	}

	// Record job_created event
	newVer, err := r.store.AppendEvent(ctx, jobID, 0, Event{
		Type: EventJobCreated,
		Payload: mustMarshal(map[string]string{
			"name": name,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("durability: append job_created: %w", err)
	}

	// Update job version to match store
	job.Version = newVer
	job.UpdatedAt = time.Now()
	if err := r.store.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("durability: save job version: %w", err)
	}

	return job, nil
}

// Execute runs the given steps sequentially with automatic checkpointing.
//
// On first call: executes all steps from the beginning.
// On resume (after crash): loads checkpoint, skips completed steps, resumes from where it left off.
//
// Returns the final state after all steps complete.
func (r *Runner) Execute(ctx context.Context, jobID string, steps []Step) (map[string]any, error) {
	// Load job metadata
	job, err := r.store.LoadJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("durability: load job: %w", err)
	}
	if job == nil {
		return nil, ErrJobNotFound
	}
	if job.State == StateCompleted {
		// Already done — return stored result
		var result map[string]any
		if job.Result != nil {
			json.Unmarshal(job.Result, &result)
		}
		return result, nil
	}
	if job.State == StateCancelled {
		return nil, fmt.Errorf("durability: job %s is cancelled", jobID)
	}

	// Sync job version with store's actual event count
	_, storeVersion, err := r.store.ListEvents(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("durability: list events: %w", err)
	}
	job.Version = storeVersion

	// Try to load checkpoint (resume point)
	var state map[string]any
	startIdx := 0

	checkpoint, err := r.store.LoadCheckpoint(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("durability: load checkpoint: %w", err)
	}

	if checkpoint != nil {
		// Resume from checkpoint
		if checkpoint.State != nil {
			if err := json.Unmarshal(checkpoint.State, &state); err != nil {
				return nil, fmt.Errorf("durability: unmarshal checkpoint state: %w", err)
			}
		}
		// Find the step index to resume from
		for i, step := range steps {
			if step.ID == checkpoint.StepID {
				startIdx = i + 1 // Resume from the step AFTER the checkpoint
				break
			}
		}
	}

	if state == nil {
		// Fresh start — parse input as initial state
		state = make(map[string]any)
		if job.Input != nil {
			json.Unmarshal(job.Input, &state)
		}
	}

	// Mark job as running
	job.State = StateRunning
	job.UpdatedAt = time.Now()
	if err := r.store.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("durability: save job running: %w", err)
	}

	// Execute steps from startIdx
	for i := startIdx; i < len(steps); i++ {
		step := steps[i]

		// Record step_started
		newVer, err := r.store.AppendEvent(ctx, jobID, job.Version, Event{
			Type:   EventStepStarted,
			StepID: step.ID,
			Payload: mustMarshal(map[string]string{
				"step_name": step.Name,
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("durability: append step_started: %w", err)
		}
		job.Version = newVer

		// Execute the step with retries
		var stepResult map[string]any
		maxRetries := step.MaxRetries
		if maxRetries <= 0 {
			maxRetries = 1
		}

		var lastErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 && step.RetryDelay > 0 {
				select {
				case <-time.After(step.RetryDelay):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}

			stepCtx := Context{
				Inner:   ctx,
				JobID:   jobID,
				StepID:  step.ID,
				State:   copyState(state),
				Version: job.Version,
			}

			stepResult, lastErr = step.Fn(stepCtx, copyState(state))
			if lastErr == nil {
				break
			}

			// Record retry
			if attempt < maxRetries-1 {
				newVer, _ = r.store.AppendEvent(ctx, jobID, job.Version, Event{
					Type:   EventStepRetried,
					StepID: step.ID,
					Payload: mustMarshal(map[string]any{
						"attempt":     attempt + 1,
						"error":       lastErr.Error(),
						"max_retries": maxRetries,
					}),
				})
				job.Version = newVer
			}
		}

		if lastErr != nil {
			// Step failed after all retries
			newVer, _ = r.store.AppendEvent(ctx, jobID, job.Version, Event{
				Type:   EventStepFailed,
				StepID: step.ID,
				Payload: mustMarshal(map[string]any{
					"error": lastErr.Error(),
				}),
			})
			job.Version = newVer
			job.State = StateFailed
			job.Error = lastErr.Error()
			job.UpdatedAt = time.Now()
			_ = r.store.SaveJob(ctx, job)
			return nil, fmt.Errorf("durability: step %s failed: %w", step.ID, lastErr)
		}

		// Merge step result into state
		for k, v := range stepResult {
			state[k] = v
		}

		// Record step_finished
		stateBytes := mustMarshal(state)
		newVer, _ = r.store.AppendEvent(ctx, jobID, job.Version, Event{
			Type:   EventStepFinished,
			StepID: step.ID,
			Payload: mustMarshal(map[string]any{
				"output_keys": keys(stepResult),
			}),
		})
		job.Version = newVer

		// Checkpoint: save state snapshot
		if err := r.store.SaveCheckpoint(ctx, jobID, step.ID, stateBytes, job.Version); err != nil {
			return nil, fmt.Errorf("durability: save checkpoint: %w", err)
		}

		// Record checkpoint event
		newVer, _ = r.store.AppendEvent(ctx, jobID, job.Version, Event{
			Type:   EventCheckpointSaved,
			StepID: step.ID,
		})
		job.Version = newVer

		// Update completed steps
		job.CompletedSteps[step.ID] = stateBytes
		job.UpdatedAt = time.Now()
		if err := r.store.SaveJob(ctx, job); err != nil {
			return nil, fmt.Errorf("durability: save job progress: %w", err)
		}
	}

	// All steps completed
	resultBytes := mustMarshal(state)
	job.State = StateCompleted
	job.Result = resultBytes
	job.UpdatedAt = time.Now()
	if err := r.store.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("durability: save job completed: %w", err)
	}

	_, _ = r.store.AppendEvent(ctx, jobID, job.Version, Event{
		Type: EventJobCompleted,
		Payload: mustMarshal(map[string]any{
			"output_keys": keys(state),
		}),
	})

	return state, nil
}

// Resume is a convenience method that loads the job and re-executes the same steps.
// Use this when you don't have the steps list handy — it replays from checkpoint.
func (r *Runner) Resume(ctx context.Context, jobID string, steps []Step) (map[string]any, error) {
	return r.Execute(ctx, jobID, steps)
}

// GetJob returns the current state of a job.
func (r *Runner) GetJob(ctx context.Context, jobID string) (*Job, error) {
	return r.store.LoadJob(ctx, jobID)
}

// GetEvents returns all events for a job (for audit/replay).
func (r *Runner) GetEvents(ctx context.Context, jobID string) ([]Event, error) {
	events, _, err := r.store.ListEvents(ctx, jobID)
	return events, err
}

// Store returns the underlying store (for advanced usage).
func (r *Runner) Store() Store {
	return r.store
}

// --- helpers ---

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func copyState(m map[string]any) map[string]any {
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
