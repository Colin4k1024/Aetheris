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

package planner

import (
	"context"
	"fmt"
	"testing"
)

type mockToolRunner struct{}

func (m *mockToolRunner) Execute(ctx context.Context, toolName string, input map[string]any) (string, error) {
	return "tool result", nil
}

type mockWorkflowRunner struct{}

func (m *mockWorkflowRunner) ExecuteWorkflow(ctx context.Context, name string, params map[string]any) (interface{}, error) {
	return "workflow result", nil
}

type mockLLMRunner struct{}

func (m *mockLLMRunner) Generate(ctx context.Context, prompt string) (string, error) {
	return "llm response", nil
}

func TestNewTaskGraphExecutor(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})

	if executor == nil {
		t.Fatal("executor should not be nil")
	}

	if executor.Tools == nil {
		t.Error("Tools should be set")
	}

	if executor.Workflow == nil {
		t.Error("Workflow should be set")
	}

	if executor.LLM == nil {
		t.Error("LLM should be set")
	}
}

func TestTaskGraphExecutor_Execute_NilGraph(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Execute(nil): %v", err)
	}
	if results != nil {
		t.Errorf("Execute(nil): expected nil results, got %v", results)
	}
}

func TestTaskGraphExecutor_Execute_EmptyGraph(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{Nodes: []TaskNode{}})
	if err != nil {
		t.Fatalf("Execute(empty): %v", err)
	}
	if results != nil {
		t.Errorf("Execute(empty): expected nil results, got %v", results)
	}
}

func TestTaskGraphExecutor_Execute_ToolNode(t *testing.T) {
	toolRunner := &mockToolRunner{}
	executor := NewTaskGraphExecutor(toolRunner, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeTool, ToolName: "test_tool", Config: map[string]any{"key": "value"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute(tool): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(tool): expected 1 result, got %d", len(results))
	}
	if results[0].NodeID != "node1" {
		t.Errorf("Execute(tool): expected node1, got %s", results[0].NodeID)
	}
	if results[0].Err != "" {
		t.Errorf("Execute(tool): expected no error, got %s", results[0].Err)
	}
	if results[0].Output != "tool result" {
		t.Errorf("Execute(tool): expected 'tool result', got %s", results[0].Output)
	}
}

func TestTaskGraphExecutor_Execute_ToolNode_NilTools(t *testing.T) {
	executor := NewTaskGraphExecutor(nil, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeTool, ToolName: "test_tool"},
		},
	})
	if err != nil {
		t.Fatalf("Execute(nil tools): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(nil tools): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "ToolRunner not configured" {
		t.Errorf("Execute(nil tools): expected 'ToolRunner not configured', got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_ToolNode_MissingToolName(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeTool, ToolName: ""},
		},
	})
	if err != nil {
		t.Fatalf("Execute(missing tool_name): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(missing tool_name): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "node missing tool_name" {
		t.Errorf("Execute(missing tool_name): expected 'node missing tool_name', got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_ToolNode_NilConfig(t *testing.T) {
	toolRunner := &mockToolRunner{}
	executor := NewTaskGraphExecutor(toolRunner, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeTool, ToolName: "test_tool", Config: nil},
		},
	})
	if err != nil {
		t.Fatalf("Execute(nil config): %v", err)
	}
	if len(results) != 1 || results[0].Err != "" {
		t.Errorf("Execute(nil config): expected success")
	}
}

type errorToolRunner struct{}

func (m *errorToolRunner) Execute(ctx context.Context, toolName string, input map[string]any) (string, error) {
	return "", fmt.Errorf("tool execution failed")
}

func TestTaskGraphExecutor_Execute_ToolNode_Error(t *testing.T) {
	executor := NewTaskGraphExecutor(&errorToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeTool, ToolName: "test_tool"},
		},
	})
	if err != nil {
		t.Fatalf("Execute(tool error): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(tool error): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "tool execution failed" {
		t.Errorf("Execute(tool error): expected 'tool execution failed', got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_WorkflowNode(t *testing.T) {
	workflowRunner := &mockWorkflowRunner{}
	executor := NewTaskGraphExecutor(&mockToolRunner{}, workflowRunner, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeWorkflow, Workflow: "test_workflow", Config: map[string]any{"param": "value"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute(workflow): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(workflow): expected 1 result, got %d", len(results))
	}
	if results[0].NodeID != "node1" {
		t.Errorf("Execute(workflow): expected node1, got %s", results[0].NodeID)
	}
	if results[0].Err != "" {
		t.Errorf("Execute(workflow): expected no error, got %s", results[0].Err)
	}
	if results[0].Output != "workflow result" {
		t.Errorf("Execute(workflow): expected 'workflow result', got %s", results[0].Output)
	}
}

func TestTaskGraphExecutor_Execute_WorkflowNode_NilWorkflow(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, nil, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeWorkflow, Workflow: "test_workflow"},
		},
	})
	if err != nil {
		t.Fatalf("Execute(nil workflow): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(nil workflow): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "WorkflowRunner not configured" {
		t.Errorf("Execute(nil workflow): expected 'WorkflowRunner not configured', got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_WorkflowNode_NilConfig(t *testing.T) {
	workflowRunner := &mockWorkflowRunner{}
	executor := NewTaskGraphExecutor(&mockToolRunner{}, workflowRunner, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeWorkflow, Workflow: "test_workflow", Config: nil},
		},
	})
	if err != nil {
		t.Fatalf("Execute(workflow nil config): %v", err)
	}
	if len(results) != 1 || results[0].Err != "" {
		t.Errorf("Execute(workflow nil config): expected success")
	}
}

