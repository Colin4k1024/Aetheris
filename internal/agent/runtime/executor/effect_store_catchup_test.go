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
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEffectStoreCatchUp_NoDoubleExecute verifies that when a tool invocation is
// recorded in EffectStore but the event stream doesn't have the corresponding
// command_committed event (crash before commit), the catch-up mechanism properly
// replays the effect without re-executing the tool.
func TestEffectStoreCatchUp_NoDoubleExecute(t *testing.T) {
	jobID := "job-catchup-1"
	taskID := "step-catchup-1"
	toolName := "test_tool"
	cfg := map[string]any{"key": "value"}
	idempotencyKey := IdempotencyKey(jobID, taskID, toolName, cfg)

	// Simulated result that was saved to EffectStore before crash
	effectResult := []byte(`{"done":true,"output":"effect-store-result"}`)

	// Create EffectStore with pre-existing effect (simulates crash before commit)
	effectStore := NewEffectStoreMem()
	ctx := context.Background()
	_ = effectStore.PutEffect(ctx, &EffectRecord{
		JobID:          jobID,
		CommandID:      taskID,
		IdempotencyKey: idempotencyKey,
		Kind:           EffectKindTool,
		Input:          []byte(`{"key":"value"}`),
		Output:         effectResult,
	})

	// Track tool execution calls
	var callCount int32

	tools := &countToolExec{count: &callCount}
	adapter := &ToolNodeAdapter{
		Tools:       tools,
		EffectStore: effectStore,
		// No InvocationLedger/InvocationStore - we're testing EffectStore catch-up only
	}

	// Create context WITHOUT CompletedToolInvocations (event stream doesn't have the result)
	// This simulates the scenario: EffectStore has entry, but event stream missing command_committed
	ctx = WithJobID(ctx, jobID)

	payload := &AgentDAGPayload{Results: make(map[string]any)}
	out, err := adapter.runNode(ctx, taskID, toolName, cfg, nil, payload)

	// Should succeed via catch-up from EffectStore
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.NotNil(t, out.Results[taskID])

	// Tool should NOT have been called (catch-up from EffectStore)
	assert.Equal(t, int32(0), atomic.LoadInt32(&callCount), "tool should not be executed during catch-up")

	// Verify result matches EffectStore output
	m, ok := out.Results[taskID].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "effect-store-result", m["output"])
}

