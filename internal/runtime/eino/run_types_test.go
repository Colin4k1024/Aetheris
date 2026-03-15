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
	"time"
)

func TestRunStatus(t *testing.T) {
	statuses := []RunStatus{
		RunStatusPending,
		RunStatusRunning,
		RunStatusPaused,
		RunStatusSucceeded,
		RunStatusFailed,
		RunStatusCanceled,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("RunStatus should not be empty")
		}
	}
}

func TestResumeMode(t *testing.T) {
	if ResumeModeFromToolCall == "" {
		t.Error("ResumeMode should not be empty")
	}
}

func TestResumeStrategy(t *testing.T) {
	strategies := []ResumeStrategy{
		ResumeStrategyReuseSuccessfulEffects,
		ResumeStrategyReexecuteFromPoint,
	}
	for _, s := range strategies {
		if s == "" {
			t.Error("ResumeStrategy should not be empty")
		}
	}
}

func TestBudgetPolicy(t *testing.T) {
	budget := BudgetPolicy{
		MaxTokens:    1000,
		MaxToolCalls: 10,
		MaxRetries:   3,
	}
	if budget.MaxTokens != 1000 {
		t.Errorf("expected 1000, got %d", budget.MaxTokens)
	}
	if budget.MaxToolCalls != 10 {
		t.Errorf("expected 10, got %d", budget.MaxToolCalls)
	}
	if budget.MaxRetries != 3 {
		t.Errorf("expected 3, got %d", budget.MaxRetries)
	}
}

func TestRun(t *testing.T) {
	now := time.Now()
	run := Run{
		ID:         "run-1",
		WorkflowID: "workflow-1",
		Status:     RunStatusPending,
		Input:      map[string]interface{}{"key": "value"},
		Budget:     BudgetPolicy{MaxTokens: 1000},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if run.ID != "run-1" {
		t.Errorf("expected run-1, got %s", run.ID)
	}
	if run.Status != RunStatusPending {
		t.Errorf("expected Pending, got %s", run.Status)
	}
}

func TestStep(t *testing.T) {
	now := time.Now()
	step := Step{
		ID:        "step-1",
		RunID:     "run-1",
		NodeName:  "node1",
		Status:    "running",
		Input:     map[string]interface{}{"key": "value"},
		StartedAt: &now,
		EndedAt:   &now,
	}
	if step.ID != "step-1" {
		t.Errorf("expected step-1, got %s", step.ID)
	}
	if step.Status != "running" {
		t.Errorf("expected running, got %s", step.Status)
	}
}

func TestToolCall(t *testing.T) {
	now := time.Now()
	tc := ToolCall{
		ID:             "tc-1",
		RunID:          "run-1",
		StepID:         "step-1",
		ToolName:       "tool1",
		Status:         "pending",
		SideEffectSafe: true,
		StartedAt:      &now,
		EndedAt:        &now,
	}
	if tc.ID != "tc-1" {
		t.Errorf("expected tc-1, got %s", tc.ID)
	}
	if !tc.SideEffectSafe {
		t.Error("expected SideEffectSafe to be true")
	}
}

func TestEventType(t *testing.T) {
	events := []EventType{
		EventTypeRunCreated,
		EventTypeRunPaused,
		EventTypeRunResumed,
		EventTypeStepStarted,
		EventTypeStepCompleted,
		EventTypeToolCallStarted,
		EventTypeToolCallEnded,
		EventTypeRunFailed,
		EventTypeRunSucceeded,
		EventTypeHumanInjected,
	}
	for _, e := range events {
		if e == "" {
			t.Error("EventType should not be empty")
		}
	}
}

func TestRuntimeEvent(t *testing.T) {
	now := time.Now()
	event := RuntimeEvent{
		ID:         "event-1",
		RunID:      "run-1",
		StepID:     "step-1",
		Type:       EventTypeRunCreated,
		Seq:        1,
		Actor:      "system",
		Payload:    map[string]interface{}{"key": "value"},
		OccurredAt: now,
	}
	if event.ID != "event-1" {
		t.Errorf("expected event-1, got %s", event.ID)
	}
	if event.Type != EventTypeRunCreated {
		t.Errorf("expected RunCreated, got %s", event.Type)
	}
}

func TestResumeRunRequest(t *testing.T) {
	req := ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc-1",
		Strategy:       ResumeStrategyReexecuteFromPoint,
		Operator:       "user1",
		Reason:         "retry",
	}
	if req.Mode != ResumeModeFromToolCall {
		t.Errorf("expected FROM_TOOL_CALL, got %s", req.Mode)
	}
	if req.Strategy != ResumeStrategyReexecuteFromPoint {
		t.Errorf("expected REEXECUTE_FROM_POINT, got %s", req.Strategy)
	}
}

func TestHumanDecision(t *testing.T) {
	decision := HumanDecision{
		TargetStepID: "step-1",
		Patch:        map[string]interface{}{"key": "new-value"},
		Operator:     "user1",
		Comment:      "approved",
	}
	if decision.TargetStepID != "step-1" {
		t.Errorf("expected step-1, got %s", decision.TargetStepID)
	}
	if decision.Operator != "user1" {
		t.Errorf("expected user1, got %s", decision.Operator)
	}
}
