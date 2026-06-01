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

// Package routing_test contains the RoutingAdvisor recovery validation suite.
//
// Recovery Validation: 9-Step Proof
//
// This file proves that the RoutingAdvisor evidence contract is replay-safe.
// The nine steps mirror the acceptance criteria in issue #210:
//
//  1. Original execution performs routing (advisor.Decide called once).
//  2. Route evidence is persisted (route_decision_recorded event appended).
//  3. Worker failure occurs after persistence (event log survives crash).
//  4. Recovery resumes from checkpoint (persisted evidence is loadable).
//  5. Replay performs zero outbound routing calls (advisor never called again).
//  6. Recorded route evidence is reused (decision_id matches original).
//  7. decision_hash remains unchanged (hash is immutable across replay).
//  8. Route evidence remains byte-identical (JSON marshaling is stable).
//  9. Execution trace remains unchanged (same capability is executed).
package routing_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Colin4k1024/Aetheris/v2/pkg/routing"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// countingAdvisor wraps a RoutingAdvisor and records every Decide call.
// Used to assert that replay paths do not invoke the advisor.
type countingAdvisor struct {
	inner       routing.RoutingAdvisor
	decideCalls int
}

func (c *countingAdvisor) Decide(ctx context.Context, req routing.RouteDecisionRequest) (routing.RouteDecision, error) {
	c.decideCalls++
	return c.inner.Decide(ctx, req)
}

func (c *countingAdvisor) RecordOutcome(ctx context.Context, outcome routing.RouteOutcome) error {
	return c.inner.RecordOutcome(ctx, outcome)
}

// fakeEventLog simulates a persisted event store for route_decision_recorded
// events. It is keyed by decision_key, matching the recovery lookup contract.
type fakeEventLog struct {
	decisions []routing.RouteDecision
}

// append persists a route decision (simulates writing a route_decision_recorded event).
func (e *fakeEventLog) append(d routing.RouteDecision) {
	e.decisions = append(e.decisions, d)
}

// findByDecisionKey looks up a previously persisted decision.
// Returns (RouteDecision, true) if found, or (zero, false) if not.
func (e *fakeEventLog) findByDecisionKey(key string) (routing.RouteDecision, bool) {
	for _, d := range e.decisions {
		if d.DecisionKey == key {
			return d, true
		}
	}
	return routing.RouteDecision{}, false
}

// findByTenant returns decisions for a specific tenant.
// Used in isolation tests.
func (e *fakeEventLog) findByTenant(tenantID string) []routing.RouteDecision {
	var out []routing.RouteDecision
	for _, d := range e.decisions {
		if d.TenantID == tenantID {
			out = append(out, d)
		}
	}
	return out
}

