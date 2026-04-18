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

// Package hermes provides the ACP client for communicating with Hermes Agent.
package hermes

import (
	"encoding/json"
	"time"
)

// JobDispatch represents a job dispatch message sent to Hermes ACP Server.
type JobDispatch struct {
	Type        string                 `json:"type"`
	JobID       string                 `json:"job_id"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	StepID      string                 `json:"step_id,omitempty"`
	Instruction string                 `json:"instruction"`
	Tools       []string               `json:"tools,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	CallbackURL string                 `json:"callback_url"`
}

// ToolCallEvent represents a tool call event from Hermes.
type ToolCallEvent struct {
	Type      string                 `json:"type"`
	JobID     string                 `json:"job_id"`
	SessionID string                 `json:"session_id"`
	CallID    string                 `json:"call_id"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToolResultEvent represents a tool result event from Hermes.
type ToolResultEvent struct {
	Type      string    `json:"type"`
	JobID     string    `json:"job_id"`
	SessionID string    `json:"session_id"`
	CallID    string    `json:"call_id"`
	ToolName  string    `json:"tool_name"`
	Result    any       `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionStartEvent represents a session start event from Hermes.
type SessionStartEvent struct {
	Type      string    `json:"type"`
	JobID     string    `json:"job_id"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionEndEvent represents a session end event from Hermes.
type SessionEndEvent struct {
	Type          string    `json:"type"`
	JobID         string    `json:"job_id"`
	SessionID     string    `json:"session_id"`
	Status        string    `json:"status"` // completed, failed, canceled
	FinalResponse string    `json:"final_response,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// CheckpointSaveEvent represents a checkpoint save event from Hermes.
type CheckpointSaveEvent struct {
	Type        string          `json:"type"`
	JobID       string          `json:"job_id"`
	SessionID   string          `json:"session_id"`
	Checkpoints *CheckpointData `json:"checkpoint,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// CheckpointData contains checkpoint information.
type CheckpointData struct {
	ID           string                   `json:"id"`
	Cursor       string                   `json:"cursor"`
	History      []map[string]interface{} `json:"history,omitempty"`
	SnapshotSize int                      `json:"snapshot_size"`
}

// ACPEvent represents a generic ACP event received from Hermes.
type ACPEvent struct {
	Type      string          `json:"type"`
	JobID     string          `json:"job_id"`
	SessionID string          `json:"session_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// EventType definitions
const (
	EventTypeJobDispatch    = "job.dispatch"
	EventTypeSessionStart   = "session.start"
	EventTypeSessionEnd     = "session.end"
	EventTypeToolCall       = "tool.call"
	EventTypeToolResult     = "tool.result"
	EventTypeCheckpointSave = "checkpoint.save"
	EventTypePing           = "ping"
	EventTypePong           = "pong"
)

// SessionStatus values
const (
	SessionStatusCompleted = "completed"
	SessionStatusFailed    = "failed"
	SessionStatusCanceled  = "canceled"
)
