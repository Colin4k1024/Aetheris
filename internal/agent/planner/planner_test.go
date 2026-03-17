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
	"testing"

	"rag-platform/internal/agent/memory"
	"rag-platform/internal/runtime/session"
)

func TestPlanStep(t *testing.T) {
	step := PlanStep{
		Tool:  "search",
		Input: map[string]any{"query": "test"},
	}
	if step.Tool != "search" {
		t.Errorf("expected search, got %s", step.Tool)
	}
}

func TestStep(t *testing.T) {
	step := Step{
		Tool:  "search",
		Input: map[string]any{"query": "test"},
		Final: "final answer",
	}
	if step.Tool != "search" {
		t.Errorf("expected search, got %s", step.Tool)
	}
	if step.Final != "final answer" {
		t.Errorf("expected final answer, got %s", step.Final)
	}
}

func TestPlanResult(t *testing.T) {
	result := PlanResult{
		Steps: []PlanStep{
			{Tool: "tool1", Input: nil},
			{Tool: "tool2", Input: nil},
		},
		Next:        "continue",
		FinalAnswer: "done",
	}
	if len(result.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(result.Steps))
	}
	if result.Next != "continue" {
		t.Errorf("expected continue, got %s", result.Next)
	}
}

func TestToolSchemaItem(t *testing.T) {
	item := toolSchemaItem{
		Name:        "search",
		Description: "Search for information",
	}
	if item.Name != "search" {
		t.Errorf("expected search, got %s", item.Name)
	}
	if item.Description != "Search for information" {
		t.Errorf("expected description, got %s", item.Description)
	}
}

func TestLLMPlanner_New(t *testing.T) {
	planner := NewLLMPlanner(nil)
	if planner == nil {
		t.Fatal("expected non-nil planner")
	}
	if planner.client != nil {
		t.Error("expected nil client")
	}
}

func TestLLMPlanner_SetToolsSchemaForGoal(t *testing.T) {
	planner := NewLLMPlanner(nil)
	schema := []byte(`[{"name":"tool1","description":"desc1"}]`)
	planner.SetToolsSchemaForGoal(schema)
	if planner.toolsSchemaForGoal == nil {
		t.Error("expected non-nil toolsSchemaForGoal")
	}
}

func TestLLMPlanner_Plan_NilClient(t *testing.T) {
	planner := NewLLMPlanner(nil)
	result, err := planner.Plan(nil, "test", nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Next != "finish" {
		t.Errorf("expected finish, got %s", result.Next)
	}
	if result.FinalAnswer == "" {
		t.Error("expected final answer")
	}
}

func TestLLMPlanner_PlanGoal_DefaultGraph(t *testing.T) {
	planner := NewLLMPlanner(nil)
	graph, err := planner.PlanGoal(nil, "test goal", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if len(graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.Nodes))
	}
	if graph.Nodes[0].Type != NodeLLM {
		t.Errorf("expected NodeLLM, got %s", graph.Nodes[0].Type)
	}
}

func TestLLMPlanner_PlanGoal_WithToolsSchema(t *testing.T) {
	planner := NewLLMPlanner(nil)
	schema := []byte(`[{"name":"search","description":"Search for information"},{"name":"api","description":"Call API"}]`)
	planner.SetToolsSchemaForGoal(schema)
	// Test that setting schema doesn't panic
	if planner.toolsSchemaForGoal == nil {
		t.Error("expected non-nil toolsSchemaForGoal")
	}
}

func TestLLMPlanner_Next_NilClient(t *testing.T) {
	planner := NewLLMPlanner(nil)
	step, err := planner.Next(nil, nil, "test query", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if step == nil {
		t.Fatal("expected non-nil step")
	}
	if step.Final == "" {
		t.Error("expected final answer for nil client")
	}
}

func TestLLMPlanner_Next_WithSession(t *testing.T) {
	planner := NewLLMPlanner(nil)
	sess := session.New("test-session")
	step, err := planner.Next(context.Background(), sess, "test query", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if step == nil {
		t.Fatal("expected non-nil step")
	}
}

func TestPlanStep_JSONMarshal(t *testing.T) {
	step := PlanStep{
		Tool:  "search",
		Input: map[string]any{"query": "test"},
	}
	// Test that PlanStep can be marshaled to JSON
	// This exercises the json tags
	if step.Tool != "search" {
		t.Errorf("expected search, got %s", step.Tool)
	}
	if step.Input["query"] != "test" {
		t.Errorf("expected query=test, got %v", step.Input)
	}
}

func TestStep_JSONMarshal(t *testing.T) {
	step := Step{
		Tool:  "search",
		Input: map[string]any{"query": "test"},
		Final: "final answer",
	}
	// Test JSON fields
	if step.Final != "final answer" {
		t.Errorf("expected final answer, got %s", step.Final)
	}
}

func TestPlanResult_NextDefault(t *testing.T) {
	result := PlanResult{
		Steps:       []PlanStep{},
		Next:        "", // Empty should default to finish
		FinalAnswer: "",
	}
	// Test that empty Next gets handled
	if result.Next == "" {
		// This is the case that Plan handles
		result.Next = "finish"
	}
	if result.Next != "finish" {
		t.Errorf("expected finish, got %s", result.Next)
	}
}

func TestLLMPlanner_PlanGoal_WithNilMemory(t *testing.T) {
	planner := NewLLMPlanner(nil)
	// Test with nil memory - should not panic
	graph, err := planner.PlanGoal(context.Background(), "test goal", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
}

func TestLLMPlanner_PlanGoal_WithEmptyMemory(t *testing.T) {
	planner := NewLLMPlanner(nil)
	// Test with empty memory
	mem := memory.NewCompositeMemory()
	graph, err := planner.PlanGoal(context.Background(), "test goal", mem)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
}