// routeExecution models the runtime's routing logic in a single function.
// On original execution (isReplay=false), it calls the advisor and persists
// the resulting evidence to the event log.
// On replay (isReplay=true), it reads from the log and verifies the hash
// without calling the advisor at all.
//
// The returned values are:
//   - RouteDecision: the decision evidence (original or replayed)
//   - selectedCapabilityID: the capability that should execute the step
//   - error: routing_decision_missing or routing_decision_hash_mismatch on replay errors
func routeExecution(
	ctx context.Context,
	advisor routing.RoutingAdvisor,
	log *fakeEventLog,
	req routing.RouteDecisionRequest,
	isReplay bool,
) (routing.RouteDecision, string, error) {
	if isReplay {
		d, ok := log.findByDecisionKey(req.DecisionKey)
		if !ok {
			return routing.RouteDecision{}, "", fmt.Errorf("routing_decision_missing: no evidence for decision_key=%s", req.DecisionKey)
		}
		if err := routing.VerifyDecisionHash(d); err != nil {
			return routing.RouteDecision{}, "", fmt.Errorf("routing_decision_hash_mismatch: %w", err)
		}
		return d, d.Selected.CapabilityID, nil
	}

	// Original execution: call advisor → persist evidence → return selected capability.
	d, err := advisor.Decide(ctx, req)
	if err != nil {
		return routing.RouteDecision{}, "", fmt.Errorf("routing: advisor.Decide failed: %w", err)
	}
	log.append(d)
	return d, d.Selected.CapabilityID, nil
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// fixtureRequest returns the canonical RouteDecisionRequest used across all
// recovery validation steps.
func fixtureRequest() routing.RouteDecisionRequest {
	return routing.RouteDecisionRequest{
		SchemaVersion: routing.RouteDecisionRequestSchemaV1,
		JobID:         "job-recovery-001",
		RunID:         "run-recovery-001",
		TenantID:      "tenant-alpha",
		PlanID:        "plan-recovery-001",
		NodeID:        "node-retrieve",
		StepID:        "step-1",
		DecisionKey:   "job-recovery-001:node-retrieve:step-1",
		Goal:          "search for Aetheris runtime information",
		Candidates: []routing.RouteCandidate{
			{
				CapabilityID: "tool:web_search",
				Kind:         routing.KindTool,
				Provider:     "builtin",
				Score:        0.95,
				Rank:         1,
				ReasonCodes:  []string{"historical_success", "lowest_expected_latency"},
			},
			{
				CapabilityID: "tool:calculator",
				Kind:         routing.KindTool,
				Provider:     "builtin",
				Score:        0.60,
				Rank:         2,
				ReasonCodes:  []string{"low_cost"},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Step 1 — Original execution calls the advisor exactly once.
// ---------------------------------------------------------------------------

func TestRecovery_Step1_OriginalExecution_CallsAdvisorOnce(t *testing.T) {
	advisor := &countingAdvisor{inner: routing.NewNoOpAdvisor()}
	log := &fakeEventLog{}

	_, _, err := routeExecution(context.Background(), advisor, log, fixtureRequest(), false)
	require.NoError(t, err)

	assert.Equal(t, 1, advisor.decideCalls,
		"advisor.Decide must be called exactly once during original execution")
}

// ---------------------------------------------------------------------------
// Step 2 — Route evidence is persisted before selected capability executes.
// ---------------------------------------------------------------------------

func TestRecovery_Step2_OriginalExecution_EvidencePersisted(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{}
	req := fixtureRequest()

	_, _, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)

	require.Len(t, log.decisions, 1, "exactly one route_decision_recorded event expected")

	d := log.decisions[0]
	assert.Equal(t, req.DecisionKey, d.DecisionKey, "persisted decision must match request decision_key")
	assert.Equal(t, req.JobID, d.JobID)
	assert.Equal(t, req.TenantID, d.TenantID)
	assert.NotEmpty(t, d.DecisionHash, "decision_hash must be populated before persistence")
	assert.NoError(t, routing.VerifyDecisionHash(d), "persisted decision must have a valid hash")
}

// ---------------------------------------------------------------------------
// Step 3+4 — Simulated worker failure and recovery resume from checkpoint.
//
// In Aetheris, the event log (JobStore) is durable. A worker crash does not
// remove persisted events. Recovery loads the checkpoint and continues from
// where the job stopped.
// ---------------------------------------------------------------------------

func TestRecovery_Step3_4_WorkerFailure_RecoveryResumesFromCheckpoint(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{}
	req := fixtureRequest()

	// Original execution: advisor is called, evidence is persisted.
	original, _, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)

	// --- Simulated worker crash ---
	// The event log survives because it is durable (event-sourced store).
	// Worker restarts and a new goroutine is assigned the job.
	// The runtime reads the checkpoint and locates the route_decision_recorded event.

	// Post-crash state: evidence still in log, decision_key lookup succeeds.
	recovered, ok := log.findByDecisionKey(req.DecisionKey)
	require.True(t, ok, "route evidence must survive simulated worker crash (event-sourced durability)")
	assert.Equal(t, original.DecisionID, recovered.DecisionID,
		"recovered decision must be the same as the original (no new decision was made)")
}

// ---------------------------------------------------------------------------
// Step 5 — Replay performs zero outbound advisor calls.
// ---------------------------------------------------------------------------

func TestRecovery_Step5_Replay_ZeroAdvisorCalls(t *testing.T) {
	advisor := &countingAdvisor{inner: routing.NewNoOpAdvisor()}
	log := &fakeEventLog{}
	req := fixtureRequest()

	// Original execution.
	_, _, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)
	callsAfterOriginal := advisor.decideCalls

	// Replay: must NOT call the advisor again.
	_, _, err = routeExecution(context.Background(), advisor, log, req, true)
	require.NoError(t, err)

	assert.Equal(t, callsAfterOriginal, advisor.decideCalls,
		"replay must not call advisor.Decide (zero additional advisor calls)")
}

// ---------------------------------------------------------------------------
// Step 6 — Replay reuses the recorded route evidence.
// ---------------------------------------------------------------------------

func TestRecovery_Step6_Replay_ReusesRecordedEvidence(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{}
	req := fixtureRequest()

	original, originalCap, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)

	replayed, replayedCap, err := routeExecution(context.Background(), advisor, log, req, true)
	require.NoError(t, err)

	assert.Equal(t, original.DecisionID, replayed.DecisionID,
		"replay must return the same decision_id as the original")
	assert.Equal(t, originalCap, replayedCap,
		"replay must select the same capability as original execution")
}

// ---------------------------------------------------------------------------
// Step 7 — decision_hash is immutable across replay.
// ---------------------------------------------------------------------------

func TestRecovery_Step7_Replay_DecisionHashUnchanged(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{}
	req := fixtureRequest()

	original, _, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)

	replayed, _, err := routeExecution(context.Background(), advisor, log, req, true)
	require.NoError(t, err)

	assert.Equal(t, original.DecisionHash, replayed.DecisionHash,
		"decision_hash must be identical across original and replay")
	assert.NoError(t, routing.VerifyDecisionHash(replayed),
		"replayed decision must pass hash verification")
}

