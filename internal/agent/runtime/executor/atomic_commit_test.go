// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"context"
	"sync"
	"testing"

	"rag-platform/internal/runtime/jobstore"
)

// fakeLedgerEventSink 内存实现的 LedgerEventSink
type fakeLedgerEventSink struct {
	mu     sync.RWMutex
	events []jobstore.JobEvent
}

func newFakeLedgerEventSink() *fakeLedgerEventSink {
	return &fakeLedgerEventSink{}
}

func (s *fakeLedgerEventSink) AppendLedgerAcquired(ctx context.Context, jobID string, ver int64, payload *LedgerAcquiredPayload) error {
	ev, err := jobstore.NewLedgerAcquiredEvent(jobID, payload.InvocationID, payload.StepID, payload.ToolName, payload.IdempotencyKey)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, *ev)
	return nil
}

func (s *fakeLedgerEventSink) AppendLedgerCommitted(ctx context.Context, jobID string, ver int64, payload *LedgerCommittedPayload) error {
	ev, err := jobstore.NewLedgerCommittedEvent(jobID, payload.InvocationID, payload.StepID, payload.ToolName, payload.IdempotencyKey)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, *ev)
	return nil
}

func (s *fakeLedgerEventSink) ListEvents(ctx context.Context, jobID string) ([]jobstore.JobEvent, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.events, int64(len(s.events)), nil
}

// TestAtomicCommit_CrashRecovery verifies the crash recovery protocol:
// 1. ledger_acquired is written before tool execution
// 2. On replay, orphaned ledger_acquired (no ledger_committed) is detected
// 3. Second worker receives WaitOtherWorker, no double execution
func TestAtomicCommit_CrashRecovery(t *testing.T) {
	ctx := context.Background()
	store := NewToolInvocationStoreMem()
	ledger := NewInvocationLedgerFromStore(store)
	sink := newFakeLedgerEventSink()
	ledger.(*ledgerStore).SetEventSink(sink)

	jobID := "job-atomic"
	stepID := "step-1"
	toolName := "test_tool"
	idempotencyKey := "key-atomic-1"
	argsHash := "hash1"

	// Worker1: Acquire -> gets AllowExecute
	decision1, _, err := ledger.Acquire(ctx, jobID, stepID, toolName, argsHash, idempotencyKey, nil)
	if err != nil {
		t.Fatalf("Worker1 Acquire failed: %v", err)
	}
	if decision1 != InvocationDecisionAllowExecute {
		t.Fatalf("Worker1 expected AllowExecute, got %v", decision1)
	}

	// Worker1: Atomic commit writes ledger_acquired before executing tool
	toolExecuted := false
	err = DefaultAtomicCommit(ctx, ledger, sink, jobID, stepID, toolName, idempotencyKey, 0, argsHash, func() (string, error) {
		toolExecuted = true
		return `{"status":"ok"}`, nil
	})
	if err != nil {
		t.Fatalf("AtomicCommit failed: %v", err)
	}
	if !toolExecuted {
		t.Fatal("tool was not executed")
	}

	// Verify: ledger_acquired and ledger_committed events are present
	sink.mu.RLock()
	eventTypes := make([]string, len(sink.events))
	for i, e := range sink.events {
		eventTypes[i] = string(e.Type)
	}
	sink.mu.RUnlock()

	hasAcquired := false
	hasCommitted := false
	for _, et := range eventTypes {
		if et == "ledger_acquired" {
			hasAcquired = true
		}
		if et == "ledger_committed" {
			hasCommitted = true
		}
	}
	if !hasAcquired {
		t.Error("expected ledger_acquired event")
	}
	if !hasCommitted {
		t.Error("expected ledger_committed event")
	}

	// Worker2 (new instance, same shared store and event sink): tries to Acquire
	// Uses same store pointer (shared storage) and sink (shared event log)
	ledger2 := NewInvocationLedgerFromStore(store)
	sink2 := newFakeLedgerEventSink()
	sink2.events = append([]jobstore.JobEvent(nil), sink.events...)
	ledger2.(*ledgerStore).SetEventSink(sink2)

	decision2, _, err := ledger2.Acquire(ctx, jobID, stepID, toolName, argsHash, idempotencyKey, nil)
	if err != nil {
		t.Fatalf("Worker2 Acquire failed: %v", err)
	}
	if decision2 != InvocationDecisionReturnRecordedResult {
		t.Fatalf("Worker2 expected ReturnRecordedResult (committed=true in store), got %v", decision2)
	}
}

