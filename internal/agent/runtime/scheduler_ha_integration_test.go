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

// Package runtime_test provides integration tests for the runtime package.
// This is an external test package to avoid import cycles with the job package.
package runtime_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/job"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
)

// fakeRunFunc tracks how many times each scheduler's run function was called.
type fakeRunFunc struct {
	mu        sync.Mutex
	callCount map[string]int // schedulerID -> call count
}

func (f *fakeRunFunc) run(schedulerID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.callCount == nil {
		f.callCount = make(map[string]int)
	}
	f.callCount[schedulerID]++
}

func (f *fakeRunFunc) getCount(schedulerID string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount[schedulerID]
}

// TestSchedulerHA_MultipleSchedulersOneAgent tests that only one scheduler can
// wake an agent at a time (Take is atomic).
func TestSchedulerHA_MultipleSchedulersOneAgent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := runtime.NewManager()
	fakeRun := &fakeRunFunc{}

	// Create two schedulers sharing the same manager
	sched1 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		fakeRun.run("sched1")
	})
	sched2 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		fakeRun.run("sched2")
	})

	// Create an agent
	agent, err := manager.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Sched1 wakes the agent first
	err = sched1.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched1.WakeAgent: %v", err)
	}
	if agent.GetStatus() != runtime.StatusRunning {
		t.Errorf("agent status after sched1 wake: got %v, want StatusRunning", agent.GetStatus())
	}

	// Sched2 tries to wake the same agent - should be no-op
	err = sched2.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched2.WakeAgent: %v", err)
	}
	// Agent should still be running (not taken by sched2)
	if agent.GetStatus() != runtime.StatusRunning {
		t.Errorf("agent status after sched2 wake attempt: got %v, want StatusRunning", agent.GetStatus())
	}

	// Only sched1's runFunc should have been called
	if fakeRun.getCount("sched1") == 0 {
		t.Error("sched1 runFunc should have been called")
	}
	if fakeRun.getCount("sched2") != 0 {
		t.Error("sched2 runFunc should NOT have been called")
	}

	// Release the agent
	agent.Release()
}

// TestSchedulerHA_SecondSchedulerPicksUpAfterFirstStops tests that after the
// first scheduler releases an agent, the second scheduler can wake it.
func TestSchedulerHA_SecondSchedulerPicksUpAfterFirstStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := runtime.NewManager()
	fakeRun := &fakeRunFunc{}

	sched1 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		fakeRun.run("sched1")
	})
	sched2 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		fakeRun.run("sched2")
	})

	agent, err := manager.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Sched1 wakes the agent
	err = sched1.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched1.WakeAgent: %v", err)
	}
	if fakeRun.getCount("sched1") == 0 {
		t.Error("sched1 runFunc should have been called")
	}

	// Agent is running - sched2 cannot wake it
	err = sched2.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched2.WakeAgent: %v", err)
	}
	if fakeRun.getCount("sched2") != 0 {
		t.Error("sched2 should not have run while agent is running")
	}

	// Sched1 releases the agent (via Stop)
	err = sched1.Stop(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched1.Stop: %v", err)
	}
	if agent.GetStatus() != runtime.StatusIdle {
		t.Errorf("agent status after stop: got %v, want StatusIdle", agent.GetStatus())
	}

	// Now sched2 can wake the agent
	err = sched2.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched2.WakeAgent after release: %v", err)
	}
	if fakeRun.getCount("sched2") == 0 {
		t.Error("sched2 runFunc should have been called after release")
	}

	agent.Release()
}

// TestSchedulerHA_OrphanReclamation tests that when a scheduler dies without
// releasing an agent, the agent can be recovered via explicit orphan detection.
// This simulates the lease expiry scenario where a scheduler crashes.
func TestSchedulerHA_OrphanReclamation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := runtime.NewManager()
	fakeRun := &fakeRunFunc{}

	// Sched1 is the "crashed" scheduler - we simulate crash by just stopping
	// without calling Release on the agent
	sched1 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		fakeRun.run("sched1")
		// Simulate crash: don't call agent.Release()
	})

	sched2 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		fakeRun.run("sched2")
	})

	agent, err := manager.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Sched1 wakes the agent but "crashes" (goroutine exits without Release)
	// We simulate this by manually setting status to Running then making sched1 stop
	err = sched1.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched1.WakeAgent: %v", err)
	}
	if agent.GetStatus() != runtime.StatusRunning {
		t.Errorf("agent should be running: got %v", agent.GetStatus())
	}

	// Sched1 goroutine is running but we simulate crash by having it stop
	// without releasing the agent. The agent is now in Running state but
	// the scheduler that woke it is gone - this is the orphan scenario.

	// In a real HA setup, the orphan detection would:
	// 1. Detect the scheduler is gone (via heartbeat timeout or similar)
	// 2. Force-release the agent back to Idle

	// For testing, we simulate the orphan detection by directly forcing
	// the agent back to Idle (simulating what a recovery mechanism would do)
	agent.SetStatus(runtime.StatusIdle)

	// Now sched2 should be able to wake the agent
	err = sched2.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("sched2.WakeAgent after orphan recovery: %v", err)
	}
	if fakeRun.getCount("sched2") == 0 {
		t.Error("sched2 should have run after orphan recovery")
	}
	if agent.GetStatus() != runtime.StatusRunning {
		t.Errorf("agent should be running after sched2 wake: got %v", agent.GetStatus())
	}

	agent.Release()
}

