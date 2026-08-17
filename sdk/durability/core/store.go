package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Store is the persistence interface for the durable execution engine.
// Implementations store the event stream and job metadata.
// The in-memory implementation is provided; postgres implementation is in sdk/postgres/.
type Store interface {
	// --- Event Stream ---

	// AppendEvent adds an event to the job's event stream.
	// It MUST be atomic with version checking: if expectedVersion != current version, return ErrVersionMismatch.
	// Returns the new version after successful append.
	AppendEvent(ctx context.Context, jobID string, expectedVersion int, event Event) (newVersion int, err error)

	// ListEvents returns all events for a job, ordered by version.
	// Returns the events and the current version (len of event stream).
	ListEvents(ctx context.Context, jobID string) ([]Event, int, error)

	// --- Job Metadata ---

	// SaveJob persists job metadata. Used for quick state lookups without replaying events.
	SaveJob(ctx context.Context, job *Job) error

	// LoadJob retrieves job metadata. Returns nil if not found.
	LoadJob(ctx context.Context, jobID string) (*Job, error)

	// --- Checkpoints ---

	// SaveCheckpoint persists a checkpoint (serialized state snapshot).
	SaveCheckpoint(ctx context.Context, jobID string, stepID string, state []byte, version int) error

	// LoadCheckpoint loads the latest checkpoint for a job.
	// Returns nil if no checkpoint exists.
	LoadCheckpoint(ctx context.Context, jobID string) (*Checkpoint, error)
}

// Checkpoint represents a point-in-time snapshot of job state.
type Checkpoint struct {
	JobID     string    `json:"job_id"`
	StepID    string    `json:"step_id"`
	State     []byte    `json:"state"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// Sentinel errors for the store.
var (
	ErrVersionMismatch = fmt.Errorf("durability: version mismatch on append")
	ErrJobNotFound     = fmt.Errorf("durability: job not found")
	ErrJobExists       = fmt.Errorf("durability: job already exists")
)

// --- In-Memory Implementation ---

// MemoryStore is a thread-safe in-memory implementation of Store.
// Perfect for development, testing, and single-process deployments.
// For production with multiple workers, use the postgres implementation.
type MemoryStore struct {
	mu          sync.RWMutex
	jobs        map[string]*Job
	events      map[string][]Event   // jobID -> events
	checkpoints map[string]*Checkpoint // jobID -> latest checkpoint
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs:        make(map[string]*Job),
		events:      make(map[string][]Event),
		checkpoints: make(map[string]*Checkpoint),
	}
}

func (s *MemoryStore) AppendEvent(ctx context.Context, jobID string, expectedVersion int, event Event) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.events[jobID]
	if len(current) != expectedVersion {
		return len(current), ErrVersionMismatch
	}

	event.JobID = jobID
	event.Version = len(current) + 1
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s-%d", jobID, event.Version)
	}

	s.events[jobID] = append(current, event)
	return event.Version, nil
}

func (s *MemoryStore) ListEvents(ctx context.Context, jobID string) ([]Event, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[jobID]
	// Return a copy to prevent external mutation
	out := make([]Event, len(events))
	copy(out, events)
	return out, len(out), nil
}

func (s *MemoryStore) SaveJob(ctx context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *job
	s.jobs[job.ID] = &cp
	return nil
}

func (s *MemoryStore) LoadJob(ctx context.Context, jobID string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job := s.jobs[jobID]
	if job == nil {
		return nil, nil
	}
	cp := *job
	// Deep copy completed steps
	if job.CompletedSteps != nil {
		cp.CompletedSteps = make(map[string]json.RawMessage, len(job.CompletedSteps))
		for k, v := range job.CompletedSteps {
			cp.CompletedSteps[k] = v
		}
	}
	return &cp, nil
}

func (s *MemoryStore) SaveCheckpoint(ctx context.Context, jobID string, stepID string, state []byte, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := &Checkpoint{
		JobID:     jobID,
		StepID:    stepID,
		Version:   version,
		CreatedAt: time.Now(),
	}
	if len(state) > 0 {
		cp.State = make([]byte, len(state))
		copy(cp.State, state)
	}
	s.checkpoints[jobID] = cp
	return nil
}

func (s *MemoryStore) LoadCheckpoint(ctx context.Context, jobID string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := s.checkpoints[jobID]
	if cp == nil {
		return nil, nil
	}
	out := *cp
	if len(cp.State) > 0 {
		out.State = make([]byte, len(cp.State))
		copy(out.State, cp.State)
	}
	return &out, nil
}
