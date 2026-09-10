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
	"fmt"
	"sync"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
)

// mockChatModel 用于测试的 Mock ChatModel
type mockChatModel struct {
	mu             sync.Mutex
	generateCnt    int
	response       *schema.Message
	secondResponse *schema.Message // returned on second+ calls
	err            error
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error) {
	m.mu.Lock()
	m.generateCnt++
	cnt := m.generateCnt
	mr := m.response
	mr2 := m.secondResponse
	me := m.err
	m.mu.Unlock()

	if me != nil {
		return nil, me
	}
	if cnt > 1 && mr2 != nil {
		return mr2, nil
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
// mockADKAgent implements ADKInvokable and ADKStreamable for testing.
type mockADKAgent struct {
	invokeResult map[string]any
	invokeErr    error
	streamChunks []map[string]any
}

func (m *mockADKAgent) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if m.invokeErr != nil {
		return nil, m.invokeErr
	}
	if m.invokeResult != nil {
		return m.invokeResult, nil
	}
	return map[string]any{"response": "real response"}, nil
}

func (m *mockADKAgent) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	for _, chunk := range m.streamChunks {
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// mockADKCheckpoint implements ADKStateful for testing.
type mockADKCheckpoint struct {
	state map[string]any
}

func (m *mockADKCheckpoint) GetState(ctx context.Context) (map[string]any, error) {
	if m.state != nil {
		return m.state, nil
	}
	return map[string]any{"status": "running"}, nil
}

func TestADKAdapter_Invoke(t *testing.T) {
	agent := &mockADKAgent{invokeResult: map[string]any{"response": "real response"}}
	adapter := NewADKAdapter(agent, nil)

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "test",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["response"] != "real response" {
		t.Fatalf("response = %v, want real response", result["response"])
	}
}

func TestADKAdapter_Invoke_NilAgent(t *testing.T) {
	adapter := NewADKAdapter(nil, nil)
	_, err := adapter.Invoke(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for nil agent")
	}
}

func TestADKAdapter_Invoke_NonInvokableAgent(t *testing.T) {
	adapter := NewADKAdapter("not-an-agent", nil)
	_, err := adapter.Invoke(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for non-invokable agent")
	}
}

func TestADKAdapter_Stream(t *testing.T) {
	agent := &mockADKAgent{
		streamChunks: []map[string]any{
			{"delta": "chunk1"},
			{"delta": "chunk2"},
		},
	}
	adapter := NewADKAdapter(agent, nil)

	var chunks []string
	err := adapter.Stream(context.Background(), map[string]any{}, func(chunk map[string]any) error {
		if delta, ok := chunk["delta"].(string); ok {
			chunks = append(chunks, delta)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Stream error = %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestADKAdapter_Stream_NilAgent(t *testing.T) {
	adapter := NewADKAdapter(nil, nil)
	err := adapter.Stream(context.Background(), map[string]any{}, func(chunk map[string]any) error { return nil })
	if err == nil {
		t.Fatal("expected error for nil agent in Stream")
	}
}

// TestADKAdapter_GetState 测试 ADK 状态
func TestADKAdapter_GetState(t *testing.T) {
	checkpoint := &mockADKCheckpoint{state: map[string]any{"status": "running"}}
	adapter := NewADKAdapter(nil, checkpoint)

	state, err := adapter.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState error = %v", err)
	}
	if state["status"] != "running" {
		t.Fatalf("status = %v, want running", state["status"])
	}
}

func TestADKAdapter_GetState_NilCheckpoint(t *testing.T) {
	adapter := NewADKAdapter(nil, nil)
	_, err := adapter.GetState(context.Background())
	if err == nil {
		t.Fatal("expected error for nil checkpoint")
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

// --- Tool Execution Tests ---

// mockTool implements ToolExecutor for testing.
type mockTool struct {
	name      string
	execFn    func(ctx context.Context, args map[string]any) (string, error)
	callCount int
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	m.callCount++
	if m.execFn != nil {
		return m.execFn(ctx, args)
	}
	return "default result", nil
}
func (m *mockTool) Calls() int { return m.callCount }

func TestReactAgentAdapter_ToolExecution(t *testing.T) {
	tool := &mockTool{
		name: "calculator",
		execFn: func(ctx context.Context, args map[string]any) (string, error) {
			return "result: 42", nil
		},
	}
	model := &mockChatModel{
		response: &schema.Message{
			Role:    schema.Assistant,
			Content: "I'll calculate that",
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "calculator",
						Arguments: `{"expr": "6*7"}`,
					},
				},
			},
		},
		secondResponse: &schema.Message{
			Role:    schema.Assistant,
			Content: "The answer is 42",
		},
	}

	adapter := NewReactAgentAdapter(model, []interface{}{tool})
	_, err := adapter.Invoke(context.Background(), map[string]any{"prompt": "What is 6*7?"})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}

	if tool.Calls() != 1 {
		t.Fatalf("expected tool to be called 1 time, got %d", tool.Calls())
	}
}

func TestReactAgentAdapter_ToolNotFound(t *testing.T) {
	model := &mockChatModel{
		response: &schema.Message{
			Role:    schema.Assistant,
			Content: "Using unknown tool",
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "nonexistent",
						Arguments: `{"x": 1}`,
					},
				},
			},
		},
		secondResponse: &schema.Message{
			Role:    schema.Assistant,
			Content: "I could not find the tool",
		},
	}

	adapter := NewReactAgentAdapter(model, nil) // no tools
	result, err := adapter.Invoke(context.Background(), map[string]any{"prompt": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The adapter should complete, with the tool error recorded in history
	if result["response"] == nil {
		t.Error("expected a response")
	}
}

func TestReactAgentAdapter_ToolBadArgs(t *testing.T) {
	tool := &mockTool{
		name:   "calculator",
		execFn: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	}
	model := &mockChatModel{
		response: &schema.Message{
			Role:    schema.Assistant,
			Content: "Calling with bad args",
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "calculator",
						Arguments: `{bad json`,
					},
				},
			},
		},
		secondResponse: &schema.Message{
			Role:    schema.Assistant,
			Content: "Bad arguments provided",
		},
	}

	adapter := NewReactAgentAdapter(model, []interface{}{tool})
	result, err := adapter.Invoke(context.Background(), map[string]any{"prompt": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The tool should not have been called (bad args caught before Execute)
	if tool.Calls() != 0 {
		t.Errorf("expected 0 tool calls (bad args caught), got %d", tool.Calls())
	}
	_ = result
}

func TestManusAgentAdapter_ToolExecution(t *testing.T) {
	tool := &mockTool{
		name: "search",
		execFn: func(ctx context.Context, args map[string]any) (string, error) {
			query, _ := args["query"].(string)
			return "search results for: " + query, nil
		},
	}
	model := &mockChatModel{
		response: &schema.Message{
			Role:    schema.Assistant,
			Content: "Searching...",
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "search",
						Arguments: `{"query": "test"}`,
					},
				},
			},
		},
		secondResponse: &schema.Message{
			Role:    schema.Assistant,
			Content: "Search complete",
		},
	}

	adapter := NewManusAgentAdapter(model, []interface{}{tool})
	_, err := adapter.Invoke(context.Background(), map[string]any{"prompt": "Search for test"})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}

	if tool.Calls() != 1 {
		t.Fatalf("expected tool to be called 1 time, got %d", tool.Calls())
	}
}

