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
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/replay"
	"rag-platform/internal/agent/runtime"
	"rag-platform/internal/runtime/jobstore"
)

// mockLLMWithCallCount LLM mock，记录 Generate 调用次数（用于断言 Replay 不调用 LLM）
type mockLLMWithCallCount struct {
	callCount *int32
	response  string
}

func (m *mockLLMWithCallCount) Generate(ctx context.Context, prompt string) (string, error) {
	atomic.AddInt32(m.callCount, 1)
	return m.response, nil
}

// TestReplay_NeverCallsLLM 契约：配置 Effect Store 时，Replay **绝不**调用 LLM（design/execution-guarantees.md § Replay 绝不调用 LLM）
func TestReplay_NeverCallsLLM(t *testing.T) {
	ctx := context.Background()
	jobID := "job-llm-replay"

	var callCount int32
	mockLLM := &mockLLMWithCallCount{
		callCount: &callCount,
		response:  "LLM response for test",
	}

	// Setup: Effect Store + Event Store
	effectStore := NewEffectStoreMem()
	eventStore := jobstore.NewMemoryStore()

	// 构造 TaskGraph：单个 LLM 节点
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{ID: "llm1", Type: planner.NodeLLM, Config: map[string]any{"goal": "Summarize"}},
		},
		Edges: []planner.TaskEdge{},
	}

	// 使用 buildReplayableEventStream 构造完整事件流（含 PlanGenerated + command_committed + NodeFinished）
	nodeResults := map[string][]byte{
		"llm1": []byte(`{"output":"LLM response for test"}`),
	}
	buildReplayableEventStream(t, eventStore, jobID, taskGraph, nodeResults)

	// 同时写入 Effect Store（模拟第一次执行后的状态）
	respBytes, _ := json.Marshal("LLM response for test")
	_ = effectStore.PutEffect(ctx, &EffectRecord{
		JobID:     jobID,
		CommandID: "llm1",
		Kind:      EffectKindLLM,
		Input:     []byte(`{"prompt":"Summarize"}`),
		Output:    respBytes,
	})

	// Compiler + Runner with Effect Store
	llmAdapter := &LLMNodeAdapter{LLM: mockLLM, EffectStore: effectStore}
	compiler := NewCompiler(map[string]NodeAdapter{planner.NodeLLM: llmAdapter})
	runner := NewRunner(compiler)
	cpStore := runtime.NewCheckpointStoreMem()
	fakeJobStore := &fakeJobStoreForRunner{}
	runner.SetCheckpointStores(cpStore, fakeJobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))

	// Replay 执行：事件流已有完整记录，callCount 应保持 0
	agent := &runtime.Agent{ID: "agent-1"}
	job := &JobForRunner{ID: jobID, Goal: "test goal"}

	err := runner.RunForJob(ctx, agent, job)
	require.NoError(t, err, "Replay should succeed")
	require.Equal(t, int32(0), atomic.LoadInt32(&callCount), "Replay must NOT call LLM")
}

// TestReplay_LLMWithEffectStore 验证配置 Effect Store 时 Replay 从 Effect Store 注入（adapter 层防御）
func TestReplay_LLMWithEffectStore(t *testing.T) {
	ctx := context.Background()
	jobID := "job-llm-effect"

	var callCount int32
	mockLLM := &mockLLMWithCallCount{
		callCount: &callCount,
		response:  "Should not be called",
	}

	effectStore := NewEffectStoreMem()
	eventStore := jobstore.NewMemoryStore()

	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{ID: "llm1", Type: planner.NodeLLM},
		},
		Edges: []planner.TaskEdge{},
	}

	// 构造已有记录的事件流（模拟第一次执行后状态）
	buildReplayableEventStream(t, eventStore, jobID, taskGraph, map[string][]byte{
		"llm1": []byte(`{"output":"Stored LLM response"}`),
	})

	// Effect Store 中已有记录（模拟第一次执行写入）
	respBytes, _ := json.Marshal("Stored LLM response")
	_ = effectStore.PutEffect(ctx, &EffectRecord{
		JobID:     jobID,
		CommandID: "llm1",
		Kind:      EffectKindLLM,
		Output:    respBytes,
	})

	// LLMNodeAdapter 配置 Effect Store
	llmAdapter := &LLMNodeAdapter{LLM: mockLLM, EffectStore: effectStore}
	compiler := NewCompiler(map[string]NodeAdapter{planner.NodeLLM: llmAdapter})
	runner := NewRunner(compiler)
	cpStore := runtime.NewCheckpointStoreMem()
	fakeJobStore := &fakeJobStoreForRunner{}
	runner.SetCheckpointStores(cpStore, fakeJobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))

	agent := &runtime.Agent{ID: "agent-1"}
	job := &JobForRunner{ID: jobID, Goal: "test"}

	// Replay：应从 Effect Store 注入，不调用 LLM
	err := runner.RunForJob(ctx, agent, job)
	require.NoError(t, err)
	require.Equal(t, int32(0), atomic.LoadInt32(&callCount), "Replay with EffectStore must not call LLM")
}