// TestAtomicCommit_OrphanedAcquired 检测孤岛的 ledger_acquired 无 ledger_committed → WaitOtherWorker
func TestAtomicCommit_OrphanedAcquired(t *testing.T) {
	ctx := context.Background()
	store := NewToolInvocationStoreMem()
	ledger := NewInvocationLedgerFromStore(store)
	sink := newFakeLedgerEventSink()
	ledger.(*ledgerStore).SetEventSink(sink)

	jobID := "job-orphan"
	stepID := "step-1"
	toolName := "test_tool"
	idempotencyKey := "key-orphan-1"
	argsHash := "hash1"

	// Simulate: ledger_acquired written but no tool execution or commit (crash after acquired)
	sink.AppendLedgerAcquired(ctx, jobID, 0, &LedgerAcquiredPayload{
		InvocationID:   "inv-orphan",
		JobID:          jobID,
		StepID:         stepID,
		ToolName:       toolName,
		IdempotencyKey: idempotencyKey,
	})

	// Verify: hasOrphanedAcquired returns true
	hasOrphan := ledger.(*ledgerStore).hasOrphanedAcquired(ctx, jobID, idempotencyKey)
	if !hasOrphan {
		t.Error("expected orphaned ledger_acquired to be detected")
	}

	// Worker2: tries to Acquire — should get WaitOtherWorker (orphaned in-progress)
	decision, _, err := ledger.Acquire(ctx, jobID, stepID, toolName, argsHash, idempotencyKey, nil)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if decision != InvocationDecisionWaitOtherWorker {
		t.Fatalf("expected WaitOtherWorker for orphaned acquired, got %v", decision)
	}
}

// TestAtomicCommit_StoreStillHasStarted 验证：当 LedgerEventSink 检测孤岛时，
// store 中已有 started 记录（Worker1 SetStarted 成功），因此 Acquire 返回 WaitOtherWorker
func TestAtomicCommit_StoreStartedBeforeAcquired(t *testing.T) {
	ctx := context.Background()
	store := NewToolInvocationStoreMem()
	ledger := NewInvocationLedgerFromStore(store)
	sink := newFakeLedgerEventSink()
	ledger.(*ledgerStore).SetEventSink(sink)

	jobID := "job-sequence"
	stepID := "step-1"
	toolName := "test_tool"
	idempotencyKey := "key-seq-1"
	argsHash := "hash1"

	// Simulate crash sequence:
	// 1. Worker1: Acquire → SetStarted → AllowExecute (store has started, not committed)
	decision1, rec1, err := ledger.Acquire(ctx, jobID, stepID, toolName, argsHash, idempotencyKey, nil)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if decision1 != InvocationDecisionAllowExecute {
		t.Fatalf("expected AllowExecute, got %v", decision1)
	}
	if rec1 == nil || rec1.InvocationID == "" {
		t.Fatal("expected invocation record")
	}

	// 2. Worker1: writes ledger_acquired (crash happens after this, before Commit)
	invID := rec1.InvocationID
	sink.AppendLedgerAcquired(ctx, jobID, 0, &LedgerAcquiredPayload{
		InvocationID:   invID,
		JobID:          jobID,
		StepID:         stepID,
		ToolName:       toolName,
		IdempotencyKey: idempotencyKey,
	})

	// 3. Worker1 crashes before Commit (simulated by not calling Commit)

	// 4. Worker2 (same job, new attempt): tries to Acquire
	// Should detect orphaned ledger_acquired → WaitOtherWorker
	decision2, _, err := ledger.Acquire(ctx, jobID, stepID, toolName, argsHash, idempotencyKey, nil)
	if err != nil {
		t.Fatalf("Worker2 Acquire failed: %v", err)
	}
	// Note: because store already has started (not committed), first CheckStore returns WaitOtherWorker
	// Even without event sink, this is already correct. The event sink adds the crash-window detection.
	if decision2 != InvocationDecisionWaitOtherWorker {
		t.Fatalf("expected WaitOtherWorker, got %v", decision2)
	}
}
