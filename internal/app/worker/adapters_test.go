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

package worker

import (
	"context"
	"testing"

	"rag-platform/internal/agent/memory"
	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/tools"
	"rag-platform/internal/runtime/session"
	"rag-platform/internal/agent/runtime"
)

// mockPlanGoalProvider implements planGoalProvider for testing
type mockPlanGoalProvider struct {
	planFunc func(ctx context.Context, goal string, mem memory.Memory) (*planner.TaskGraph, error)
}

func (m *mockPlanGoalProvider) PlanGoal(ctx context.Context, goal string, mem memory.Memory) (*planner.TaskGraph, error) {
	if m.planFunc != nil {
		return m.planFunc(ctx, goal, mem)
	}
	return nil, nil
}

func TestNewPlannerProviderAdapter(t *testing.T) {
	provider := &mockPlanGoalProvider{
		planFunc: func(ctx context.Context, goal string, mem memory.Memory) (*planner.TaskGraph, error) {
			return &planner.TaskGraph{}, nil
		},
	}

	adapter := newPlannerProviderAdapter(provider)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestPlannerProviderAdapter_Plan_WithContext(t *testing.T) {
	provider := &mockPlanGoalProvider{
		planFunc: func(ctx context.Context, goal string, mem memory.Memory) (*planner.TaskGraph, error) {
			if goal != "test goal" {
				t.Errorf("expected 'test goal', got %s", goal)
			}
			return &planner.TaskGraph{}, nil
		},
	}

	adapter := newPlannerProviderAdapter(provider)
	result, err := adapter.Plan(context.Background(), "test goal", memory.NewCompositeMemory())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestPlannerProviderAdapter_Plan_NilContext(t *testing.T) {
	provider := &mockPlanGoalProvider{
		planFunc: func(ctx context.Context, goal string, mem memory.Memory) (*planner.TaskGraph, error) {
			return &planner.TaskGraph{}, nil
		},
	}

	adapter := newPlannerProviderAdapter(provider)
	result, err := adapter.Plan(nil, "test goal", memory.NewCompositeMemory())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestPlannerProviderAdapter_Plan_NilMemory(t *testing.T) {
	provider := &mockPlanGoalProvider{
		planFunc: func(ctx context.Context, goal string, mem memory.Memory) (*planner.TaskGraph, error) {
			if mem == nil {
				t.Error("expected non-nil memory")
			}
			return &planner.TaskGraph{}, nil
		},
	}

	adapter := newPlannerProviderAdapter(provider)
	result, err := adapter.Plan(context.Background(), "test goal", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestNewToolsProviderAdapter(t *testing.T) {
	registry := tools.NewRegistry()
	adapter := newToolsProviderAdapter(registry)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestToolsProviderAdapter_Get(t *testing.T) {
	registry := tools.NewRegistry()
	// Register a test tool
	registry.Register(&testTool{name: "test_tool"})

	adapter := newToolsProviderAdapter(registry)
	result, ok := adapter.Get("test_tool")
	if !ok {
		t.Error("expected to find test_tool")
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestToolsProviderAdapter_Get_NotFound(t *testing.T) {
	registry := tools.NewRegistry()
	adapter := newToolsProviderAdapter(registry)
	result, ok := adapter.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent tool")
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestToolsProviderAdapter_List(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testTool{name: "tool1"})
	registry.Register(&testTool{name: "tool2"})

	adapter := newToolsProviderAdapter(registry)
	list := adapter.List()
	if len(list) != 2 {
		t.Errorf("expected 2 tools, got %d", len(list))
	}
}

func TestToolsProviderAdapter_List_Empty(t *testing.T) {
	registry := tools.NewRegistry()
	adapter := newToolsProviderAdapter(registry)
	list := adapter.List()
	if len(list) != 0 {
		t.Errorf("expected 0 tools, got %d", len(list))
	}
}

// testTool implements tools.Tool for testing
type testTool struct {
	name        string
	description string
}

func (t *testTool) Name() string        { return t.name }
func (t *testTool) Description() string { return t.description }
func (t *testTool) Schema() map[string]any { return nil }
func (t *testTool) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
	return nil, nil
}

// Compile-time check that testTool implements tools.Tool
var _ tools.Tool = (*testTool)(nil)

// Compile-time check that adapter implements runtime interfaces
var _ runtime.PlannerProvider = (*plannerProviderAdapter)(nil)
var _ runtime.ToolsProvider = (*toolsProviderAdapter)(nil)
