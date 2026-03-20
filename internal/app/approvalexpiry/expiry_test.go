package approvalexpiry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"rag-platform/internal/agent/job"
	"rag-platform/internal/runtime/jobstore"
)

func TestExpireApprovalWaitsOnce_SettlesExpiredApproval(t *testing.T) {
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	events := jobstore.NewMemoryStore()
	wakeup := job.NewWakeupQueueMem(4)
	jobID, err := meta.Create(ctx, &job.Job{ID: "job-approval-expired", AgentID: "a1", Goal: "g1", Status: job.StatusWaiting})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := meta.UpdateStatus(ctx, jobID, job.StatusWaiting); err != nil {
		t.Fatalf("UpdateStatus waiting: %v", err)
	}
	appendEvent(t, ctx, events, jobID, jobstore.JobCreated, nil)
	appendEvent(t, ctx, events, jobID, jobstore.JobRunning, nil)
	appendWaiting(t, ctx, events, jobID, jobstore.JobWaitingPayload{
		NodeID:           "approval-node",
		WaitType:         "signal",
		WaitKind:         "human",
		CorrelationKey:   "approval-123",
		Reason:           "approval_required",
		ExpiresAtRFC3339: time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339),
	})

	n, err := ExpireApprovalWaitsOnce(ctx, meta, events, wakeup)
	if err != nil {
		t.Fatalf("ExpireApprovalWaitsOnce: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count = %d, want 1", n)
	}
	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Status != job.StatusPending {
		t.Fatalf("job status = %v, want Pending", stored.Status)
	}
	list, _, err := events.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	last := list[len(list)-1]
	if last.Type != jobstore.WaitCompleted {
		t.Fatalf("last event = %v, want WaitCompleted", last.Type)
	}
	payload, err := jobstore.ParseWaitCompletedPayload(last.Payload)
	if err != nil {
		t.Fatalf("ParseWaitCompletedPayload: %v", err)
	}
	if payload.Approval.Decision != "expired" || payload.Approval.Reason != "approval_expired" {
		t.Fatalf("unexpected approval metadata: %+v", payload.Approval)
	}
	var resumed map[string]any
	if err := json.Unmarshal(payload.Payload, &resumed); err != nil {
		t.Fatalf("unmarshal wait_completed payload: %v", err)
	}
	if resumed["decision"] != "expired" || resumed["expiry_action"] != "expired" {
		t.Fatalf("unexpected resume payload: %+v", resumed)
	}
	if jobIDReady, ok := wakeup.Receive(ctx, 10*time.Millisecond); !ok || jobIDReady != jobID {
		t.Fatalf("wakeup queue should receive expired job, got jobID=%q ok=%v", jobIDReady, ok)
	}
}

func TestExpireApprovalWaitsOnce_RejectsExpiredApproval(t *testing.T) {
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	events := jobstore.NewMemoryStore()
	wakeup := job.NewWakeupQueueMem(4)
	jobID, err := meta.Create(ctx, &job.Job{ID: "job-approval-rejected", AgentID: "a1", Goal: "g1", Status: job.StatusWaiting})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := meta.UpdateStatus(ctx, jobID, job.StatusWaiting); err != nil {
		t.Fatalf("UpdateStatus waiting: %v", err)
	}
	appendEvent(t, ctx, events, jobID, jobstore.JobCreated, nil)
	appendWaiting(t, ctx, events, jobID, jobstore.JobWaitingPayload{
		NodeID:           "approval-node",
		WaitType:         "signal",
		WaitKind:         "human",
		CorrelationKey:   "approval-reject-123",
		Reason:           "approval_required",
		ExpiresAtRFC3339: time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339),
		ResumptionContext: mustJSON(t, map[string]any{
			"expiry_action": "rejected",
		}),
	})

	n, err := ExpireApprovalWaitsOnce(ctx, meta, events, wakeup)
	if err != nil {
		t.Fatalf("ExpireApprovalWaitsOnce: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count = %d, want 1", n)
	}
	list, _, err := events.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	last := list[len(list)-1]
	if last.Type != jobstore.WaitCompleted {
		t.Fatalf("last event = %v, want WaitCompleted", last.Type)
	}
	payload, err := jobstore.ParseWaitCompletedPayload(last.Payload)
	if err != nil {
		t.Fatalf("ParseWaitCompletedPayload: %v", err)
	}
	if payload.Approval.Decision != "rejected" || payload.Approval.Reason != "approval_expired_rejected" {
		t.Fatalf("unexpected approval metadata: %+v", payload.Approval)
	}
	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Status != job.StatusPending {
		t.Fatalf("job status = %v, want Pending", stored.Status)
	}
	if jobIDReady, ok := wakeup.Receive(ctx, 10*time.Millisecond); !ok || jobIDReady != jobID {
		t.Fatalf("wakeup queue should receive rejected-expired job, got jobID=%q ok=%v", jobIDReady, ok)
	}
}

