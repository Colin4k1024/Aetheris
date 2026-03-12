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

// fakeBedrockClient Bedrock Client 的 mock 实现
type fakeBedrockClient struct {
	mu                sync.Mutex
	invokeCnt         int
	createSessionFunc func(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error)
	invokeFunc        func(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error)
	streamFunc        func(ctx context.Context, agentID, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error
	sessionFunc       func(ctx context.Context, agentID, sessionID string) (map[string]any, error)
}

func (f *fakeBedrockClient) CreateAgentSession(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error) {
	if f.createSessionFunc != nil {
		return f.createSessionFunc(ctx, agentID, sessionConfig)
	}
	return "session-" + agentID, nil
}

func (f *fakeBedrockClient) Invoke(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error) {
	f.mu.Lock()
	f.invokeCnt++
	fn := f.invokeFunc
	f.mu.Unlock()
	if fn == nil {
		return map[string]any{"response": "ok", "session_id": sessionID}, nil
	}
	return fn(ctx, agentID, sessionID, input)
}

func (f *fakeBedrockClient) InvokeWithResponseStream(ctx context.Context, agentID, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error {
	if f.streamFunc != nil {
		return f.streamFunc(ctx, agentID, sessionID, input, onChunk)
	}
	if onChunk != nil {
		return onChunk(map[string]any{"chunk": "data", "session_id": sessionID})
	}
	return nil
}

func (f *fakeBedrockClient) GetAgentSession(ctx context.Context, agentID, sessionID string) (map[string]any, error) {
	if f.sessionFunc != nil {
		return f.sessionFunc(ctx, agentID, sessionID)
	}
	return map[string]any{
		"session_id": sessionID,
		"agent_id":   agentID,
		"status":     "active",
	}, nil
}

func (f *fakeBedrockClient) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.invokeCnt
}

// TestBedrockNodeAdapter_SessionManagement 测试 Session 管理
func TestBedrockNodeAdapter_SessionManagement(t *testing.T) {
	createdSession := ""
	client := &fakeBedrockClient{
		createSessionFunc: func(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error) {
			createdSession = "new-session-123"
			return createdSession, nil
		},
		invokeFunc: func(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error) {
			return map[string]any{
				"response":  "test response",
				"session":  sessionID,
				"agent_id": agentID,
			}, nil
		},
	}

	adapter := &BedrockNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "test query", Results: map[string]any{}}
	config := map[string]any{
		"agent_id": "my-agent",
	}

	result, err := adapter.runNode(context.Background(), "br1", config, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["br1"] == nil {
		t.Fatal("result should be in payload")
	}
	resultMap := result.Results["br1"].(map[string]any)
	if resultMap["response"] != "test response" {
		t.Fatalf("response = %v, want test response", resultMap["response"])
	}
	if createdSession != "new-session-123" {
		t.Fatalf("session not created, got %s", createdSession)
	}
}

// TestBedrockNodeAdapter_ReuseExistingSession 测试重用已有 Session
func TestBedrockNodeAdapter_ReuseExistingSession(t *testing.T) {
	createSessionCalled := false
	client := &fakeBedrockClient{
		createSessionFunc: func(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error) {
			createSessionCalled = true
			return "new-session", nil
		},
		invokeFunc: func(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error) {
			return map[string]any{
				"response":  "test response",
				"session":  sessionID,
				"agent_id": agentID,
			}, nil
		},
	}

	adapter := &BedrockNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "test query", Results: map[string]any{}}
	config := map[string]any{
		"agent_id":   "my-agent",
		"session_id": "existing-session",
	}

	result, err := adapter.runNode(context.Background(), "br1", config, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["br1"] == nil {
		t.Fatal("result should be in payload")
	}
	// 验证没有创建新 session
	if createSessionCalled {
		t.Fatal("should not create new session when session_id is provided")
	}
	resultMap := result.Results["br1"].(map[string]any)
	if resultMap["session"] != "existing-session" {
		t.Fatalf("session = %v, want existing-session", resultMap["session"])
	}
}

// TestBedrockNodeAdapter_StreamResponse 测试流式响应
func TestBedrockNodeAdapter_StreamResponse(t *testing.T) {
	chunksReceived := 0
	client := &fakeBedrockClient{
		streamFunc: func(ctx context.Context, agentID, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error {
			if onChunk != nil {
				_ = onChunk(map[string]any{"chunk": "part1"})
				_ = onChunk(map[string]any{"chunk": "part2"})
				_ = onChunk(map[string]any{"chunk": "part3"})
				chunksReceived = 3
			}
			return nil
		},
	}

	err := client.InvokeWithResponseStream(context.Background(), "agent-1", "session-1", map[string]any{"goal": "test"}, func(chunk map[string]any) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Stream error = %v", err)
	}
	if chunksReceived != 3 {
		t.Fatalf("chunksReceived = %d, want 3", chunksReceived)
	}
}

// TestBedrockNodeAdapter_ErrorHandling 测试错误处理
func TestBedrockNodeAdapter_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "retryable",
			err:  &BedrockError{Code: BedrockErrorRetryable, Message: "throttled"},
			assertErr: func(t *testing.T, err error) {
				var be *BedrockError
				if !errors.As(err, &be) {
					t.Fatalf("error = %v, want BedrockError", err)
				}
			},
		},
		{
			name: "permanent",
			err:  &BedrockError{Code: BedrockErrorPermanent, Message: "invalid agent"},
			assertErr: func(t *testing.T, err error) {
				var be *BedrockError
				if !errors.As(err, &be) {
					t.Fatalf("error = %v, want BedrockError", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeBedrockClient{
				createSessionFunc: func(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error) {
					return "session-1", nil
				},
				invokeFunc: func(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error) {
					return nil, tt.err
				},
			}
			adapter := &BedrockNodeAdapter{Client: client}
			payload := &AgentDAGPayload{Goal: "g", Results: map[string]any{}}
			_, err := adapter.runNode(context.Background(), "br1", map[string]any{"agent_id": "my-agent"}, payload)
			if err == nil {
				t.Fatalf("expected error")
			}
			tt.assertErr(t, err)
		})
	}
}

// TestBedrockNodeAdapter_WithEffectStore 测试 EffectStore 重放
func TestBedrockNodeAdapter_WithEffectStore(t *testing.T) {
	ctx := context.Background()
	jobID := "job-br-effect"

	effectStore := NewEffectStoreMem()

	// 先写入 effect
	effectRecord := &EffectRecord{
		JobID:     jobID,
		CommandID: "br1",
		Kind:      EffectKindTool,
		Input:     []byte(`{"goal":"test"}`),
		Output:    []byte(`{"cached":true}`),
		Metadata:  map[string]any{"adapter": "bedrock"},
	}
	if err := effectStore.PutEffect(ctx, effectRecord); err != nil {
		t.Fatalf("PutEffect error = %v", err)
	}

	// 创建 client（应该不会被调用）
	client := &fakeBedrockClient{
		invokeFunc: func(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error) {
			t.Fatal("client should not be called when effect exists")
			return nil, nil
		},
	}

	adapter := &BedrockNodeAdapter{
		Client:      client,
		EffectStore: effectStore,
	}

	// 模拟带 jobID 的 context
	ctx = WithJobID(ctx, jobID)
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(ctx, "br1", map[string]any{"agent_id": "my-agent"}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	// 验证结果来自 effect store
	resultMap := result.Results["br1"].(map[string]any)
	if resultMap["cached"] != true {
		t.Fatalf("result should be from effect store, got %v", resultMap)
	}
	if client.Calls() != 0 {
		t.Fatalf("client should not be called, got %d calls", client.Calls())
	}
}

// TestBedrockNodeAdapter_ClientNotConfigured 测试未配置 client 的情况
func TestBedrockNodeAdapter_ClientNotConfigured(t *testing.T) {
	adapter := &BedrockNodeAdapter{Client: nil}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	_, err := adapter.runNode(context.Background(), "br1", map[string]any{"agent_id": "my-agent"}, payload)
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
	if err.Error() != "BedrockNodeAdapter: Client not configured" {
		t.Fatalf("error = %v, want 'Client not configured'", err)
	}
}

// TestBedrockNodeAdapter_GetSession 测试获取 Session 状态
func TestBedrockNodeAdapter_GetSession(t *testing.T) {
	client := &fakeBedrockClient{
		sessionFunc: func(ctx context.Context, agentID, sessionID string) (map[string]any, error) {
			return map[string]any{
				"session_id": sessionID,
				"agent_id":   agentID,
				"status":     "active",
				"history":    []string{"msg1"},
			}, nil
		},
	}

	session, err := client.GetAgentSession(context.Background(), "agent-1", "session-1")
	if err != nil {
		t.Fatalf("GetAgentSession error = %v", err)
	}
	if session["session_id"] != "session-1" {
		t.Fatalf("session_id = %v, want session-1", session["session_id"])
	}
	if session["status"] != "active" {
		t.Fatalf("status = %v, want active", session["status"])
	}
}

// TestBedrockNodeAdapter_ToDAGNode 测试 ToDAGNode 方法
func TestBedrockNodeAdapter_ToDAGNode(t *testing.T) {
	client := &fakeBedrockClient{}
	adapter := &BedrockNodeAdapter{Client: client}

	task := &planner.TaskNode{
		ID:   "br1",
		Type: planner.NodeBedrock,
		Config: map[string]any{
			"agent_id": "my-agent",
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

// TestBedrockNodeAdapter_ToNodeRunner 测试 ToNodeRunner 方法
func TestBedrockNodeAdapter_ToNodeRunner(t *testing.T) {
	client := &fakeBedrockClient{}
	adapter := &BedrockNodeAdapter{Client: client}

	task := &planner.TaskNode{
		ID:   "br1",
		Type: planner.NodeBedrock,
		Config: map[string]any{
			"agent_id": "my-agent",
		},
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
	if result.Results["br1"] == nil {
		t.Fatal("result should be in payload")
	}
}

// TestBedrockNodeAdapter_NormalExecution 测试正常执行流程
func TestBedrockNodeAdapter_NormalExecution(t *testing.T) {
	client := &fakeBedrockClient{}

	adapter := &BedrockNodeAdapter{Client: client}
	payload := &AgentDAGPayload{Goal: "test query", Results: map[string]any{}}
	config := map[string]any{
		"agent_id": "my-agent",
	}

	result, err := adapter.runNode(context.Background(), "br1", config, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["br1"] == nil {
		t.Fatal("result should be in payload")
	}
	resultMap := result.Results["br1"].(map[string]any)
	if resultMap["response"] != "ok" {
		t.Fatalf("response = %v, want ok", resultMap["response"])
	}
}