// TestEffectStoreCatchUp_WithInvocationLedger verifies that when EffectStore has an
// entry AND InvocationLedger also has a committed record, the ledger takes precedence
// and prevents re-execution. The EffectStore catch-up only applies when there's no
// ledger record - with ledger, the ledger is authoritative.
func TestEffectStoreCatchUp_WithInvocationLedger(t *testing.T) {
	jobID := "job-catchup-2"
	taskID := "step-catchup-2"
	toolName := "test_tool"
	cfg := map[string]any{"key": "value"}
	idempotencyKey := IdempotencyKey(jobID, taskID, toolName, cfg)

	effectResult := []byte(`{"done":true,"output":"with-ledger-result"}`)

	// Create EffectStore with pre-existing effect (simulates crash before commit)
	effectStore := NewEffectStoreMem()
	invocationStore := NewToolInvocationStoreMem()
	ledger := NewInvocationLedgerFromStore(invocationStore)

	ctx := context.Background()

	// Simulate the ledger already has a committed record (crash after ledger Commit,
	// before event stream write). Must call SetStarted first to properly set JobID.
	_ = invocationStore.SetStarted(ctx, &ToolInvocationRecord{
		InvocationID:   "inv-recovered",
		JobID:          jobID,
		StepID:         taskID,
		ToolName:       toolName,
		ArgsHash:       ArgumentsHash(cfg),
		IdempotencyKey: idempotencyKey,
		Status:         ToolInvocationStatusStarted,
	})
	_ = invocationStore.SetFinished(ctx, idempotencyKey, ToolInvocationStatusSuccess, effectResult, true, "")

	// Also put in EffectStore for completeness
	_ = effectStore.PutEffect(ctx, &EffectRecord{
		JobID:          jobID,
		CommandID:      taskID,
		IdempotencyKey: idempotencyKey,
		Kind:           EffectKindTool,
		Output:         effectResult,
	})

	var callCount int32
	tools := &countToolExec{count: &callCount}
	adapter := &ToolNodeAdapter{
		Tools:            tools,
		EffectStore:      effectStore,
		InvocationLedger: ledger,
		InvocationStore:  invocationStore,
	}

	ctx = WithJobID(ctx, jobID)
	payload := &AgentDAGPayload{Results: make(map[string]any)}

	out, err := adapter.runNode(ctx, taskID, toolName, cfg, nil, payload)
	assert.NoError(t, err)
	assert.NotNil(t, out)

	// Tool should NOT have been called - ledger has committed record
	assert.Equal(t, int32(0), atomic.LoadInt32(&callCount), "tool should not be executed when ledger has committed record")

	// Verify the result was properly injected from ledger store
	m, ok := out.Results[taskID].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "with-ledger-result", m["output"])

	// Verify subsequent Acquire returns the committed result
	decision, rec, err := ledger.Acquire(ctx, jobID, taskID, toolName, ArgumentsHash(cfg), idempotencyKey, nil)
	assert.NoError(t, err)
	assert.Equal(t, InvocationDecisionReturnRecordedResult, decision)
	assert.NotNil(t, rec)
	assert.Equal(t, string(effectResult), string(rec.Result))
}

// TestEffectStoreCatchUp_CompletedSetPreventsDoubleExecute verifies that the
// completedSet (from event stream) takes precedence and prevents double execution
// even when EffectStore has a different result.
func TestEffectStoreCatchUp_CompletedSetPreventsDoubleExecute(t *testing.T) {
	jobID := "job-catchup-3"
	taskID := "step-catchup-3"
	toolName := "test_tool"
	cfg := map[string]any{"key": "value"}
	idempotencyKey := IdempotencyKey(jobID, taskID, toolName, cfg)

	// EffectStore has one result
	effectStoreResult := []byte(`{"done":true,"output":"from-effect-store"}`)
	// completedSet (from event stream) has a different result
	completedSetResult := []byte(`{"done":true,"output":"from-event-stream"}`)

	effectStore := NewEffectStoreMem()
	ctx := context.Background()
	_ = effectStore.PutEffect(ctx, &EffectRecord{
		JobID:          jobID,
		CommandID:      taskID,
		IdempotencyKey: idempotencyKey,
		Kind:           EffectKindTool,
		Output:         effectStoreResult,
	})

	var callCount int32
	tools := &countToolExec{count: &callCount}
	adapter := &ToolNodeAdapter{
		Tools:       tools,
		EffectStore: effectStore,
	}

	// Context WITH CompletedToolInvocations - this should take precedence
	ctx = WithJobID(ctx, jobID)
	ctx = WithCompletedToolInvocations(ctx, map[string][]byte{idempotencyKey: completedSetResult})

	payload := &AgentDAGPayload{Results: make(map[string]any)}
	out, err := adapter.runNode(ctx, taskID, toolName, cfg, nil, payload)
	assert.NoError(t, err)
	assert.NotNil(t, out)

	// Tool should NOT have been called
	assert.Equal(t, int32(0), atomic.LoadInt32(&callCount), "tool should not be executed when completedSet has result")

	// Result should be from completedSet (event stream), NOT from EffectStore
	m, ok := out.Results[taskID].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "from-event-stream", m["output"], "completedSet should take precedence over EffectStore")
}