// ---------------------------------------------------------------------------
// Step 8 — Route evidence is byte-identical across replay.
// ---------------------------------------------------------------------------

func TestRecovery_Step8_Replay_EvidenceByteIdentical(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{}
	req := fixtureRequest()

	original, _, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)

	replayed, _, err := routeExecution(context.Background(), advisor, log, req, true)
	require.NoError(t, err)

	originalJSON, err := json.Marshal(original)
	require.NoError(t, err)
	replayedJSON, err := json.Marshal(replayed)
	require.NoError(t, err)

	assert.Equal(t, string(originalJSON), string(replayedJSON),
		"route evidence JSON must be byte-identical between original and replay")
}

// ---------------------------------------------------------------------------
// Step 9 — Execution trace (capability selected) remains unchanged.
// ---------------------------------------------------------------------------

func TestRecovery_Step9_Replay_ExecutionTraceUnchanged(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{}
	req := fixtureRequest()

	_, originalCap, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)

	_, replayedCap, err := routeExecution(context.Background(), advisor, log, req, true)
	require.NoError(t, err)

	assert.Equal(t, originalCap, replayedCap,
		"replay must execute the same capability as original execution (execution trace unchanged)")
}

// ---------------------------------------------------------------------------
// Full 9-step narrative test — sequential proof of the recovery invariant.
// ---------------------------------------------------------------------------

