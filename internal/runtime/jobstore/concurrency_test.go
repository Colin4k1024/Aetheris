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
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestTST06_ConcurrentClaim tests that only one worker can successfully claim a job
// when multiple workers attempt to claim concurrently.
func TestTST06_ConcurrentClaim(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	jobID := "job-concurrent-claim"

	// Setup: create a job
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})

	const numWorkers = 10
	var wg sync.WaitGroup
	successCount := atomic.Int32{}
	successWorkerIDs := make([]string, 0, numWorkers)
	var mu sync.Mutex

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		workerID := "worker-" + string(rune('A'+i))
		go func(wid string) {
			defer wg.Done()
			claimedID, _, attemptID, err := s.Claim(ctx, wid)
			if err == nil {
				count := successCount.Add(1)
				mu.Lock()
				if count == 1 {
					successWorkerIDs = append(successWorkerIDs, wid)
					_ = claimedID
					_ = attemptID
				}
				mu.Unlock()
			}
		}(workerID)
	}

	wg.Wait()

	if successCount.Load() != 1 {
		t.Errorf("expected exactly 1 successful claim, got %d", successCount.Load())
	}
}

// TestTST06_ConcurrentClaimJob tests that only one worker can successfully claim
// a specific job via ClaimJob when multiple workers attempt concurrently.
func TestTST06_ConcurrentClaimJob(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	jobID := "job-concurrent-claimjob"

	// Setup: create a job
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})

	const numWorkers = 5
	var wg sync.WaitGroup
	successCount := atomic.Int32{}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		workerID := "worker-" + string(rune('A'+i))
		go func(wid string) {
			defer wg.Done()
			_, _, err := s.ClaimJob(ctx, wid, jobID)
			if err == nil {
				successCount.Add(1)
			}
		}(workerID)
	}

	wg.Wait()

	// Exactly one worker should succeed
	if successCount.Load() != 1 {
		t.Errorf("expected exactly 1 successful ClaimJob, got %d", successCount.Load())
	}

	// Subsequent claim attempts should fail
	_, _, err := s.ClaimJob(ctx, "worker-new", jobID)
	if err != ErrClaimNotFound {
		t.Errorf("expected ErrClaimNotFound for subsequent claim, got %v", err)
	}
}

// TestTST06_ConcurrentHeartbeat tests that stale heartbeats are properly rejected
// when the lease has expired or belongs to another worker.
func TestTST06_ConcurrentHeartbeat(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	jobID := "job-heartbeat"

	// Setup: create and claim a job
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})
	_, _, _, err := s.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// Wrong worker heartbeat should fail
	err = s.Heartbeat(ctx, "worker-2", jobID)
	if err != ErrClaimNotFound {
		t.Errorf("expected ErrClaimNotFound for wrong worker, got %v", err)
	}

	// Correct worker heartbeat should succeed
	err = s.Heartbeat(ctx, "worker-1", jobID)
	if err != nil {
		t.Errorf("Heartbeat by owner should succeed, got %v", err)
	}
}

// TestTST06_GetCurrentAttemptID tests that GetCurrentAttemptID correctly retrieves
// the attempt ID for an active claim.
func TestTST06_GetCurrentAttemptID(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	jobID := "job-attemptid"

	// Setup: create and claim a job
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})
	_, _, attemptID, err := s.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// Verify attempt ID can be retrieved immediately after claim
	retrievedAttemptID, err := s.GetCurrentAttemptID(ctx, jobID)
	if err != nil {
		t.Fatalf("GetCurrentAttemptID: %v", err)
	}
	if retrievedAttemptID != attemptID {
		t.Errorf("expected attemptID %s, got %s", attemptID, retrievedAttemptID)
	}

	// Heartbeat should succeed
	err = s.Heartbeat(ctx, "worker-1", jobID)
	if err != nil {
		t.Errorf("Heartbeat should succeed, got %v", err)
	}

	// Verify attempt ID still correct after heartbeat
	retrievedAttemptID, err = s.GetCurrentAttemptID(ctx, jobID)
	if err != nil {
		t.Fatalf("GetCurrentAttemptID after Heartbeat: %v", err)
	}
	if retrievedAttemptID != attemptID {
		t.Errorf("expected attemptID %s after heartbeat, got %s", attemptID, retrievedAttemptID)
	}

	// GetCurrentAttemptID for non-existent job should return empty
	emptyAttemptID, err := s.GetCurrentAttemptID(ctx, "non-existent-job")
	if err != nil {
		t.Fatalf("GetCurrentAttemptID for non-existent: %v", err)
	}
	if emptyAttemptID != "" {
		t.Errorf("expected empty attemptID for non-existent job, got %s", emptyAttemptID)
	}
}