type errorWorkflowRunner struct{}

func (m *errorWorkflowRunner) ExecuteWorkflow(ctx context.Context, name string, params map[string]any) (interface{}, error) {
	return nil, fmt.Errorf("workflow execution failed")
}

func TestTaskGraphExecutor_Execute_WorkflowNode_Error(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &errorWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeWorkflow, Workflow: "test_workflow"},
		},
	})
	if err != nil {
		t.Fatalf("Execute(workflow error): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(workflow error): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "workflow execution failed" {
		t.Errorf("Execute(workflow error): expected 'workflow execution failed', got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_LLMNode(t *testing.T) {
	llmRunner := &mockLLMRunner{}
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, llmRunner)
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeLLM, Config: map[string]any{"goal": "test prompt"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute(llm): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(llm): expected 1 result, got %d", len(results))
	}
	if results[0].NodeID != "node1" {
		t.Errorf("Execute(llm): expected node1, got %s", results[0].NodeID)
	}
	if results[0].Err != "" {
		t.Errorf("Execute(llm): expected no error, got %s", results[0].Err)
	}
	if results[0].Output != "llm response" {
		t.Errorf("Execute(llm): expected 'llm response', got %s", results[0].Output)
	}
}

func TestTaskGraphExecutor_Execute_LLMNode_NilLLM(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, nil)
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeLLM, Config: map[string]any{"goal": "test"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute(nil llm): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(nil llm): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "LLMRunner not configured" {
		t.Errorf("Execute(nil llm): expected 'LLMRunner not configured', got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_LLMNode_NoGoal(t *testing.T) {
	llmRunner := &mockLLMRunner{}
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, llmRunner)
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeLLM, Config: nil},
		},
	})
	if err != nil {
		t.Fatalf("Execute(llm no goal): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(llm no goal): expected 1 result, got %d", len(results))
	}
	// Empty prompt should still work with mock
	if results[0].Err != "" {
		t.Errorf("Execute(llm no goal): expected no error, got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_LLMNode_NonStringGoal(t *testing.T) {
	llmRunner := &mockLLMRunner{}
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, llmRunner)
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeLLM, Config: map[string]any{"goal": 123}},
		},
	})
	if err != nil {
		t.Fatalf("Execute(llm non-string goal): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(llm non-string goal): expected 1 result, got %d", len(results))
	}
	// Non-string goal should use empty prompt
	if results[0].Output != "llm response" {
		t.Errorf("Execute(llm non-string goal): expected response, got %s", results[0].Output)
	}
}

type errorLLMRunner struct{}

func (m *errorLLMRunner) Generate(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("llm generation failed")
}

func TestTaskGraphExecutor_Execute_LLMNode_Error(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, &errorLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeLLM, Config: map[string]any{"goal": "test"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute(llm error): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(llm error): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "llm generation failed" {
		t.Errorf("Execute(llm error): expected 'llm generation failed', got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_UnknownNodeType(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: "unknown_type"},
		},
	})
	if err != nil {
		t.Fatalf("Execute(unknown type): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Execute(unknown type): expected 1 result, got %d", len(results))
	}
	if results[0].Err != "Unknown node type: unknown_type" {
		t.Errorf("Execute(unknown type): expected unknown type error, got %s", results[0].Err)
	}
}

func TestTaskGraphExecutor_Execute_StopOnError(t *testing.T) {
	executor := NewTaskGraphExecutor(&errorToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeTool, ToolName: "test_tool"},
			{ID: "node2", Type: NodeTool, ToolName: "test_tool2"},
			{ID: "node3", Type: NodeTool, ToolName: "test_tool3"},
		},
	})
	if err != nil {
		t.Fatalf("Execute(stop on error): %v", err)
	}
	// Should stop at first error, only 1 result
	if len(results) != 1 {
		t.Fatalf("Execute(stop on error): expected 1 result, got %d", len(results))
	}
	if results[0].NodeID != "node1" {
		t.Errorf("Execute(stop on error): expected node1, got %s", results[0].NodeID)
	}
}

func TestTaskGraphExecutor_Execute_MultipleNodes(t *testing.T) {
	toolRunner := &mockToolRunner{}
	workflowRunner := &mockWorkflowRunner{}
	llmRunner := &mockLLMRunner{}
	executor := NewTaskGraphExecutor(toolRunner, workflowRunner, llmRunner)
	ctx := context.Background()

	results, err := executor.Execute(ctx, &TaskGraph{
		Nodes: []TaskNode{
			{ID: "node1", Type: NodeTool, ToolName: "tool1"},
			{ID: "node2", Type: NodeWorkflow, Workflow: "wf1"},
			{ID: "node3", Type: NodeLLM, Config: map[string]any{"goal": "prompt"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute(multiple): %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("Execute(multiple): expected 3 results, got %d", len(results))
	}
	if results[0].Output != "tool result" {
		t.Errorf("Execute(multiple): node1 expected 'tool result', got %s", results[0].Output)
	}
	if results[1].Output != "workflow result" {
		t.Errorf("Execute(multiple): node2 expected 'workflow result', got %s", results[1].Output)
	}
	if results[2].Output != "llm response" {
		t.Errorf("Execute(multiple): node3 expected 'llm response', got %s", results[2].Output)
	}
}
