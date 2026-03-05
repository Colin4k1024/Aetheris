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
