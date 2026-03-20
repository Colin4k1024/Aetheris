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

package job

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"rag-platform/internal/runtime/jobstore"
)

type fakeEventStore struct {
	expiredIDs []string
	events     map[string][]jobstore.JobEvent
}

func (f *fakeEventStore) ListJobIDsWithExpiredClaim(ctx context.Context) ([]string, error) {
	return f.expiredIDs, nil
}

func (f *fakeEventStore) ListEvents(ctx context.Context, jobID string) ([]jobstore.JobEvent, int, error) {
	events := f.events[jobID]
	return events, len(events), nil
}

func (f *fakeEventStore) Append(ctx context.Context, jobID string, expectedVersion int, event jobstore.JobEvent) (int, error) {
	panic("not used")
}
func (f *fakeEventStore) Claim(ctx context.Context, workerID string) (string, int, string, error) {
	panic("not used")
}
func (f *fakeEventStore) ClaimJob(ctx context.Context, workerID string, jobID string) (int, string, error) {
	panic("not used")
}
func (f *fakeEventStore) Heartbeat(ctx context.Context, workerID string, jobID string) error {
	panic("not used")
}
func (f *fakeEventStore) Watch(ctx context.Context, jobID string) (<-chan jobstore.JobEvent, error) {
	panic("not used")
}

// Snapshot methods (2.0-M1 additions)
func (f *fakeEventStore) CreateSnapshot(ctx context.Context, jobID string, upToVersion int, snapshot []byte) error {
	return nil
}

func (f *fakeEventStore) GetLatestSnapshot(ctx context.Context, jobID string) (*jobstore.JobSnapshot, error) {
	return nil, nil
}

func (f *fakeEventStore) DeleteSnapshotsBefore(ctx context.Context, jobID string, beforeVersion int) error {
	return nil
}

func (f *fakeEventStore) GetCurrentAttemptID(ctx context.Context, jobID string) (string, error) {
	return "", nil
}

func TestReclaimOrphanedFromEventStore(t *testing.T) {
	ctx := context.Background()
	metadata := NewJobStoreMem()

	// Job j1: Running in metadata, expired claim, not blocked -> should reclaim
	j1 := &Job{ID: "j1", AgentID: "a1", Goal: "g1", Status: StatusRunning}
	_, _ = metadata.Create(ctx, j1)
	_ = metadata.UpdateStatus(ctx, "j1", StatusRunning)

	now := time.Now()
	ev := func(typ jobstore.EventType) jobstore.JobEvent {
		return jobstore.JobEvent{JobID: "j1", Type: typ, CreatedAt: now}
	}

	eventStore := &fakeEventStore{
		expiredIDs: []string{"j1"},
		events: map[string][]jobstore.JobEvent{
			"j1": {ev(jobstore.JobCreated), ev(jobstore.JobRunning)},
		},
	}
	n, err := ReclaimOrphanedFromEventStore(ctx, metadata, eventStore)
	if err != nil {
		t.Fatalf("ReclaimOrphanedFromEventStore: %v", err)
	}
	if n != 1 {
		t.Errorf("reclaimed = %d, want 1", n)
	}
	j, _ := metadata.Get(ctx, "j1")
	if j != nil && j.Status != StatusPending {
		t.Errorf("j1 status = %v, want Pending", j.Status)
	}
}

func TestReclaimOrphanedFromEventStore_SkipsBlocked(t *testing.T) {
	ctx := context.Background()
	metadata := NewJobStoreMem()
	j1 := &Job{ID: "j1", AgentID: "a1", Goal: "g1", Status: StatusRunning}
	_, _ = metadata.Create(ctx, j1)
	_ = metadata.UpdateStatus(ctx, "j1", StatusRunning)

	now := time.Now()
	ev := func(typ jobstore.EventType) jobstore.JobEvent {
		return jobstore.JobEvent{JobID: "j1", Type: typ, CreatedAt: now}
	}
	eventStore := &fakeEventStore{
		expiredIDs: []string{"j1"},
		events: map[string][]jobstore.JobEvent{
			"j1": {ev(jobstore.JobCreated), ev(jobstore.JobRunning), ev(jobstore.JobWaiting)},
		},
	}
	n, err := ReclaimOrphanedFromEventStore(ctx, metadata, eventStore)
	if err != nil {
		t.Fatalf("ReclaimOrphanedFromEventStore: %v", err)
	}
	if n != 0 {
		t.Errorf("reclaimed = %d, want 0 (blocked)", n)
	}
	j, _ := metadata.Get(ctx, "j1")
	if j != nil && j.Status != StatusRunning {
		t.Errorf("j1 status = %v, want still Running", j.Status)
	}
}