func TestManusAgentAdapter_ToolNotFound(t *testing.T) {
	model := &mockChatModel{
		response: &schema.Message{
			Role:    schema.Assistant,
			Content: "Using unknown tool",
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "nonexistent",
						Arguments: `{"x": 1}`,
					},
				},
			},
		},
		secondResponse: &schema.Message{
			Role:    schema.Assistant,
			Content: "Tool not available",
		},
	}

	adapter := NewManusAgentAdapter(model, nil)
	result, err := adapter.Invoke(context.Background(), map[string]any{"prompt": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["response"] == nil {
		t.Error("expected a response")
	}
}

func TestExecuteToolFromList_Success(t *testing.T) {
	tool := &mockTool{
		name: "test_tool",
		execFn: func(ctx context.Context, args map[string]any) (string, error) {
			return "executed: " + args["input"].(string), nil
		},
	}

	result, err := executeToolFromList(context.Background(), []interface{}{tool}, "test_tool", `{"input":"hello"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "executed: hello" {
		t.Errorf("expected 'executed: hello', got %s", result)
	}
}

func TestExecuteToolFromList_NotFound(t *testing.T) {
	_, err := executeToolFromList(context.Background(), nil, "nonexistent", `{}`)
	if err == nil {
		t.Error("expected error for tool not found")
	}
}

func TestExecuteToolFromList_BadJSON(t *testing.T) {
	tool := &mockTool{name: "test_tool"}
	_, err := executeToolFromList(context.Background(), []interface{}{tool}, "test_tool", `{bad json`)
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestExecuteToolFromList_NilTool(t *testing.T) {
	tool := &mockTool{name: "real_tool"}
	tools := []interface{}{nil, tool}
	_, err := executeToolFromList(context.Background(), tools, "real_tool", `{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteToolFromList_NotToolExecutor(t *testing.T) {
	// Non-ToolExecutor in the list should be skipped, not crash
	tools := []interface{}{"not-a-tool", 42}
	_, err := executeToolFromList(context.Background(), tools, "nonexistent", `{}`)
	if err == nil {
		t.Error("expected error for tool not found")
	}
}

func TestExecuteToolFromList_EmptyArgs(t *testing.T) {
	tool := &mockTool{
		name: "no_args_tool",
		execFn: func(ctx context.Context, args map[string]any) (string, error) {
			if len(args) == 0 {
				return "no args needed", nil
			}
			return "has args", nil
		},
	}

	result, err := executeToolFromList(context.Background(), []interface{}{tool}, "no_args_tool", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "no args needed" {
		t.Errorf("expected 'no args needed', got %s", result)
	}
}

// TestGraphAdapter_DiamondDAG 测试菱形 DAG：A→B, A→C, B→D, C→D
func TestGraphAdapter_DiamondDAG(t *testing.T) {
	adapter := NewGraphAdapter()

	adapter.AddNode("a", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"a_val": "from_a"}, nil
	})
	adapter.AddNode("b", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"b_val": "from_b"}, nil
	})
	adapter.AddNode("c", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"c_val": "from_c"}, nil
	})
	adapter.AddNode("d", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"d_val": "from_d"}, nil
	})

	adapter.AddEdge("a", "b")
	adapter.AddEdge("a", "c")
	adapter.AddEdge("b", "d")
	adapter.AddEdge("c", "d")
	adapter.SetEntry("a")

	result, err := adapter.Invoke(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["a_val"] != "from_a" {
		t.Errorf("a_val = %v, want from_a", result["a_val"])
	}
	if result["b_val"] != "from_b" {
		t.Errorf("b_val = %v, want from_b", result["b_val"])
	}
	if result["c_val"] != "from_c" {
		t.Errorf("c_val = %v, want from_c", result["c_val"])
	}
	if result["d_val"] != "from_d" {
		t.Errorf("d_val = %v, want from_d", result["d_val"])
	}
}