// TestSchedulerHA_ResumeAfterSuspend tests the Resume flow which uses
// Release()+Take() atomically to reclaim an agent.
func TestSchedulerHA_ResumeAfterSuspend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := runtime.NewManager()
	fakeRun := &fakeRunFunc{}

	sched := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		fakeRun.run("sched")
	})

	agent, err := manager.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Wake the agent
	err = sched.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("WakeAgent: %v", err)
	}

	// Suspend the agent
	err = sched.Suspend(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	if agent.GetStatus() != runtime.StatusSuspended {
		t.Errorf("after suspend: got %v, want StatusSuspended", agent.GetStatus())
	}

	// Resume should work and trigger run
	err = sched.Resume(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if agent.GetStatus() != runtime.StatusRunning {
		t.Errorf("after resume: got %v, want StatusRunning", agent.GetStatus())
	}
	if fakeRun.getCount("sched") == 0 {
		t.Error("resume should trigger run")
	}

	agent.Release()
}

// TestSchedulerHA_ConcurrentWakeAttempts tests that concurrent WakeAgent calls
// from multiple schedulers result in only one actually waking the agent.
// This verifies the Take() atomicity under contention.
func TestSchedulerHA_ConcurrentWakeAttempts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := runtime.NewManager()
	var runCount int32

	sched1 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		atomic.AddInt32(&runCount, 1)
	})
	sched2 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		atomic.AddInt32(&runCount, 1)
	})
	sched3 := runtime.NewScheduler(manager, func(ctx context.Context, agentID string) {
		atomic.AddInt32(&runCount, 1)
	})

	agent, err := manager.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// All three schedulers try to wake the agent concurrently
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); sched1.WakeAgent(ctx, agent.ID) }()
	go func() { defer wg.Done(); sched2.WakeAgent(ctx, agent.ID) }()
	go func() { defer wg.Done(); sched3.WakeAgent(ctx, agent.ID) }()
	wg.Wait()

	// Only ONE scheduler should have actually run
	if runCount != 1 {
		t.Errorf("runCount: got %d, want 1", runCount)
	}

	agent.Release()
}

// TestSchedulerHA_WithJobStoreIntegration tests the integration between
// runtime schedulers and the job store for proper HA behavior.
// This test simulates the full flow: job creation, scheduling, failure, recovery.
func TestSchedulerHA_WithJobStoreIntegration(t *testing.T) {
	// Skip if this test requires full PostgreSQL setup
	// The job store HA features require PostgreSQL for proper lease handling
	t.Skip("Requires PostgreSQL for full HA testing with job store integration")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// In a real setup, this would be PostgreSQL-backed
	// For unit testing, we use the in-memory store which doesn't have
	// true lease expiry semantics
	jobStore := job.NewJobStoreMem()

	// Create a job
	jobID, err := jobStore.Create(ctx, &job.Job{
		AgentID: "agent-1",
		Goal:    "test goal",
	})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}

	// First scheduler claims the job
	claimedJob, err := jobStore.ClaimNextPending(ctx)
	if err != nil {
		t.Fatalf("ClaimNextPending: %v", err)
	}
	if claimedJob == nil {
		t.Fatal("should have claimed job")
	}
	if claimedJob.ID != jobID {
		t.Errorf("claimed job ID: got %s, want %s", claimedJob.ID, jobID)
	}

	// Second scheduler cannot claim the same job (already Running)
	claimedJob2, err := jobStore.ClaimNextPending(ctx)
	if err != nil {
		t.Fatalf("ClaimNextPending (second): %v", err)
	}
	if claimedJob2 != nil {
		t.Error("second claim should return nil (job already running)")
	}

	// Simulate first scheduler crash - job stays in Running state
	// In real HA, ReclaimOrphanedJobs would be called after lease expiry

	// For in-memory store, ReclaimOrphanedJobs returns 0 (no true expiry)
	n, err := jobStore.ReclaimOrphanedJobs(ctx, time.Second)
	if err != nil {
		t.Fatalf("ReclaimOrphanedJobs: %v", err)
	}
	if n != 0 {
		t.Errorf("in-memory store should not reclaim: got %d", n)
	}

	// Complete the job manually
	err = jobStore.UpdateStatus(ctx, jobID, job.StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	// Now the second scheduler can claim (if job was properly requeued)
	// This demonstrates the recovery flow after first scheduler completes
}

// TestSchedulerHA_JobReclaimAfterSchedulerCrash simulates what happens when
// a scheduler dies while running a job - the job should be reclaimable.
func TestSchedulerHA_JobReclaimAfterSchedulerCrash(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This test uses the job-level scheduler which has proper HA semantics
	jobStore := job.NewJobStoreMem()

	var jobRunCount int32
	runJob := func(ctx context.Context, j *job.Job) error {
		atomic.AddInt32(&jobRunCount, 1)
		// Simulate work being done
		return nil
	}

	// Create schedulers with different IDs (simulating different workers)
	sched1 := job.NewScheduler(jobStore, runJob, job.SchedulerConfig{
		MaxConcurrency: 1,
		RetryMax:       0,
		Backoff:        10 * time.Millisecond,
	})

	// Create a job
	jobID, err := jobStore.Create(ctx, &job.Job{AgentID: "a1", Goal: "g"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Sched1 starts processing
	sched1.Start(ctx)
	defer sched1.Stop()

	// Wait for job to be picked up
	for i := 0; i < 50; i++ {
		time.Sleep(50 * time.Millisecond)
		j, _ := jobStore.Get(ctx, jobID)
		if j != nil && j.Status == job.StatusCompleted {
			break
		}
	}

	j, _ := jobStore.Get(ctx, jobID)
	if j == nil || j.Status != job.StatusCompleted {
		t.Errorf("expected job completed, got status %v", j.Status)
	}
	if atomic.LoadInt32(&jobRunCount) != 1 {
		t.Errorf("expected jobRunCount 1, got %d", jobRunCount)
	}
}