func TestReclaimOrphanedFromEventStore_ReconcilesTerminalStatusFromEvents(t *testing.T) {
	ctx := context.Background()
	metadata := NewJobStoreMem()
	j1 := &Job{ID: "j1", AgentID: "a1", Goal: "g1", Status: StatusRunning}
	_, _ = metadata.Create(ctx, j1)
	_ = metadata.UpdateStatus(ctx, "j1", StatusRunning)

	now := time.Now()
	ev := func(typ jobstore.EventType) jobstore.JobEvent {
		return jobstore.JobEvent{JobID: "j1", Type: typ, CreatedAt: now}
	}
	eventStore := &fakeEventStore{
		expiredIDs: []string{"j1"},
		events: map[string][]jobstore.JobEvent{
			"j1": {ev(jobstore.JobCreated), ev(jobstore.JobRunning), ev(jobstore.JobCompleted)},
		},
	}
	n, err := ReclaimOrphanedFromEventStore(ctx, metadata, eventStore)
	if err != nil {
		t.Fatalf("ReclaimOrphanedFromEventStore: %v", err)
	}
	if n != 1 {
		t.Errorf("reclaimed = %d, want 1", n)
	}
	j, _ := metadata.Get(ctx, "j1")
	if j == nil || j.Status != StatusCompleted {
		t.Fatalf("j1 status = %v, want Completed", j.Status)
	}
}

func TestReclaimOrphanedFromEventStore_TerminalEventPreventsRerunAfterRestart(t *testing.T) {
	tests := []struct {
		name      string
		eventType jobstore.EventType
		want      JobStatus
	}{
		{name: "completed", eventType: jobstore.JobCompleted, want: StatusCompleted},
		{name: "failed", eventType: jobstore.JobFailed, want: StatusFailed},
		{name: "cancelled", eventType: jobstore.JobCancelled, want: StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			metadata := NewJobStoreMem()
			j1 := &Job{ID: "j1", AgentID: "a1", Goal: "g1", Status: StatusRunning}
			_, _ = metadata.Create(ctx, j1)
			_ = metadata.UpdateStatus(ctx, "j1", StatusRunning)

			now := time.Now()
			ev := func(typ jobstore.EventType) jobstore.JobEvent {
				return jobstore.JobEvent{JobID: "j1", Type: typ, CreatedAt: now}
			}
			eventStore := &fakeEventStore{
				expiredIDs: []string{"j1"},
				events: map[string][]jobstore.JobEvent{
					"j1": {ev(jobstore.JobCreated), ev(jobstore.JobRunning), ev(tt.eventType)},
				},
			}

			n, err := ReclaimOrphanedFromEventStore(ctx, metadata, eventStore)
			if err != nil {
				t.Fatalf("ReclaimOrphanedFromEventStore: %v", err)
			}
			if n != 1 {
				t.Fatalf("reclaimed = %d, want 1", n)
			}

			var runCount int32
			sched := NewScheduler(metadata, func(_ context.Context, j *Job) error {
				atomic.AddInt32(&runCount, 1)
				return nil
			}, SchedulerConfig{MaxConcurrency: 1, RetryMax: 0, Backoff: 10 * time.Millisecond})
			sched.Start(ctx)
			defer sched.Stop()

			time.Sleep(150 * time.Millisecond)
			if atomic.LoadInt32(&runCount) != 0 {
				t.Fatalf("%s job should not rerun after reclaim, runCount=%d", tt.name, runCount)
			}
			j, _ := metadata.Get(ctx, "j1")
			if j == nil || j.Status != tt.want {
				t.Fatalf("j1 status = %v, want %v", j.Status, tt.want)
			}
			if pending, err := metadata.ClaimNextPending(ctx); err != nil || pending != nil {
				t.Fatalf("%s job should not be claimable after reclaim, pending=%+v err=%v", tt.name, pending, err)
			}
		})
	}
}
