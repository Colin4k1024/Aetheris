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
	"context"
	"encoding/json"
	"testing"

	"rag-platform/internal/runtime/session"
)

// mockRuntimeTool 用于测试的 Mock Tool
type mockRuntimeTool struct {
	name   string
	desc   string
	schema map[string]any
	execFn func(ctx context.Context, input map[string]any) (any, error)
}

func (m *mockRuntimeTool) Name() string           { return m.name }
func (m *mockRuntimeTool) Description() string    { return m.desc }
func (m *mockRuntimeTool) Schema() map[string]any { return m.schema }
func (m *mockRuntimeTool) Execute(ctx context.Context, _ *session.Session, input map[string]any, _ interface{}) (any, error) {
	if m.execFn != nil {
		return m.execFn(ctx, input)
	}
	return "mock result", nil
}

// mockRegistry 用于测试的 Mock Registry
type mockRegistry struct {
	tools []RuntimeTool
}

func (r *mockRegistry) List() []RuntimeTool {
	return r.tools
}

func TestRegistryToolBridge_EinoTools(t *testing.T) {
	reg := &mockRegistry{
		tools: []RuntimeTool{
			&mockRuntimeTool{
				name: "search",
				desc: "搜索工具",
				schema: map[string]any{
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "搜索关键词",
						},
					},
					"required": []any{"query"},
				},
			},
			&mockRuntimeTool{
				name:   "calculator",
				desc:   "计算器",
				schema: map[string]any{"expression": "数学表达式"},
			},
		},
	}

	bridge := NewRegistryToolBridge(reg)
	tools := bridge.EinoTools()

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	// 验证第一个工具
	info0, err := tools[0].Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error: %v", err)
	}
	if info0.Name != "search" {
		t.Errorf("expected name 'search', got %q", info0.Name)
	}
	if info0.Desc != "搜索工具" {
		t.Errorf("expected desc '搜索工具', got %q", info0.Desc)
	}

	// 验证第二个工具
	info1, err := tools[1].Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error: %v", err)
	}
	if info1.Name != "calculator" {
		t.Errorf("expected name 'calculator', got %q", info1.Name)
	}
}

func TestRegistryToolBridge_NilRegistry(t *testing.T) {
	bridge := NewRegistryToolBridge(nil)
	tools := bridge.EinoTools()
	if tools != nil {
		t.Errorf("expected nil tools for nil registry, got %v", tools)
	}
}

func TestRegistryToolBridge_EmptyRegistry(t *testing.T) {
	reg := &mockRegistry{tools: nil}
	bridge := NewRegistryToolBridge(reg)
	tools := bridge.EinoTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestRegistryToolAdapter_InvokableRun(t *testing.T) {
	called := false
	tool := &mockRuntimeTool{
		name: "echo",
		desc: "回声工具",
		schema: map[string]any{
			"properties": map[string]any{
				"text": map[string]any{"type": "string", "description": "输入文本"},
			},
		},
		execFn: func(ctx context.Context, input map[string]any) (any, error) {
			called = true
			text, _ := input["text"].(string)
			return "echo: " + text, nil
		},
	}

	adapter := &registryToolAdapter{tool: tool}

	// 测试 InvokableRun
	result, err := adapter.InvokableRun(context.Background(), `{"text":"hello"}`)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}
	if !called {
		t.Error("tool Execute was not called")
	}
	if result != "echo: hello" {
		t.Errorf("expected 'echo: hello', got %q", result)
	}
}

func TestRegistryToolAdapter_InvokableRunWithSession(t *testing.T) {
	var gotSession *session.Session
	tool := &mockRuntimeTool{
		name:   "session_tool",
		desc:   "Session 感知",
		schema: map[string]any{},
	}
	// Override Execute to capture session
	tool.execFn = func(ctx context.Context, input map[string]any) (any, error) {
		// Can't directly access session from execFn, but the adapter passes nil for mockRuntimeTool
		// This tests the context-based session passing
		gotSession = sessionFromContext(ctx)
		return "ok", nil
	}
	// Override the mock's Execute to use context
	sessionTool := &sessionAwareMock{
		mockRuntimeTool: *tool,
	}

	adapter := &registryToolAdapter{tool: sessionTool}

	// 创建带 Session 的 context
	sess := session.New("test-session")
	ctx := WithSession(context.Background(), sess)

	_, err := adapter.InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}
	if gotSession == nil || gotSession.ID != "test-session" {
		t.Error("session was not passed through context")
	}
}

type sessionAwareMock struct {
	mockRuntimeTool
}

func (m *sessionAwareMock) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
	if m.execFn != nil {
		// Store session in context for verification
		return m.execFn(ctx, input)
	}
	return "mock", nil
}

func TestRegistryToolAdapter_InvokableRunInvalidJSON(t *testing.T) {
	tool := &mockRuntimeTool{
		name: "test",
		desc: "test",
		execFn: func(ctx context.Context, input map[string]any) (any, error) {
			// Should get "raw" key when JSON is invalid
			if _, ok := input["raw"]; !ok {
				t.Error("expected 'raw' key for invalid JSON")
			}
			return "handled", nil
		},
	}
	adapter := &registryToolAdapter{tool: tool}
	result, err := adapter.InvokableRun(context.Background(), "not json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "handled" {
		t.Errorf("expected 'handled', got %q", result)
	}
}

func TestRegistryToolAdapter_InvokableRunReturnsToolResult(t *testing.T) {
	tool := &mockRuntimeTool{
		name: "test",
		desc: "test",
		execFn: func(ctx context.Context, input map[string]any) (any, error) {
			return RuntimeToolResult{Done: true, Output: "tool output"}, nil
		},
	}
	adapter := &registryToolAdapter{tool: tool}
	result, err := adapter.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "tool output" {
		t.Errorf("expected 'tool output', got %q", result)
	}
}

func TestRegistryToolAdapter_InvokableRunReturnsMap(t *testing.T) {
	tool := &mockRuntimeTool{
		name: "test",
		desc: "test",
		execFn: func(ctx context.Context, input map[string]any) (any, error) {
			return map[string]any{"key": "value"}, nil
		},
	}
	adapter := &registryToolAdapter{tool: tool}
	result, err := adapter.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value, got %v", m)
	}
}

func TestSchemaMapToParams_JSONSchemaFormat(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "搜索关键词",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "返回数量",
			},
		},
		"required": []any{"query"},
	}
	params := schemaMapToParams(s)
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if !params["query"].Required {
		t.Error("query should be required")
	}
	if params["top_k"].Required {
		t.Error("top_k should not be required")
	}
}

func TestSchemaMapToParams_FlatFormat(t *testing.T) {
	s := map[string]any{
		"query": "搜索关键词",
		"top_k": "返回数量",
	}
	params := schemaMapToParams(s)
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
}

func TestSchemaMapToParams_Nil(t *testing.T) {
	params := schemaMapToParams(nil)
	if params != nil {
		t.Errorf("expected nil, got %v", params)
	}
}

func TestWithSession_RoundTrip(t *testing.T) {
	sess := session.New("test-123")
	ctx := WithSession(context.Background(), sess)
	got := sessionFromContext(ctx)
	if got == nil {
		t.Fatal("session not found in context")
	}
	if got.ID != "test-123" {
		t.Errorf("expected session ID 'test-123', got %q", got.ID)
	}
}

func TestSessionFromContext_NoSession(t *testing.T) {
	got := sessionFromContext(context.Background())
	if got != nil {
		t.Error("expected nil session from empty context")
	}
}
