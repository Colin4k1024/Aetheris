package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
