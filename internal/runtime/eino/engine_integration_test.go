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

package eino

import (
	"testing"
)

// TestWorkflowExecutionIntegration 测试工作流执行集成
func TestWorkflowExecutionIntegration(t *testing.T) {
	// 创建工作流
	wf := CreateWorkflow("test-workflow", "Test workflow")

	// 添加节点
	err := wf.AddNode("input", "validate", &NodeConfig{Name: "input"})
	if err != nil {
		t.Fatalf("AddNode input: %v", err)
	}

	err = wf.AddNode("process", "format", &NodeConfig{Name: "process"})
	if err != nil {
		t.Fatalf("AddNode process: %v", err)
	}

	// 添加边
	err = wf.AddEdge("input", "process")
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}

	t.Logf("Workflow created with nodes: input -> process")
}

// TestWorkflowExecutionIntegration_MultipleEdges 测试多边工作流
func TestWorkflowExecutionIntegration_MultipleEdges(t *testing.T) {
	wf := CreateWorkflow("multi-edge", "Multiple edges workflow")

	// 创建三个节点
	wf.AddNode("start", "validate", &NodeConfig{Name: "start"})
	wf.AddNode("branch-a", "format", &NodeConfig{Name: "branch-a"})
	wf.AddNode("branch-b", "format", &NodeConfig{Name: "branch-b"})
	wf.AddNode("end", "format", &NodeConfig{Name: "end"})

	// start -> branch-a, start -> branch-b
	if err := wf.AddEdge("start", "branch-a"); err != nil {
		t.Fatalf("AddEdge start->branch-a: %v", err)
	}
	if err := wf.AddEdge("start", "branch-b"); err != nil {
		t.Fatalf("AddEdge start->branch-b: %v", err)
	}

	// branch-a -> end, branch-b -> end
	if err := wf.AddEdge("branch-a", "end"); err != nil {
		t.Fatalf("AddEdge branch-a->end: %v", err)
	}
	if err := wf.AddEdge("branch-b", "end"); err != nil {
		t.Fatalf("AddEdge branch-b->end: %v", err)
	}

	t.Log("Multi-edge workflow created: start -> {branch-a, branch-b} -> end")
}

// TestWorkflowExecutionIntegration_LinearChain 测试线性链式工作流
func TestWorkflowExecutionIntegration_LinearChain(t *testing.T) {
	wf := CreateWorkflow("linear-chain", "Linear chain workflow")

	// 创建线性链: input -> process -> output
	wf.AddNode("input", "validate", &NodeConfig{Name: "input"})
	wf.AddNode("process", "format", &NodeConfig{Name: "process"})
	wf.AddNode("output", "format", &NodeConfig{Name: "output"})

	// 添加线性边
	if err := wf.AddEdge("input", "process"); err != nil {
		t.Fatalf("AddEdge input->process: %v", err)
	}
	if err := wf.AddEdge("process", "output"); err != nil {
		t.Fatalf("AddEdge process->output: %v", err)
	}

	t.Log("Linear chain workflow: input -> process -> output")
}
