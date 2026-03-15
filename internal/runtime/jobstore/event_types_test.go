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
	"encoding/json"
	"testing"
	"time"
)

func TestJobEventTypes(t *testing.T) {
	events := []EventType{
		JobCreated,
		PlanGenerated,
		NodeStarted,
		NodeFinished,
		CommandEmitted,
		CommandCommitted,
		ToolCalled,
		ToolReturned,
		ToolInvocationStarted,
		ToolInvocationFinished,
		StepCommitted,
		JobCompleted,
		JobFailed,
		JobCancelled,
		JobQueued,
		JobLeased,
		JobRunning,
		JobWaiting,
		JobRequeued,
		JobRetrying,
		WaitCompleted,
		JobParked,
		JobResumed,
		StepStarted,
		StepFinished,
		StepFailed,
		StepRetried,
		CheckpointSaved,
		TimerFired,
		RandomRecorded,
		UUIDRecorded,
		HTTPRecorded,
		AgentMessage,
		StateCheckpointed,
		AgentThoughtRecorded,
		DecisionMade,
		ToolSelected,
		ToolResultSummarized,
		RecoveryStarted,
		RecoveryCompleted,
	}
	for _, e := range events {
		if e == "" {
			t.Error("EventType should not be empty")
		}
	}
}

func TestJobEvent_MarshalJSON(t *testing.T) {
	event := JobEvent{
		ID:        "event-1",
		JobID:     "job-1",
		Type:      JobCreated,
		Payload:   []byte(`{"key":"value"}`),
		CreatedAt: time.Now(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestJobEvent_UnmarshalJSON(t *testing.T) {
	// Payload is []byte which gets base64 encoded in JSON
	jsonData := `{"id":"event-1","job_id":"job-1","type":"job_created","payload":"eyJrZXkiOiJ2YWx1ZSJ9"}` // base64 of {"key":"value"}
	var event JobEvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "event-1" {
		t.Errorf("expected event-1, got %s", event.ID)
	}
	if event.Type != JobCreated {
		t.Errorf("expected job_created, got %s", event.Type)
	}
	if string(event.Payload) != `{"key":"value"}` {
		t.Errorf("expected payload {\"key\":\"value\"}, got %s", string(event.Payload))
	}
}

func TestDecisionSnapshot(t *testing.T) {
	event := JobEvent{
		ID:        "1",
		JobID:     "job-1",
		Type:      DecisionSnapshot,
		Payload:   json.RawMessage(`{"goal":"test goal","task_graph_summary":"summary"}`),
		CreatedAt: time.Now(),
	}
	if event.Type != DecisionSnapshot {
		t.Errorf("expected DecisionSnapshot, got %s", event.Type)
	}
}
