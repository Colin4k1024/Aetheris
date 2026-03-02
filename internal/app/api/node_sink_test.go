package api

import (
	"context"
	"testing"

	agentexec "rag-platform/internal/agent/runtime/executor"
	"rag-platform/internal/runtime/eino"
	"rag-platform/internal/runtime/jobstore"
)

func TestNodeEventSink_SyncToolInvocationToRunStore(t *testing.T) {
	ctx := context.Background()
	runStore := eino.NewMemoryRunStore()
	sink := NewNodeEventSinkWithRunStore(jobstore.NewMemoryStore(), runStore)

	payload := &agentexec.ToolInvocationStartedPayload{
		InvocationID:   "tc-auto-1",
		ToolName:       "search",
		IdempotencyKey: "idem-1",
		StartedAt:      "2026-02-24T10:00:00Z",
	}
	if err := sink.AppendToolInvocationStarted(ctx, "job-1", "node-search", payload); err != nil {
		t.Fatalf("AppendToolInvocationStarted: %v", err)
	}

	run, err := runStore.GetRun(ctx, "job-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.WorkflowID != "agent_job" {
		t.Fatalf("run workflow_id=%q, want agent_job", run.WorkflowID)
	}
	if _, err := runStore.PauseRun(ctx, "job-1", "manual", "alice"); err != nil {
		t.Fatalf("PauseRun: %v", err)
	}
	if _, err := runStore.ResumeRun(ctx, "job-1", eino.ResumeRunRequest{
		Mode:           eino.ResumeModeFromToolCall,
		FromToolCallID: "tc-auto-1",
		Strategy:       eino.ResumeStrategyReuseSuccessfulEffects,
		Operator:       "alice",
	}); err != nil {
		t.Fatalf("ResumeRun: %v", err)
	}

	finished := &agentexec.ToolInvocationFinishedPayload{
		InvocationID:   "tc-auto-1",
		IdempotencyKey: "idem-1",
		Outcome:        agentexec.ToolInvocationOutcomeSuccess,
		Result:         []byte(`{"ok":true}`),
		FinishedAt:     "2026-02-24T10:01:00Z",
	}
	if err := sink.AppendToolInvocationFinished(ctx, "job-1", "node-search", finished); err != nil {
		t.Fatalf("AppendToolInvocationFinished: %v", err)
	}
	events, _, err := runStore.ListEvents(ctx, "job-1", 0, 50)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	foundEnded := false
	for _, e := range events {
		if e.Type == eino.EventTypeToolCallEnded {
			if got, _ := e.Payload["status"].(string); got == "SUCCEEDED" {
				foundEnded = true
				break
			}
		}
	}
	if !foundEnded {
		t.Fatalf("expected tool_call_ended event with SUCCEEDED status, events=%v", events)
	}
}
