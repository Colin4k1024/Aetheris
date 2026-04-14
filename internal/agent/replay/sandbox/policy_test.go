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

package sandbox

import (
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/replay"
)

func TestOperationKindConstants(t *testing.T) {
	tests := []struct {
		kind     OperationKind
		expected OperationKind
	}{
		{Deterministic, Deterministic},
		{SideEffect, SideEffect},
		{External, External},
	}

	for _, tt := range tests {
		if tt.kind != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.kind)
		}
	}
}

func TestReplayDecision(t *testing.T) {
	decision := ReplayDecision{
		Kind:   SideEffect,
		Inject: true,
		Result: []byte(`{"result": "test"}`),
	}

	if decision.Kind != SideEffect {
		t.Errorf("expected side_effect, got %s", decision.Kind)
	}
	if !decision.Inject {
		t.Error("expected Inject to be true")
	}
	if string(decision.Result) != `{"result": "test"}` {
		t.Errorf("expected result, got %s", string(decision.Result))
	}
}

func TestDefaultPolicy_Decide_NilReplayContext(t *testing.T) {
	policy := DefaultPolicy{}

	// With nil replayCtx - should return decision without injection
	decision := policy.Decide("node-1", "cmd-1", planner.NodeTool, nil)

	if decision.Kind != SideEffect {
		t.Errorf("expected SideEffect, got %s", decision.Kind)
	}
	if decision.Inject {
		t.Error("expected Inject to be false when replayCtx is nil")
	}
}

func TestDefaultPolicy_Decide_WithReplayContext(t *testing.T) {
	policy := DefaultPolicy{}

	// With replayCtx that has no results
	decision := policy.Decide("node-1", "cmd-1", planner.NodeTool, &replay.ReplayContext{
		CommandResults: map[string][]byte{},
	})

	if decision.Kind != SideEffect {
		t.Errorf("expected SideEffect, got %s", decision.Kind)
	}
	if decision.Inject {
		t.Error("expected Inject to be false when no results")
	}
}

func TestDefaultPolicy_Decide_WithResult(t *testing.T) {
	policy := DefaultPolicy{}

	result := []byte(`{"status": "success"}`)
	decision := policy.Decide("node-1", "cmd-1", planner.NodeTool, &replay.ReplayContext{
		CommandResults: map[string][]byte{
			"cmd-1": result,
		},
	})

	if decision.Kind != SideEffect {
		t.Errorf("expected SideEffect, got %s", decision.Kind)
	}
	if !decision.Inject {
		t.Error("expected Inject to be true when result exists")
	}
	if string(decision.Result) != string(result) {
		t.Errorf("expected result, got %s", string(decision.Result))
	}
}

func TestDefaultPolicy_Decide_LLMNode(t *testing.T) {
	policy := DefaultPolicy{}

	result := []byte(`{"content": "hello"}`)
	decision := policy.Decide("node-1", "cmd-1", planner.NodeLLM, &replay.ReplayContext{
		CommandResults: map[string][]byte{
			"cmd-1": result,
		},
	})

	if decision.Kind != SideEffect {
		t.Errorf("expected SideEffect, got %s", decision.Kind)
	}
	if !decision.Inject {
		t.Error("expected Inject to be true when result exists")
	}
}

func TestDefaultPolicy_Decide_WorkflowNode(t *testing.T) {
	policy := DefaultPolicy{}

	result := []byte(`{"output": "done"}`)
	decision := policy.Decide("node-1", "cmd-1", planner.NodeWorkflow, &replay.ReplayContext{
		CommandResults: map[string][]byte{
			"cmd-1": result,
		},
	})

	if decision.Kind != SideEffect {
		t.Errorf("expected SideEffect, got %s", decision.Kind)
	}
	if !decision.Inject {
		t.Error("expected Inject to be true when result exists")
	}
}

func TestDefaultPolicy_Decide_UnknownNodeType(t *testing.T) {
	policy := DefaultPolicy{}

	result := []byte(`{"data": "test"}`)
	decision := policy.Decide("node-1", "cmd-1", "unknown_type", &replay.ReplayContext{
		CommandResults: map[string][]byte{
			"cmd-1": result,
		},
	})

	if decision.Kind != SideEffect {
		t.Errorf("expected SideEffect, got %s", decision.Kind)
	}
	if !decision.Inject {
		t.Error("expected Inject to be true when result exists")
	}
}

func TestKindForNodeType(t *testing.T) {
	tests := []struct {
		nodeType string
		expected OperationKind
	}{
		{planner.NodeTool, SideEffect},
		{planner.NodeLLM, SideEffect},
		{planner.NodeWorkflow, SideEffect},
		{"unknown", SideEffect},
	}

	for _, tt := range tests {
		result := kindForNodeType(tt.nodeType)
		if result != tt.expected {
			t.Errorf("for nodeType %s: expected %s, got %s", tt.nodeType, tt.expected, result)
		}
	}
}