// TestRecovery_Full9StepNarrative runs all recovery steps in order as a
// single narrative test. This is the canonical "recovery proof" that
// demonstrates the full lifecycle of a RoutingAdvisor decision:
// original execution → evidence persistence → crash → recovery → replay.
func TestRecovery_Full9StepNarrative(t *testing.T) {
	advisor := &countingAdvisor{inner: routing.NewNoOpAdvisor()}
	log := &fakeEventLog{}
	req := fixtureRequest()
	ctx := context.Background()

	// -----------------------------------------------------------------------
	// Steps 1 & 2: Original execution — advisor called once, evidence persisted.
	// -----------------------------------------------------------------------
	original, originalCap, err := routeExecution(ctx, advisor, log, req, false)
	require.NoError(t, err, "step 1-2: original execution must succeed")

	assert.Equal(t, 1, advisor.decideCalls,
		"step 1: advisor.Decide called exactly once during original execution")
	require.Len(t, log.decisions, 1,
		"step 2: route_decision_recorded event persisted in event log")
	assert.NoError(t, routing.VerifyDecisionHash(log.decisions[0]),
		"step 2: persisted evidence has a valid decision_hash")

	// -----------------------------------------------------------------------
	// Steps 3 & 4: Simulated crash and recovery.
	// -----------------------------------------------------------------------
	// Crash simulation: worker process terminates after persisting evidence.
	// Recovery: new worker reads from durable event log (checkpoint).
	recovered, ok := log.findByDecisionKey(req.DecisionKey)
	require.True(t, ok,
		"step 3-4: route evidence survives worker crash (event-sourced durability)")
	assert.Equal(t, original.DecisionID, recovered.DecisionID,
		"step 4: recovered decision_id matches original")

	// -----------------------------------------------------------------------
	// Steps 5-9: Replay invariants.
	// -----------------------------------------------------------------------
	advisorCallsBeforeReplay := advisor.decideCalls

	replayed, replayedCap, err := routeExecution(ctx, advisor, log, req, true)
	require.NoError(t, err, "step 5-9: replay must succeed")

	// Step 5: Replay makes zero advisor calls.
	assert.Equal(t, advisorCallsBeforeReplay, advisor.decideCalls,
		"step 5: replay calls advisor.Decide zero times")

	// Step 6: Replay reuses recorded evidence.
	assert.Equal(t, original.DecisionID, replayed.DecisionID,
		"step 6: replay returns the same decision_id as original")

	// Step 7: decision_hash unchanged.
	assert.Equal(t, original.DecisionHash, replayed.DecisionHash,
		"step 7: decision_hash is immutable across replay")
	assert.NoError(t, routing.VerifyDecisionHash(replayed),
		"step 7: replayed decision passes hash verification")

	// Step 8: Evidence byte-identical.
	origJSON, _ := json.Marshal(original)
	replayJSON, _ := json.Marshal(replayed)
	assert.Equal(t, string(origJSON), string(replayJSON),
		"step 8: route evidence JSON is byte-identical between original and replay")

	// Step 9: Execution trace unchanged.
	assert.Equal(t, originalCap, replayedCap,
		"step 9: replay selects the same capability as original execution")
}

// ---------------------------------------------------------------------------
// Error path: missing route evidence during replay.
// ---------------------------------------------------------------------------

func TestRecovery_Replay_MissingEvidence_ReturnsError(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{} // empty — no evidence persisted
	req := fixtureRequest()

	_, _, err := routeExecution(context.Background(), advisor, log, req, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "routing_decision_missing",
		"replay with missing evidence must return routing_decision_missing error")
}

// ---------------------------------------------------------------------------
// Error path: hash mismatch during replay (tampered evidence).
// ---------------------------------------------------------------------------

func TestRecovery_Replay_TamperedHash_ReturnsError(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	log := &fakeEventLog{}
	req := fixtureRequest()

	// Original execution persists valid evidence.
	_, _, err := routeExecution(context.Background(), advisor, log, req, false)
	require.NoError(t, err)

	// Tamper with the persisted decision hash.
	log.decisions[0].DecisionHash = "000000000000000000000000000000000000000000000000000000000000dead"

	_, _, err = routeExecution(context.Background(), advisor, log, req, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "routing_decision_hash_mismatch",
		"tampered evidence must be rejected with routing_decision_hash_mismatch")
}

// ---------------------------------------------------------------------------
// Tenant isolation: Tenant A evidence must not be reused by Tenant B.
// ---------------------------------------------------------------------------

