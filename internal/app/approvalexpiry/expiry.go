package approvalexpiry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"rag-platform/internal/agent/job"
	"rag-platform/internal/runtime/jobstore"
)

type blockedJobLister interface {
	ListByStatuses(ctx context.Context, statuses []job.JobStatus, tenantID string) ([]*job.Job, error)
}

func ExpireApprovalWaitsOnce(ctx context.Context, metaStore job.JobStore, eventStore jobstore.JobStore, wakeupQueue job.WakeupQueue) (int, error) {
	if metaStore == nil || eventStore == nil {
		return 0, nil
	}
	lister, ok := metaStore.(blockedJobLister)
	if !ok {
		return 0, nil
	}
	jobs, err := lister.ListByStatuses(ctx, []job.JobStatus{job.StatusWaiting, job.StatusParked}, "")
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	expired := 0
	for _, j := range jobs {
		settled, settleErr := expireSingleApprovalWait(ctx, metaStore, eventStore, wakeupQueue, now, j)
		if settleErr != nil {
			continue
		}
		if settled {
			expired++
		}
	}
	return expired, nil
}

func expireSingleApprovalWait(ctx context.Context, metaStore job.JobStore, eventStore jobstore.JobStore, wakeupQueue job.WakeupQueue, now time.Time, j *job.Job) (bool, error) {
	if j == nil {
		return false, nil
	}
	events, ver, err := eventStore.ListEvents(ctx, j.ID)
	if err != nil {
		return false, err
	}
	wait, ok := latestPendingApprovalWait(events)
	if !ok || wait.CorrelationKey == "" || wait.ExpiresAtRFC3339 == "" {
		return false, nil
	}
	expiresAt, err := time.Parse(time.RFC3339, wait.ExpiresAtRFC3339)
	if err != nil || expiresAt.After(now) {
		return false, nil
	}
	action := expiryAction(wait)
	if lastEventIsWaitCompletedWithCorrelationKey(events, wait.CorrelationKey) {
		_ = metaStore.UpdateStatus(ctx, j.ID, job.StatusPending)
		if wakeupQueue != nil {
			_ = wakeupQueue.NotifyReady(ctx, j.ID)
		}
		return true, nil
	}
	if action == "cancelled" {
		return cancelExpiredApprovalWait(ctx, metaStore, eventStore, now, j, wait, expiresAt, ver, events)
	}
	decision := "expired"
	reason := "approval_expired"
	if action == "rejected" {
		decision = "rejected"
		reason = "approval_expired_rejected"
	}
	payloadBytes, err := json.Marshal(map[string]any{
		"approved":      false,
		"decision":      decision,
		"reason":        reason,
		"expired":       true,
		"expiry_action": action,
		"expires_at":    expiresAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return false, err
	}
	completedPayload, err := json.Marshal(jobstore.WaitCompletedPayload{
		NodeID:         wait.NodeID,
		Payload:        payloadBytes,
		CorrelationKey: wait.CorrelationKey,
		Approval: jobstore.ApprovalMetadata{
			Decision:          decision,
			Reason:            reason,
			ApprovedAtRFC3339: now.Format(time.RFC3339),
		},
	})
	if err != nil {
		return false, err
	}
	_, err = eventStore.Append(ctx, j.ID, ver, jobstore.JobEvent{JobID: j.ID, Type: jobstore.WaitCompleted, Payload: completedPayload})
	if err != nil {
		if errors.Is(err, jobstore.ErrVersionMismatch) {
			latestEvents, _, listErr := eventStore.ListEvents(ctx, j.ID)
			if listErr == nil && lastEventIsWaitCompletedWithCorrelationKey(latestEvents, wait.CorrelationKey) {
				_ = metaStore.UpdateStatus(ctx, j.ID, job.StatusPending)
				if wakeupQueue != nil {
					_ = wakeupQueue.NotifyReady(ctx, j.ID)
				}
				return true, nil
			}
		}
		return false, err
	}
	if err := metaStore.UpdateStatus(ctx, j.ID, job.StatusPending); err != nil {
		return false, err
	}
	if wakeupQueue != nil {
		_ = wakeupQueue.NotifyReady(ctx, j.ID)
	}
	return true, nil
}

func cancelExpiredApprovalWait(ctx context.Context, metaStore job.JobStore, eventStore jobstore.JobStore, now time.Time, j *job.Job, wait jobstore.JobWaitingPayload, expiresAt time.Time, ver int, events []jobstore.JobEvent) (bool, error) {
	payload, err := json.Marshal(map[string]any{
		"node_id":         wait.NodeID,
		"correlation_key": wait.CorrelationKey,
		"decision":        "cancelled",
		"reason":          "approval_expired_cancelled",
		"expired":         true,
		"expiry_action":   "cancelled",
		"expires_at":      expiresAt.UTC().Format(time.RFC3339),
		"cancelled_at":    now.Format(time.RFC3339),
	})
	if err != nil {
		return false, err
	}
	_, err = eventStore.Append(ctx, j.ID, ver, jobstore.JobEvent{JobID: j.ID, Type: jobstore.JobCancelled, Payload: payload})
	if err != nil {
		if errors.Is(err, jobstore.ErrVersionMismatch) {
			latestEvents, _, listErr := eventStore.ListEvents(ctx, j.ID)
			if listErr == nil && lastEventIsCancelledForCorrelationKey(latestEvents, wait.CorrelationKey) {
				_ = metaStore.UpdateStatus(ctx, j.ID, job.StatusCancelled)
				return true, nil
			}
		}
		return false, err
	}
	if err := metaStore.UpdateStatus(ctx, j.ID, job.StatusCancelled); err != nil {
		return false, err
	}
	return true, nil
}

func latestPendingApprovalWait(events []jobstore.JobEvent) (jobstore.JobWaitingPayload, bool) {
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Type {
		case jobstore.WaitCompleted, jobstore.JobCompleted, jobstore.JobFailed, jobstore.JobCancelled:
			return jobstore.JobWaitingPayload{}, false
		case jobstore.JobWaiting:
			wait, err := jobstore.ParseJobWaitingPayload(e.Payload)
			if err != nil || wait.CorrelationKey == "" || !isApprovalWait(wait) {
				return jobstore.JobWaitingPayload{}, false
			}
			return wait, true
		}
	}
	return jobstore.JobWaitingPayload{}, false
}

