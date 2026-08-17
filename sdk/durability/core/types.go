// Package core provides the fundamental types and interfaces for durable agent execution.
//
// Aetheris SDK is a framework-agnostic library for adding crash recovery, checkpointing,
// and idempotency to any Go agent. Zero dependencies on eino, hertz, or any framework.
//
// Quick start:
//
//	store := core.NewMemoryStore()
//	runner := core.NewRunner(store)
//	job, _ := runner.Start(ctx, "my-job", map[string]any{"input": "hello"})
//	result, _ := runner.Resume(ctx, job.ID, myStep)
package core

import (
	"context"
	"encoding/json"
	"time"
)

// EventType categorizes an immutable event in the job's event stream.
type EventType string

const (
	// Lifecycle events
	EventJobCreated   EventType = "job_created"
	EventJobStarted   EventType = "job_started"
	EventJobCompleted EventType = "job_completed"
	EventJobFailed    EventType = "job_failed"
	EventJobCancelled EventType = "job_cancelled"

	// Step events
	EventStepStarted   EventType = "step_started"
	EventStepFinished  EventType = "step_finished"
	EventStepFailed    EventType = "step_failed"
	EventStepRetried   EventType = "step_retried"
	EventStepSkipped   EventType = "step_skipped"

	// Checkpoint events
	EventCheckpointSaved   EventType = "checkpoint_saved"
	EventCheckpointLoaded  EventType = "checkpoint_loaded"

	// Effect events (idempotency)
	EventEffectRecorded EventType = "effect_recorded"
)

// JobState represents the current state of a durable job.
type JobState string

const (
	StateCreated   JobState = "created"
	StateRunning   JobState = "running"
	StateWaiting   JobState = "waiting"
	StateCompleted JobState = "completed"
	StateFailed    JobState = "failed"
	StateCancelled JobState = "cancelled"
)

// Event is an immutable record in the job's event stream.
// The event stream is the source of truth — the job's state is derived from replaying its events.
type Event struct {
	ID        string          `json:"id"`
	JobID     string          `json:"job_id"`
	Type      EventType       `json:"type"`
	StepID    string          `json:"step_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"created_at"`

	// Proof chain for tamper detection (optional)
	PrevHash string `json:"prev_hash,omitempty"`
	Hash     string `json:"hash,omitempty"`
}

// Job represents a durable unit of work.
// A job progresses through steps, with each transition recorded as an event.
type Job struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     JobState  `json:"state"`
	Version   int       `json:"version"`
	Input     []byte    `json:"input,omitempty"`
	Result    []byte    `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Steps records which steps have been completed (step_id -> output).
	// Used by the runner to determine which steps to skip on resume.
	CompletedSteps map[string]json.RawMessage `json:"completed_steps,omitempty"`
}

// Step represents a single unit of work within a job.
// A step is a function that takes state and returns updated state.
type Step struct {
	ID          string
	Name        string
	Fn          StepFunc
	MaxRetries  int
	RetryDelay  time.Duration
}

// StepFunc is the function signature for a durable step.
// It receives the current state and returns updated state.
// The runner handles checkpointing — the step function just does its work.
type StepFunc func(ctx Context, state map[string]any) (map[string]any, error)

// Context provides the step with access to the current job context.
// It wraps context.Context and adds job-specific information.
type Context struct {
	// Inner is the underlying context.Context (cancellation, deadlines).
	Inner context.Context
	// JobID is the current job's ID.
	JobID string
	// StepID is the current step's ID.
	StepID string
	// State is the current job state (read-only snapshot).
	State map[string]any
	// Version is the current event stream version.
	Version int
}

// Deadline implements context.Context.
func (c Context) Deadline() (time.Time, bool) { return c.Inner.Deadline() }

// Done implements context.Context.
func (c Context) Done() <-chan struct{} { return c.Inner.Done() }

// Err implements context.Context.
func (c Context) Err() error { return c.Inner.Err() }

// Value implements context.Context.
func (c Context) Value(key any) any { return c.Inner.Value(key) }
