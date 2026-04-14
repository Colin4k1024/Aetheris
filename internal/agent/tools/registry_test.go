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

package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/session"
)

type mockTool struct {
	name, desc string
	schema     map[string]any
}

func (m mockTool) Name() string        { return m.name }
func (m mockTool) Description() string { return m.desc }
func (m mockTool) Schema() map[string]any {
	if m.schema == nil {
		return map[string]any{}
	}
	return m.schema
}
func (m mockTool) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
	return nil, nil
}

func TestRegistry_Register_Get_List(t *testing.T) {
	r := NewRegistry()
	t1 := mockTool{name: "tool1", desc: "desc1"}
	t2 := mockTool{name: "tool2", desc: "desc2"}
	r.Register(t1)
	r.Register(t2)
	got, ok := r.Get("tool1")
	if !ok || got.Name() != "tool1" {
		t.Errorf("Get tool1: ok=%v got=%v", ok, got)
	}
	_, ok = r.Get("missing")
	if ok {
		t.Error("Get missing should be false")
	}
	list := r.List()
	if len(list) != 2 {
		t.Errorf("List: expected 2, got %d", len(list))
	}
}

func TestRegistry_SchemasForLLM(t *testing.T) {
	r := NewRegistry()
	r.Register(mockTool{name: "t1", desc: "d1", schema: map[string]any{"type": "string"}})
	data, err := r.SchemasForLLM()
	if err != nil {
		t.Fatalf("SchemasForLLM: %v", err)
	}
	var list []ToolSchemaForLLM
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 1 || list[0].Name != "t1" || list[0].Description != "d1" {
		t.Errorf("SchemasForLLM: %+v", list)
	}
}

func TestRegistry_GetCapability(t *testing.T) {
	r := NewRegistry()
	r.Register(mockTool{name: "tool1", desc: "desc1"})

	// Tool without capability returns name
	cap := r.GetCapability("tool1")
	if cap != "tool1" {
		t.Errorf("expected tool1, got %s", cap)
	}

	// Non-existent tool returns name
	cap = r.GetCapability("missing")
	if cap != "missing" {
		t.Errorf("expected missing, got %s", cap)
	}
}

func TestRegistry_Manifests(t *testing.T) {
	r := NewRegistry()
	r.Register(mockTool{name: "tool1", desc: "desc1"})

	manifests := r.Manifests()
	if len(manifests) != 1 {
		t.Errorf("expected 1 manifest, got %d", len(manifests))
	}
}

func TestRegistry_Manifest(t *testing.T) {
	r := NewRegistry()
	r.Register(mockTool{name: "tool1", desc: "desc1"})

	// Existing tool
	m := r.Manifest("tool1")
	if m == nil {
		t.Fatal("expected non-nil manifest")
	}
	if m.Name != "tool1" {
		t.Errorf("expected tool1, got %s", m.Name)
	}

	// Non-existing tool
	m = r.Manifest("missing")
	if m != nil {
		t.Error("expected nil manifest for missing tool")
	}
}