func isApprovalWait(wait jobstore.JobWaitingPayload) bool {
	if strings.EqualFold(wait.WaitKind, "human") {
		return true
	}
	switch wait.Reason {
	case "approval_required", "capability_approval":
		return true
	}
	return strings.HasPrefix(wait.CorrelationKey, "cap-approval-")
}

func expiryAction(wait jobstore.JobWaitingPayload) string {
	if len(wait.ResumptionContext) == 0 {
		return "expired"
	}
	var ctxPayload map[string]any
	if err := json.Unmarshal(wait.ResumptionContext, &ctxPayload); err != nil {
		return "expired"
	}
	value, _ := ctxPayload["expiry_action"].(string)
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "rejected", "reject":
		return "rejected"
	case "cancelled", "cancel":
		return "cancelled"
	default:
		return "expired"
	}
}

func lastEventIsWaitCompletedWithCorrelationKey(events []jobstore.JobEvent, correlationKey string) bool {
	if len(events) == 0 {
		return false
	}
	last := events[len(events)-1]
	if last.Type != jobstore.WaitCompleted {
		return false
	}
	payload, err := jobstore.ParseWaitCompletedPayload(last.Payload)
	if err != nil {
		return false
	}
	return payload.CorrelationKey == correlationKey
}

func lastEventIsCancelledForCorrelationKey(events []jobstore.JobEvent, correlationKey string) bool {
	if len(events) == 0 {
		return false
	}
	last := events[len(events)-1]
	if last.Type != jobstore.JobCancelled {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(last.Payload, &payload); err != nil {
		return false
	}
	value, _ := payload["correlation_key"].(string)
	return value == correlationKey
}