// TestEffectStoreCatchUp_WriteCatchUpFinished verifies that writeCatchUpFinished
// properly writes both tool_invocation_finished and command_committed events.
func TestEffectStoreCatchUp_WriteCatchUpFinished(t *testing.T) {
	jobID := "job-catchup-4"
	taskID := "step-catchup-4"
	stepIDForLedger := "step-override-4"
	toolName := "test_tool"
	cfg := map[string]any{"key": "value"}
	idempotencyKey := IdempotencyKey(jobID, stepIDForLedger, toolName, cfg)

	effectResult := []byte(`{"done":true,"output":"catchup-finished"}`)

	// Track what events are written
	var toolFinishedCalled bool
	var commandCommittedCalled bool
	var capturedToolPayload *ToolInvocationFinishedPayload
	var capturedCommandPayload []byte

	toolSink := &mockToolEventSink{
		onAppendToolInvocationFinished: func(ctx context.Context, jobID string, nodeID string, payload *ToolInvocationFinishedPayload) error {
			toolFinishedCalled = true
			capturedToolPayload = payload
			return nil
		},
	}

	cmdSink := &mockCommandEventSink{
		onAppendCommandCommitted: func(ctx context.Context, jobID string, nodeID string, commandID string, result []byte, inputHash string) error {
			commandCommittedCalled = true
			capturedCommandPayload = result
			return nil
		},
	}

	effectStore := NewEffectStoreMem()
	ctx := context.Background()
	_ = effectStore.PutEffect(ctx, &EffectRecord{
		JobID:          jobID,
		CommandID:      stepIDForLedger,
		IdempotencyKey: idempotencyKey,
		Kind:           EffectKindTool,
		Output:         effectResult,
	})

	var callCount int32
	tools := &countToolExec{count: &callCount}
	adapter := &ToolNodeAdapter{
		Tools:            tools,
		EffectStore:      effectStore,
		ToolEventSink:    toolSink,
		CommandEventSink: cmdSink,
	}

	ctx = WithJobID(ctx, jobID)
	ctx = WithExecutionStepID(ctx, stepIDForLedger) // Different from taskID
	payload := &AgentDAGPayload{Results: make(map[string]any)}

	out, err := adapter.runNode(ctx, taskID, toolName, cfg, nil, payload)
	assert.NoError(t, err)
	assert.NotNil(t, out)

	// Verify tool was NOT executed
	assert.Equal(t, int32(0), atomic.LoadInt32(&callCount))

	// Verify both events were written via catch-up
	assert.True(t, toolFinishedCalled, "tool_invocation_finished should be written via catch-up")
	assert.True(t, commandCommittedCalled, "command_committed should be written via catch-up")

	// Verify tool_invocation_finished payload
	assert.NotNil(t, capturedToolPayload)
	assert.Equal(t, "catchup-"+idempotencyKey, capturedToolPayload.InvocationID)
	assert.Equal(t, idempotencyKey, capturedToolPayload.IdempotencyKey)
	assert.Equal(t, ToolInvocationOutcomeSuccess, capturedToolPayload.Outcome)
	assert.Equal(t, string(effectResult), string(capturedToolPayload.Result))

	// Verify command_committed payload
	assert.Equal(t, string(effectResult), string(capturedCommandPayload))
}

// TestEffectStoreCatchUp_NoEffectStore_NoCatchUp verifies that when EffectStore
// is not configured, the system falls back to normal execution path.
func TestEffectStoreCatchUp_NoEffectStore_NoCatchUp(t *testing.T) {
	jobID := "job-catchup-5"
	taskID := "step-catchup-5"
	toolName := "test_tool"
	cfg := map[string]any{"key": "value"}

	// EffectStore is NOT configured
	var callCount int32
	tools := &countToolExec{count: &callCount}
	adapter := &ToolNodeAdapter{
		Tools: tools,
		// EffectStore is nil - no catch-up possible
	}

	ctx := context.Background()
	ctx = WithJobID(ctx, jobID)
	payload := &AgentDAGPayload{Results: make(map[string]any)}

	out, err := adapter.runNode(ctx, taskID, toolName, cfg, nil, payload)
	assert.NoError(t, err)
	assert.NotNil(t, out)

	// Tool SHOULD have been called (no EffectStore, normal execution)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "tool should be executed when no EffectStore catch-up available")

	// Verify result from actual tool execution
	m, ok := out.Results[taskID].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "called", m["output"])
}

