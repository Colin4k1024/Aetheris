package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentexec "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/tools"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/session"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func TestExternalAgentCallTool_Execute(t *testing.T) {
	t.Setenv("CUSTOMER_BOT_TOKEN", "secret-token")
	var got struct {
		Auth           string
		IdempotencyKey string
		JobID          string
		AgentID        string
		Body           externalAgentRequest
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Auth = r.Header.Get("Authorization")
		got.IdempotencyKey = r.Header.Get("Idempotency-Key")
		got.JobID = r.Header.Get("X-Aetheris-Job-ID")
		got.AgentID = r.Header.Get("X-Aetheris-Agent-ID")
		if err := json.NewDecoder(r.Body).Decode(&got.Body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"answer":   "hello from existing agent",
			"final":    true,
			"metadata": map[string]any{"source": "test"},
		})
	}))
	defer server.Close()

	tool := NewExternalAgentCallTool(map[string]config.AgentExternalConfig{
		"customer_support_bot": {
			URL:      server.URL,
			Timeout:  "2s",
			TokenEnv: "CUSTOMER_BOT_TOKEN",
		},
	})
	ctx := agentexec.WithJobID(context.Background(), "job-123")
	out, err := tool.Execute(ctx, session.New("sess-123"), map[string]any{
		"agent_id":        "customer_support_bot",
		"message":         "hi",
		"idempotency_key": "idem-123",
	}, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	result, ok := out.(tools.ToolResult)
	if !ok {
		t.Fatalf("expected map result, got %T", out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Output), &payload); err != nil {
		t.Fatalf("decode tool output: %v", err)
	}
	if payload["answer"] != "hello from existing agent" {
		t.Errorf("unexpected answer: %v", payload["answer"])
	}
	if got.Auth != "Bearer secret-token" {
		t.Errorf("expected bearer token, got %q", got.Auth)
	}
	if got.IdempotencyKey != "idem-123" {
		t.Errorf("expected idempotency key, got %q", got.IdempotencyKey)
	}
	if got.JobID != "job-123" {
		t.Errorf("expected job id header, got %q", got.JobID)
	}
	if got.AgentID != "customer_support_bot" {
		t.Errorf("expected agent id header, got %q", got.AgentID)
	}
	if got.Body.Message != "hi" {
		t.Errorf("expected message body, got %q", got.Body.Message)
	}
	if got.Body.SessionID != "sess-123" {
		t.Errorf("expected session id body, got %q", got.Body.SessionID)
	}
	if got.Body.Metadata["agent_id"] != "customer_support_bot" {
		t.Errorf("expected metadata agent_id, got %v", got.Body.Metadata["agent_id"])
	}
}

func TestExternalAgentCallTool_Execute_ErrorMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	tool := NewExternalAgentCallTool(map[string]config.AgentExternalConfig{
		"customer_support_bot": {URL: server.URL, Timeout: "2s"},
	})
	_, err := tool.Execute(context.Background(), session.New("sess-123"), map[string]any{
		"agent_id": "customer_support_bot",
		"message":  "hi",
	}, nil)
	if err == nil {
		t.Fatalf("expected upstream error")
	}
}

func TestExternalAgentCallTool_SSELegacy_HappyPath(t *testing.T) {
	var gotAccept string
	var gotBody sseAgentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: Hello\n\ndata: World\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()

	tool := NewExternalAgentCallTool(map[string]config.AgentExternalConfig{
		"sse_agent": {URL: server.URL, Protocol: "sse_legacy", AgentID: "research-agent"},
	})
	out, err := tool.Execute(context.Background(), session.New("sess-42"), map[string]any{
		"agent_id": "sse_agent",
		"message":  "summarise this",
	}, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	result, ok := out.(tools.ToolResult)
	if !ok {
		t.Fatalf("expected ToolResult, got %T", out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["answer"] != "HelloWorld" {
		t.Errorf("unexpected answer: %v", payload["answer"])
	}
	if gotAccept != "text/event-stream" {
		t.Errorf("expected Accept: text/event-stream, got %q", gotAccept)
	}
	if gotBody.AgentID != "research-agent" {
		t.Errorf("expected agent_id=research-agent, got %q", gotBody.AgentID)
	}
	if gotBody.SessionID != "sess-42" {
		t.Errorf("expected session_id=sess-42, got %q", gotBody.SessionID)
	}
	if gotBody.Message != "summarise this" {
		t.Errorf("expected message=summarise this, got %q", gotBody.Message)
	}
}

func TestExternalAgentCallTool_SSELegacy_MissingDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Stream ends without [DONE]
		fmt.Fprint(w, "data: partial\n\n")
	}))
	defer server.Close()

	tool := NewExternalAgentCallTool(map[string]config.AgentExternalConfig{
		"sse_agent": {URL: server.URL, Protocol: "sse_legacy"},
	})
	_, err := tool.Execute(context.Background(), session.New("s"), map[string]any{
		"agent_id": "sse_agent",
		"message":  "hello",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing [DONE], got nil")
	}
	if !strings.Contains(err.Error(), "[DONE]") {
		t.Errorf("error should mention [DONE]: %v", err)
	}
}

func TestExternalAgentCallTool_SSELegacy_ByteLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send a response that exceeds maxExternalResponseBytes (10 MiB).
		// We use many small tokens to build up the limit.
		token := strings.Repeat("x", 1024) // 1 KiB token
		for i := 0; i < 11*1024; i++ {     // 11 MiB total
			fmt.Fprintf(w, "data: %s\n\n", token)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	tool := NewExternalAgentCallTool(map[string]config.AgentExternalConfig{
		"sse_agent": {URL: server.URL, Protocol: "sse_legacy"},
	})
	_, err := tool.Execute(context.Background(), session.New("s"), map[string]any{
		"agent_id": "sse_agent",
		"message":  "hello",
	}, nil)
	if err == nil {
		t.Fatal("expected error for oversized SSE response, got nil")
	}
	if !strings.Contains(err.Error(), "MiB limit") {
		t.Errorf("error should mention MiB limit: %v", err)
	}
}

func TestExternalAgentCallTool_SSELegacy_MetadataRejected(t *testing.T) {
	tool := NewExternalAgentCallTool(map[string]config.AgentExternalConfig{
		"sse_agent": {URL: "http://localhost:9999", Protocol: "sse_legacy"},
	})
	_, err := tool.Execute(context.Background(), session.New("s"), map[string]any{
		"agent_id": "sse_agent",
		"message":  "hello",
		"metadata": map[string]any{"key": "value"},
	}, nil)
	if err == nil {
		t.Fatal("expected error when metadata provided for sse_legacy, got nil")
	}
	if !strings.Contains(err.Error(), "metadata") {
		t.Errorf("error should mention metadata: %v", err)
	}
}

func TestExternalAgentCallTool_UnknownProtocol(t *testing.T) {
	tool := NewExternalAgentCallTool(map[string]config.AgentExternalConfig{
		"bad_agent": {URL: "http://localhost:9999", Protocol: "grpc"},
	})
	_, err := tool.Execute(context.Background(), session.New("s"), map[string]any{
		"agent_id": "bad_agent",
		"message":  "hello",
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported protocol, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported protocol") {
		t.Errorf("error should mention 'unsupported protocol': %v", err)
	}
}
