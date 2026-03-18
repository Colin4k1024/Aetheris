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

func TestJobSnapshot(t *testing.T) {
	now := time.Now()
	snapshot := JobSnapshot{
		JobID:     "job-1",
		Version:   10,
		Snapshot:  []byte(`{"key":"value"}`),
		CreatedAt: now,
	}

	if snapshot.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", snapshot.JobID)
	}
	if snapshot.Version != 10 {
		t.Errorf("expected 10, got %d", snapshot.Version)
	}
	if string(snapshot.Snapshot) != `{"key":"value"}` {
		t.Errorf("expected snapshot, got %s", snapshot.Snapshot)
	}
	if snapshot.CreatedAt != now {
		t.Errorf("expected %v, got %v", now, snapshot.CreatedAt)
	}
}

func TestSnapshotPayload(t *testing.T) {
	payload := SnapshotPayload{
		TaskGraphState:         json.RawMessage(`{"nodes":[]}`),
		CursorNode:             "node-1",
		CompletedNodeIDs:       []string{"n1", "n2"},
		CompletedCommandIDs:    []string{"c1"},
		PendingToolInvocations: []string{"t1"},
		Phase:                  2,
	}

	if payload.CursorNode != "node-1" {
		t.Errorf("expected node-1, got %s", payload.CursorNode)
	}
	if len(payload.CompletedNodeIDs) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(payload.CompletedNodeIDs))
	}
	if payload.Phase != 2 {
		t.Errorf("expected phase 2, got %d", payload.Phase)
	}
}

func TestSnapshotPayload_JSONMarshal(t *testing.T) {
	payload := SnapshotPayload{
		TaskGraphState:   json.RawMessage(`{"nodes":[]}`),
		CursorNode:       "node-1",
		CompletedNodeIDs: []string{"n1"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed SnapshotPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.CursorNode != "node-1" {
		t.Errorf("expected node-1, got %s", parsed.CursorNode)
	}
}

func TestCompactionConfig(t *testing.T) {
	config := CompactionConfig{
		EnableAutoCompaction: true,
		EventCountThreshold:  500,
		TimeIntervalHours:    12,
		KeepSnapshotCount:    5,
	}

	if !config.EnableAutoCompaction {
		t.Error("expected EnableAutoCompaction to be true")
	}
	if config.EventCountThreshold != 500 {
		t.Errorf("expected 500, got %d", config.EventCountThreshold)
	}
	if config.TimeIntervalHours != 12 {
		t.Errorf("expected 12, got %d", config.TimeIntervalHours)
	}
	if config.KeepSnapshotCount != 5 {
		t.Errorf("expected 5, got %d", config.KeepSnapshotCount)
	}
}

func TestDefaultCompactionConfig(t *testing.T) {
	config := DefaultCompactionConfig()

	if config.EnableAutoCompaction {
		t.Error("expected EnableAutoCompaction to be false by default")
	}
	if config.EventCountThreshold != 1000 {
		t.Errorf("expected 1000, got %d", config.EventCountThreshold)
	}
	if config.TimeIntervalHours != 24 {
		t.Errorf("expected 24, got %d", config.TimeIntervalHours)
	}
	if config.KeepSnapshotCount != 3 {
		t.Errorf("expected 3, got %d", config.KeepSnapshotCount)
	}
}
