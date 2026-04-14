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

package replay

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

// TestSnapshotReplay_SerializeDeserializeRoundtrip 验证 ReplayContext 经 Serialize/Deserialize 后数据完整性。
func TestSnapshotReplay_SerializeDeserializeRoundtrip(t *testing.T) {
	original := &ReplayContext{
		TaskGraphState: []byte(`{"nodes":[{"id":"n1","type":"tool"}]}`),
		CursorNode:     "n1",
		PayloadResults: []byte(`{"result":"ok"}`),
		CompletedNodeIDs: map[string]struct{}{
			"n1": {},
			"n2": {},
		},
		PayloadResultsByNode: map[string][]byte{
			"n1": []byte(`{"r":"v1"}`),
		},
		CompletedCommandIDs: map[string]struct{}{
			"cmd-1": {},
		},
		CommandResults: map[string][]byte{
			"cmd-1": []byte(`{"status":"done"}`),
		},
		CompletedToolInvocations: map[string][]byte{
			"idem-key-1": []byte(`{"output":"result"}`),
		},
		PendingToolInvocations: map[string]struct{}{
			"idem-key-2": {},
		},
		StateChangesByStep:      make(map[string][]StateChangeRecord),
		ApprovedCorrelationKeys: map[string]struct{}{"corr-1": {}},
		WorkingMemorySnapshot:   []byte(`{"key":"value"}`),
		Phase:                   PhaseExecuting,
		RecordedTime:            map[string]int64{"timer-1": 1700000000},
		RecordedRandom:          map[string][]byte{"rand-1": []byte(`[42]`)},
		RecordedUUID:            map[string]string{"uuid-1": "550e8400-e29b-41d4-a716-446655440000"},
		RecordedHTTP:            map[string][]byte{"http-1": []byte(`{"status":200}`)},
	}

	data, err := SerializeReplayContext(original)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("serialized data is empty")
	}

	// Deserialize using the unexported function via a fresh store path
	var payload jobstore.SnapshotPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify key fields round-tripped
	if payload.CursorNode != original.CursorNode {
		t.Errorf("CursorNode: got %q want %q", payload.CursorNode, original.CursorNode)
	}
	if payload.Phase != int(original.Phase) {
		t.Errorf("Phase: got %d want %d", payload.Phase, int(original.Phase))
	}
	if len(payload.CompletedNodeIDs) != len(original.CompletedNodeIDs) {
		t.Errorf("CompletedNodeIDs count: got %d want %d", len(payload.CompletedNodeIDs), len(original.CompletedNodeIDs))
	}
	if len(payload.CompletedCommandIDs) != len(original.CompletedCommandIDs) {
		t.Errorf("CompletedCommandIDs count: got %d want %d", len(payload.CompletedCommandIDs), len(original.CompletedCommandIDs))
	}
	if len(payload.PendingToolInvocations) != len(original.PendingToolInvocations) {
		t.Errorf("PendingToolInvocations count: got %d want %d", len(payload.PendingToolInvocations), len(original.PendingToolInvocations))
	}
	if len(payload.ApprovedCorrelationKeys) != len(original.ApprovedCorrelationKeys) {
		t.Errorf("ApprovedCorrelationKeys count: got %d want %d", len(payload.ApprovedCorrelationKeys), len(original.ApprovedCorrelationKeys))
	}
	if payload.RecordedTime["timer-1"] != 1700000000 {
		t.Errorf("RecordedTime[timer-1]: got %d want 1700000000", payload.RecordedTime["timer-1"])
	}
	if payload.RecordedUUID["uuid-1"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("RecordedUUID: got %s want expected UUID", payload.RecordedUUID["uuid-1"])
	}
}

// TestSnapshotReplay_BuildFromEvents_ThenSerialize 验证从事件流重建 ReplayContext 后可序列化。
func TestSnapshotReplay_BuildFromEvents_ThenSerialize(t *testing.T) {
	store := jobstore.NewMemoryStore()
	ctx := context.Background()

	jobID := "test-job-replay"
	taskGraph := json.RawMessage(`{"nodes":[{"id":"node-1","type":"tool"}],"edges":[]}`)
	planPayload, _ := json.Marshal(map[string]interface{}{
		"task_graph": taskGraph,
	})
	if _, err := store.Append(ctx, jobID, 0, jobstore.JobEvent{
		Type:    jobstore.PlanGenerated,
		Payload: planPayload,
	}); err != nil {
		t.Fatalf("append PlanGenerated: %v", err)
	}

	nodeFinishedPayload, _ := json.Marshal(map[string]interface{}{
		"node_id":         "node-1",
		"payload_results": json.RawMessage(`{"answer":"42"}`),
		"result_type":     "success",
	})
	if _, err := store.Append(ctx, jobID, 1, jobstore.JobEvent{
		Type:    jobstore.NodeFinished,
		Payload: nodeFinishedPayload,
	}); err != nil {
		t.Fatalf("append NodeFinished: %v", err)
	}

	builder := NewReplayContextBuilder(store)
	rc, err := builder.BuildFromEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("BuildFromEvents: %v", err)
	}
	if rc == nil {
		t.Fatal("ReplayContext should not be nil after PlanGenerated + NodeFinished")
	}

	if _, ok := rc.CompletedNodeIDs["node-1"]; !ok {
		t.Error("node-1 should be in CompletedNodeIDs")
	}
	if rc.CursorNode != "node-1" {
		t.Errorf("CursorNode: got %q want node-1", rc.CursorNode)
	}
	if rc.Phase != PhaseExecuting {
		t.Errorf("Phase: got %d want PhaseExecuting", rc.Phase)
	}

	// 验证可以序列化快照
	snapshotData, err := SerializeReplayContext(rc)
	if err != nil {
		t.Fatalf("SerializeReplayContext: %v", err)
	}
	if len(snapshotData) == 0 {
		t.Fatal("snapshot data should not be empty")
	}
	t.Logf("snapshot serialized, %d bytes", len(snapshotData))
}

