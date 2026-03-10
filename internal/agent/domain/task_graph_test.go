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

package domain

import (
	"testing"
)

func TestTaskGraphExecution_AddNode(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1", Type: "tool"})
	g.AddNode(GraphNode{ID: "node2", Type: "llm"})

	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes))
	}
}

func TestTaskGraphExecution_AddEdge(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1"})
	g.AddNode(GraphNode{ID: "node2"})
	g.AddEdge(GraphEdge{From: "node1", To: "node2"})

	if len(g.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(g.Edges))
	}
}

func TestTaskGraphExecution_BuildIndex(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1"})
	g.AddNode(GraphNode{ID: "node2"})
	g.AddEdge(GraphEdge{From: "node1", To: "node2"})

	err := g.BuildIndex()
	if err != nil {
		t.Fatalf("BuildIndex failed: %v", err)
	}

	// node1 没有依赖，应该是 Ready
	node1 := g.GetNode("node1")
	if node1.State != NodeExecutionStateReady {
		t.Errorf("expected node1 to be ready, got %v", node1.State)
	}

	// node2 依赖 node1，应该是 Pending
	node2 := g.GetNode("node2")
	if node2.State != NodeExecutionStatePending {
		t.Errorf("expected node2 to be pending, got %v", node2.State)
	}
}

func TestTaskGraphExecution_TopologicalSort(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "a"})
	g.AddNode(GraphNode{ID: "b"})
	g.AddNode(GraphNode{ID: "c"})
	g.AddEdge(GraphEdge{From: "a", To: "b"})
	g.AddEdge(GraphEdge{From: "b", To: "c"})

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Errorf("expected order [a, b, c], got %v", order)
	}
}

func TestTaskGraphExecution_TopologicalSort_Cycle(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "a"})
	g.AddNode(GraphNode{ID: "b"})
	g.AddEdge(GraphEdge{From: "a", To: "b"})
	g.AddEdge(GraphEdge{From: "b", To: "a"})

	_, err := g.TopologicalSort()
	if err == nil {
		t.Error("expected error for cycle, got nil")
	}
}

func TestTaskGraphExecution_GetReadyNodes(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1"})
	g.AddNode(GraphNode{ID: "node2"})
	g.AddNode(GraphNode{ID: "node3"})
	g.AddEdge(GraphEdge{From: "node1", To: "node2"})
	g.AddEdge(GraphEdge{From: "node2", To: "node3"})
	g.BuildIndex()

	ready := g.GetReadyNodes()
	if len(ready) != 1 || ready[0] != "node1" {
		t.Errorf("expected [node1], got %v", ready)
	}
}

func TestTaskGraphExecution_MarkNodeCompleted(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1"})
	g.AddNode(GraphNode{ID: "node2"})
	g.AddEdge(GraphEdge{From: "node1", To: "node2"})
	g.BuildIndex()

	// 标记 node1 完成
	err := g.MarkNodeCompleted("node1", map[string]interface{}{"result": "ok"})
	if err != nil {
		t.Fatalf("MarkNodeCompleted failed: %v", err)
	}

	// node2 应该变为 Ready
	node2 := g.GetNode("node2")
	if node2.State != NodeExecutionStateReady {
		t.Errorf("expected node2 to be ready, got %v", node2.State)
	}

	ready := g.GetReadyNodes()
	if len(ready) != 1 || ready[0] != "node2" {
		t.Errorf("expected [node2], got %v", ready)
	}
}

func TestTaskGraphExecution_MarkNodeFailed(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1"})
	g.AddNode(GraphNode{ID: "node2"})
	g.AddEdge(GraphEdge{From: "node1", To: "node2"})
	g.BuildIndex()

	// 标记 node1 失败
	err := g.MarkNodeFailed("node1", "some error")
	if err != nil {
		t.Fatalf("MarkNodeFailed failed: %v", err)
	}

	// node2 应该被跳过
	node2 := g.GetNode("node2")
	if node2.State != NodeExecutionStateSkipped {
		t.Errorf("expected node2 to be skipped, got %v", node2.State)
	}
}

func TestTaskGraphExecution_IsCompleted(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1"})
	g.AddNode(GraphNode{ID: "node2"})
	g.BuildIndex()

	if g.IsCompleted() {
		t.Error("expected not completed initially")
	}

	_ = g.MarkNodeCompleted("node1", nil)
	if g.IsCompleted() {
		t.Error("expected not completed after one node")
	}

	_ = g.MarkNodeCompleted("node2", nil)
	if !g.IsCompleted() {
		t.Error("expected completed after all nodes")
	}
}

func TestTaskGraphExecution_GetProgress(t *testing.T) {
	g := NewTaskGraphExecution()
	g.AddNode(GraphNode{ID: "node1"})
	g.AddNode(GraphNode{ID: "node2"})
	g.AddNode(GraphNode{ID: "node3"})
	g.BuildIndex()

	completed, total, failed, skipped := g.GetProgress()
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if completed != 0 || failed != 0 || skipped != 0 {
		t.Errorf("expected all zeros initially, got %d, %d, %d", completed, failed, skipped)
	}

	_ = g.MarkNodeCompleted("node1", nil)
	_ = g.MarkNodeFailed("node2", "error")

	completed, total, failed, skipped = g.GetProgress()
	if completed != 1 {
		t.Errorf("expected 1 completed, got %d", completed)
	}
	if failed != 1 {
		t.Errorf("expected 1 failed, got %d", failed)
	}
}
