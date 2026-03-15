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
	"testing"

	"rag-platform/internal/runtime/jobstore"
)

type mockJobStore struct {
	events []jobstore.JobEvent
	err    error
}

func (m *mockJobStore) ListEvents(ctx context.Context, jobID string) ([]jobstore.JobEvent, int, error) {
	return m.events, len(m.events), m.err
}

func (m *mockJobStore) Append(ctx context.Context, jobID string, expectedVersion int, event jobstore.JobEvent) (int, error) {
	return 0, nil
}

func (m *mockJobStore) Claim(ctx context.Context, workerID string) (string, int, string, error) {
	return "", 0, "", nil
}

func (m *mockJobStore) ClaimJob(ctx context.Context, workerID string, jobID string) (int, string, error) {
	return 0, "", nil
}

func (m *mockJobStore) Heartbeat(ctx context.Context, workerID string, jobID string) error {
	return nil
}

func (m *mockJobStore) Watch(ctx context.Context, jobID string) (<-chan jobstore.JobEvent, error) {
	return nil, nil
}

func (m *mockJobStore) ListJobIDsWithExpiredClaim(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockJobStore) GetCurrentAttemptID(ctx context.Context, jobID string) (string, error) {
	return "", nil
}

func (m *mockJobStore) CreateSnapshot(ctx context.Context, jobID string, upToVersion int, snapshot []byte) error {
	return nil
}

func (m *mockJobStore) GetLatestSnapshot(ctx context.Context, jobID string) (*jobstore.JobSnapshot, error) {
	return nil, nil
}

func (m *mockJobStore) DeleteSnapshotsBefore(ctx context.Context, jobID string, beforeVersion int) error {
	return nil
}

// Ensure mockJobStore implements the interface
var _ jobstore.JobStore = (*mockJobStore)(nil)

func TestNewJobHistory_Empty(t *testing.T) {
	h, err := NewJobHistory(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Len() != 0 {
		t.Errorf("expected 0, got %d", h.Len())
	}
}

func TestNewJobHistory_WithEvents(t *testing.T) {
	store := &mockJobStore{
		events: []jobstore.JobEvent{
			{ID: "1", JobID: "job-1", Type: jobstore.JobCreated},
			{ID: "2", JobID: "job-1", Type: jobstore.PlanGenerated},
		},
	}
	h, err := NewJobHistory(context.Background(), store, "job-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Len() != 2 {
		t.Errorf("expected 2, got %d", h.Len())
	}
}

func TestJobHistory_Next(t *testing.T) {
	store := &mockJobStore{
		events: []jobstore.JobEvent{
			{ID: "1", JobID: "job-1", Type: jobstore.JobCreated},
			{ID: "2", JobID: "job-1", Type: jobstore.PlanGenerated},
		},
	}
	h, _ := NewJobHistory(context.Background(), store, "job-1")

	event, ok := h.Next()
	if !ok {
		t.Fatal("expected to get first event")
	}
	if event.ID != "1" {
		t.Errorf("expected 1, got %s", event.ID)
	}
	if h.Index() != 1 {
		t.Errorf("expected index 1, got %d", h.Index())
	}
}

func TestJobHistory_Next_Exhausted(t *testing.T) {
	store := &mockJobStore{
		events: []jobstore.JobEvent{
			{ID: "1", JobID: "job-1", Type: jobstore.JobCreated},
		},
	}
	h, _ := NewJobHistory(context.Background(), store, "job-1")

	h.Next()              // consume first
	event, ok := h.Next() // should be exhausted
	if ok {
		t.Error("expected no more events")
	}
	if event != nil {
		t.Error("expected nil event")
	}
}

func TestJobHistory_Peek(t *testing.T) {
	store := &mockJobStore{
		events: []jobstore.JobEvent{
			{ID: "1", JobID: "job-1", Type: jobstore.JobCreated},
			{ID: "2", JobID: "job-1", Type: jobstore.PlanGenerated},
		},
	}
	h, _ := NewJobHistory(context.Background(), store, "job-1")

	event, ok := h.Peek()
	if !ok {
		t.Fatal("expected to peek first event")
	}
	if event.ID != "1" {
		t.Errorf("expected 1, got %s", event.ID)
	}
	// Peek should not advance
	if h.Index() != 0 {
		t.Errorf("expected index 0, got %d", h.Index())
	}
}

func TestJobHistory_Peek_Exhausted(t *testing.T) {
	store := &mockJobStore{
		events: []jobstore.JobEvent{
			{ID: "1", JobID: "job-1", Type: jobstore.JobCreated},
		},
	}
	h, _ := NewJobHistory(context.Background(), store, "job-1")

	h.Next()              // consume first
	event, ok := h.Peek() // should be exhausted
	if ok {
		t.Error("expected no more events")
	}
	if event != nil {
		t.Error("expected nil event")
	}
}

func TestJobHistory_Index(t *testing.T) {
	store := &mockJobStore{
		events: []jobstore.JobEvent{
			{ID: "1", JobID: "job-1", Type: jobstore.JobCreated},
			{ID: "2", JobID: "job-1", Type: jobstore.PlanGenerated},
		},
	}
	h, _ := NewJobHistory(context.Background(), store, "job-1")

	if h.Index() != 0 {
		t.Errorf("expected 0, got %d", h.Index())
	}

	h.Next()
	if h.Index() != 1 {
		t.Errorf("expected 1, got %d", h.Index())
	}

	h.Next()
	if h.Index() != 2 {
		t.Errorf("expected 2, got %d", h.Index())
	}
}

func TestJobHistory_Len(t *testing.T) {
	store := &mockJobStore{
		events: []jobstore.JobEvent{
			{ID: "1", JobID: "job-1", Type: jobstore.JobCreated},
			{ID: "2", JobID: "job-1", Type: jobstore.PlanGenerated},
			{ID: "3", JobID: "job-1", Type: jobstore.JobCompleted},
		},
	}
	h, _ := NewJobHistory(context.Background(), store, "job-1")

	if h.Len() != 3 {
		t.Errorf("expected 3, got %d", h.Len())
	}
}
