package eino

import (
	"context"
	"testing"
)

func TestMemoryRunStore_ResumeRequiresOwnedToolCall(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	run, err := store.CreateRun(ctx, &Run{WorkflowID: "wf_resume"})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if _, err := store.PauseRun(ctx, run.ID, "manual", "alice"); err != nil {
		t.Fatalf("PauseRun: %v", err)
	}

	_, err = store.ResumeRun(ctx, run.ID, ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc_missing",
		Strategy:       ResumeStrategyReuseSuccessfulEffects,
	})
	if err == nil || err != ErrToolCallNotFound {
		t.Fatalf("ResumeRun without owned tool call err=%v, want %v", err, ErrToolCallNotFound)
	}

	if _, err := store.UpsertToolCall(ctx, &ToolCall{ID: "tc_ok", RunID: run.ID, ToolName: "search"}); err != nil {
		t.Fatalf("UpsertToolCall: %v", err)
	}
	if _, err := store.ResumeRun(ctx, run.ID, ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc_ok",
		Strategy:       ResumeStrategyReuseSuccessfulEffects,
	}); err != nil {
		t.Fatalf("ResumeRun with owned tool call: %v", err)
	}
}

func TestMemoryRunStore_UpsertToolCallWritesEndedEvent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	run, err := store.CreateRun(ctx, &Run{WorkflowID: "wf_tool_event"})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	call := &ToolCall{ID: "tc-1", RunID: run.ID, ToolName: "search", Status: "STARTED"}
	if _, err := store.UpsertToolCall(ctx, call); err != nil {
		t.Fatalf("UpsertToolCall started: %v", err)
	}
	call.Status = "SUCCEEDED"
	if _, err := store.UpsertToolCall(ctx, call); err != nil {
		t.Fatalf("UpsertToolCall succeeded: %v", err)
	}

	events, _, err := store.ListEvents(ctx, run.ID, 0, 20)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("events len=%d, want >=3", len(events))
	}
	last := events[len(events)-1]
	if last.Type != EventTypeToolCallEnded {
		t.Fatalf("last event type=%s, want %s", last.Type, EventTypeToolCallEnded)
	}
	if got, _ := last.Payload["status"].(string); got != "SUCCEEDED" {
		t.Fatalf("last event status=%q, want SUCCEEDED", got)
	}
}
