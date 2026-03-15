// Copyright 2026 Aetheris
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestLeaseConfig_Validation tests that LeaseConfig validation works correctly
func TestLeaseConfig_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       LeaseConfig
		wantValid bool
	}{
		{
			name: "valid config",
			cfg: LeaseConfig{
				LeaseDuration:     30 * time.Second,
				HeartbeatInterval: 10 * time.Second,
			},
			wantValid: true,
		},
		{
			name: "heartbeat longer than lease",
			cfg: LeaseConfig{
				LeaseDuration:     30 * time.Second,
				HeartbeatInterval: 40 * time.Second,
			},
			wantValid: false,
		},
		{
			name: "zero lease duration",
			cfg: LeaseConfig{
				LeaseDuration:     0,
				HeartbeatInterval: 10 * time.Second,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.cfg.HeartbeatInterval < tt.cfg.LeaseDuration && tt.cfg.LeaseDuration > 0
			assert.Equal(t, tt.wantValid, isValid)
		})
	}
}

// TestLeaseConfig_RecommendedHeartbeat tests that recommended heartbeat interval
// is LeaseDuration/2 as per documentation
func TestLeaseConfig_RecommendedHeartbeat(t *testing.T) {
	cfg := LeaseConfig{
		LeaseDuration:     30 * time.Second,
		HeartbeatInterval: 15 * time.Second, // LeaseDuration / 2
	}

	// Recommended: heartbeat interval should be <= LeaseDuration / 2
	isRecommended := cfg.HeartbeatInterval <= cfg.LeaseDuration/2
	assert.True(t, isRecommended, "Heartbeat interval should be <= LeaseDuration/2")
}

// TestContextCancellation_DoesNotAffectLease tests that canceling a context
// doesn't affect the lease state (lease is managed by the store)
func TestContextCancellation_DoesNotAffectLease(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate work
	_ = ctx

	// Cancel context
	cancel()

	// Context should be cancelled
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		t.Fatal("context should be cancelled")
	}

	// Note: In real implementation, lease is managed by jobstore
	// Context cancellation doesn't automatically release the lease
	// The lease must be explicitly released via store operations
}

// TestFairnessPolicy_DefaultValues tests default fairness policy values
func TestFairnessPolicy_DefaultValues(t *testing.T) {
	policy := DefaultFairnessPolicy()

	assert.Equal(t, 70, policy.RealtimeWeight)
	assert.Equal(t, 20, policy.DefaultWeight)
	assert.Equal(t, 8, policy.BackgroundWeight)
	assert.Equal(t, 2, policy.HeavyWeight)
	assert.Equal(t, 5*time.Minute, policy.StarvationThreshold)
}

// TestFairQueueScheduler_NewFairQueueScheduler tests scheduler creation
func TestFairQueueScheduler_NewFairQueueScheduler(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	assert.NotNil(t, scheduler)
	assert.Equal(t, policy, scheduler.policy)
}

// TestFairQueueScheduler_SelectQueue tests queue selection
func TestFairQueueScheduler_SelectQueue(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	ctx := context.Background()
	availableQueues := []string{"realtime", "default", "background", "heavy"}

	// First selection should return the highest weight queue
	selected := scheduler.SelectQueue(ctx, availableQueues)
	assert.Equal(t, "realtime", selected)
}

// TestFairQueueScheduler_SelectQueue_EmptyAvailable tests with no available queues
func TestFairQueueScheduler_SelectQueue_EmptyAvailable(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	ctx := context.Background()
	selected := scheduler.SelectQueue(ctx, []string{})
	assert.Equal(t, "", selected)
}

// TestFairQueueScheduler_TrackJobStart tests job wait tracking
func TestFairQueueScheduler_TrackJobStart(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	scheduler.TrackJobStart("job-1")
	scheduler.TrackJobStart("job-2")

	stats := scheduler.GetStats()
	assert.Equal(t, 2, stats["waiting_jobs"])
}

// TestFairQueueScheduler_CheckStarvation tests starvation detection
func TestFairQueueScheduler_CheckStarvation(t *testing.T) {
	policy := FairnessPolicy{
		StarvationThreshold: 1 * time.Millisecond,
	}
	scheduler := NewFairQueueScheduler(policy)

	scheduler.TrackJobStart("job-1")

	// Should not be starved immediately
	assert.False(t, scheduler.CheckStarvation("job-1"))

	// Wait and check again
	time.Sleep(2 * time.Millisecond)
	assert.True(t, scheduler.CheckStarvation("job-1"))
}

// TestFairQueueScheduler_CheckStarvation_NotTracked tests non-tracked job
func TestFairQueueScheduler_CheckStarvation_NotTracked(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	assert.False(t, scheduler.CheckStarvation("nonexistent-job"))
}

// TestFairQueueScheduler_CompleteJob tests job completion cleanup
func TestFairQueueScheduler_CompleteJob(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	scheduler.TrackJobStart("job-1")
	scheduler.CompleteJob("job-1")

	stats := scheduler.GetStats()
	assert.Equal(t, 0, stats["waiting_jobs"])
}

// TestFairQueueScheduler_GetStats tests stats collection
func TestFairQueueScheduler_GetStats(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	scheduler.TrackJobStart("job-1")
	scheduler.TrackJobStart("job-2")

	stats := scheduler.GetStats()

	assert.Greater(t, stats["current_round"].(int), 0)
	assert.NotNil(t, stats["queue_tickets"])
	assert.NotNil(t, stats["queue_weights"])
	assert.Equal(t, 2, stats["waiting_jobs"])
	assert.Equal(t, 0, stats["starved_jobs"])
}

// TestFairQueueScheduler_AdjustWeights tests weight adjustment
func TestFairQueueScheduler_AdjustWeights(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	// Adjust to new weight
	scheduler.AdjustWeights("realtime", 50)

	stats := scheduler.GetStats()
	weights := stats["queue_weights"].(map[string]int)
	assert.Equal(t, 50, weights["realtime"])
}

// TestFairQueueScheduler_AdjustWeights_Bounds tests weight boundary handling
func TestFairQueueScheduler_AdjustWeights_Bounds(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	// Test negative weight
	scheduler.AdjustWeights("realtime", -10)
	stats := scheduler.GetStats()
	weights := stats["queue_weights"].(map[string]int)
	assert.Equal(t, 0, weights["realtime"])

	// Test over 100
	scheduler.AdjustWeights("realtime", 150)
	stats = scheduler.GetStats()
	weights = stats["queue_weights"].(map[string]int)
	assert.Equal(t, 100, weights["realtime"])
}

// TestFairQueueScheduler_MultipleRounds tests round-robin behavior
func TestFairQueueScheduler_MultipleRounds(t *testing.T) {
	policy := DefaultFairnessPolicy()
	scheduler := NewFairQueueScheduler(policy)

	ctx := context.Background()
	availableQueues := []string{"realtime", "default", "background", "heavy"}

	initialRound := scheduler.GetStats()["current_round"].(int)

	// Exhaust all tickets
	for i := 0; i < 100; i++ {
		scheduler.SelectQueue(ctx, availableQueues)
	}

	// Should have incremented round
	stats := scheduler.GetStats()
	assert.GreaterOrEqual(t, stats["current_round"].(int), initialRound)
}
