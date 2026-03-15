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
	"testing"
)

func TestTaskGraph_Marshal_Unmarshal(t *testing.T) {
	g := &TaskGraph{
		Nodes: []TaskNode{
			{ID: "n1", Type: NodeLLM, Config: map[string]any{"goal": "g1"}},
			{ID: "n2", Type: NodeTool, ToolName: "search"},
		},
		Edges: []TaskEdge{{From: "n1", To: "n2"}},
	}
	data, err := g.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Marshal returned empty")
	}
	var out TaskGraph
	if err := out.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out.Nodes) != 2 || len(out.Edges) != 1 {
		t.Errorf("Unmarshal: nodes=%d edges=%d", len(out.Nodes), len(out.Edges))
	}
	if out.Nodes[0].ID != "n1" || out.Nodes[0].Type != NodeLLM {
		t.Errorf("node0: %+v", out.Nodes[0])
	}
	if out.Edges[0].From != "n1" || out.Edges[0].To != "n2" {
		t.Errorf("edge: %+v", out.Edges[0])
	}
}

func TestTaskGraph_Unmarshal_Empty(t *testing.T) {
	var g TaskGraph
	if err := g.Unmarshal([]byte("{}")); err != nil {
		t.Fatalf("Unmarshal empty: %v", err)
	}
	if g.Nodes != nil || g.Edges != nil {
		t.Errorf("expected nil nodes/edges, got %+v", g)
	}
}

func TestTaskNodeTypes(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{string(NodeTool), "tool"},
		{string(NodeWorkflow), "workflow"},
		{string(NodeLLM), "llm"},
		{string(NodeWait), "wait"},
		{string(NodeApproval), "approval"},
		{string(NodeCondition), "condition"},
		{string(NodeLangChainGo), "langchaingo"},
		{string(NodeLangGraphGo), "langgraphgo"},
		{string(NodeADK), "adk"},
		{string(NodeGenkit), "genkit"},
		{string(NodeEinoReact), "eino_react"},
		{string(NodeEinoDEER), "eino_deer"},
		{string(NodeEinoManus), "eino_manus"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestWaitKindConstants(t *testing.T) {
	if WaitKindUserInput != "user_input" {
		t.Errorf("expected user_input, got %s", WaitKindUserInput)
	}
	if WaitKindWebhook != "webhook" {
		t.Errorf("expected webhook, got %s", WaitKindWebhook)
	}
	if WaitKindSchedule != "schedule" {
		t.Errorf("expected schedule, got %s", WaitKindSchedule)
	}
	if WaitKindCondition != "condition" {
		t.Errorf("expected condition, got %s", WaitKindCondition)
	}
	if WaitKindMessage != "message" {
		t.Errorf("expected message, got %s", WaitKindMessage)
	}
}

func TestTaskNode(t *testing.T) {
	node := TaskNode{
		ID:       "test-node",
		Type:     NodeTool,
		Config:   map[string]any{"key": "value"},
		ToolName: "search",
	}
	if node.ID != "test-node" {
		t.Errorf("expected test-node, got %s", node.ID)
	}
	if node.Type != NodeTool {
		t.Errorf("expected tool, got %s", node.Type)
	}
	if node.ToolName != "search" {
		t.Errorf("expected search, got %s", node.ToolName)
	}
}

func TestTaskEdge(t *testing.T) {
	edge := TaskEdge{
		From: "node1",
		To:   "node2",
	}
	if edge.From != "node1" {
		t.Errorf("expected node1, got %s", edge.From)
	}
	if edge.To != "node2" {
		t.Errorf("expected node2, got %s", edge.To)
	}
}

func TestTaskGraph(t *testing.T) {
	graph := TaskGraph{
		Nodes: []TaskNode{
			{ID: "n1", Type: NodeLLM},
			{ID: "n2", Type: NodeTool, ToolName: "search"},
		},
		Edges: []TaskEdge{
			{From: "n1", To: "n2"},
		},
	}
	if len(graph.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(graph.Edges))
	}
}
