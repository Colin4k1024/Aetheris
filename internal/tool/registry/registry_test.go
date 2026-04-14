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

package registry

import (
	"context"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
)

type mockTool struct {
	name        string
	description string
	schema      tool.Schema
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Schema() tool.Schema { return m.schema }
func (m *mockTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
	return tool.ToolResult{Content: "ok"}, nil
}

func TestNewRegistry(t *testing.T) {
	reg := New()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestRegistry_Register(t *testing.T) {
	reg := New()
	t1 := &mockTool{name: "tool1", description: "tool 1"}
	reg.Register(t1)

	// Get tool
	got, ok := reg.Get("tool1")
	if !ok {
		t.Error("expected to find tool1")
	}
	if got.Name() != "tool1" {
		t.Errorf("expected tool1, got %s", got.Name())
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	reg := New()
	got, ok := reg.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
	if got != nil {
		t.Error("expected nil")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := New()
	reg.Register(&mockTool{name: "tool1"})
	reg.Register(&mockTool{name: "tool2"})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("expected 2 tools, got %d", len(list))
	}
}

func TestRegistry_List_Empty(t *testing.T) {
	reg := New()
	list := reg.List()
	if len(list) != 0 {
		t.Errorf("expected 0 tools, got %d", len(list))
	}
}

func TestRegistry_SchemasForLLM(t *testing.T) {
	reg := New()
	reg.Register(&mockTool{
		name:        "search",
		description: "Search the web",
		schema:      tool.Schema{Type: "object"},
	})

	data, err := reg.SchemasForLLM()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}
