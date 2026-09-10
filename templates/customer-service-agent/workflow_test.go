// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
)

func TestCreateCustomerServiceWorkflow_Nodes(t *testing.T) {
	wf := createCustomerServiceWorkflow()
	if wf == nil {
		t.Fatal("expected non-nil workflow")
	}
	if len(wf.Nodes) != 8 {
		t.Errorf("expected 8 nodes, got %d", len(wf.Nodes))
	}
	expectedIDs := []string{"triage", "technical_support", "billing", "refund_request",
		"general_inquiry", "human_review", "format_response", "send_response"}
	nodeIDs := make(map[string]bool)
	for _, n := range wf.Nodes {
		nodeIDs[n.ID] = true
	}
	for _, id := range expectedIDs {
		if !nodeIDs[id] {
			t.Errorf("missing node: %s", id)
		}
	}
}

func TestCreateCustomerServiceWorkflow_EntryAndEdges(t *testing.T) {
	wf := createCustomerServiceWorkflow()
	if len(wf.Edges) == 0 {
		t.Fatal("expected non-empty edges")
	}

	// Verify triage has 4 outgoing edges (branching)
	triageEdges := 0
	for _, e := range wf.Edges {
		if e.From == "triage" {
			triageEdges++
		}
	}
	if triageEdges != 4 {
		t.Errorf("expected 4 outgoing edges from triage, got %d", triageEdges)
	}

	// Verify all edges reference existing nodes
	nodeSet := make(map[string]bool)
	for _, n := range wf.Nodes {
		nodeSet[n.ID] = true
	}
	for _, e := range wf.Edges {
		if !nodeSet[e.From] {
			t.Errorf("edge references unknown source node: %s", e.From)
		}
		if !nodeSet[e.To] {
			t.Errorf("edge references unknown target node: %s", e.To)
		}
	}
}

func TestCustomerServiceWorkflow_NodeTypes(t *testing.T) {
	wf := createCustomerServiceWorkflow()
	for _, n := range wf.Nodes {
		switch n.ID {
		case "triage", "technical_support", "billing", "general_inquiry", "format_response":
			if n.Type != planner.NodeLLM {
				t.Errorf("node %s: expected type %s, got %s", n.ID, planner.NodeLLM, n.Type)
			}
		case "refund_request", "human_review":
			if n.Type != planner.NodeWait {
				t.Errorf("node %s: expected type %s, got %s", n.ID, planner.NodeWait, n.Type)
			}
		case "send_response":
			if n.Type != planner.NodeTool {
				t.Errorf("node %s: expected type %s, got %s", n.ID, planner.NodeTool, n.Type)
			}
		}
	}
}

func TestCustomerServiceWorkflow_ExecutorWithoutRunners(t *testing.T) {
	wf := createCustomerServiceWorkflow()
	executor := planner.NewTaskGraphExecutor(nil, nil, nil)
	results, err := executor.Execute(context.Background(), wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	// Without runners, nodes should report errors, not succeed silently
	if results[0].Err == "" {
		t.Error("expected error for first node without configured runner")
	}
}