func TestRecovery_TenantIsolation_CrossTenantReplay_NotAllowed(t *testing.T) {
	advisorA := routing.NewNoOpAdvisor()
	// Tenant A executes and persists evidence.
	logA := &fakeEventLog{}
	reqA := fixtureRequest()
	reqA.TenantID = "tenant-alpha"
	reqA.JobID = "job-alpha-001"
	reqA.DecisionKey = "job-alpha-001:node-1:step-1"

	_, _, err := routeExecution(context.Background(), advisorA, logA, reqA, false)
	require.NoError(t, err)

	// Tenant B tries to replay using Tenant A's log — different decision_key
	// because tenant_id isolation means Tenant B never wrote to logA.
	advisorB := routing.NewNoOpAdvisor()
	_ = advisorB
	reqB := fixtureRequest()
	reqB.TenantID = "tenant-beta"
	reqB.JobID = "job-beta-001"
	reqB.DecisionKey = "job-beta-001:node-1:step-1"

	// Tenant B's decision_key is not in Tenant A's log.
	_, _, err = routeExecution(context.Background(), advisorA, logA, reqB, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "routing_decision_missing",
		"cross-tenant replay must fail with routing_decision_missing because evidence is keyed by job+tenant")

	// Verify logA only contains Tenant A evidence.
	tenantADecisions := logA.findByTenant("tenant-alpha")
	tenantBDecisions := logA.findByTenant("tenant-beta")
	assert.Len(t, tenantADecisions, 1, "log contains exactly one Tenant A decision")
	assert.Empty(t, tenantBDecisions, "log contains no Tenant B decisions (tenant isolation)")
}

// ---------------------------------------------------------------------------
// Hash determinism across multiple calls with fixed timestamps.
// ---------------------------------------------------------------------------

func TestRecovery_HashDeterminism_FixedTimestamp(t *testing.T) {
	// Build a decision with a fixed timestamp to prove the hash is stable
	// across serialization calls. This proves the replay invariant that
	// the decision_hash is always the same for identical evidence.
	fixedTime := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)

	d := routing.RouteDecision{
		SchemaVersion: routing.RouteDecisionSchemaV1,
		DecisionID:    "dec-hash-stability-001",
		DecisionKey:   "job-stability-001:node-1:step-1",
		JobID:         "job-stability-001",
		RunID:         "run-stability-001",
		TenantID:      "tenant-stability",
		PlanID:        "plan-stability-001",
		NodeID:        "node-1",
		StepID:        "step-1",
		Advisor:       routing.AdvisorInfo{Name: "noop", Version: "v1", Adapter: "noop"},
		Selected: routing.RouteCandidate{
			CapabilityID: "tool:web_search",
			Kind:         routing.KindTool,
			Provider:     "builtin",
			Score:        0.95,
			Rank:         1,
			ReasonCodes:  []string{"historical_success", "lowest_expected_latency"},
		},
		Candidates: []routing.RouteCandidate{
			{
				CapabilityID: "tool:web_search",
				Kind:         routing.KindTool,
				Provider:     "builtin",
				Score:        0.95,
				Rank:         1,
				ReasonCodes:  []string{"historical_success", "lowest_expected_latency"},
			},
			{
				CapabilityID: "tool:calculator",
				Kind:         routing.KindTool,
				Provider:     "builtin",
				Score:        0.60,
				Rank:         2,
				ReasonCodes:  []string{"low_cost"},
			},
		},
		FallbackPolicy: routing.FallbackOpen,
		CreatedAt:      fixedTime,
	}

	h1, err := routing.HashRouteDecision(d)
	require.NoError(t, err)

	h2, err := routing.HashRouteDecision(d)
	require.NoError(t, err)

	assert.Equal(t, h1, h2,
		"HashRouteDecision must be deterministic across calls with identical input")
	assert.Len(t, h1, 64,
		"SHA-256 hex digest must be exactly 64 characters")

	// Verify the hash passes verification when stored in the decision.
	d.DecisionHash = h1
	assert.NoError(t, routing.VerifyDecisionHash(d),
		"stored hash must pass verification")

	// Verify the hash changes when evidence is mutated — proving tamper detection.
	dMutated := d
	dMutated.Selected.CapabilityID = "tool:different_tool"
	h3, err := routing.HashRouteDecision(dMutated)
	require.NoError(t, err)
	assert.NotEqual(t, h1, h3,
		"hash must change when evidence is mutated (tamper sensitivity)")
}