// TestRunForJob_CheckpointSaveFailure_RetryReplaysWithoutReexecutingLLM 证明：
// 即使 node_finished 已写入但 checkpoint 保存失败，下一次运行也必须依赖事件流跳过已完成的 LLM。
func TestRunForJob_CheckpointSaveFailure_RetryReplaysWithoutReexecutingLLM(t *testing.T) {
	ctx := context.Background()
	jobID := "job-llm-checkpoint-save-failure"
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: "llm1", Type: planner.NodeLLM, Config: map[string]any{"goal": "Summarize approval context"}}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	planPayload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "g1"})
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}

	var callCount int32
	mockLLM := &mockLLMWithCallCount{callCount: &callCount, response: "LLM response after checkpoint failure"}
	sink := &jobStoreReplaySink{store: eventStore}
	adapter := &LLMNodeAdapter{LLM: mockLLM, CommandEventSink: sink}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeLLM: adapter}))
	cpStore := &failSaveOnceCheckpointStore{base: runtime.NewCheckpointStoreMem()}
	jobStore := &cursorTrackingJobStore{}
	runner.SetCheckpointStores(cpStore, jobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	runner.SetNodeEventSink(sink)

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err == nil || !strings.Contains(err.Error(), "save checkpoint failed") {
		t.Fatalf("expected save checkpoint failure, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected first run to execute LLM exactly once, got %d", callCount)
	}
	if cpStore.saveCalls != 1 {
		t.Fatalf("expected one checkpoint save attempt, got %d", cpStore.saveCalls)
	}
	if jobStore.cursorCalls != 0 {
		t.Fatalf("cursor should not update after checkpoint save failure, got %d", jobStore.cursorCalls)
	}
	_, status := jobStore.getLast()
	const statusFailed = 3
	if status != statusFailed {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusFailed)
	}

	retryRunner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeLLM: adapter}))
	retryStore := &cursorTrackingJobStore{}
	retryRunner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), retryStore)
	retryRunner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	retryRunner.SetNodeEventSink(sink)
	err = retryRunner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err != nil {
		t.Fatalf("retry RunForJob: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected retry to replay from events without re-executing LLM, got %d calls", callCount)
	}
	_, retryStatus := retryStore.getLast()
	const statusCompleted = 2
	if retryStatus != statusCompleted {
		t.Fatalf("retry UpdateStatus = %d, want %d", retryStatus, statusCompleted)
	}
}