// TestGraphAdapter_LinearChain 测试线性链 A→B→C
func TestGraphAdapter_LinearChain(t *testing.T) {
	adapter := NewGraphAdapter()

	adapter.AddNode("a", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"step": "a"}, nil
	})
	adapter.AddNode("b", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"step": "b"}, nil
	})
	adapter.AddNode("c", func(ctx context.Context, input any) (any, error) {
		return map[string]any{"step": "c"}, nil
	})

	adapter.AddEdge("a", "b")
	adapter.AddEdge("b", "c")
	adapter.SetEntry("a")

	result, err := adapter.Invoke(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if result["step"] != "c" {
		t.Errorf("step = %v, want c", result["step"])
	}
}

// TestGraphAdapter_FindNextNodes 测试 findNextNodes 返回所有下游节点
func TestGraphAdapter_FindNextNodes(t *testing.T) {
	adapter := NewGraphAdapter()
	adapter.AddEdge("a", "b")
	adapter.AddEdge("a", "c")
	adapter.AddEdge("a", "d")

	nexts := adapter.findNextNodes("a")
	if len(nexts) != 3 {
		t.Fatalf("findNextNodes len = %d, want 3", len(nexts))
	}
}

// TestGraphAdapter_FindNextNodes_None 测试无下游节点
func TestGraphAdapter_FindNextNodes_None(t *testing.T) {
	adapter := NewGraphAdapter()
	adapter.AddEdge("a", "b")

	nexts := adapter.findNextNodes("b")
	if len(nexts) != 0 {
		t.Fatalf("findNextNodes len = %d, want 0", len(nexts))
	}
}

// TestGraphAdapter_NodeError 测试节点执行错误传播
func TestGraphAdapter_NodeError(t *testing.T) {
	adapter := NewGraphAdapter()

	adapter.AddNode("a", func(ctx context.Context, input any) (any, error) {
		return nil, fmt.Errorf("boom")
	})
	adapter.SetEntry("a")

	_, err := adapter.Invoke(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
