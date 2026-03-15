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

package http

import (
	"encoding/json"
	"testing"
	"time"

	"rag-platform/internal/runtime/jobstore"
)

func TestExecutionNode(t *testing.T) {
	node := ExecutionNode{
		SpanID:    "span-1",
		Type:      "tool",
		ToolName:  "test_tool",
		StepIndex: 1,
		Children:  []*ExecutionNode{},
	}
	if node.SpanID != "span-1" {
		t.Errorf("expected span-1, got %s", node.SpanID)
	}
	if node.Type != "tool" {
		t.Errorf("expected tool, got %s", node.Type)
	}
	if node.ToolName != "test_tool" {
		t.Errorf("expected test_tool, got %s", node.ToolName)
	}
}

func TestExecutionNode_WithParent(t *testing.T) {
	parentID := "parent-span"
	node := ExecutionNode{
		SpanID:   "child-span",
		ParentID: &parentID,
		Type:     "node",
		NodeID:   "node-1",
	}
	if node.ParentID == nil {
		t.Error("expected non-nil ParentID")
	}
	if *node.ParentID != "parent-span" {
		t.Errorf("expected parent-span, got %s", *node.ParentID)
	}
}

func TestExecutionNode_WithTimes(t *testing.T) {
	now := time.Now()
	node := ExecutionNode{
		SpanID:    "span-1",
		Type:      "tool",
		StartTime: &now,
	}
	if node.StartTime == nil {
		t.Error("expected non-nil StartTime")
	}
}

func TestBuildExecutionTree_Empty(t *testing.T) {
	tree := BuildExecutionTree(nil)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if tree.SpanID != "root" {
		t.Errorf("expected root, got %s", tree.SpanID)
	}
	if tree.Type != "job" {
		t.Errorf("expected job, got %s", tree.Type)
	}
}

func TestBuildExecutionTree_WithEvents(t *testing.T) {
	now := time.Now()
	events := []jobstore.JobEvent{
		{
			ID:        "1",
			JobID:     "job-1",
			Type:      jobstore.JobCreated,
			Payload:   json.RawMessage(`{}`),
			CreatedAt: now,
		},
		{
			ID:        "2",
			JobID:     "job-1",
			Type:      jobstore.PlanGenerated,
			Payload:   json.RawMessage(`{"trace_span_id":"plan-1","node_id":"test"}`),
			CreatedAt: now,
		},
	}

	tree := BuildExecutionTree(events)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if len(tree.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(tree.Children))
	}
}

func TestBuildExecutionTree_PlanGenerated(t *testing.T) {
	now := time.Now()
	events := []jobstore.JobEvent{
		{
			ID:        "1",
			JobID:     "job-1",
			Type:      jobstore.PlanGenerated,
			Payload:   json.RawMessage(`{"trace_span_id":"plan-1"}`),
			CreatedAt: now,
		},
	}

	tree := BuildExecutionTree(events)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if len(tree.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(tree.Children))
	}
	if tree.Children[0].Type != "plan" {
		t.Errorf("expected plan type, got %s", tree.Children[0].Type)
	}
}

func TestBuildExecutionTree_NodeStarted_NoPlan(t *testing.T) {
	now := time.Now()
	events := []jobstore.JobEvent{
		{
			ID:        "1",
			JobID:     "job-1",
			Type:      jobstore.NodeStarted,
			Payload:   json.RawMessage(`{"trace_span_id":"node-1","node_id":"test_node"}`),
			CreatedAt: now,
		},
	}

	tree := BuildExecutionTree(events)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	// Without plan, node goes to root children
	if len(tree.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(tree.Children))
	}
}

func TestBuildExecutionTree_NodeStarted_WithPlan(t *testing.T) {
	now := time.Now()
	events := []jobstore.JobEvent{
		{
			ID:        "1",
			JobID:     "job-1",
			Type:      jobstore.PlanGenerated,
			Payload:   json.RawMessage(`{"trace_span_id":"plan-1"}`),
			CreatedAt: now,
		},
		{
			ID:        "2",
			JobID:     "job-1",
			Type:      jobstore.NodeStarted,
			Payload:   json.RawMessage(`{"trace_span_id":"node-1","node_id":"test_node","parent_span_id":"plan-1"}`),
			CreatedAt: now,
		},
	}

	tree := BuildExecutionTree(events)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if len(tree.Children) != 1 {
		t.Errorf("expected 1 child (plan), got %d", len(tree.Children))
	}
	if tree.Children[0].Type != "plan" {
		t.Errorf("expected plan type, got %s", tree.Children[0].Type)
	}
	if len(tree.Children[0].Children) != 1 {
		t.Errorf("expected 1 child (node), got %d", len(tree.Children[0].Children))
	}
}

func TestBuildExecutionTree_ToolCalls(t *testing.T) {
	now := time.Now()
	events := []jobstore.JobEvent{
		{
			ID:        "1",
			JobID:     "job-1",
			Type:      jobstore.PlanGenerated,
			Payload:   json.RawMessage(`{"trace_span_id":"plan-1"}`),
			CreatedAt: now,
		},
		{
			ID:        "2",
			JobID:     "job-1",
			Type:      jobstore.NodeStarted,
			Payload:   json.RawMessage(`{"trace_span_id":"node-1","node_id":"test_node","parent_span_id":"plan-1"}`),
			CreatedAt: now,
		},
		{
			ID:        "3",
			JobID:     "job-1",
			Type:      jobstore.ToolCalled,
			Payload:   json.RawMessage(`{"trace_span_id":"tool-1","tool_name":"my_tool","parent_span_id":"node-1"}`),
			CreatedAt: now,
		},
		{
			ID:        "4",
			JobID:     "job-1",
			Type:      jobstore.ToolReturned,
			Payload:   json.RawMessage(`{"trace_span_id":"tool-1","tool_name":"my_tool","output":"result"}`),
			CreatedAt: now.Add(time.Second),
		},
	}

	tree := BuildExecutionTree(events)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	// Should have tool as child of node
	if len(tree.Children[0].Children[0].Children) != 1 {
		t.Errorf("expected 1 tool child, got %d", len(tree.Children[0].Children[0].Children))
	}
}

func TestBuildExecutionTree_DecisionSnapshot(t *testing.T) {
	now := time.Now()
	events := []jobstore.JobEvent{
		{
			ID:        "1",
			JobID:     "job-1",
			Type:      jobstore.PlanGenerated,
			Payload:   json.RawMessage(`{"trace_span_id":"plan-1"}`),
			CreatedAt: now,
		},
		{
			ID:        "2",
			JobID:     "job-1",
			Type:      jobstore.DecisionSnapshot,
			Payload:   json.RawMessage(`{"goal":"test goal"}`),
			CreatedAt: now,
		},
	}

	tree := BuildExecutionTree(events)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if tree.Children[0].DecisionSnapshot == nil {
		t.Error("expected DecisionSnapshot to be set")
	}
}
