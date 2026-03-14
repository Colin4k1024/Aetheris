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

package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockTransport is a test Transport that returns preconfigured responses.
type mockTransport struct {
	mu        sync.Mutex
	requests  []*JSONRPCRequest
	responses []*JSONRPCResponse
	idx       int
	closed    bool
}

func newMockTransport(responses ...*JSONRPCResponse) *mockTransport {
	return &mockTransport{responses: responses}
}

func (m *mockTransport) Send(msg *JSONRPCRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, msg)
	return nil
}

func (m *mockTransport) Receive() (*JSONRPCResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx >= len(m.responses) {
		return nil, nil
	}
	resp := m.responses[m.idx]
	m.idx++
	return resp, nil
}

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockTransport) sentRequests() []*JSONRPCRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*JSONRPCRequest(nil), m.requests...)
}

func makeJSONRaw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestClient_Initialize(t *testing.T) {
	initResult := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapability{
			Tools: &ToolsCapability{ListChanged: false},
		},
		ServerInfo: Implementation{Name: "test-server", Version: "1.0"},
	}
	toolsList := ToolsListResult{
		Tools: []MCPToolDef{
			{Name: "read_file", Description: "Read a file", InputSchema: map[string]any{"type": "object"}},
			{Name: "write_file", Description: "Write a file", InputSchema: map[string]any{"type": "object"}},
		},
	}

	transport := newMockTransport(
		&JSONRPCResponse{JSONRPC: "2.0", ID: 1, Result: makeJSONRaw(initResult)},
		&JSONRPCResponse{JSONRPC: "2.0", ID: 2, Result: makeJSONRaw(toolsList)},
	)

	client := NewClient(ClientConfig{
		Name:        "test",
		Transport:   transport,
		InitTimeout: 5 * time.Second,
		CallTimeout: 5 * time.Second,
	})

	err := client.Initialize(context.Background())
	require.NoError(t, err)

	// Check server info
	info := client.ServerInfo()
	require.NotNil(t, info)
	require.Equal(t, "test-server", info.ServerInfo.Name)

	// Check discovered tools
	tools := client.Tools()
	require.Len(t, tools, 2)
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	require.True(t, names["read_file"])
	require.True(t, names["write_file"])

	// Verify requests sent
	reqs := transport.sentRequests()
	require.Len(t, reqs, 2)
	require.Equal(t, MethodInitialize, reqs[0].Method)
	require.Equal(t, MethodToolsList, reqs[1].Method)
}

func TestClient_CallTool(t *testing.T) {
	callResult := ToolsCallResult{
		Content: []ContentBlock{{Type: "text", Text: "file content here"}},
	}

	transport := newMockTransport(
		&JSONRPCResponse{JSONRPC: "2.0", ID: 1, Result: makeJSONRaw(callResult)},
	)

	client := NewClient(ClientConfig{
		Name:        "test",
		Transport:   transport,
		CallTimeout: 5 * time.Second,
	})

	result, err := client.CallTool(context.Background(), "read_file", map[string]any{"path": "/tmp/test.txt"})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)
	require.Equal(t, "file content here", result.Content[0].Text)
}

func TestClient_CallTool_Error(t *testing.T) {
	callResult := ToolsCallResult{
		Content: []ContentBlock{{Type: "text", Text: "file not found"}},
		IsError: true,
	}

	transport := newMockTransport(
		&JSONRPCResponse{JSONRPC: "2.0", ID: 1, Result: makeJSONRaw(callResult)},
	)

	client := NewClient(ClientConfig{
		Name:        "test",
		Transport:   transport,
		CallTimeout: 5 * time.Second,
	})

	result, err := client.CallTool(context.Background(), "read_file", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "file not found")
	require.NotNil(t, result) // result is still returned even on error
}

func TestClient_CallTool_RPCError(t *testing.T) {
	transport := newMockTransport(
		&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error:   &JSONRPCError{Code: ErrCodeMethodNotFound, Message: "tool not found"},
		},
	)

	client := NewClient(ClientConfig{
		Name:        "test",
		Transport:   transport,
		CallTimeout: 5 * time.Second,
	})

	_, err := client.CallTool(context.Background(), "nonexistent", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tool not found")
}

func TestClient_Close(t *testing.T) {
	transport := newMockTransport()
	client := NewClient(ClientConfig{
		Name:      "test",
		Transport: transport,
	})

	err := client.Close()
	require.NoError(t, err)
	require.True(t, transport.closed)
}

func TestManager_CallTool_NotConnected(t *testing.T) {
	mgr := NewManager(nil)
	_, err := mgr.CallTool(context.Background(), "unknown-server", "tool", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not connected")
}

func TestManager_EmptyConfig(t *testing.T) {
	mgr := NewManager(nil)
	err := mgr.ConnectAll(context.Background(), ManagerConfig{})
	require.NoError(t, err)
	require.Empty(t, mgr.ServerNames())
}
