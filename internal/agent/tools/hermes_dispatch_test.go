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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/session"
)

func TestNewHermesDispatchTool_MissingEndpoint(t *testing.T) {
	_, err := NewHermesDispatchTool(HermesDispatchConfig{})
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}

func TestNewHermesDispatchTool_DefaultTimeout(t *testing.T) {
	tool, err := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: "http://localhost:8765"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.httpClient.Timeout != hermesDefaultACPTimeout {
		t.Errorf("expected default timeout %v, got %v", hermesDefaultACPTimeout, tool.httpClient.Timeout)
	}
}

func TestNewHermesDispatchTool_CustomTimeout(t *testing.T) {
	custom := 30 * time.Second
	tool, err := NewHermesDispatchTool(HermesDispatchConfig{
		Endpoint: "http://localhost:8765",
		Timeout:  custom,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.httpClient.Timeout != custom {
		t.Errorf("expected timeout %v, got %v", custom, tool.httpClient.Timeout)
	}
}

func TestHermesDispatchTool_Metadata(t *testing.T) {
	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: "http://localhost:8765"})

	if tool.Name() != HermesDispatchToolName {
		t.Errorf("expected name %q, got %q", HermesDispatchToolName, tool.Name())
	}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
	if tool.Protocol() != "acp" {
		t.Errorf("expected protocol 'acp', got %q", tool.Protocol())
	}
	if tool.Source() != "hermes" {
		t.Errorf("expected source 'hermes', got %q", tool.Source())
	}
	if tool.RequiredCapability() != "hermes.dispatch" {
		t.Errorf("expected capability 'hermes.dispatch', got %q", tool.RequiredCapability())
	}
}

func TestHermesDispatchTool_Schema(t *testing.T) {
	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: "http://localhost:8765"})
	schema := tool.Schema()

	if schema["type"] != "object" {
		t.Error("expected schema type 'object'")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map in schema")
	}
	if _, ok := props["task"]; !ok {
		t.Error("expected 'task' property in schema")
	}
	if _, ok := props["tools"]; !ok {
		t.Error("expected 'tools' property in schema")
	}
	if _, ok := props["context"]; !ok {
		t.Error("expected 'context' property in schema")
	}
	required, ok := schema["required"].([]any)
	if !ok || len(required) == 0 || required[0] != "task" {
		t.Error("expected 'task' to be required")
	}
}

func TestHermesDispatchTool_Execute_MissingTask(t *testing.T) {
	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: "http://localhost:8765"})
	_, err := tool.Execute(context.Background(), nil, map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for missing 'task' input")
	}
}

func TestHermesDispatchTool_Execute_Success(t *testing.T) {
	// Stand up a mock Hermes ACP server.
	var receivedReq hermesDispatchRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != hermesACPDispatchPath {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hermesDispatchResponse{
			SessionID: "hermes-session-42",
			Output:    "Code review complete",
			Done:      true,
		})
	}))
	defer srv.Close()

	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: srv.URL})
	sess := session.New("job-123")

	result, err := tool.Execute(context.Background(), sess, map[string]any{
		"task":  "Review the pull request and identify security issues",
		"tools": []any{"terminal", "file"},
		"context": map[string]any{
			"pr_number": 42,
		},
	}, nil)
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
	if m["output"] != "Code review complete" {
		t.Errorf("unexpected output: %v", m["output"])
	}
	if m["session_id"] != "hermes-session-42" {
		t.Errorf("unexpected session_id: %v", m["session_id"])
	}
	if m["source"] != "hermes" {
		t.Errorf("expected source='hermes', got %v", m["source"])
	}
	if m["job_id"] != "job-123" {
		t.Errorf("expected job_id='job-123', got %v", m["job_id"])
	}
	if receivedReq.Task != "Review the pull request and identify security issues" {
		t.Errorf("unexpected task forwarded: %q", receivedReq.Task)
	}
	if receivedReq.JobID != "job-123" {
		t.Errorf("expected job_id='job-123' in request, got %q", receivedReq.JobID)
	}
	if len(receivedReq.Tools) != 2 {
		t.Errorf("expected 2 tools forwarded, got %d", len(receivedReq.Tools))
	}
}

func TestHermesDispatchTool_Execute_RemoteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hermesDispatchResponse{
			Error: "hermes agent crashed",
			Done:  false,
		})
	}))
	defer srv.Close()

	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: srv.URL})
	_, err := tool.Execute(context.Background(), nil, map[string]any{"task": "do something"}, nil)
	if err == nil {
		t.Fatal("expected error for remote error response")
	}
}

func TestHermesDispatchTool_Execute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: srv.URL})
	_, err := tool.Execute(context.Background(), nil, map[string]any{"task": "do something"}, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 503 response")
	}
}

func TestHermesDispatchTool_Execute_NoSession(t *testing.T) {
	var receivedReq hermesDispatchRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedReq)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hermesDispatchResponse{Done: true, Output: "ok"})
	}))
	defer srv.Close()

	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: srv.URL})
	_, err := tool.Execute(context.Background(), nil, map[string]any{"task": "do something"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedReq.JobID != "" {
		t.Errorf("expected empty job_id for nil session, got %q", receivedReq.JobID)
	}
}

func TestHermesDispatchTool_ImplementsInterfaces(t *testing.T) {
	tool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: "http://localhost:8765"})

	var _ Tool = tool
	var _ ToolWithCapability = tool
	var _ ToolWithMetadata = tool
}

func TestHermesDispatchTool_Registerable(t *testing.T) {
	dispatchTool, _ := NewHermesDispatchTool(HermesDispatchConfig{Endpoint: "http://localhost:8765"})
	reg := NewRegistry()
	reg.Register(dispatchTool)

	got, ok := reg.Get(HermesDispatchToolName)
	if !ok {
		t.Fatal("expected to retrieve hermes_dispatch from registry")
	}
	if got.Name() != HermesDispatchToolName {
		t.Errorf("unexpected name: %q", got.Name())
	}
}