func TestExpireApprovalWaitsOnce_CancelsExpiredApproval(t *testing.T) {
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	events := jobstore.NewMemoryStore()
	wakeup := job.NewWakeupQueueMem(4)
	jobID, err := meta.Create(ctx, &job.Job{ID: "job-approval-cancelled", AgentID: "a1", Goal: "g1", Status: job.StatusWaiting})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := meta.UpdateStatus(ctx, jobID, job.StatusWaiting); err != nil {
		t.Fatalf("UpdateStatus waiting: %v", err)
	}
	appendEvent(t, ctx, events, jobID, jobstore.JobCreated, nil)
	appendWaiting(t, ctx, events, jobID, jobstore.JobWaitingPayload{
		NodeID:           "approval-node",
		WaitType:         "signal",
		WaitKind:         "human",
		CorrelationKey:   "approval-cancel-123",
		Reason:           "approval_required",
		ExpiresAtRFC3339: time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339),
		ResumptionContext: mustJSON(t, map[string]any{
			"expiry_action": "cancelled",
		}),
	})

	n, err := ExpireApprovalWaitsOnce(ctx, meta, events, wakeup)
	if err != nil {
		t.Fatalf("ExpireApprovalWaitsOnce: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count = %d, want 1", n)
	}
	stored, err := meta.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Status != job.StatusCancelled {
		t.Fatalf("job status = %v, want Cancelled", stored.Status)
	}
	list, _, err := events.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	last := list[len(list)-1]
	if last.Type != jobstore.JobCancelled {
		t.Fatalf("last event = %v, want JobCancelled", last.Type)
	}
	if jobIDReady, ok := wakeup.Receive(ctx, 10*time.Millisecond); ok {
		t.Fatalf("wakeup queue should stay empty for cancelled job, got jobID=%q", jobIDReady)
	}
}

func TestExpireApprovalWaitsOnce_SkipsNonApprovalWait(t *testing.T) {
	ctx := context.Background()
	meta := job.NewJobStoreMem()
	events := jobstore.NewMemoryStore()
	jobID, err := meta.Create(ctx, &job.Job{ID: "job-non-approval", AgentID: "a1", Goal: "g1", Status: job.StatusWaiting})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := meta.UpdateStatus(ctx, jobID, job.StatusWaiting); err != nil {
		t.Fatalf("UpdateStatus waiting: %v", err)
	}
	appendEvent(t, ctx, events, jobID, jobstore.JobCreated, nil)
	appendWaiting(t, ctx, events, jobID, jobstore.JobWaitingPayload{
		NodeID:           "wait-node",
		WaitType:         "signal",
		WaitKind:         "signal",
		CorrelationKey:   "signal-123",
		Reason:           "signal_wait",
		ExpiresAtRFC3339: time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339),
	})

	n, err := ExpireApprovalWaitsOnce(ctx, meta, events, nil)
	if err != nil {
		t.Fatalf("ExpireApprovalWaitsOnce: %v", err)
	}
	if n != 0 {
		t.Fatalf("expired count = %d, want 0", n)
	}
	stored, _ := meta.Get(ctx, jobID)
	if stored.Status != job.StatusWaiting {
		t.Fatalf("job status = %v, want Waiting", stored.Status)
	}
}

func appendEvent(t *testing.T, ctx context.Context, store jobstore.JobStore, jobID string, eventType jobstore.EventType, payload []byte) {
	t.Helper()
	_, ver, err := store.ListEvents(ctx, jobID)
	if err != nil {
		ver = 0
	}
	if _, err := store.Append(ctx, jobID, ver, jobstore.JobEvent{JobID: jobID, Type: eventType, Payload: payload}); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
}

func appendWaiting(t *testing.T, ctx context.Context, store jobstore.JobStore, jobID string, wait jobstore.JobWaitingPayload) {
	t.Helper()
	payload, err := json.Marshal(wait)
	if err != nil {
		t.Fatalf("marshal wait: %v", err)
	}
	appendEvent(t, ctx, store, jobID, jobstore.JobWaiting, payload)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return payload
}
