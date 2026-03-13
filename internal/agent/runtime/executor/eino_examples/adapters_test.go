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

package eino_examples

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cloudwego/eino/schema"

	"rag-platform/internal/agent/runtime/executor"
)

// mockChatModel 用于测试的 Mock ChatModel
type mockChatModel struct {
	mu          sync.Mutex
	generateCnt int
	response    *schema.Message
	err         error
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error) {
	m.mu.Lock()
	m.generateCnt++
	mr := m.response
	me := m.err
	m.mu.Unlock()

	if me != nil {
		return nil, me
	}
	if mr != nil {
		return mr, nil
	}
	return &schema.Message{
		Content: "mock response",
	}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

func (m *mockChatModel) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateCnt
}

// TestReactAgentAdapter_Invoke 测试 ReAct Agent 执行
func TestReactAgentAdapter_Invoke(t *testing.T) {
	model := &mockChatModel{
		response: &schema.Message{Content: "test response"},
	}
	adapter := NewReactAgentAdapter(model, nil)

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test prompt",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["response"] != "test response" {
		t.Fatalf("response = %v, want test response", result["response"])
	}
	if model.Calls() != 1 {
		t.Fatalf("model calls = %d, want 1", model.Calls())
	}
}

// TestReactAgentAdapter_ModelNotConfigured 测试未配置模型的情况
func TestReactAgentAdapter_ModelNotConfigured(t *testing.T) {
	adapter := NewReactAgentAdapter(nil, nil)

	_, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test",
	})
	if err == nil {
		t.Fatal("expected error when model is nil")
	}
	if err.Error() != "ReactAgentAdapter: Model not configured" {
		t.Fatalf("error = %v, want 'Model not configured'", err)
	}
}

// TestReactAgentAdapter_GetState 测试获取状态
func TestReactAgentAdapter_GetState(t *testing.T) {
	adapter := NewReactAgentAdapter(nil, nil)

	state, err := adapter.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState error = %v", err)
	}
	if state["status"] != "ready" {
		t.Fatalf("status = %v, want ready", state["status"])
	}
}

// TestDEERAgentAdapter_Invoke 测试 DEER-Go Agent 执行
func TestDEERAgentAdapter_Invoke(t *testing.T) {
	model := &mockChatModel{
		response: &schema.Message{Content: "deer response"},
	}
	adapter := NewDEERAgentAdapter(model, nil)

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test prompt",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["response"] != "deer response" {
		t.Fatalf("response = %v, want deer response", result["response"])
	}
}

// TestDEERAgentAdapter_ModelNotConfigured 测试未配置模型
func TestDEERAgentAdapter_ModelNotConfigured(t *testing.T) {
	adapter := NewDEERAgentAdapter(nil, nil)

	_, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test",
	})
	if err == nil {
		t.Fatal("expected error when model is nil")
	}
}

// TestManusAgentAdapter_Invoke 测试 Manus Agent 执行
func TestManusAgentAdapter_Invoke(t *testing.T) {
	model := &mockChatModel{
		response: &schema.Message{Content: "manus response"},
	}
	adapter := NewManusAgentAdapter(model, nil)

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test prompt",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["response"] != "manus response" {
		t.Fatalf("response = %v, want manus response", result["response"])
	}
}

// TestManusAgentAdapter_ModelNotConfigured 测试未配置模型
func TestManusAgentAdapter_ModelNotConfigured(t *testing.T) {
	adapter := NewManusAgentAdapter(nil, nil)

	_, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test",
	})
	if err == nil {
		t.Fatal("expected error when model is nil")
	}
}

// TestADKAdapter_Invoke 测试 ADK 执行
func TestADKAdapter_Invoke(t *testing.T) {
	adapter := NewADKAdapter(nil, nil)

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["response"] != "ADK response" {
		t.Fatalf("response = %v, want ADK response", result["response"])
	}
}