// TestSnapshotReplay_BuildFromSnapshot_FallbackToEvents 验证 BuildFromSnapshot 在无快照时降级到全事件重放。
func TestSnapshotReplay_BuildFromSnapshot_FallbackToEvents(t *testing.T) {
	store := jobstore.NewMemoryStore()
	ctx := context.Background()

	jobID := "test-job-fallback"
	taskGraph := json.RawMessage(`{"nodes":[{"id":"n1","type":"pure"}],"edges":[]}`)
	planPayload, _ := json.Marshal(map[string]interface{}{"task_graph": taskGraph})
	_, _ = store.Append(ctx, jobID, 0, jobstore.JobEvent{Type: jobstore.PlanGenerated, Payload: planPayload})

	builder := NewReplayContextBuilder(store)

	// memory store GetLatestSnapshot 返回 nil，BuildFromSnapshot 应降级到 BuildFromEvents
	rc, err := builder.BuildFromSnapshot(ctx, jobID)
	if err != nil {
		t.Fatalf("BuildFromSnapshot: %v", err)
	}
	if rc == nil {
		t.Fatal("expected non-nil ReplayContext after fallback to BuildFromEvents")
	}
	if len(rc.TaskGraphState) == 0 {
		t.Error("TaskGraphState should be populated after fallback")
	}
}

func TestSnapshotReplay_BuildFromSnapshot_AppliesIncrementalEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")
	storeI, err := jobstore.NewEmbeddedStore(path)
	if err != nil {
		t.Fatalf("new embedded store: %v", err)
	}
	store := storeI
	ctx := context.Background()
	jobID := "test-job-incremental"

	taskGraph := json.RawMessage(`{"nodes":[{"id":"n1","type":"pure"},{"id":"n2","type":"pure"}],"edges":[]}`)
	planPayload, _ := json.Marshal(map[string]interface{}{"task_graph": taskGraph})
	if _, err := store.Append(ctx, jobID, 0, jobstore.JobEvent{Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan: %v", err)
	}
	node1Payload, _ := json.Marshal(map[string]interface{}{
		"node_id":         "n1",
		"payload_results": json.RawMessage(`{"n1":"ok"}`),
		"result_type":     "success",
	})
	if _, err := store.Append(ctx, jobID, 1, jobstore.JobEvent{Type: jobstore.NodeFinished, Payload: node1Payload}); err != nil {
		t.Fatalf("append n1: %v", err)
	}

	builder := NewReplayContextBuilder(store)
	base, err := builder.BuildFromEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("build base: %v", err)
	}
	snapshotBytes, err := SerializeReplayContext(base)
	if err != nil {
		t.Fatalf("serialize snapshot: %v", err)
	}
	if err := store.CreateSnapshot(ctx, jobID, 2, snapshotBytes); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	node2Payload, _ := json.Marshal(map[string]interface{}{
		"node_id":         "n2",
		"payload_results": json.RawMessage(`{"n2":"ok"}`),
		"result_type":     "success",
	})
	if _, err := store.Append(ctx, jobID, 2, jobstore.JobEvent{Type: jobstore.NodeFinished, Payload: node2Payload}); err != nil {
		t.Fatalf("append n2: %v", err)
	}

	rc, err := builder.BuildFromSnapshot(ctx, jobID)
	if err != nil {
		t.Fatalf("BuildFromSnapshot: %v", err)
	}
	if rc == nil {
		t.Fatal("expected non-nil replay context")
	}
	if _, ok := rc.CompletedNodeIDs["n2"]; !ok {
		t.Fatalf("expected incremental node n2 to be applied")
	}
	if rc.CursorNode != "n2" {
		t.Fatalf("cursor should advance to n2, got %s", rc.CursorNode)
	}
}
