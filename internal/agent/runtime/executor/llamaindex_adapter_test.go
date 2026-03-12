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
	"errors"
	"sync"
	"testing"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/runtime"
)

// fakeLlamaIndexClient LlamaIndex Client 的 mock 实现
type fakeLlamaIndexClient struct {
	mu          sync.Mutex
	invokeCnt   int
	invokeFunc  func(ctx context.Context, input map[string]any) (map[string]any, error)
	streamFunc  func(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
	stateFunc   func(ctx context.Context, sessionID string) (map[string]any, error)
}

func (f *fakeLlamaIndexClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	f.mu.Lock()
	f.invokeCnt++
	fn := f.invokeFunc
	f.mu.Unlock()
	if fn == nil {
		return map[string]any{"result": "ok"}, nil
	}
	return fn(ctx, input)
}

func (f *fakeLlamaIndexClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	if f.streamFunc != nil {
		return f.streamFunc(ctx, input, onChunk)
	}
	if onChunk != nil {
		return onChunk(map[string]any{"chunk": "data"})
	}
	return nil
}

func (f *fakeLlamaIndexClient) GetState(ctx context.Context, sessionID string) (map[string]any, error) {
	if f.stateFunc != nil {
		return f.stateFunc(ctx, sessionID)
	}
	return map[string]any{"session_id": sessionID, "status": "active"}, nil
}

func (f *fakeLlamaIndexClient) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.invokeCnt
}

// TestLlamaIndexNodeAdapter_NormalExecution 测试正常执行流程
func TestLlamaIndexNodeAdapter_NormalExecution(t *testing.T) {
	client := &fakeLlamaIndexClient{
		invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			if input["goal"] == nil {
				t.Fatal("goal should be set")
			}
			return map[string]any{
				"answer": "test response",
			}, nil
		},
	}

	adapter := &LlamaIndexNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "test query", Results: map[string]any{}}

	result, err := adapter.runNode(context.Background(), "li1", map[string]any{}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["li1"] == nil {
		t.Fatal("result should be in payload")
	}
	resultMap := result.Results["li1"].(map[string]any)
	if resultMap["answer"] != "test response" {
		t.Fatalf("answer = %v, want test response", resultMap["answer"])
	}
	if client.Calls() != 1 {
		t.Fatalf("invoke calls = %d, want 1", client.Calls())
	}
}

// TestLlamaIndexNodeAdapter_StreamMode 测试流式输出
func TestLlamaIndexNodeAdapter_StreamMode(t *testing.T) {
	chunkReceived := false
	client := &fakeLlamaIndexClient{
		streamFunc: func(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
			if onChunk != nil {
				err := onChunk(map[string]any{"chunk": "part1"})
				if err != nil {
					return err
				}
				err = onChunk(map[string]any{"chunk": "part2"})
				if err != nil {
					return err
				}
				chunkReceived = true
			}
			return nil
		},
	}

	err := client.Stream(context.Background(), map[string]any{"goal": "test"}, func(chunk map[string]any) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Stream error = %v", err)
	}
	if !chunkReceived {
		t.Fatal("chunks should be received")
	}
}

// TestLlamaIndexNodeAdapter_StateManagement 测试状态管理
func TestLlamaIndexNodeAdapter_StateManagement(t *testing.T) {
	client := &fakeLlamaIndexClient{
		stateFunc: func(ctx context.Context, sessionID string) (map[string]any, error) {
			return map[string]any{
				"session_id": sessionID,
				"status":     "active",
				"history":    []string{"msg1", "msg2"},
			}, nil
		},
	}

	state, err := client.GetState(context.Background(), "session-123")
	if err != nil {
		t.Fatalf("GetState error = %v", err)
	}
	if state["session_id"] != "session-123" {
		t.Fatalf("session_id = %v, want session-123", state["session_id"])
	}
	if state["status"] != "active" {
		t.Fatalf("status = %v, want active", state["status"])
	}
}

// TestLlamaIndexNodeAdapter_ErrorMapping 测试错误映射
func TestLlamaIndexNodeAdapter_ErrorMapping(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "retryable error wrapped",
			err:  &LlamaIndexError{Code: LlamaIndexErrorRetryable, Message: "temporary"},
			assertErr: func(t *testing.T, err error) {
				// LlamaIndex 适配器将错误包装为 LlamaIndexError 返回
				var liErr *LlamaIndexError
				if !errors.As(err, &liErr) {
					t.Fatalf("error = %v, want LlamaIndexError", err)
				}
			},
		},
		{
			name: "permanent error wrapped",
			err:  &LlamaIndexError{Code: LlamaIndexErrorPermanent, Message: "invalid query"},
			assertErr: func(t *testing.T, err error) {
				var liErr *LlamaIndexError
				if !errors.As(err, &liErr) {
					t.Fatalf("error = %v, want LlamaIndexError", err)
				}
			},
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			assertErr: func(t *testing.T, err error) {
				// 通用错误也会被包装为 LlamaIndexError
				var liErr *LlamaIndexError
				if !errors.As(err, &liErr) {
					t.Fatalf("error = %v, want LlamaIndexError", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeLlamaIndexClient{invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
				return nil, tt.err
			}}
			adapter := &LlamaIndexNodeAdapter{Client: client}
			payload := &AgentDAGPayload{Goal: "g", Results: map[string]any{}}
			_, err := adapter.runNode(context.Background(), "li1", map[string]any{}, payload)
			if err == nil {
				t.Fatalf("expected error")
			}
			tt.assertErr(t, err)
		})
	}
}