// TestADKAdapter_GetState 测试 ADK 状态
func TestADKAdapter_GetState(t *testing.T) {
	adapter := NewADKAdapter(nil, nil)

	state, err := adapter.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState error = %v", err)
	}
	if state["status"] != "ready" {
		t.Fatalf("status = %v, want ready", state["status"])
	}
}

// TestChainAdapter_Invoke 测试 Chain 执行
func TestChainAdapter_Invoke(t *testing.T) {
	adapter := NewChainAdapter()

	// 添加一个简单的节点
	adapter.AddNode("step1", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"result": "step1 done"}, nil
	})

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"input": "test",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["input"] != "test" {
		t.Fatalf("input = %v, want test", result["input"])
	}
}

// TestChainAdapter_GetState 测试 Chain 状态
func TestChainAdapter_GetState(t *testing.T) {
	adapter := NewChainAdapter()

	state, err := adapter.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState error = %v", err)
	}
	if state["status"] != "ready" {
		t.Fatalf("status = %v, want ready", state["status"])
	}
}

// TestGraphAdapter_Invoke 测试 Graph 执行
func TestGraphAdapter_Invoke(t *testing.T) {
	adapter := NewGraphAdapter()

	adapter.AddNode("node1", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"result": "node1 done"}, nil
	})
	adapter.AddEdge("node1", "node2")
	adapter.SetEntry("node1")

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"input": "test",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["input"] != "test" {
		t.Fatalf("input = %v, want test", result["input"])
	}
}

// TestGraphAdapter_Edges 测试 Graph 边
func TestGraphAdapter_Edges(t *testing.T) {
	adapter := NewGraphAdapter()

	adapter.AddEdge("node1", "node2")
	adapter.AddEdge("node2", "node3")

	if len(adapter.Edges) != 2 {
		t.Fatalf("edges len = %d, want 2", len(adapter.Edges))
	}
}

// TestWorkflowAdapter_Invoke 测试 Workflow 执行
func TestWorkflowAdapter_Invoke(t *testing.T) {
	adapter := NewWorkflowAdapter()

	adapter.AddNode("step1", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"result": "step1 done"}, nil
	})

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"input": "test",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["input"] != "test" {
		t.Fatalf("input = %v, want test", result["input"])
	}
}

// TestWorkflowAdapter_GetState 测试 Workflow 状态
func TestWorkflowAdapter_GetState(t *testing.T) {
	adapter := NewWorkflowAdapter()

	state, err := adapter.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState error = %v", err)
	}
	if state["status"] != "ready" {
		t.Fatalf("status = %v, want ready", state["status"])
	}
}

// TestToNodeRunner 测试转换为 NodeRunner
func TestToNodeRunner(t *testing.T) {
	adapter := NewReactAgentAdapter(&mockChatModel{
		response: &schema.Message{Content: "test response"},
	}, nil)

	runner := ToNodeRunner(adapter)

	payload := &executor.AgentDAGPayload{
		Goal:    "test goal",
		Results: make(map[string]any),
	}

	result, err := runner(context.Background(), payload)
	if err != nil {
		t.Fatalf("runner error = %v", err)
	}
	if result.Results["eino"] == nil {
		t.Fatal("result should have eino key")
	}
}

// TestConvertToPlannerTaskNode 测试转换为 TaskNode
func TestConvertToPlannerTaskNode(t *testing.T) {
	adapter := NewReactAgentAdapter(nil, nil)
	config := map[string]any{
		"model": "test-model",
	}

	node := ConvertToPlannerTaskNode(adapter, "react", config)
	if node == nil {
		t.Fatal("node should not be nil")
	}
	if node.Type != "react" {
		t.Fatalf("type = %v, want react", node.Type)
	}
	if node.Config["model"] != "test-model" {
		t.Fatalf("config model = %v, want test-model", node.Config["model"])
	}
}

// TestJSONMarshal 测试 JSON 序列化
func TestJSONMarshal(t *testing.T) {
	data := map[string]any{"key": "value"}
	result := JSONMarshal(data)
	if string(result) != `{"key":"value"}` {
		t.Fatalf("result = %s, want {\"key\":\"value\"}", string(result))
	}
}