// TestEffectStoreCatchUp_EmptyOutputNoCatchUp verifies that EffectStore entries
// with empty output don't trigger catch-up, allowing normal execution.
func TestEffectStoreCatchUp_EmptyOutputNoCatchUp(t *testing.T) {
	jobID := "job-catchup-6"
	taskID := "step-catchup-6"
	toolName := "test_tool"
	cfg := map[string]any{"key": "value"}
	idempotencyKey := IdempotencyKey(jobID, taskID, toolName, cfg)

	// EffectStore has entry but with empty output (inflight state)
	effectStore := NewEffectStoreMem()
	ctx := context.Background()
	_ = effectStore.PutEffect(ctx, &EffectRecord{
		JobID:          jobID,
		CommandID:      taskID,
		IdempotencyKey: idempotencyKey,
		Kind:           EffectKindTool,
		Input:          []byte(`{"key":"value"}`),
		Output:         []byte{}, // Empty - indicates inflight, not committed
		Error:          "inflight",
	})

	var callCount int32
	tools := &countToolExec{count: &callCount}
	adapter := &ToolNodeAdapter{
		Tools:       tools,
		EffectStore: effectStore,
	}

	ctx = WithJobID(ctx, jobID)
	payload := &AgentDAGPayload{Results: make(map[string]any)}

	out, err := adapter.runNode(ctx, taskID, toolName, cfg, nil, payload)
	assert.NoError(t, err)
	assert.NotNil(t, out)

	// Tool SHOULD be called because EffectStore output is empty
	// (empty output means it wasn't committed, just inflight)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "tool should execute when EffectStore entry has empty output")
}

// mockToolEventSink implements ToolEventSink for testing
type mockToolEventSink struct {
	onAppendToolInvocationFinished func(ctx context.Context, jobID string, nodeID string, payload *ToolInvocationFinishedPayload) error
}

func (m *mockToolEventSink) AppendToolCalled(ctx context.Context, jobID string, nodeID string, toolName string, input []byte) error {
	return nil
}

func (m *mockToolEventSink) AppendToolReturned(ctx context.Context, jobID string, nodeID string, output []byte) error {
	return nil
}

func (m *mockToolEventSink) AppendToolResultSummarized(ctx context.Context, jobID string, nodeID string, toolName string, summary string, errMsg string, idempotent bool) error {
	return nil
}

func (m *mockToolEventSink) AppendToolInvocationStarted(ctx context.Context, jobID string, nodeID string, payload *ToolInvocationStartedPayload) error {
	return nil
}

func (m *mockToolEventSink) AppendToolInvocationFinished(ctx context.Context, jobID string, nodeID string, payload *ToolInvocationFinishedPayload) error {
	if m.onAppendToolInvocationFinished != nil {
		return m.onAppendToolInvocationFinished(ctx, jobID, nodeID, payload)
	}
	return nil
}

// mockCommandEventSink implements CommandEventSink for testing
type mockCommandEventSink struct {
	onAppendCommandCommitted func(ctx context.Context, jobID string, nodeID string, commandID string, result []byte, inputHash string) error
}

func (m *mockCommandEventSink) AppendCommandEmitted(ctx context.Context, jobID string, nodeID string, commandID string, kind string, input []byte) error {
	return nil
}

func (m *mockCommandEventSink) AppendCommandCommitted(ctx context.Context, jobID string, nodeID string, commandID string, result []byte, inputHash string) error {
	if m.onAppendCommandCommitted != nil {
		return m.onAppendCommandCommitted(ctx, jobID, nodeID, commandID, result, inputHash)
	}
	return nil
}
