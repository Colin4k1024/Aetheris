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

package tool

import (
	"context"
	"testing"
)

// mockTool implements Tool interface for testing
type mockTool struct {
	name        string
	description string
	schema      Schema
	executeFunc func(ctx context.Context, input map[string]any) (ToolResult, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Schema() Schema      { return m.schema }
func (m *mockTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	return m.executeFunc(ctx, input)
}

func TestSchema(t *testing.T) {
	s := Schema{
		Type:        "object",
		Description: "A test schema",
		Properties: map[string]SchemaProperty{
			"name": {
				Type:        "string",
				Description: "The name",
			},
		},
		Required: []string{"name"},
	}

	if s.Type != "object" {
		t.Errorf("expected object type, got %s", s.Type)
	}
	if s.Description != "A test schema" {
		t.Errorf("expected description, got %s", s.Description)
	}
	if len(s.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(s.Properties))
	}
	if len(s.Required) != 1 || s.Required[0] != "name" {
		t.Errorf("expected required ['name'], got %v", s.Required)
	}
}

func TestSchemaProperty(t *testing.T) {
	prop := SchemaProperty{
		Type:        "string",
		Description: "A property",
	}

	if prop.Type != "string" {
		t.Errorf("expected string type, got %s", prop.Type)
	}
	if prop.Description != "A property" {
		t.Errorf("expected description, got %s", prop.Description)
	}
}

func TestToolResult(t *testing.T) {
	result := ToolResult{
		Content: "success",
		Err:     "",
	}

	if result.Content != "success" {
		t.Errorf("expected success, got %s", result.Content)
	}

	resultWithErr := ToolResult{
		Content: "",
		Err:     "error occurred",
	}

	if resultWithErr.Err != "error occurred" {
		t.Errorf("expected error, got %s", resultWithErr.Err)
	}
}

func TestToolInterface(t *testing.T) {
	// Test that mockTool implements Tool interface
	var _ Tool = &mockTool{}

	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		schema: Schema{
			Type: "object",
			Properties: map[string]SchemaProperty{
				"query": {Type: "string"},
			},
		},
		executeFunc: func(ctx context.Context, input map[string]any) (ToolResult, error) {
			return ToolResult{Content: "executed"}, nil
		},
	}

	if tool.Name() != "test_tool" {
		t.Errorf("expected test_tool, got %s", tool.Name())
	}
	if tool.Description() != "A test tool" {
		t.Errorf("expected A test tool, got %s", tool.Description())
	}
	if tool.Schema().Type != "object" {
		t.Errorf("expected object type, got %s", tool.Schema().Type)
	}

	result, err := tool.Execute(context.Background(), map[string]any{"query": "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Content != "executed" {
		t.Errorf("expected executed, got %s", result.Content)
	}
}
