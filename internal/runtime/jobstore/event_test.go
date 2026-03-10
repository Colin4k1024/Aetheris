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

package jobstore

import (
	"testing"
)

func TestNewJobParkedEvent(t *testing.T) {
	event, err := NewJobParkedEvent("job-1", "waiting for approval", "corr-key-1", "human")
	if err != nil {
		t.Fatalf("NewJobParkedEvent failed: %v", err)
	}
	if event.Type != JobParked {
		t.Errorf("expected type JobParked, got %v", event.Type)
	}
	if event.JobID != "job-1" {
		t.Errorf("expected JobID job-1, got %s", event.JobID)
	}

	var payload JobParkedPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("UnmarshalPayload failed: %v", err)
	}
	if payload.Reason != "waiting for approval" {
		t.Errorf("expected reason 'waiting for approval', got %s", payload.Reason)
	}
	if payload.CorrelationKey != "corr-key-1" {
		t.Errorf("expected correlation key 'corr-key-1', got %s", payload.CorrelationKey)
	}
	if payload.WaitType != "human" {
		t.Errorf("expected wait type 'human', got %s", payload.WaitType)
	}
}

func TestNewJobResumedEvent(t *testing.T) {
	event, err := NewJobResumedEvent("job-1", "corr-key-1", "signal")
	if err != nil {
		t.Fatalf("NewJobResumedEvent failed: %v", err)
	}
	if event.Type != JobResumed {
		t.Errorf("expected type JobResumed, got %v", event.Type)
	}

	var payload JobResumedPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("UnmarshalPayload failed: %v", err)
	}
	if payload.CorrelationKey != "corr-key-1" {
		t.Errorf("expected correlation key 'corr-key-1', got %s", payload.CorrelationKey)
	}
	if payload.ResumedBy != "signal" {
		t.Errorf("expected resumed by 'signal', got %s", payload.ResumedBy)
	}
}

func TestNewStepStartedEvent(t *testing.T) {
	event, err := NewStepStartedEvent("job-1", "step-1", "node-1", 0, `{"input":"test"}`)
	if err != nil {
		t.Fatalf("NewStepStartedEvent failed: %v", err)
	}
	if event.Type != StepStarted {
		t.Errorf("expected type StepStarted, got %v", event.Type)
	}

	var payload StepStartedPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("UnmarshalPayload failed: %v", err)
	}
	if payload.StepID != "step-1" {
		t.Errorf("expected step ID 'step-1', got %s", payload.StepID)
	}
	if payload.NodeID != "node-1" {
		t.Errorf("expected node ID 'node-1', got %s", payload.NodeID)
	}
	if payload.StepIndex != 0 {
		t.Errorf("expected step index 0, got %d", payload.StepIndex)
	}
}

func TestNewStepFinishedEvent(t *testing.T) {
	event, err := NewStepFinishedEvent("job-1", "step-1", "node-1", 0, `{"result":"ok"}`, 1500)
	if err != nil {
		t.Fatalf("NewStepFinishedEvent failed: %v", err)
	}
	if event.Type != StepFinished {
		t.Errorf("expected type StepFinished, got %v", event.Type)
	}

	var payload StepFinishedPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("UnmarshalPayload failed: %v", err)
	}
	if payload.DurationMs != 1500 {
		t.Errorf("expected duration 1500, got %d", payload.DurationMs)
	}
}

func TestNewStepFailedEvent(t *testing.T) {
	event, err := NewStepFailedEvent("job-1", "step-1", "node-1", 0, "timeout error", true)
	if err != nil {
		t.Fatalf("NewStepFailedEvent failed: %v", err)
	}
	if event.Type != StepFailed {
		t.Errorf("expected type StepFailed, got %v", event.Type)
	}

	var payload StepFailedPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("UnmarshalPayload failed: %v", err)
	}
	if payload.Error != "timeout error" {
		t.Errorf("expected error 'timeout error', got %s", payload.Error)
	}
	if !payload.Retryable {
		t.Errorf("expected retryable=true")
	}
}

func TestNewStepRetriedEvent(t *testing.T) {
	event, err := NewStepRetriedEvent("job-1", "step-1", "node-1", 0, 1, 3, "timeout error")
	if err != nil {
		t.Fatalf("NewStepRetriedEvent failed: %v", err)
	}
	if event.Type != StepRetried {
		t.Errorf("expected type StepRetried, got %v", event.Type)
	}

	var payload StepRetriedPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("UnmarshalPayload failed: %v", err)
	}
	if payload.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", payload.RetryCount)
	}
	if payload.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", payload.MaxRetries)
	}
}

func TestNewCheckpointSavedEvent(t *testing.T) {
	event, err := NewCheckpointSavedEvent("job-1", "cp-123", "session-1", "cursor-abc", 5, 1024)
	if err != nil {
		t.Fatalf("NewCheckpointSavedEvent failed: %v", err)
	}
	if event.Type != CheckpointSaved {
		t.Errorf("expected type CheckpointSaved, got %v", event.Type)
	}

	var payload CheckpointSavedPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("UnmarshalPayload failed: %v", err)
	}
	if payload.CheckpointID != "cp-123" {
		t.Errorf("expected checkpoint ID 'cp-123', got %s", payload.CheckpointID)
	}
	if payload.Cursor != "cursor-abc" {
		t.Errorf("expected cursor 'cursor-abc', got %s", payload.Cursor)
	}
	if payload.StepIndex != 5 {
		t.Errorf("expected step index 5, got %d", payload.StepIndex)
	}
	if payload.SnapshotSize != 1024 {
		t.Errorf("expected snapshot size 1024, got %d", payload.SnapshotSize)
	}
}
