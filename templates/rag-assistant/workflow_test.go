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

func TestCreateRAGWorkflow_Nodes(t *testing.T) {
	wf := createRAGWorkflow()
	if wf == nil {
		t.Fatal("expected non-nil workflow")
	}
	if len(wf.Nodes) != 8 {
		t.Errorf("expected 8 nodes, got %d", len(wf.Nodes))
	}
	expectedIDs := []string{"query_rewrite", "retrieve_v1", "web_search",
		"rerank", "synthesize", "generate_answer", "quality_check", "finalize"}
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

func TestCreateRAGWorkflow_Edges(t *testing.T) {
	wf := createRAGWorkflow()
	if len(wf.Edges) == 0 {
		t.Fatal("expected non-empty edges")
	}

	// Verify query_rewrite has 2 outgoing edges (branching to retrieve_v1 and web_search)
	qrEdges := 0
	for _, e := range wf.Edges {
		if e.From == "query_rewrite" {
			qrEdges++
		}
	}
	if qrEdges != 2 {
		t.Errorf("expected 2 outgoing edges from query_rewrite, got %d", qrEdges)
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

func TestCreateRAGWorkflow_NodeTypes(t *testing.T) {
	wf := createRAGWorkflow()
	for _, n := range wf.Nodes {
		switch n.ID {
		case "retrieve_v1", "web_search":
			if n.Type != planner.NodeTool {
				t.Errorf("node %s: expected type %s, got %s", n.ID, planner.NodeTool, n.Type)
			}
		case "query_rewrite", "rerank", "synthesize", "generate_answer", "quality_check", "finalize":
			if n.Type != planner.NodeLLM {
				t.Errorf("node %s: expected type %s, got %s", n.ID, planner.NodeLLM, n.Type)
			}
		}
	}
}

func TestCreateQueryWorkflow_Executes(t *testing.T) {
	ctx := context.Background()
	runnable, err := CreateQueryWorkflow(ctx)
	if err != nil {
		t.Fatalf("CreateQueryWorkflow error: %v", err)
	}
	if runnable == nil {
		t.Fatal("expected non-nil runnable")
	}

	output, err := runnable.Invoke(ctx, &RAGInput{Query: "test query"})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.Answer == "" {
		t.Error("expected non-empty answer")
	}
	if len(output.Sources) == 0 {
		t.Error("expected non-empty sources")
	}
}
