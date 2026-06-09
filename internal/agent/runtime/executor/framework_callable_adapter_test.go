package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
)

func TestFrameworkCallableNodeAdapter_Invoke(t *testing.T) {
	var got frameworkCallableRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aetheris/nodes/load_question/invoke" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{"prompt": "search this"},
			"final":  true,
		})
	}))
	defer server.Close()

	adapter := &FrameworkCallableNodeAdapter{}
	run, err := adapter.ToNodeRunner(&planner.TaskNode{
		ID:   "load_question",
		Type: planner.NodeFrameworkCallable,
		Config: map[string]any{
			"url":       server.URL,
			"framework": "langchain",
		},
	}, nil)
	if err != nil {
		t.Fatalf("ToNodeRunner: %v", err)
	}
	payload := NewAgentDAGPayload("goal", "agent", "sess-1")
	payload.Results["previous"] = map[string]any{"ok": true}
	out, err := run(WithJobID(context.Background(), "job-1"), payload)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if got.JobID != "job-1" || got.NodeID != "load_question" || got.SessionID != "sess-1" {
		t.Fatalf("unexpected request envelope: %+v", got)
	}
	if got.Input["goal"] != "goal" || got.Input["message"] != "goal" {
		t.Fatalf("expected original goal in input, got %+v", got.Input)
	}
	if _, leaked := got.Input["url"]; leaked {
		t.Fatalf("transport config leaked into callable input: %+v", got.Input)
	}
	result, ok := out.Results["load_question"].(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", out.Results["load_question"])
	}
	output, ok := result["output"].(map[string]any)
	if !ok || output["prompt"] != "search this" {
		t.Fatalf("unexpected output: %+v", result)
	}
}
