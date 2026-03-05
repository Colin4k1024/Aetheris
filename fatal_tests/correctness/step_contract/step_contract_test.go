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

package step_contract

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rag-platform/internal/agent/runtime"
)

// TestStepContract_S1_DeterministicTime tests that Clock provides deterministic
// time during replay (no side effects from time.Now()).
func TestStepContract_S1_DeterministicTime(t *testing.T) {
	jobID, stepID := "job-s1-test", "step-1"

	// Create ReplayClock - should return same time for same job+step
	replayClock1 := runtime.ReplayClock(jobID, stepID)
	t1 := replayClock1()

	replayClock2 := runtime.ReplayClock(jobID, stepID)
	t2 := replayClock2()

	// Verification: same job+step should produce identical time
	assert.Equal(t, t1, t2, "ReplayClock must be deterministic for same job+step")

	// Different step should produce potentially different time
	replayClock3 := runtime.ReplayClock(jobID, "step-2")
	t3 := replayClock3()

	// Note: In practice, deterministic replay may produce same or different time
	// depending on implementation. The key is it's reproducible.
	_ = t3
}

// TestStepContract_S2_DeterministicRandom tests that RNG provides deterministic
// random numbers during replay.
func TestStepContract_S2_DeterministicRandom(t *testing.T) {
	ctx := context.Background()
	jobID, stepID := "job-s2-test", "step-1"
	n := 100

	// Create ReplayRNG - should return same sequence for same job+step
	replayRNG1 := runtime.ReplayRNG(jobID, stepID)
	ctx1 := runtime.WithRNG(ctx, replayRNG1)

	replayRNG2 := runtime.ReplayRNG(jobID, stepID)
	ctx2 := runtime.WithRNG(ctx, replayRNG2)

	// Generate same sequence
	values1 := make([]int, 5)
	values2 := make([]int, 5)

	for i := 0; i < 5; i++ {
		values1[i] = runtime.RandIntn(ctx1, n)
		values2[i] = runtime.RandIntn(ctx2, n)
	}

	// Verification: same job+step should produce identical sequence
	assert.Equal(t, values1, values2, "ReplayRNG must produce deterministic sequence")
}

// TestStepContract_S3_TimeInjectedDuringReplay tests that Clock can be
// injected during replay mode.
func TestStepContract_S3_TimeInjectedDuringReplay(t *testing.T) {
	ctx := context.Background()
	jobID, stepID := "job-s3-test", "step-1"

	// Simulate replay: inject deterministic clock
	replayClock := runtime.ReplayClock(jobID, stepID)
	ctxWithClock := runtime.WithClock(ctx, replayClock)

	// Get clock from context
	gotClock := runtime.Clock(ctxWithClock)

	// Verification: should return the injected deterministic time
	assert.Equal(t, gotClock, replayClock(), "Clock should return injected deterministic time")
}

// TestStepContract_S4_RNGInjectedDuringReplay tests that RNG can be
// injected during replay mode.
func TestStepContract_S4_RNGInjectedDuringReplay(t *testing.T) {
	ctx := context.Background()
	jobID, stepID := "job-s4-test", "step-1"

	// Simulate replay: inject deterministic RNG
	replayRNG := runtime.ReplayRNG(jobID, stepID)
	ctxWithRNG := runtime.WithRNG(ctx, replayRNG)

	// Get RNG from context and use it
	val := runtime.RandIntn(ctxWithRNG, 1000)

	// Verification: should work with injected RNG
	require.GreaterOrEqual(t, val, 0)
	require.Less(t, val, 1000)
}

// TestStepContract_VerifyContractCompliance tests that the runtime
// properly enforces the Step Contract.
func TestStepContract_VerifyContractCompliance(t *testing.T) {
	// This test verifies that the Step Contract interface is properly implemented

	ctx := context.Background()
	jobID := "job-contract-test"

	// Test 1: Clock should be available
	_ = runtime.Clock(ctx)

	// Test 2: RandIntn should be available
	_ = runtime.RandIntn(ctx, 100)

	// Test 3: ReplayClock should be a function
	replayClock := runtime.ReplayClock(jobID, "step-1")
	require.NotNil(t, replayClock)
	_ = replayClock()

	// Test 4: ReplayRNG should be a function
	replayRNG := runtime.ReplayRNG(jobID, "step-1")
	require.NotNil(t, replayRNG)

	// Test 5: WithClock should inject clock into context
	injectedClock := runtime.ReplayClock(jobID, "step-2")
	ctxWithClock := runtime.WithClock(ctx, injectedClock)
	resultClock := runtime.Clock(ctxWithClock)
	assert.Equal(t, injectedClock(), resultClock)

	// Test 6: WithRNG should inject RNG into context
	injectedRNG := runtime.ReplayRNG(jobID, "step-2")
	ctxWithRNG := runtime.WithRNG(ctx, injectedRNG)
	resultRNG := runtime.RandIntn(ctxWithRNG, 100)
	assert.GreaterOrEqual(t, resultRNG, 0)
	assert.Less(t, resultRNG, 100)

	t.Log("Step Contract interface verified successfully")
}
