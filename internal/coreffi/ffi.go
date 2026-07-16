// Package coreffi provides Go bindings for the Rust Aetheris core library.
//
// This package wraps the C ABI exported by aetheris-ffi, providing
// idiomatic Go interfaces for JobStore, EffectStore, and Executor.
package coreffi

/*
#cgo LDFLAGS: -L${SRCDIR}/../../aetheris-core/target/release -laetheris_ffi
#cgo darwin LDFLAGS: -framework Security -framework Foundation
#include "../../aetheris-core/include/aetheris_core.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"
)

// Error codes matching Rust AetherisError.
const (
	ErrSuccess         = 0
	ErrInvalidInput    = -1
	ErrVersionConflict = -2
	ErrDatabase        = -3
	ErrTimeout         = -4
	ErrInternal        = -5
)

// CoreError represents an error from the Rust core library.
type CoreError struct {
	Code    int
	Message string
}

func (e *CoreError) Error() string {
	return fmt.Sprintf("core error (code %d): %s", e.Code, e.Message)
}

func errorFromCode(code C.int32_t) error {
	if code == 0 {
		return nil
	}
	return &CoreError{Code: int(code)}
}

// ── JobStore ──────────────────────────────────────────────────

// JobStore wraps the Rust JobStore handle.
type JobStore struct {
	handle C.uintptr_t
	mu     sync.Mutex
}

// JobEvent represents a job event (mirrors Rust JobEvent).
type JobEvent struct {
	JobID     string `json:"job_id"`
	Version   int    `json:"version"`
	Type      string `json:"event_type"`
	Payload   []byte `json:"payload"`
	PrevHash  string `json:"prev_hash"`
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp_ms"`
}

// ClaimResult represents a successful claim.
type ClaimResult struct {
	JobID     string `json:"job_id"`
	Version   int    `json:"version"`
	AttemptID string `json:"attempt_id"`
}

// NewJobStore creates a new JobStore connected to the given DSN.
func NewJobStore(dsn string) (*JobStore, error) {
	cDSN := C.CString(dsn)
	defer C.free(unsafe.Pointer(cDSN))

	var handle C.uintptr_t
	rc := C.aetheris_jobstore_new(cDSN, &handle)
	if rc != 0 {
		return nil, errorFromCode(rc)
	}
	return &JobStore{handle: handle}, nil
}

// Append adds an event to a job's event stream with optimistic concurrency.
func (s *JobStore) Append(jobID string, expectedVersion int, event JobEvent) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cJobID := C.CString(jobID)
	defer C.free(unsafe.Pointer(cJobID))

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return 0, fmt.Errorf("marshal event: %w", err)
	}

	var newVersion C.int32_t
	rc := C.aetheris_jobstore_append(
		s.handle,
		cJobID,
		C.int32_t(expectedVersion),
		(*C.uint8_t)(C.CBytes(eventJSON)),
		C.size_t(len(eventJSON)),
		&newVersion,
	)
	if rc != 0 {
		return 0, errorFromCode(rc)
	}
	return int(newVersion), nil
}

// Claim claims the next available job for a worker.
func (s *JobStore) Claim(workerID string) (*ClaimResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cWorkerID := C.CString(workerID)
	defer C.free(unsafe.Pointer(cWorkerID))

	var outJSON *C.uint8_t
	var outLen C.size_t

	rc := C.aetheris_jobstore_claim(s.handle, cWorkerID, &outJSON, &outLen)
	if rc != 0 {
		return nil, errorFromCode(rc)
	}

	if outJSON == nil || outLen == 0 {
		return nil, nil // No job available
	}

	defer C.aetheris_free(outJSON, outLen)
	data := C.GoBytes(unsafe.Pointer(outJSON), C.int(outLen))

	var result ClaimResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal claim result: %w", err)
	}
	return &result, nil
}

// Close frees the Rust JobStore handle.
func (s *JobStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle != 0 {
		C.aetheris_jobstore_free(s.handle)
		s.handle = 0
	}
}

// ── EffectStore ───────────────────────────────────────────────

// EffectStore wraps the Rust EffectStore handle.
type EffectStore struct {
	handle C.uintptr_t
	mu     sync.Mutex
}

// NewEffectStore creates a new in-memory EffectStore.
func NewEffectStore() (*EffectStore, error) {
	var handle C.uintptr_t
	rc := C.aetheris_effectstore_new(&handle)
	if rc != 0 {
		return nil, errorFromCode(rc)
	}
	return &EffectStore{handle: handle}, nil
}

// RecordPending records a pending effect (Phase 1 of 2PC).
func (s *EffectStore) RecordPending(jobID, attemptID, kind string, input []byte, idempotencyKey string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cJobID := C.CString(jobID)
	cAttemptID := C.CString(attemptID)
	cKind := C.CString(kind)
	cKey := C.CString(idempotencyKey)
	defer C.free(unsafe.Pointer(cJobID))
	defer C.free(unsafe.Pointer(cAttemptID))
	defer C.free(unsafe.Pointer(cKind))
	defer C.free(unsafe.Pointer(cKey))

	var outID *C.char
	rc := C.aetheris_effectstore_record_pending(
		s.handle,
		cJobID,
		cAttemptID,
		cKind,
		(*C.uint8_t)(C.CBytes(input)),
		C.size_t(len(input)),
		cKey,
		&outID,
	)
	if rc != 0 {
		return "", errorFromCode(rc)
	}
	defer C.aetheris_free_string(outID)
	return C.GoString(outID), nil
}

// Confirm commits a pending effect (Phase 2 — commit).
func (s *EffectStore) Confirm(effectID string, output []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cEffectID := C.CString(effectID)
	defer C.free(unsafe.Pointer(cEffectID))

	rc := C.aetheris_effectstore_confirm(
		s.handle,
		cEffectID,
		(*C.uint8_t)(C.CBytes(output)),
		C.size_t(len(output)),
	)
	return errorFromCode(rc)
}

// Rollback aborts a pending effect (Phase 2 — abort).
func (s *EffectStore) Rollback(effectID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cEffectID := C.CString(effectID)
	cReason := C.CString(reason)
	defer C.free(unsafe.Pointer(cEffectID))
	defer C.free(unsafe.Pointer(cReason))

	rc := C.aetheris_effectstore_rollback(s.handle, cEffectID, cReason)
	return errorFromCode(rc)
}

// Close frees the Rust EffectStore handle.
func (s *EffectStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle != 0 {
		C.aetheris_effectstore_free(s.handle)
		s.handle = 0
	}
}

// ── Executor ──────────────────────────────────────────────────

// Executor wraps the Rust Executor handle.
type Executor struct {
	handle C.uintptr_t
	mu     sync.Mutex
}

// NewExecutor creates a new Executor backed by a JobStore.
func NewExecutor(jobStore *JobStore) (*Executor, error) {
	var handle C.uintptr_t
	rc := C.aetheris_executor_new(jobStore.handle, &handle)
	if rc != 0 {
		return nil, errorFromCode(rc)
	}
	return &Executor{handle: handle}, nil
}

// RunStep executes a single step and returns the result JSON.
func (e *Executor) RunStep(jobID string, input []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	cJobID := C.CString(jobID)
	defer C.free(unsafe.Pointer(cJobID))

	var outResult *C.uint8_t
	var outLen C.size_t

	rc := C.aetheris_executor_run_step(
		e.handle,
		cJobID,
		(*C.uint8_t)(C.CBytes(input)),
		C.size_t(len(input)),
		&outResult,
		&outLen,
	)
	if rc != 0 {
		return nil, errorFromCode(rc)
	}
	defer C.aetheris_free(outResult, outLen)
	return C.GoBytes(unsafe.Pointer(outResult), C.int(outLen)), nil
}

// Close frees the Rust Executor handle.
func (e *Executor) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.handle != 0 {
		C.aetheris_executor_free(e.handle)
		e.handle = 0
	}
}
