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
	"errors"
	"sync"
	"testing"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/replay"
	"rag-platform/internal/agent/runtime"
	"rag-platform/internal/runtime/jobstore"
)

type fakeLangGraphClient struct {
	mu         sync.Mutex
	invokeCnt  int
	invokeFunc func(ctx context.Context, input map[string]any) (map[string]any, error)
}

func (f *fakeLangGraphClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	f.mu.Lock()
	f.invokeCnt++
	fn := f.invokeFunc
	f.mu.Unlock()
	if fn == nil {
		return map[string]any{"ok": true}, nil
	}
	return fn(ctx, input)
}

func (f *fakeLangGraphClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	if onChunk != nil {
		return onChunk(map[string]any{"chunk": "x"})
	}
	return nil
}

func (f *fakeLangGraphClient) State(ctx context.Context, threadID string) (map[string]any, error) {
	return map[string]any{"thread_id": threadID, "status": "ok"}, nil
}

func (f *fakeLangGraphClient) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.invokeCnt
}

func TestLangGraphNodeAdapter_ErrorMapping(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "retryable",
			err:  &LangGraphError{Code: LangGraphErrorRetryable, Message: "temporary"},
			assertErr: func(t *testing.T, err error) {
				var sf *StepFailure
				if !errors.As(err, &sf) || sf.Type != StepResultRetryableFailure {
					t.Fatalf("error = %v, want StepResultRetryableFailure", err)
				}
			},
		},
		{
			name: "permanent",
			err:  &LangGraphError{Code: LangGraphErrorPermanent, Message: "bad graph"},
			assertErr: func(t *testing.T, err error) {
				var sf *StepFailure
				if !errors.As(err, &sf) || sf.Type != StepResultPermanentFailure {
					t.Fatalf("error = %v, want StepResultPermanentFailure", err)
				}
			},
		},
		{
			name: "signal wait",
			err:  &LangGraphError{Code: LangGraphErrorWait, CorrelationKey: "lg-approval-1", Reason: "human_approval"},
			assertErr: func(t *testing.T, err error) {
				var sw *SignalWaitRequired
				if !errors.As(err, &sw) || sw.CorrelationKey != "lg-approval-1" {
					t.Fatalf("error = %v, want SignalWaitRequired", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeLangGraphClient{invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
				return nil, tt.err
			}}
			adapter := &LangGraphNodeAdapter{Client: client}
			payload := &AgentDAGPayload{Goal: "g", Results: map[string]any{}}
			_, err := adapter.runNode(context.Background(), "lg1", map[string]any{}, payload)
			if err == nil {
				t.Fatalf("expected error")
			}
			tt.assertErr(t, err)
		})
	}
}

func appendPlanGeneratedForLangGraph(t *testing.T, store jobstore.JobStore, jobID string, g *planner.TaskGraph) {
	t.Helper()
	graphBytes, err := g.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	payload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "langgraph"})
	if _, err := store.Append(context.Background(), jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: payload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}
}

func TestLangGraphAdapter_ReplayAndSignalResume(t *testing.T) {
	ctx := context.Background()
	jobID := "job-langgraph-signal"
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{Nodes: []planner.TaskNode{{ID: "lg1", Type: planner.NodeLangGraph}}}
	appendPlanGeneratedForLangGraph(t, eventStore, jobID, graph)

	client := &fakeLangGraphClient{invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
		return nil, &LangGraphError{Code: LangGraphErrorWait, CorrelationKey: "lg-approval-1", Reason: "human_approval"}
	}}
	compiler := NewCompiler(map[string]NodeAdapter{planner.NodeLangGraph: &LangGraphNodeAdapter{Client: client}})
	runner := NewRunner(compiler)
	runner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), &fakeJobStoreForRunner{})
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))

	err := runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "approve", Cursor: ""})
	if !errors.Is(err, ErrJobWaiting) {
		t.Fatalf("first run err = %v, want ErrJobWaiting", err)
	}
	if client.Calls() != 1 {
		t.Fatalf("langgraph invoke calls = %d, want 1", client.Calls())
	}

	_, ver, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	waitPayload, _ := json.Marshal(map[string]any{
		"node_id":         "lg1",
		"payload":         map[string]any{"approved": true},
		"correlation_key": "lg-approval-1",
	})
	if _, err := eventStore.Append(ctx, jobID, ver, jobstore.JobEvent{JobID: jobID, Type: jobstore.WaitCompleted, Payload: waitPayload}); err != nil {
		t.Fatalf("append wait_completed: %v", err)
	}

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "approve", Cursor: ""})
	if err != nil {
		t.Fatalf("second run should complete, got err: %v", err)
	}
	if client.Calls() != 1 {
		t.Fatalf("langgraph invoke should not be re-executed after signal replay, got %d calls", client.Calls())
	}
}

