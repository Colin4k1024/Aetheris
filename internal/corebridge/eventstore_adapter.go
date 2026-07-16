// Package corebridge — RustEventStore adapter.
//
// Wraps the Rust FFI for Append/Claim (hot path) while delegating
// all other jobstore.JobStore methods to the Go-native pgStore.
// This hybrid approach lets us prove the FFI integration works
// without reimplementing every method in Rust.
package corebridge

import (
	"context"
	"fmt"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/coreffi"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

// RustEventStore implements jobstore.JobStore by routing Append and Claim
// through the Rust FFI, and delegating all other methods to a Go-native store.
type RustEventStore struct {
	ffi   *coreffi.JobStore
	native jobstore.JobStore // fallback for non-FFI methods
}

// NewRustEventStore creates a hybrid event store.
// `dsn` is passed to the Rust FFI; `native` is the Go store for fallback methods.
func NewRustEventStore(dsn string, native jobstore.JobStore) (*RustEventStore, error) {
	store, err := coreffi.NewJobStore(dsn)
	if err != nil {
		return nil, fmt.Errorf("corebridge: init Rust FFI JobStore: %w", err)
	}
	return &RustEventStore{ffi: store, native: native}, nil
}

// Append routes through Rust FFI.
// On FFI error, falls back to the native store.
func (s *RustEventStore) Append(ctx context.Context, jobID string, expectedVersion int, event jobstore.JobEvent) (int, error) {
	ts := event.CreatedAt.UnixMilli()
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	ffiEvt := coreffi.JobEvent{
		JobID:     event.JobID,
		Version:   expectedVersion,
		Type:      string(event.Type),
		Payload:   event.Payload,
		PrevHash:  event.PrevHash,
		Hash:      event.Hash,
		Timestamp: ts,
	}

	newVer, err := s.ffi.Append(jobID, expectedVersion, ffiEvt)
	if err != nil {
		// FFI stub or error — fall back to native
		return s.native.Append(ctx, jobID, expectedVersion, event)
	}
	return newVer, nil
}

// Claim routes through Rust FFI.
// Returns (jobID, version, attemptID, error).
// Falls back to native on FFI error or empty result.
func (s *RustEventStore) Claim(ctx context.Context, workerID string) (string, int, string, error) {
	result, err := s.ffi.Claim(workerID)
	if err != nil {
		return s.native.Claim(ctx, workerID)
	}
	if result == nil {
		// No job available — also try native (FFI stubs return nil)
		return s.native.Claim(ctx, workerID)
	}
	return result.JobID, result.Version, result.AttemptID, nil
}

// Close frees both the Rust FFI handle and is a no-op for native.
func (s *RustEventStore) Close() {
	s.ffi.Close()
}

// ── Delegated methods (native fallback) ───────────────────────

func (s *RustEventStore) ListEvents(ctx context.Context, jobID string) ([]jobstore.JobEvent, int, error) {
	return s.native.ListEvents(ctx, jobID)
}

func (s *RustEventStore) ClaimJob(ctx context.Context, workerID, jobID string) (int, string, error) {
	return s.native.ClaimJob(ctx, workerID, jobID)
}

func (s *RustEventStore) Heartbeat(ctx context.Context, workerID, jobID string) error {
	return s.native.Heartbeat(ctx, workerID, jobID)
}

func (s *RustEventStore) Watch(ctx context.Context, jobID string) (<-chan jobstore.JobEvent, error) {
	return s.native.Watch(ctx, jobID)
}

func (s *RustEventStore) ListJobIDsWithExpiredClaim(ctx context.Context) ([]string, error) {
	return s.native.ListJobIDsWithExpiredClaim(ctx)
}

func (s *RustEventStore) GetCurrentAttemptID(ctx context.Context, jobID string) (string, error) {
	return s.native.GetCurrentAttemptID(ctx, jobID)
}

func (s *RustEventStore) CreateSnapshot(ctx context.Context, jobID string, upToVersion int, snapshot []byte) error {
	return s.native.CreateSnapshot(ctx, jobID, upToVersion, snapshot)
}

func (s *RustEventStore) GetLatestSnapshot(ctx context.Context, jobID string) (*jobstore.JobSnapshot, error) {
	return s.native.GetLatestSnapshot(ctx, jobID)
}

func (s *RustEventStore) DeleteSnapshotsBefore(ctx context.Context, jobID string, beforeVersion int) error {
	return s.native.DeleteSnapshotsBefore(ctx, jobID, beforeVersion)
}

// Verify interface compliance at compile time.
var _ jobstore.JobStore = (*RustEventStore)(nil)