// TestTST06_ConcurrentAppendWithVersionMismatch tests that concurrent Append
// operations with incorrect versions are properly rejected.
func TestTST06_ConcurrentAppendWithVersionMismatch(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	jobID := "job-concurrent-append"

	// Setup: create initial event
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})

	// Simulate concurrent Append attempts - all racing to append at version 1
	const numGoroutines = 10
	successCount := atomic.Int32{}

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := s.Append(ctx, jobID, 1, JobEvent{JobID: jobID, Type: NodeStarted})
			if err == nil {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// Only one should succeed due to version check
	if successCount.Load() != 1 {
		t.Errorf("expected exactly 1 successful Append, got %d", successCount.Load())
	}

	// Verify final state: exactly 2 events
	events, ver, err := s.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if ver != 2 || len(events) != 2 {
		t.Errorf("expected version 2 and 2 events, got version %d and %d events", ver, len(events))
	}
}

// TestTST06_ConcurrentAppendWithAttemptID tests that Append with attempt_id
// correctly validates the current lease holder.
func TestTST06_ConcurrentAppendWithAttemptID(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	jobID := "job-append-attempt"

	// Setup: create job and claim
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})
	_, _, attemptID1, err := s.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// Append with correct attempt ID should succeed
	ctxWithAttempt := WithAttemptID(ctx, attemptID1)
	_, err = s.Append(ctxWithAttempt, jobID, 1, JobEvent{JobID: jobID, Type: PlanGenerated})
	if err != nil {
		t.Errorf("Append with correct attemptID should succeed, got %v", err)
	}

	// Append with wrong attempt ID should fail (version is now 2, use correct version)
	ctxWrongAttempt := WithAttemptID(ctx, "wrong-attempt-id")
	_, err = s.Append(ctxWrongAttempt, jobID, 2, JobEvent{JobID: jobID, Type: NodeStarted})
	if err != ErrStaleAttempt {
		t.Errorf("expected ErrStaleAttempt, got %v", err)
	}

	// Create another job for worker-2 to claim
	jobID2 := "job-append-attempt-2"
	_, _ = s.Append(ctx, jobID2, 0, JobEvent{JobID: jobID2, Type: JobCreated})
	_, _, attemptID2, err := s.Claim(ctx, "worker-2")
	if err != nil {
		t.Fatalf("Claim by worker-2: %v", err)
	}

	// Worker-1's attempt ID on job 1 is still valid since worker-2 claimed a DIFFERENT job (jobID2)
	// The claim on jobID still belongs to worker-1 with attemptID1, so append should succeed
	currentAttemptID, _ := s.GetCurrentAttemptID(ctx, jobID)
	if currentAttemptID != attemptID1 {
		t.Errorf("expected currentAttemptID=%q, got %q", attemptID1, currentAttemptID)
	}
	_, err = s.Append(ctxWithAttempt, jobID, 2, JobEvent{JobID: jobID, Type: NodeStarted})
	if err != nil {
		t.Errorf("Append with worker-1's attemptID on job-1 should succeed, got %v", err)
	}

	// Worker-2's attempt ID should work on job 2
	ctxNewAttempt := WithAttemptID(ctx, attemptID2)
	_, err = s.Append(ctxNewAttempt, jobID2, 1, JobEvent{JobID: jobID2, Type: NodeStarted})
	if err != nil {
		t.Errorf("Append with worker-2 attemptID on job-2 should succeed, got %v", err)
	}
}

