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
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
)

type testMockTool struct{}

func (m *testMockTool) Name() string        { return "mock_tool" }
func (m *testMockTool) Description() string { return "A mock tool for testing" }
func (m *testMockTool) Schema() tool.Schema { return tool.Schema{} }
func (m *testMockTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
	return tool.ToolResult{Content: "executed"}, nil
}

func TestWrap(t *testing.T) {
	mock := &testMockTool{}
	wrapped := Wrap(mock)

	if wrapped.Name() != "mock_tool" {
		t.Errorf("expected name 'mock_tool', got '%s'", wrapped.Name())
	}

	if wrapped.Description() != "A mock tool for testing" {
		t.Errorf("expected description, got '%s'", wrapped.Description())
	}
}

func TestWrappedTool_Execute(t *testing.T) {
	mock := &testMockTool{}
	wrapped := Wrap(mock)

	result, err := wrapped.Execute(context.Background(), nil, map[string]any{"key": "value"}, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Result is tool.ToolResult which has String() method
	tr, ok := result.(tool.ToolResult)
	if !ok {
		t.Fatalf("expected tool.ToolResult, got %T", result)
	}
	if tr.Content != "executed" {
		t.Errorf("expected 'executed', got '%s'", tr.Content)
	}
}

func TestWrappedTool_Schema(t *testing.T) {
	mock := &testMockTool{}
	wrapped := Wrap(mock)

	schema := wrapped.Schema()
	if schema == nil {
		t.Error("expected non-nil schema")
	}
}
