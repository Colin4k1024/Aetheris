// Copyright 2026 Aetheris
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

package lease_fencing

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Lease represents a job lease
type Lease struct {
	JobID       string
	WorkerID    string
	ExpiresAt   time.Time
	Token       string
}

// LeaseManager simulates lease fencing for job execution
type LeaseManager struct {
	mu     sync.RWMutex
	leases map[string]*Lease
}

// NewLeaseManager creates a new lease manager
func NewLeaseManager() *LeaseManager {
	return &LeaseManager{
		leases: make(map[string]*Lease),
	}
}

// Acquire attempts to acquire a lease for a job
// Returns: lease, bool (true if acquired, false if denied)
func (lm *LeaseManager) Acquire(ctx context.Context, jobID, workerID string, duration time.Duration) (*Lease, bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Check if lease already exists
	if existing, found := lm.leases[jobID]; found {
		if time.Now().Before(existing.ExpiresAt) {
			// Lease still valid, cannot acquire
			return nil, false
		}
		// Lease expired, can acquire
	}

	// Acquire new lease
	lease := &Lease{
		JobID:     jobID,
		WorkerID:  workerID,
		ExpiresAt: time.Now().Add(duration),
		Token:     jobID + "|" + workerID + "|" + nowUnix(),
	}
	lm.leases[jobID] = lease
	return lease, true
}

// Renew renews an existing lease
func (lm *LeaseManager) Renew(ctx context.Context, jobID, workerID, token string, duration time.Duration) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	existing, found := lm.leases[jobID]
	if !found {
		return false
	}

	// Verify token matches
	if existing.Token != token {
		return false
	}

	// Verify worker matches
	if existing.WorkerID != workerID {
		return false
	}

	// Check if still valid
	if time.Now().After(existing.ExpiresAt) {
		return false
	}

	// Renew
	existing.ExpiresAt = time.Now().Add(duration)
	return true
}

// Release releases a lease
func (lm *LeaseManager) Release(ctx context.Context, jobID, workerID, token string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	existing, found := lm.leases[jobID]
	if !found {
		return false
	}

	if existing.WorkerID != workerID || existing.Token != token {
		return false
	}

	delete(lm.leases, jobID)
	return true
}

// GetLease returns the current lease for a job
func (lm *LeaseManager) GetLease(jobID string) (*Lease, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lease, found := lm.leases[jobID]
	if found && time.Now().After(lease.ExpiresAt) {
		// Lease expired
		return nil, false
	}
	return lease, found
}

func nowUnix() string {
	return time.Now().Format("20060102150405")
}

// TestLeaseFencing_F11_LeaseExpiryDuringExecution tests that when a lease
// expires during execution, another worker can take over.
func TestLeaseFencing_F11_LeaseExpiryDuringExecution(t *testing.T) {
	ctx := context.Background()
	manager := NewLeaseManager()

	jobID := "job-f11-test"
	worker1 := "worker-1"

	// Worker 1 acquires lease
	lease1, acquired := manager.Acquire(ctx, jobID, worker1, 1*time.Second)
	require.True(t, acquired, "Worker 1 should acquire lease")
	require.NotNil(t, lease1)

	// Simulate: worker executing (sleep)
	time.Sleep(500 * time.Millisecond)

	// Lease should still be valid
	currentLease, valid := manager.GetLease(jobID)
	assert.True(t, valid, "Lease should still be valid")
	assert.Equal(t, worker1, currentLease.WorkerID)

	// Wait for lease to expire
	time.Sleep(600 * time.Millisecond)

	// Lease should now be expired
	_, valid = manager.GetLease(jobID)
	assert.False(t, valid, "Lease should be expired")

	// Worker 2 should be able to acquire
	worker2 := "worker-2"
	lease2, acquired := manager.Acquire(ctx, jobID, worker2, 1*time.Second)
	require.True(t, acquired, "Worker 2 should acquire expired lease")
	require.NotNil(t, lease2)
	assert.Equal(t, worker2, lease2.WorkerID)
}

