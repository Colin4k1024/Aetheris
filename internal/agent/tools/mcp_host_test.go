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

	"rag-platform/internal/runtime/session"
)

type mockMCPInvoker struct {
	result any
	err    error
}

func (m *mockMCPInvoker) CallTool(ctx context.Context, server string, tool string, input map[string]any) (any, error) {
	return m.result, m.err
}

func TestNewMCPHost(t *testing.T) {
	invoker := &mockMCPInvoker{}
	host := NewMCPHost(invoker)
	if host == nil {
		t.Fatal("expected non-nil MCPHost")
	}
	if host.invoker != invoker {
		t.Error("invoker not set correctly")
	}
}

func TestMCPHost_RegisterServer(t *testing.T) {
	host := NewMCPHost(&mockMCPInvoker{})

	host.RegisterServer("server1")
	if !host.HasServer("server1") {
		t.Error("server1 should be registered")
	}

	host.RegisterServer("")
	if host.HasServer("") {
		t.Error("empty server name should not be registered")
	}
}

func TestMCPHost_HasServer(t *testing.T) {
	host := NewMCPHost(&mockMCPInvoker{})

	if host.HasServer("server1") {
		t.Error("server1 should not exist initially")
	}

	host.RegisterServer("server1")
	if !host.HasServer("server1") {
		t.Error("server1 should be registered")
	}
}

func TestMCPHost_RegisterToolAdapter(t *testing.T) {
	invoker := &mockMCPInvoker{}
	host := NewMCPHost(invoker)
	host.RegisterServer("server1")

	reg := NewRegistry()
	manifest := ToolManifest{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]any{"type": "object"},
	}

	err := host.RegisterToolAdapter(reg, "server1", manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Register without server should fail
	err = host.RegisterToolAdapter(reg, "nonexistent", manifest)
	if err == nil {
		t.Error("expected error for nonexistent server")
	}

	// Register without registry should fail
	err = host.RegisterToolAdapter(nil, "server1", manifest)
	if err == nil {
		t.Error("expected error for nil registry")
	}

	// Register without name should fail
	err = host.RegisterToolAdapter(reg, "server1", ToolManifest{})
	if err == nil {
		t.Error("expected error for empty name")
	}

	// Register without invoker should fail
	host2 := NewMCPHost(nil)
	err = host2.RegisterToolAdapter(reg, "server1", manifest)
	if err == nil {
		t.Error("expected error for nil invoker")
	}
}

func TestMCPHost_RegisterToolAdapter_Retrieve(t *testing.T) {
	invoker := &mockMCPInvoker{}
	host := NewMCPHost(invoker)
	host.RegisterServer("server1")

	reg := NewRegistry()
	manifest := ToolManifest{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]any{"type": "object"},
	}

	err := host.RegisterToolAdapter(reg, "server1", manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tool, ok := reg.Get("test-tool")
	if !ok || tool == nil {
		t.Fatal("expected to retrieve registered tool")
	}
	if tool.Name() != "test-tool" {
		t.Errorf("expected name test-tool, got %s", tool.Name())
	}
	if tool.Description() != "A test tool" {
		t.Errorf("expected description 'A test tool', got %s", tool.Description())
	}
}

func TestMCPToolAdapter_Name(t *testing.T) {
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1"},
		invoker:  &mockMCPInvoker{},
	}
	if adapter.Name() != "tool1" {
		t.Errorf("expected tool1, got %s", adapter.Name())
	}
}

func TestMCPToolAdapter_Description(t *testing.T) {
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1", Description: "desc"},
		invoker:  &mockMCPInvoker{},
	}
	if adapter.Description() != "desc" {
		t.Errorf("expected desc, got %s", adapter.Description())
	}
}

func TestMCPToolAdapter_Schema(t *testing.T) {
	schema := map[string]any{"type": "object"}
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1", InputSchema: schema},
		invoker:  &mockMCPInvoker{},
	}
	if adapter.Schema()["type"] != "object" {
		t.Error("schema not set correctly")
	}
}

func TestMCPToolAdapter_RequiredCapability(t *testing.T) {
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1"},
		invoker:  &mockMCPInvoker{},
	}
	cap := adapter.RequiredCapability()
	if cap != "mcp.server1.tool1" {
		t.Errorf("expected mcp.server1.tool1, got %s", cap)
	}
}

func TestMCPToolAdapter_Protocol(t *testing.T) {
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1"},
		invoker:  &mockMCPInvoker{},
	}
	if adapter.Protocol() != "mcp" {
		t.Errorf("expected mcp, got %s", adapter.Protocol())
	}
}

func TestMCPToolAdapter_Source(t *testing.T) {
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1"},
		invoker:  &mockMCPInvoker{},
	}
	if adapter.Source() != "server1" {
		t.Errorf("expected server1, got %s", adapter.Source())
	}
}

func TestMCPToolAdapter_Execute(t *testing.T) {
	invoker := &mockMCPInvoker{result: "test result"}
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1"},
		invoker:  invoker,
	}

	ctx := context.Background()
	sess := session.New("test-session")
	result, err := adapter.Execute(ctx, sess, map[string]any{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	if m["done"] != true {
		t.Error("expected done=true")
	}
	if m["output"] != "test result" {
		t.Errorf("expected test result, got %v", m["output"])
	}
}

func TestMCPToolAdapter_Execute_NilInvoker(t *testing.T) {
	adapter := &mcpToolAdapter{
		server:   "server1",
		manifest: ToolManifest{Name: "tool1"},
		invoker:  nil,
	}

	ctx := context.Background()
	sess := session.New("test-session")
	_, err := adapter.Execute(ctx, sess, nil, nil)
	if err == nil {
		t.Error("expected error for nil invoker")
	}
}