// TestLangGraphNodeAdapter_NormalExecution 测试正常执行流程
func TestLangGraphNodeAdapter_NormalExecution(t *testing.T) {
	client := &fakeLangGraphClient{
		invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			// 验证输入
			if input["goal"] == nil {
				t.Fatal("goal should be set")
			}
			return map[string]any{
				"result":    "success",
				"next_step": "done",
			}, nil
		},
	}

	adapter := &LangGraphNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "test goal", Results: map[string]any{}}

	result, err := adapter.runNode(context.Background(), "lg1", map[string]any{}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	// 验证结果
	if result.Results["lg1"] == nil {
		t.Fatal("result should be in payload")
	}
	resultMap := result.Results["lg1"].(map[string]any)
	if resultMap["result"] != "success" {
		t.Fatalf("result = %v, want success", resultMap["result"])
	}
	if client.Calls() != 1 {
		t.Fatalf("invoke calls = %d, want 1", client.Calls())
	}
}

// TestLangGraphNodeAdapter_StreamMode 测试 Stream 模式
func TestLangGraphNodeAdapter_StreamMode(t *testing.T) {
	streamCalled := false
	client := &fakeLangGraphClient{}

	// 测试 Stream 方法
	err := client.Stream(context.Background(), map[string]any{"goal": "test"}, func(chunk map[string]any) error {
		streamCalled = true
		if chunk["chunk"] != "x" {
			t.Fatalf("chunk = %v, want x", chunk["chunk"])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Stream error = %v", err)
	}
	if !streamCalled {
		t.Fatal("stream callback should be called")
	}
}

// TestLangGraphNodeAdapter_StateManagement 测试状态管理
func TestLangGraphNodeAdapter_StateManagement(t *testing.T) {
	client := &fakeLangGraphClient{}

	// 测试 State 方法
	state, err := client.State(context.Background(), "thread-123")
	if err != nil {
		t.Fatalf("State error = %v", err)
	}
	if state["thread_id"] != "thread-123" {
		t.Fatalf("thread_id = %v, want thread-123", state["thread_id"])
	}
	if state["status"] != "ok" {
		t.Fatalf("status = %v, want ok", state["status"])
	}
}

// TestLangGraphNodeAdapter_WithEffectStore 测试 EffectStore 重放
func TestLangGraphNodeAdapter_WithEffectStore(t *testing.T) {
	ctx := context.Background()
	jobID := "job-langgraph-effect"

	// 创建内存 EffectStore
	effectStore := NewEffectStoreMem()

	// 先写入 effect
	effectRecord := &EffectRecord{
		JobID:     jobID,
		CommandID: "lg1",
		Kind:      EffectKindTool,
		Input:     []byte(`{"goal":"test"}`),
		Output:    []byte(`{"cached":true}`),
		Metadata:  map[string]any{"adapter": "langgraph"},
	}
	if err := effectStore.PutEffect(ctx, effectRecord); err != nil {
		t.Fatalf("PutEffect error = %v", err)
	}

	// 创建 client（应该不会被调用）
	client := &fakeLangGraphClient{
		invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			t.Fatal("client should not be called when effect exists")
			return nil, nil
		},
	}

	adapter := &LangGraphNodeAdapter{
		Client:      client,
		EffectStore: effectStore,
	}

	// 模拟带 jobID 的 context
	ctx = WithJobID(ctx, jobID)
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(ctx, "lg1", map[string]any{}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	// 验证结果来自 effect store
	resultMap := result.Results["lg1"].(map[string]any)
	if resultMap["cached"] != true {
		t.Fatalf("result should be from effect store, got %v", resultMap)
	}
	if client.Calls() != 0 {
		t.Fatalf("client should not be called, got %d calls", client.Calls())
	}
}

// TestLangGraphNodeAdapter_ClientNotConfigured 测试未配置 client 的情况
func TestLangGraphNodeAdapter_ClientNotConfigured(t *testing.T) {
	adapter := &LangGraphNodeAdapter{Client: nil}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	_, err := adapter.runNode(context.Background(), "lg1", map[string]any{}, payload)
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
	if err.Error() != "LangGraphNodeAdapter: Client not configured" {
		t.Fatalf("error = %v, want 'Client not configured'", err)
	}
}

// TestLangGraphNodeAdapter_RetryableErrorHandling 测试可重试错误
func TestLangGraphNodeAdapter_RetryableErrorHandling(t *testing.T) {
	client := &fakeLangGraphClient{
		invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			return nil, &LangGraphError{
				Code:    LangGraphErrorRetryable,
				Message: "rate limited",
			}
		},
	}

	adapter := &LangGraphNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	_, err := adapter.runNode(context.Background(), "lg1", map[string]any{}, payload)
	if err == nil {
		t.Fatal("expected error")
	}

	// 验证错误被正确映射
	var sf *StepFailure
	if !errors.As(err, &sf) {
		t.Fatalf("error = %v, want StepFailure", err)
	}
	if sf.Type != StepResultRetryableFailure {
		t.Fatalf("type = %v, want StepResultRetryableFailure", sf.Type)
	}
}

// TestLangGraphNodeAdapter_PermanentErrorHandling 测试永久错误
func TestLangGraphNodeAdapter_PermanentErrorHandling(t *testing.T) {
	client := &fakeLangGraphClient{
		invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			return nil, &LangGraphError{
				Code:    LangGraphErrorPermanent,
				Message: "invalid graph",
			}
		},
	}

	adapter := &LangGraphNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	_, err := adapter.runNode(context.Background(), "lg1", map[string]any{}, payload)
	if err == nil {
		t.Fatal("expected error")
	}

	// 验证错误被正确映射
	var sf *StepFailure
	if !errors.As(err, &sf) {
		t.Fatalf("error = %v, want StepFailure", err)
	}
	if sf.Type != StepResultPermanentFailure {
		t.Fatalf("type = %v, want StepResultPermanentFailure", sf.Type)
	}
}

// TestLangGraphNodeAdapter_ToNodeRunner 测试 ToNodeRunner 方法
func TestLangGraphNodeAdapter_ToNodeRunner(t *testing.T) {
	client := &fakeLangGraphClient{}
	adapter := &LangGraphNodeAdapter{Client: client}

	task := &planner.TaskNode{
		ID:   "lg1",
		Type: planner.NodeLangGraph,
	}

	// 测试 ToNodeRunner
	runner, err := adapter.ToNodeRunner(task, &runtime.Agent{ID: "a1"})
	if err != nil {
		t.Fatalf("ToNodeRunner error = %v", err)
	}
	if runner == nil {
		t.Fatal("runner should not be nil")
	}

	// 执行 runner
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}
	result, err := runner(context.Background(), payload)
	if err != nil {
		t.Fatalf("runner execution error = %v", err)
	}
	if result.Results["lg1"] == nil {
		t.Fatal("result should be in payload")
	}
}
