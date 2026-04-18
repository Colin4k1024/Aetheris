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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

// ACPEventRequest represents an incoming ACP event from Hermes.
type ACPEventRequest struct {
	Type      string                 `json:"type"`
	JobID     string                 `json:"job_id"`
	SessionID string                 `json:"session_id,omitempty"`
	CallID    string                 `json:"call_id,omitempty"`
	ToolName  string                 `json:"tool_name,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Result    any                    `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
}

// ACPCheckpointRequest represents an incoming checkpoint from Hermes.
type ACPCheckpointRequest struct {
	JobID      string          `json:"job_id"`
	SessionID  string          `json:"session_id"`
	Checkpoint json.RawMessage `json:"checkpoint"`
	Timestamp  string          `json:"timestamp,omitempty"`
}

// ACPStatusResponse represents the response for job status queries.
type ACPStatusResponse struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	IsCanceled bool   `json:"is_canceled"`
}

// HermesACPToolCall represents a tool call event for the audit trail.
// This is used by HermesRunner to store tool call events in the Event Store.
type HermesACPToolCall struct {
	RunID           string                 `json:"run_id"`
	StepID          string                 `json:"step_id"`
	ToolName        string                 `json:"tool_name"`
	RequestPayload  map[string]interface{} `json:"request_payload,omitempty"`
	ResponsePayload map[string]interface{} `json:"response_payload,omitempty"`
	IdempotencyKey  string                 `json:"idempotency_key,omitempty"`
	StartedAt       int64                  `json:"started_at,omitempty"`
	EndedAt         int64                  `json:"ended_at,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// ACPSEventStore is an interface for storing ACP events.
// This allows the ACP handlers to work with any event store implementation.
type ACPSEventStore interface {
	Append(ctx context.Context, jobID string, prevSeq int64, event jobstore.JobEvent) (int64, error)
	StoreACPToolCall(ctx context.Context, call *HermesACPToolCall) error
}

// acpEventStore is the global event store for ACP events.
var acpEventStore ACPSEventStore

// acpJobEventStore is the global job event store for job status lookups.
var acpJobEventStore jobstore.JobStore

// SetACPSEventStore sets the event store for ACP handlers.
func SetACPSEventStore(store ACPSEventStore) {
	acpEventStore = store
}

// SetACPJobEventStore sets the job event store for job status lookups.
func SetACPJobEventStore(store jobstore.JobStore) {
	acpJobEventStore = store
}

// HandleACPEvents handles POST /api/acp/events
// Receives tool.call, tool.result, session.start, session.end events from Hermes.
func HandleACPEvents(ctx context.Context, c *app.RequestContext) {
	var req ACPEventRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": fmt.Sprintf("Failed to parse request body: %v", err),
		})
		return
	}

	if req.JobID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing required field: job_id",
		})
		return
	}

	if req.Type == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing required field: type",
		})
		return
	}

	// Parse timestamp or use current time
	timestamp := time.Now()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			timestamp = t
		}
	}

	// Convert event to job event based on type
	var eventType jobstore.EventType
	var payload any

	switch req.Type {
	case "session.start":
		eventType = jobstore.JobRunning
		payload = map[string]any{
			"job_id":     req.JobID,
			"session_id": req.SessionID,
			"source":     "hermes",
		}

	case "session.end":
		switch req.Status {
		case "completed":
			eventType = jobstore.JobCompleted
		case "failed":
			eventType = jobstore.JobFailed
		case "canceled":
			eventType = jobstore.JobCancelled
		default:
			eventType = jobstore.JobCompleted
		}
		payload = map[string]any{
			"job_id":         req.JobID,
			"session_id":     req.SessionID,
			"final_response": req.Result,
			"status":         req.Status,
		}

	case "tool.call":
		eventType = jobstore.ToolCalled
		payload = map[string]any{
			"job_id":     req.JobID,
			"session_id": req.SessionID,
			"call_id":    req.CallID,
			"tool_name":  req.ToolName,
			"arguments":  req.Arguments,
		}

	case "tool.result":
		eventType = jobstore.ToolReturned
		payload = map[string]any{
			"job_id":     req.JobID,
			"session_id": req.SessionID,
			"call_id":    req.CallID,
			"tool_name":  req.ToolName,
			"result":     req.Result,
			"error":      req.Error,
		}

	default:
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "unknown_event_type",
			"message": fmt.Sprintf("Unknown event type: %s", req.Type),
		})
		return
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to marshal ACP event payload: %v", err)
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to process event",
		})
		return
	}

	// Create job event
	event := &jobstore.JobEvent{
		JobID:     req.JobID,
		Type:      eventType,
		Payload:   payloadBytes,
		CreatedAt: timestamp,
	}

	// Write to event store if available
	if acpEventStore != nil {
		if _, err := acpEventStore.Append(ctx, req.JobID, 0, *event); err != nil {
			hlog.CtxWarnf(ctx, "Failed to write ACP event to store: %v", err)
			// Don't fail the request - Hermes is fire-and-forget with these
		}

		// Also store tool call events in the dedicated table for audit
		if (req.Type == "tool.call" || req.Type == "tool.result") && req.CallID != "" {
			toolCall := &HermesACPToolCall{
				RunID:  req.JobID,
				StepID: req.SessionID, // Use session_id as step_id in this context
			}
			if req.Type == "tool.call" {
				toolCall.ToolName = req.ToolName
				toolCall.RequestPayload = req.Arguments
				// Generate idempotency key for tool.call
				toolCall.IdempotencyKey = fmt.Sprintf("%s:%s", req.CallID, req.JobID)
			} else {
				toolCall.ToolName = req.ToolName
				toolCall.ResponsePayload = map[string]any{"result": req.Result, "error": req.Error}
			}
			if err := acpEventStore.StoreACPToolCall(ctx, toolCall); err != nil {
				hlog.CtxWarnf(ctx, "Failed to store ACP tool call: %v", err)
			}
		}
	}

	c.JSON(consts.StatusOK, map[string]string{
		"status": "ok",
	})
}

// HandleACPCheckpoints handles POST /api/acp/checkpoints
// Receives checkpoint data from Hermes for durable storage.
func HandleACPCheckpoints(ctx context.Context, c *app.RequestContext) {
	var req ACPCheckpointRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": fmt.Sprintf("Failed to parse request body: %v", err),
		})
		return
	}

	if req.JobID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing required field: job_id",
		})
		return
	}

	if req.SessionID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing required field: session_id",
		})
		return
	}

	// Parse timestamp or use current time
	timestamp := time.Now()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			timestamp = t
		}
	}

	// Parse checkpoint data
	var checkpointData map[string]any
	if req.Checkpoint != nil {
		if err := json.Unmarshal(req.Checkpoint, &checkpointData); err != nil {
			hlog.CtxWarnf(ctx, "Failed to parse checkpoint data: %v", err)
		}
	}

	// Create checkpoint saved event
	payload := map[string]any{
		"job_id":     req.JobID,
		"session_id": req.SessionID,
		"checkpoint": checkpointData,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to marshal checkpoint payload: %v", err)
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to process checkpoint",
		})
		return
	}

	event := &jobstore.JobEvent{
		JobID:     req.JobID,
		Type:      jobstore.CheckpointSaved,
		Payload:   payloadBytes,
		CreatedAt: timestamp,
	}

	if acpEventStore != nil {
		if _, err := acpEventStore.Append(ctx, req.JobID, 0, *event); err != nil {
			hlog.CtxWarnf(ctx, "Failed to write checkpoint to store: %v", err)
		}
	}

	// Extract checkpoint_id if available
	checkpointID := ""
	if checkpointData != nil {
		if id, ok := checkpointData["id"].(string); ok {
			checkpointID = id
		}
	}

	c.JSON(consts.StatusOK, map[string]string{
		"status":        "ok",
		"checkpoint_id": checkpointID,
	})
}

// HandleACPJobStatus handles GET /api/acp/jobs/:job_id/status
// Returns the current status of a job (running, canceled, etc.).
func HandleACPJobStatus(ctx context.Context, c *app.RequestContext) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing job_id parameter",
		})
		return
	}

	// Query jobstore for actual job status
	status := "unknown"
	isCanceled := false

	if acpJobEventStore != nil {
		events, _, err := acpJobEventStore.ListEvents(ctx, jobID)
		if err != nil {
			hlog.CtxWarnf(ctx, "Failed to list events for job %s: %v", jobID, err)
			// Return "unknown" on error - Hermes will keep polling
		} else if len(events) > 0 {
			// Determine status from the latest events
			status = determineJobStatus(events)
			isCanceled = status == "canceled"
		}
		// If no events found, status remains "unknown"
	}

	c.JSON(consts.StatusOK, ACPStatusResponse{
		JobID:      jobID,
		Status:     status,
		IsCanceled: isCanceled,
	})
}

// determineJobStatus derives the job status from its event stream.
// It looks at the most recent terminal event to determine the current state.
func determineJobStatus(events []jobstore.JobEvent) string {
	// Look for terminal events first (job_complete, job_failed, job_cancelled)
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Type {
		case jobstore.JobCompleted:
			return "completed"
		case jobstore.JobFailed:
			return "failed"
		case jobstore.JobCancelled:
			return "canceled"
		}
	}

	// Check for parked/waiting states
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Type {
		case jobstore.JobParked:
			return "parked"
		case jobstore.JobWaiting:
			return "waiting"
		}
	}

	// Check if job is currently running
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.Type == jobstore.JobRunning {
			return "running"
		}
	}

	// Check for queued/leased states
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Type {
		case jobstore.JobQueued:
			return "queued"
		case jobstore.JobLeased:
			return "leased"
		case jobstore.JobRetrying:
			return "retrying"
		}
	}

	// Default to running if we have events but no clear terminal state
	if len(events) > 0 {
		return "running"
	}

	return "unknown"
}

// redacredactCredentials redacts sensitive values from context for logging.
// It checks if key contains "token" or "secret" (case-insensitive partial match).
func redactCredentials(context map[string]any) map[string]any {
	if context == nil {
		return nil
	}
	result := make(map[string]any)
	for k, v := range context {
		if strings.Contains(strings.ToLower(k), "token") || strings.Contains(strings.ToLower(k), "secret") {
			result[k] = "[REDACTED]"
		} else if m, ok := v.(map[string]any); ok {
			result[k] = redactCredentials(m)
		} else {
			result[k] = v
		}
	}
	return result
}
