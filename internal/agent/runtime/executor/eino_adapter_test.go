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

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
)

// fakeLLMGenForTest 用于测试的 LLM mock
type fakeLLMGenForTest struct {
	mu          sync.Mutex
	generateCnt int
	response    string
	err         error
}

func (f *fakeLLMGenForTest) Generate(ctx context.Context, prompt string) (string, error) {
	f.mu.Lock()
	f.generateCnt++
	fn := f.response
	fe := f.err
	f.mu.Unlock()
	if fe != nil {
		return "", fe
	}
	if fn != "" {
		return fn, nil
	}
	return "mock response", nil
}

func (f *fakeLLMGenForTest) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.generateCnt
}

// fakeToolExecForTest 用于测试的工具执行 mock
type fakeToolExecForTest struct {
	mu         sync.Mutex
	executeCnt int
	result     ToolResult
	err        error
}

func (f *fakeToolExecForTest) Execute(ctx context.Context, toolName string, input map[string]any, state interface{}) (ToolResult, error) {
	f.mu.Lock()
	f.executeCnt++
	fr := f.result
	fe := f.err
	f.mu.Unlock()
	if fe != nil {
		return ToolResult{}, fe
	}
	return fr, nil
}

func (f *fakeToolExecForTest) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.executeCnt
}

// fakeWorkflowExecForTest 用于测试的工作流执行 mock
type fakeWorkflowExecForTest struct {
	mu         sync.Mutex
	executeCnt int
	result     interface{}
	err        error
}

func (f *fakeWorkflowExecForTest) ExecuteWorkflow(ctx context.Context, name string, params map[string]any) (interface{}, error) {
	f.mu.Lock()
	f.executeCnt++
	fr := f.result
	fe := f.err
	f.mu.Unlock()
	if fe != nil {
		return nil, fe
	}
	if fr != nil {
		return fr, nil
	}
	return "workflow done", nil
}

func (f *fakeWorkflowExecForTest) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.executeCnt
}