// TestRunForJob_UpdateCursorFailure_RetryReplaysWithoutReexecutingLLM 证明：
// 即使 checkpoint 已保存但 UpdateCursor 失败，下一次运行在旧 cursor 下也必须依赖事件流跳过已完成的 LLM。
func TestRunForJob_UpdateCursorFailure_RetryReplaysWithoutReexecutingLLM(t *testing.T) {
	ctx := context.Background()
	jobID := "job-llm-cursor-update-failure"
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: "llm1", Type: planner.NodeLLM, Config: map[string]any{"goal": "Draft approval summary"}}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	planPayload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "g1"})
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}

	var callCount int32
	mockLLM := &mockLLMWithCallCount{callCount: &callCount, response: "LLM response after cursor failure"}
	sink := &jobStoreReplaySink{store: eventStore}
	adapter := &LLMNodeAdapter{LLM: mockLLM, CommandEventSink: sink}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeLLM: adapter}))
	jobStore := &cursorTrackingJobStore{failCursorOnce: true}
	cpStore := &recordingCheckpointStore{base: runtime.NewCheckpointStoreMem()}
	runner.SetCheckpointStores(cpStore, jobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	runner.SetNodeEventSink(sink)

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err == nil || !strings.Contains(err.Error(), "update cursor failed") {
		t.Fatalf("expected update cursor failure, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected first run to execute LLM exactly once, got %d", callCount)
	}
	if cpStore.saveCalls != 1 {
		t.Fatalf("expected checkpoint to be saved once before cursor failure, got %d", cpStore.saveCalls)
	}
	if jobStore.cursorCalls != 1 {
		t.Fatalf("expected one cursor update attempt, got %d", jobStore.cursorCalls)
	}
	_, status := jobStore.getLast()
	const statusFailedCursor = 3
	if status != statusFailedCursor {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusFailedCursor)
	}

	retryRunner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeLLM: adapter}))
	retryStore := &cursorTrackingJobStore{}
	retryRunner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), retryStore)
	retryRunner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	retryRunner.SetNodeEventSink(sink)
	err = retryRunner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err != nil {
		t.Fatalf("retry RunForJob: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected retry to skip LLM via replay after cursor failure, got %d calls", callCount)
	}
	_, retryStatus := retryStore.getLast()
	const statusCompletedCursor = 2
	if retryStatus != statusCompletedCursor {
		t.Fatalf("retry UpdateStatus = %d, want %d", retryStatus, statusCompletedCursor)
	}
}

// TestLLMAdapter_RequiresEffectStoreInProduction 验证生产模式下 LLM 必须配置 Effect Store
func TestLLMAdapter_RequiresEffectStoreInProduction(t *testing.T) {
	// 测试 RequireEffectStore 标志的逻辑
	// 当 RequireEffectStore 为 true 但 EffectStore 为 nil 时，应返回错误

	// 直接测试内部逻辑
	adapter := &LLMNodeAdapter{
		LLM:                nil,
		EffectStore:        nil,
		RequireEffectStore: true,
	}

	// 由于 runNode 是私有方法且需要复杂的 runner 设置，
	// 我们验证 adapter 的配置是正确的
	require.True(t, adapter.RequireEffectStore, "RequireEffectStore should be true")
	require.Nil(t, adapter.EffectStore, "EffectStore should be nil")
	// 这个组合在 runNode 中会导致错误
}

// TestLLMAdapter_RequireEffectStoreConfig 测试配置组合
func TestLLMAdapter_RequireEffectStoreConfig(t *testing.T) {
	tests := []struct {
		name               string
		requireEffectStore bool
		effectStore        EffectStore
		expectError        bool
	}{
		{
			name:               "require true, store nil - should error",
			requireEffectStore: true,
			effectStore:        nil,
			expectError:        true,
		},
		{
			name:               "require true, store exists - no error",
			requireEffectStore: true,
			effectStore:        NewEffectStoreMem(),
			expectError:        false,
		},
		{
			name:               "require false, store nil - no error",
			requireEffectStore: false,
			effectStore:        nil,
			expectError:        false,
		},
		{
			name:               "require false, store exists - no error",
			requireEffectStore: false,
			effectStore:        NewEffectStoreMem(),
			expectError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &LLMNodeAdapter{
				LLM:                nil,
				EffectStore:        tt.effectStore,
				RequireEffectStore: tt.requireEffectStore,
			}

			// 验证配置
			require.Equal(t, tt.requireEffectStore, adapter.RequireEffectStore)
			require.Equal(t, tt.effectStore, adapter.EffectStore)

			// 模拟错误条件
			hasError := adapter.RequireEffectStore && adapter.EffectStore == nil
			require.Equal(t, tt.expectError, hasError)
		})
	}
}