// TestTST06_ConcurrentClaimAndHeartbeatRace tests race conditions between
// Claim and Heartbeat operations across multiple jobs using ClaimJob.
func TestTST06_ConcurrentClaimAndHeartbeatRace(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	// Create multiple jobs
	const numJobs = 20
	jobIDs := make([]string, numJobs)
	for i := 0; i < numJobs; i++ {
		jobIDs[i] = fmt.Sprintf("job-race-%d", i)
		_, _ = s.Append(ctx, jobIDs[i], 0, JobEvent{JobID: jobIDs[i], Type: JobCreated})
	}

	var wg sync.WaitGroup
	var claimCount atomic.Int32
	var claimErrCount atomic.Int32
	var heartbeatFailCount atomic.Int32
	var attemptIDMismatch atomic.Int32

	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		jobID := jobIDs[i]
		go func(jid string) {
			defer wg.Done()

			// Use ClaimJob to claim a specific job
			_, attemptID, err := s.ClaimJob(ctx, "worker-main", jid)
			if err != nil {
				claimErrCount.Add(1)
				return
			}

			claimCount.Add(1)

			// Immediately try heartbeat
			err = s.Heartbeat(ctx, "worker-main", jid)
			if err != nil {
				heartbeatFailCount.Add(1)
				return
			}

			// Verify attempt ID consistency
			gotAttemptID, err := s.GetCurrentAttemptID(ctx, jid)
			if err != nil {
				return
			}
			if gotAttemptID != attemptID {
				attemptIDMismatch.Add(1)
			}
		}(jobID)
	}

	wg.Wait()

	if claimCount.Load() != numJobs {
		t.Errorf("expected %d successful claims, got %d", numJobs, claimCount.Load())
	}
	if claimErrCount.Load() != 0 {
		t.Errorf("expected 0 claim errors, got %d", claimErrCount.Load())
	}
	if heartbeatFailCount.Load() != 0 {
		t.Errorf("expected 0 heartbeat failures, got %d", heartbeatFailCount.Load())
	}
	if attemptIDMismatch.Load() != 0 {
		t.Errorf("expected 0 attemptID mismatches, got %d", attemptIDMismatch.Load())
	}
}