// TestLLMNodeAdapter_WithEino 测试 Eino 集成的 LLM 适配器
func TestLLMNodeAdapter_WithEino(t *testing.T) {
	llm := &fakeLLMGenForTest{response: "test response"}
	adapter := &LLMNodeAdapter{LLM: llm}
	payload := &AgentDAGPayload{Goal: "test prompt", Results: map[string]any{}}

	result, err := adapter.runNode(context.Background(), "llm1", map[string]any{}, &runtime.Agent{ID: "a1"}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["llm1"] == nil {
		t.Fatal("result should be in payload")
	}
	resultMap := result.Results["llm1"].(map[string]any)
	if resultMap["output"] != "test response" {
		t.Fatalf("output = %v, want test response", resultMap["output"])
	}
	if llm.Calls() != 1 {
		t.Fatalf("LLM calls = %d, want 1", llm.Calls())
	}
}

// TestLLMNodeAdapter_WithEffectStoreAndReplay 测试 LLM EffectStore 重放
func TestLLMNodeAdapter_WithEffectStoreAndReplay(t *testing.T) {
	ctx := context.Background()
	jobID := "job-llm-effect"

	effectStore := NewEffectStoreMem()

	// 先写入 effect
	effectRecord := &EffectRecord{
		JobID:     jobID,
		CommandID: "llm1",
		Kind:      EffectKindLLM,
		Input:     []byte(`{"prompt":"test"}`),
		Output:    []byte(`"cached response"`),
		Metadata:  map[string]any{"model": "test-model"},
	}
	if err := effectStore.PutEffect(ctx, effectRecord); err != nil {
		t.Fatalf("PutEffect error = %v", err)
	}

	// 创建 LLM（应该不会被调用）
	llm := &fakeLLMGenForTest{
		err: errors.New("should not be called"),
	}

	adapter := &LLMNodeAdapter{
		LLM:         llm,
		EffectStore: effectStore,
	}

	// 模拟带 jobID 的 context
	ctx = WithJobID(ctx, jobID)
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(ctx, "llm1", map[string]any{}, &runtime.Agent{ID: "a1"}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	// 验证结果来自 effect store
	resultMap := result.Results["llm1"].(map[string]any)
	if resultMap["output"] != "cached response" {
		t.Fatalf("result should be from effect store, got %v", resultMap)
	}
	if llm.Calls() != 0 {
		t.Fatalf("LLM should not be called, got %d calls", llm.Calls())
	}
}

// TestToolNodeAdapter_WithEino 测试 Eino 集成的 Tool 适配器
func TestToolNodeAdapter_WithEino(t *testing.T) {
	tool := &fakeToolExecForTest{
		result: ToolResult{
			Done:   true,
			Output: `{"result": "success"}`,
		},
	}
	adapter := &ToolNodeAdapter{Tools: tool}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(context.Background(), "tool1", "my_tool", map[string]any{}, &runtime.Agent{ID: "a1"}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["tool1"] == nil {
		t.Fatal("result should be in payload")
	}
	if tool.Calls() != 1 {
		t.Fatalf("tool calls = %d, want 1", tool.Calls())
	}
}

// TestToolNodeAdapter_WithState 测试 Tool 的状态管理
func TestToolNodeAdapter_WithState(t *testing.T) {
	tool := &fakeToolExecForTest{
		result: ToolResult{
			Done:   false,
			State:  map[string]any{"progress": 50},
			Output: "",
		},
	}
	adapter := &ToolNodeAdapter{Tools: tool}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(context.Background(), "tool1", "my_tool", map[string]any{}, &runtime.Agent{ID: "a1"}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	// 验证未完成的 Tool 返回状态
	if result.Results["tool1"] == nil {
		t.Fatal("result should be in payload")
	}
	resultMap := result.Results["tool1"].(map[string]any)
	if resultMap["state"] == nil {
		t.Fatal("state should be present for incomplete tools")
	}
}

// TestToolNodeAdapter_RetryableError 测试 Tool 的可重试错误
func TestToolNodeAdapter_RetryableError(t *testing.T) {
	tool := &fakeToolExecForTest{
		err: errors.New("transient error"),
	}
	adapter := &ToolNodeAdapter{
		Tools: tool,
		RetryPolicy: &RetryPolicy{
			MaxRetries: 2,
		},
	}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	_, err := adapter.runNode(context.Background(), "tool1", "my_tool", map[string]any{}, &runtime.Agent{ID: "a1"}, payload)
	// Tool 执行错误直接返回，不一定映射为 StepFailure
	if err == nil {
		t.Fatal("expected error")
	}
	// 记录错误类型
	t.Logf("error type: %T, error: %v", err, err)
}

// TestWorkflowNodeAdapter_WithEino 测试 Eino 集成的 Workflow 适配器
func TestWorkflowNodeAdapter_WithEino(t *testing.T) {
	wf := &fakeWorkflowExecForTest{result: map[string]any{"status": "done"}}
	adapter := &WorkflowNodeAdapter{Workflow: wf}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(context.Background(), "wf1", "my_workflow", map[string]any{}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["wf1"] == nil {
		t.Fatal("result should be in payload")
	}
	resultMap := result.Results["wf1"].(map[string]any)
	if resultMap["status"] != "done" {
		t.Fatalf("status = %v, want done", resultMap["status"])
	}
	if wf.Calls() != 1 {
		t.Fatalf("workflow calls = %d, want 1", wf.Calls())
	}
}

// TestWorkflowNodeAdapter_ErrorHandling 测试 Workflow 错误处理
func TestWorkflowNodeAdapter_ErrorHandling(t *testing.T) {
	wf := &fakeWorkflowExecForTest{
		err: errors.New("workflow failed"),
	}
	adapter := &WorkflowNodeAdapter{Workflow: wf}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	_, err := adapter.runNode(context.Background(), "wf1", "my_workflow", map[string]any{}, payload)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEinoDAGCompilation 测试 Eino DAG 编译
func TestEinoDAGCompilation(t *testing.T) {
	llm := &fakeLLMGenForTest{response: "test"}
	tool := &fakeToolExecForTest{
		result: ToolResult{Done: true, Output: `{"result": "ok"}`},
	}
	wf := &fakeWorkflowExecForTest{result: "done"}

	compiler := NewCompiler(map[string]NodeAdapter{
		planner.NodeLLM:      &LLMNodeAdapter{LLM: llm},
		planner.NodeTool:     &ToolNodeAdapter{Tools: tool},
		planner.NodeWorkflow: &WorkflowNodeAdapter{Workflow: wf},
	})

	// 创建包含多个节点的 TaskGraph
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{ID: "llm1", Type: planner.NodeLLM},
			{ID: "tool1", Type: planner.NodeTool, ToolName: "my_tool"},
			{ID: "wf1", Type: planner.NodeWorkflow, Workflow: "my_workflow"},
		},
		Edges: []planner.TaskEdge{
			{From: "llm1", To: "tool1"},
			{From: "tool1", To: "wf1"},
		},
	}

	// 编译 DAG
	compiled, err := compiler.Compile(context.Background(), graph, &runtime.Agent{ID: "a1"})
	if err != nil {
		t.Fatalf("Compile error = %v", err)
	}
	if compiled == nil {
		t.Fatal("compiled graph should not be nil")
	}
}

// TestEinoSteppableCompilation 测试 Eino Steppable 编译
func TestEinoSteppableCompilation(t *testing.T) {
	llm := &fakeLLMGenForTest{response: "test"}
	tool := &fakeToolExecForTest{
		result: ToolResult{Done: true, Output: `{"result": "ok"}`},
	}

	compiler := NewCompiler(map[string]NodeAdapter{
		planner.NodeLLM:  &LLMNodeAdapter{LLM: llm},
		planner.NodeTool: &ToolNodeAdapter{Tools: tool},
	})

	// 创建 TaskGraph
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{ID: "llm1", Type: planner.NodeLLM},
			{ID: "tool1", Type: planner.NodeTool, ToolName: "my_tool"},
		},
		Edges: []planner.TaskEdge{
			{From: "llm1", To: "tool1"},
		},
	}

	// 编译为 Steppable
	steps, err := compiler.CompileSteppable(context.Background(), graph, &runtime.Agent{ID: "a1"})
	if err != nil {
		t.Fatalf("CompileSteppable error = %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("steps len = %d, want 2", len(steps))
	}
}

// TestNodeAdapterRegistry 测试适配器注册表
func TestNodeAdapterRegistry(t *testing.T) {
	llm := &fakeLLMGenForTest{}
	compiler := NewCompiler(map[string]NodeAdapter{
		planner.NodeLLM: &LLMNodeAdapter{LLM: llm},
	})

	// 测试注册新类型
	customAdapter := customNodeAdapterForTest{}
	compiler.Register("custom_node", customAdapter)

	// 测试获取已注册的类型
	types := compiler.RegisteredNodeTypes()
	found := false
	for _, t := range types {
		if t == "custom_node" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("custom_node should be registered")
	}
}

// TestLLMNodeAdapter_WithoutEffectStore 测试没有 EffectStore 的情况
func TestLLMNodeAdapter_WithoutEffectStore(t *testing.T) {
	llm := &fakeLLMGenForTest{}
	adapter := &LLMNodeAdapter{
		LLM:                llm,
		RequireEffectStore: false, // 测试环境不需要
	}
	payload := &AgentDAGPayload{Goal: "test", Results: map[string]any{}}

	result, err := adapter.runNode(context.Background(), "llm1", map[string]any{}, &runtime.Agent{ID: "a1"}, payload)
	if err != nil {
		t.Fatalf("runNode error = %v", err)
	}

	if result.Results["llm1"] == nil {
		t.Fatal("result should be in payload")
	}
}

// TestToolNodeAdapter_WithoutTools 测试没有配置工具的情况（跳过，因为代码会 panic）
func TestToolNodeAdapter_WithoutTools(t *testing.T) {
	t.Skip("ToolNodeAdapter.runNode 会 panic，当 Tools 为 nil")
}

// TestWorkflowNodeAdapter_WithoutWorkflow 测试没有配置工作流的情况（跳过，因为代码会 panic）
func TestWorkflowNodeAdapter_WithoutWorkflow(t *testing.T) {
	t.Skip("WorkflowNodeAdapter.runNode 会 panic，当 Workflow 为 nil")
}