// TestLlamaIndexNodeAdapter_WithEffectStore 测试 EffectStore 重放
func TestLlamaIndexNodeAdapter_WithEffectStore(t *testing.T) {
	ctx := context.Background()
	jobID := "job-li-effect"

	effectStore := NewEffectStoreMem()

	// 先写入 effect
	effectRecord := &EffectRecord{
		JobID:     jobID,
		CommandID: "li1",
		Kind:      EffectKindTool,
		Input:     []byte(`{"goal":"test"}`),
		Output:    []byte(`{"cached":true}`),
		Metadata:  map[string]any{"adapter": "llamaindex"},
	}
	if err := effectStore.PutEffect(ctx, effectRecord); err != nil {
		t.Fatalf("PutEffect error = %v", err)
	}

	// 创建 client（应该不会被调用）
	client := &fakeLlamaIndexClient{
		invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			t.Fatal("client should not be called when effect exists")
			return nil, nil
		},
	}

	adapter := &LlamaIndexNodeAdapter{
		Client:      client,
		EffectStore: effectStore,
	}

	// 模拟带 jobID 的 context
	ctx = WithJobID(ctx, jobID)
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(ctx, "li1", map[string]any{}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	// 验证结果来自 effect store
	resultMap := result.Results["li1"].(map[string]any)
	if resultMap["cached"] != true {
		t.Fatalf("result should be from effect store, got %v", resultMap)
	}
	if client.Calls() != 0 {
		t.Fatalf("client should not be called, got %d calls", client.Calls())
	}
}

// TestLlamaIndexNodeAdapter_ClientNotConfigured 测试未配置 client 的情况
func TestLlamaIndexNodeAdapter_ClientNotConfigured(t *testing.T) {
	adapter := &LlamaIndexNodeAdapter{Client: nil}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	_, err := adapter.runNode(context.Background(), "li1", map[string]any{}, payload)
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
	if err.Error() != "LlamaIndexNodeAdapter: Client not configured" {
		t.Fatalf("error = %v, want 'Client not configured'", err)
	}
}

// TestLlamaIndexNodeAdapter_ToDAGNode 测试 ToDAGNode 方法
func TestLlamaIndexNodeAdapter_ToDAGNode(t *testing.T) {
	client := &fakeLlamaIndexClient{}
	adapter := &LlamaIndexNodeAdapter{Client: client}

	task := &planner.TaskNode{
		ID:   "li1",
		Type: planner.NodeLlamaIndex,
		Config: map[string]any{
			"index_name": "my-index",
		},
	}

	// 测试 ToDAGNode 返回非 nil
	node, err := adapter.ToDAGNode(task, &runtime.Agent{ID: "a1"})
	if err != nil {
		t.Fatalf("ToDAGNode error = %v", err)
	}
	if node == nil {
		t.Fatal("node should not be nil")
	}
}

// TestLlamaIndexNodeAdapter_ToNodeRunner 测试 ToNodeRunner 方法
func TestLlamaIndexNodeAdapter_ToNodeRunner(t *testing.T) {
	client := &fakeLlamaIndexClient{}
	adapter := &LlamaIndexNodeAdapter{Client: client}

	task := &planner.TaskNode{
		ID:   "li1",
		Type: planner.NodeLlamaIndex,
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
	if result.Results["li1"] == nil {
		t.Fatal("result should be in payload")
	}
}

// TestLlamaIndexNodeAdapter_InputConfig 测试输入配置
func TestLlamaIndexNodeAdapter_InputConfig(t *testing.T) {
	inputReceived := make(map[string]any)
	client := &fakeLlamaIndexClient{
		invokeFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			inputReceived = input
			return map[string]any{"result": "ok"}, nil
		},
	}

	adapter := &LlamaIndexNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "original goal", Results: map[string]any{"prev": "result"}}

	config := map[string]any{
		"input": map[string]any{
			"custom_key": "custom_value",
		},
	}
	result, err := adapter.runNode(context.Background(), "li1", config, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	// 验证配置中的 input 被合并到请求中
	if inputReceived["custom_key"] != "custom_value" {
		t.Fatalf("custom_key = %v, want custom_value", inputReceived["custom_key"])
	}
	// 验证原始 goal 和 results 也在
	if inputReceived["goal"] != "original goal" {
		t.Fatalf("goal = %v, want original goal", inputReceived["goal"])
	}
	if inputReceived["results"] == nil {
		t.Fatal("results should be in input")
	}
	_ = result
}