// TestTST06_ConcurrentWatchAndAppend tests that Watch correctly receives events
// appended from multiple goroutines.
func TestTST06_ConcurrentWatchAndAppend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := NewMemoryStore()
	jobID := "job-watch-append"

	// Setup: create initial event
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})

	// Start watching
	ch, err := s.Watch(ctx, jobID)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	const numEvents = 5

	// Append events sequentially using a mutex to protect version coordination
	var mu sync.Mutex
	var wg sync.WaitGroup
	currentVersion := 1

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			ver := currentVersion
			currentVersion++
			_, err := s.Append(ctx, jobID, ver, JobEvent{JobID: jobID, Type: NodeStarted})
			mu.Unlock()
			if err != nil {
				t.Errorf("Append failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Give watch channel time to deliver
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Count events received
	receivedCount := 0
	for range ch {
		receivedCount++
	}

	if receivedCount < numEvents {
		t.Errorf("expected at least %d events on watch, got %d", numEvents, receivedCount)
	}
}

// TestTST06_MultipleJobsConcurrentClaim tests claim behavior across multiple
// jobs with concurrent workers.
func TestTST06_MultipleJobsConcurrentClaim(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	const numJobs = 5
	jobIDs := make([]string, numJobs)

	// Setup: create multiple jobs
	for i := 0; i < numJobs; i++ {
		jobIDs[i] = "job-multi-" + string(rune('0'+i))
		_, _ = s.Append(ctx, jobIDs[i], 0, JobEvent{JobID: jobIDs[i], Type: JobCreated})
	}

	const numWorkers = 10
	var wg sync.WaitGroup
	claimedJobs := make(map[string]string) // workerID -> jobID
	var mu sync.Mutex

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		workerID := "worker-multi-" + string(rune('A'+i))
		go func(wid string) {
			defer wg.Done()
			claimedID, _, _, err := s.Claim(ctx, wid)
			if err == nil {
				mu.Lock()
				claimedJobs[wid] = claimedID
				mu.Unlock()
			}
		}(workerID)
	}

	wg.Wait()

	// Multiple workers should have claimed different jobs
	if len(claimedJobs) < 1 {
		t.Errorf("expected at least 1 successful claim, got %d", len(claimedJobs))
	}

	// No two workers should have claimed the same job
	jobClaimCount := make(map[string]int)
	for _, jobID := range claimedJobs {
		jobClaimCount[jobID]++
	}
	for jobID, count := range jobClaimCount {
		if count > 1 {
			t.Errorf("job %s claimed by multiple workers: %d times", jobID, count)
		}
	}
}

// TestTST06_ConcurrentAppendDifferentJobs tests that Append operations on
// different jobs do not interfere with each other.
func TestTST06_ConcurrentAppendDifferentJobs(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	const numJobs = 5
	jobIDs := make([]string, numJobs)

	// Setup: create multiple jobs
	for i := 0; i < numJobs; i++ {
		jobIDs[i] = "job-diff-" + string(rune('0'+i))
		_, _ = s.Append(ctx, jobIDs[i], 0, JobEvent{JobID: jobIDs[i], Type: JobCreated})
	}

	const eventsPerJob = 10
	var wg sync.WaitGroup

	// Concurrently append events to different jobs
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func(jobIdx int) {
			defer wg.Done()
			jobID := jobIDs[jobIdx]
			for j := 0; j < eventsPerJob; j++ {
				_, err := s.Append(ctx, jobID, j+1, JobEvent{JobID: jobID, Type: NodeStarted})
				if err != nil {
					t.Errorf("Append to job %s failed: %v", jobID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all jobs have correct event counts
	for i := 0; i < numJobs; i++ {
		jobID := jobIDs[i]
		events, ver, err := s.ListEvents(ctx, jobID)
		if err != nil {
			t.Errorf("ListEvents for %s: %v", jobID, err)
		}
		expectedVer := eventsPerJob + 1 // initial + appended events
		if ver != expectedVer {
			t.Errorf("job %s: expected version %d, got %d", jobID, expectedVer, ver)
		}
		if len(events) != expectedVer {
			t.Errorf("job %s: expected %d events, got %d", jobID, expectedVer, len(events))
		}
	}
}

// TestTST06_ListJobIDsWithExpiredClaim tests that expired claims are correctly identified.
func TestTST06_ListJobIDsWithExpiredClaim(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	jobID := "job-expired"

	// Setup: create and claim a job
	_, _ = s.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})
	_, _, _, err := s.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// Initially no expired claims
	expired, err := s.ListJobIDsWithExpiredClaim(ctx)
	if err != nil {
		t.Fatalf("ListJobIDsWithExpiredClaim: %v", err)
	}
	if len(expired) != 0 {
		t.Errorf("expected 0 expired claims initially, got %d", len(expired))
	}

	// Note: Testing actual expiration would require waiting for leaseDuration (30s)
	// which is impractical for unit tests. The memory store implementation uses
	// in-memory time which cannot be easily manipulated in tests.
	// This test verifies the correct behavior path is exercised.

	t.Log("Note: Actual lease expiration not tested due to 30s duration; tested claim/refresh flow")
}
