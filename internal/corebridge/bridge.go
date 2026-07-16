// Package corebridge provides a feature-flag-controlled bridge between
// the Rust core implementation and the Go native implementation.
//
// Set AETHERIS_USE_RUST_CORE=true to use the Rust FFI backend.
// Default: false (Go native).
package corebridge

import (
	"os"
	"sync"

	"github.com/Colin4k1024/Aetheris/v2/internal/coreffi"
)

// UseRustCore returns true if the Rust core should be used.
func UseRustCore() bool {
	return os.Getenv("AETHERIS_USE_RUST_CORE") == "true"
}

// ── JobStore Bridge ───────────────────────────────────────────

// JobStore is the interface that both Rust and Go implementations satisfy.
type JobStore interface {
	Append(jobID string, expectedVersion int, event coreffi.JobEvent) (int, error)
	Claim(workerID string) (*coreffi.ClaimResult, error)
	Close()
}

// RustJobStore wraps the Rust FFI JobStore.
type RustJobStore struct {
	inner *coreffi.JobStore
}

func (s *RustJobStore) Append(jobID string, expectedVersion int, event coreffi.JobEvent) (int, error) {
	return s.inner.Append(jobID, expectedVersion, event)
}

func (s *RustJobStore) Claim(workerID string) (*coreffi.ClaimResult, error) {
	return s.inner.Claim(workerID)
}

func (s *RustJobStore) Close() {
	s.inner.Close()
}

// NewJobStore creates a JobStore using the appropriate backend.
func NewJobStore(dsn string) (JobStore, error) {
	if UseRustCore() {
		store, err := coreffi.NewJobStore(dsn)
		if err != nil {
			return nil, err
		}
		return &RustJobStore{inner: store}, nil
	}
	// Go native fallback — would connect to the existing Go JobStore
	// For now, return a stub that indicates Go native mode
	return &GoNativeJobStore{dsn: dsn}, nil
}

// GoNativeJobStore is a placeholder for the Go native implementation.
// In production, this would delegate to internal/runtime/jobstore.
type GoNativeJobStore struct {
	dsn string
}

func (s *GoNativeJobStore) Append(jobID string, expectedVersion int, event coreffi.JobEvent) (int, error) {
	// TODO: delegate to Go native jobstore
	return 0, nil
}

func (s *GoNativeJobStore) Claim(workerID string) (*coreffi.ClaimResult, error) {
	// TODO: delegate to Go native jobstore
	return nil, nil
}

func (s *GoNativeJobStore) Close() {
	// No-op for Go native
}

// ── EffectStore Bridge ────────────────────────────────────────

// EffectStore is the interface for effect operations.
type EffectStore interface {
	RecordPending(jobID, attemptID, kind string, input []byte, idempotencyKey string) (string, error)
	Confirm(effectID string, output []byte) error
	Rollback(effectID, reason string) error
	Close()
}

// RustEffectStore wraps the Rust FFI EffectStore.
type RustEffectStore struct {
	inner *coreffi.EffectStore
}

func (s *RustEffectStore) RecordPending(jobID, attemptID, kind string, input []byte, idempotencyKey string) (string, error) {
	return s.inner.RecordPending(jobID, attemptID, kind, input, idempotencyKey)
}

func (s *RustEffectStore) Confirm(effectID string, output []byte) error {
	return s.inner.Confirm(effectID, output)
}

func (s *RustEffectStore) Rollback(effectID, reason string) error {
	return s.inner.Rollback(effectID, reason)
}

func (s *RustEffectStore) Close() {
	s.inner.Close()
}

// NewEffectStore creates an EffectStore using the appropriate backend.
func NewEffectStore() (EffectStore, error) {
	if UseRustCore() {
		store, err := coreffi.NewEffectStore()
		if err != nil {
			return nil, err
		}
		return &RustEffectStore{inner: store}, nil
	}
	return &GoNativeEffectStore{}, nil
}

// GoNativeEffectStore is a placeholder for the Go native implementation.
type GoNativeEffectStore struct{}

func (s *GoNativeEffectStore) RecordPending(jobID, attemptID, kind string, input []byte, idempotencyKey string) (string, error) {
	// TODO: delegate to Go native effects
	return "", nil
}

func (s *GoNativeEffectStore) Confirm(effectID string, output []byte) error {
	return nil
}

func (s *GoNativeEffectStore) Rollback(effectID, reason string) error {
	return nil
}

func (s *GoNativeEffectStore) Close() {}

// ── Executor Bridge ───────────────────────────────────────────

// Executor is the interface for execution operations.
type Executor interface {
	RunStep(jobID string, input []byte) ([]byte, error)
	Close()
}

// RustExecutor wraps the Rust FFI Executor.
type RustExecutor struct {
	inner *coreffi.Executor
}

func (e *RustExecutor) RunStep(jobID string, input []byte) ([]byte, error) {
	return e.inner.RunStep(jobID, input)
}

func (e *RustExecutor) Close() {
	e.inner.Close()
}

// NewExecutor creates an Executor using the appropriate backend.
func NewExecutor(jobStore JobStore) (Executor, error) {
	if UseRustCore() {
		if rustStore, ok := jobStore.(*RustJobStore); ok {
			exec, err := coreffi.NewExecutor(rustStore.inner)
			if err != nil {
				return nil, err
			}
			return &RustExecutor{inner: exec}, nil
		}
	}
	return &GoNativeExecutor{}, nil
}

// GoNativeExecutor is a placeholder for the Go native implementation.
type GoNativeExecutor struct{}

func (e *GoNativeExecutor) RunStep(jobID string, input []byte) ([]byte, error) {
	// TODO: delegate to Go native executor
	return []byte(`{"type":"SUCCESS","output":null}`), nil
}

func (e *GoNativeExecutor) Close() {}

// ── Initialization ────────────────────────────────────────────

var (
	initOnce sync.Once
	initErr  error
)

// Init initializes the core bridge. Call once at startup.
func Init() error {
	initOnce.Do(func() {
		if UseRustCore() {
			// Register Rust→Go callbacks if needed
			initErr = nil
		}
	})
	return initErr
}
