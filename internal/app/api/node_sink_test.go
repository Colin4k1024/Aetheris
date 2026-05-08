package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

func TestAppendJobCompleted_IncludesCommittedAnswer(t *testing.T) {
	store := jobstore.NewMemoryStore()
	sink := &nodeEventSinkImpl{store: store}
	result, _ := json.Marshal(map[string]any{
		"done":   true,
		"output": `{"answer":"hello from existing agent","final":true}`,
	})
	if err := sink.AppendCommandCommitted(context.Background(), "job-1", "external_agent_call", "external_agent_call", result, ""); err != nil {
		t.Fatalf("AppendCommandCommitted returned error: %v", err)
	}
	if err := sink.AppendJobCompleted(context.Background(), "job-1", "goal"); err != nil {
		t.Fatalf("AppendJobCompleted returned error: %v", err)
	}
	events, _, err := store.ListEvents(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	last := events[len(events)-1]
	if last.Type != jobstore.JobCompleted {
		t.Fatalf("expected last event job_completed, got %s", last.Type)
	}
	var payload map[string]any
	if err := json.Unmarshal(last.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["answer"] != "hello from existing agent" {
		t.Errorf("expected answer in job_completed payload, got %v", payload["answer"])
	}
	if payload["result"] != "hello from existing agent" {
		t.Errorf("expected result in job_completed payload, got %v", payload["result"])
	}
}