// TestLeaseFencing_F12_WorkerHeartbeatFailure tests that when worker heartbeat
// fails, the lease is released and another worker can take over.
func TestLeaseFencing_F12_WorkerHeartbeatFailure(t *testing.T) {
	ctx := context.Background()
	manager := NewLeaseManager()

	jobID := "job-f12-test"
	worker1 := "worker-1"

	// Worker 1 acquires lease
	lease1, acquired := manager.Acquire(ctx, jobID, worker1, 2*time.Second)
	require.True(t, acquired)
	require.NotNil(t, lease1)

	// Simulate heartbeat
	isAlive := manager.Renew(ctx, jobID, worker1, lease1.Token, 2*time.Second)
	assert.True(t, isAlive, "Heartbeat should succeed")

	// Simulate heartbeat failure (worker crashed)
	// In real system: heartbeat monitor detects failure

	// Wait for lease to expire
	time.Sleep(2500 * time.Millisecond)

	// Worker 2 should be able to acquire
	worker2 := "worker-2"
	lease2, acquired := manager.Acquire(ctx, jobID, worker2, 1*time.Second)
	require.True(t, acquired, "Worker 2 should acquire after heartbeat failure")
	assert.NotNil(t, lease2)
}

// TestLeaseFencing_DoubleWorkerClaim tests that two workers cannot
// both claim the same job.
func TestLeaseFencing_DoubleWorkerClaim(t *testing.T) {
	ctx := context.Background()
	manager := NewLeaseManager()

	jobID := "job-double-test"

	// Worker 1 acquires
	lease1, acquired := manager.Acquire(ctx, jobID, "worker-1", 5*time.Second)
	require.True(t, acquired)
	_ = lease1

	// Worker 2 tries to acquire (should fail)
	lease2, acquired := manager.Acquire(ctx, jobID, "worker-2", 5*time.Second)
	assert.False(t, acquired, "Worker 2 should not acquire when lease held")
	assert.Nil(t, lease2)

	// Verify only worker 1 has lease
	currentLease, valid := manager.GetLease(jobID)
	assert.True(t, valid)
	assert.Equal(t, "worker-1", currentLease.WorkerID)
}

// TestLeaseFencing_RenewWithWrongToken tests that renewal fails with wrong token
func TestLeaseFencing_RenewWithWrongToken(t *testing.T) {
	ctx := context.Background()
	manager := NewLeaseManager()

	jobID := "job-token-test"

	// Worker 1 acquires
	lease1, _ := manager.Acquire(ctx, jobID, "worker-1", 5*time.Second)
	_ = lease1

	// Worker 2 tries to renew with wrong token
	success := manager.Renew(ctx, jobID, "worker-2", "wrong-token", 5*time.Second)
	assert.False(t, success, "Renew should fail with wrong token")

	// Verify lease still belongs to worker 1
	currentLease, valid := manager.GetLease(jobID)
	assert.True(t, valid)
	assert.Equal(t, "worker-1", currentLease.WorkerID)
}

// TestLeaseFencing_ConcurrentClaim tests concurrent lease acquisition
func TestLeaseFencing_ConcurrentClaim(t *testing.T) {
	ctx := context.Background()
	manager := NewLeaseManager()

	jobID := "job-concurrent-test"

	var wg sync.WaitGroup
	results := make(chan string, 5)

	// 5 workers trying to claim simultaneously
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			lease, acquired := manager.Acquire(ctx, jobID, workerID, 5*time.Second)
			_ = lease
			if acquired {
				results <- workerID
			} else {
				// Try to get current lease
				if l, ok := manager.GetLease(jobID); ok {
					results <- "denied:" + l.WorkerID
				} else {
					results <- "denied:expired"
				}
			}
		}("worker-" + string(rune('0'+i)))
	}

	wg.Wait()
	close(results)

	// Count how many got the lease
	acquiredCount := 0
	var leaseHolder string
	for result := range results {
		if result == "worker-1" || result == "worker-2" || result == "worker-3" ||
			result == "worker-4" || result == "worker-5" {
			acquiredCount++
			leaseHolder = result
		}
	}

	// Only one should acquire
	assert.Equal(t, 1, acquiredCount, "Only one worker should acquire the lease")
	t.Logf("Lease acquired by: %s", leaseHolder)
}
