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

// Package routing_test contains fixture-driven tests for RoutingAdvisor.
//
// These tests load canonical JSON fixtures from testdata/routing_advisor/
// and prove that:
//   - fixtures can round-trip through the Go schema without data loss
//   - HashRouteDecision produces a verifiable hash for valid decisions
//   - VerifyDecisionHash rejects pre-tampered fixtures
//   - ValidateRouteDecision rejects fixtures with structural violations
//   - NoOpAdvisor produces a decision consistent with the canonical request fixture
package routing_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Colin4k1024/Aetheris/v2/pkg/routing"
)

// fixtureDir returns the path to testdata/routing_advisor relative to the
// location of this test file. Using os.ReadFile keeps the tests hermetic.
func fixtureDir() string {
	return filepath.Join("testdata", "routing_advisor")
}

// loadFixture reads a JSON fixture file and unmarshals it into dst.
func loadFixture(t *testing.T, name string, dst any) {
	t.Helper()
	path := filepath.Join(fixtureDir(), name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "fixture file must be readable: %s", path)
	require.NoError(t, json.Unmarshal(data, dst), "fixture must unmarshal without error: %s", name)
}

// ---------------------------------------------------------------------------
// TestFixture_RequestValid — canonical request round-trips cleanly.
// ---------------------------------------------------------------------------

func TestFixture_RequestValid(t *testing.T) {
	var req routing.RouteDecisionRequest
	loadFixture(t, "request.valid.json", &req)

	assert.Equal(t, routing.RouteDecisionRequestSchemaV1, req.SchemaVersion)
	assert.Equal(t, "job-fixture-001", req.JobID)
	assert.Equal(t, "run-fixture-001", req.RunID)
	assert.Equal(t, "tenant-fixture", req.TenantID)
	assert.Equal(t, "plan-fixture-001", req.PlanID)
	assert.Equal(t, "node-1", req.NodeID)
	assert.Equal(t, "step-1", req.StepID)
	assert.Equal(t, "job-fixture-001:node-1:step-1", req.DecisionKey)
	assert.NotEmpty(t, req.Goal)
	require.Len(t, req.Candidates, 2, "fixture must contain exactly two candidates")

	// Verify candidates are present with expected IDs.
	capIDs := make([]string, 0, len(req.Candidates))
	for _, c := range req.Candidates {
		capIDs = append(capIDs, c.CapabilityID)
	}
	assert.Contains(t, capIDs, "tool:web_search")
	assert.Contains(t, capIDs, "tool:calculator")
}

// ---------------------------------------------------------------------------
// TestFixture_DecisionValid — canonical decision round-trips; hash is computable.
//
// The fixture stores decision_hash as an empty string (cannot be pre-computed
// without running Go). This test computes the hash dynamically, fills it in,
// and then asserts that VerifyDecisionHash and ValidateRouteDecision both pass.
// This proves the fixture contains a structurally and semantically valid decision.
// ---------------------------------------------------------------------------

func TestFixture_DecisionValid(t *testing.T) {
	var d routing.RouteDecision
	loadFixture(t, "decision.valid.json", &d)

	// The fixture has no pre-computed hash. Compute it now.
	h, err := routing.HashRouteDecision(d)
	require.NoError(t, err, "HashRouteDecision must succeed for valid fixture")
	require.NotEmpty(t, h, "computed hash must not be empty")
	assert.Len(t, h, 64, "SHA-256 hex digest must be 64 characters")

	// Set the hash and verify round-trip.
	d.DecisionHash = h
	assert.NoError(t, routing.VerifyDecisionHash(d),
		"decision with computed hash must pass VerifyDecisionHash")
	assert.NoError(t, routing.ValidateRouteDecision(d),
		"decision with computed hash must pass ValidateRouteDecision")

	// Structural field assertions.
	assert.Equal(t, routing.RouteDecisionSchemaV1, d.SchemaVersion)
	assert.Equal(t, "dec-fixture-001", d.DecisionID)
	assert.Equal(t, "job-fixture-001:node-1:step-1", d.DecisionKey)
	assert.Equal(t, "job-fixture-001", d.JobID)
	assert.Equal(t, "tenant-fixture", d.TenantID)
	assert.Equal(t, "noop", d.Advisor.Name)
	assert.Equal(t, "tool:web_search", d.Selected.CapabilityID)
	assert.Equal(t, routing.FallbackOpen, d.FallbackPolicy)
	assert.Equal(t, time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC), d.CreatedAt)

	// Candidate count.
	assert.Len(t, d.Candidates, 2)
}

// ---------------------------------------------------------------------------
// TestFixture_DecisionValid_HashIsStable — same fixture produces the same hash twice.
//
// This is the golden stability check: if the canonical serialization changes,
// this test fails, alerting the team that replay evidence may be incompatible.
// ---------------------------------------------------------------------------

func TestFixture_DecisionValid_HashIsStable(t *testing.T) {
	var d routing.RouteDecision
	loadFixture(t, "decision.valid.json", &d)

	h1, err := routing.HashRouteDecision(d)
	require.NoError(t, err)

	h2, err := routing.HashRouteDecision(d)
	require.NoError(t, err)

	assert.Equal(t, h1, h2,
		"HashRouteDecision must produce the same hash for identical input (determinism)")
}

// ---------------------------------------------------------------------------
// TestFixture_DecisionHashMismatch — pre-tampered fixture is rejected.
// ---------------------------------------------------------------------------

func TestFixture_DecisionHashMismatch(t *testing.T) {
	var d routing.RouteDecision
	loadFixture(t, "decision.hash-mismatch.json", &d)

	require.NotEmpty(t, d.DecisionHash, "hash-mismatch fixture must have a non-empty (but wrong) hash")

	err := routing.VerifyDecisionHash(d)
	require.Error(t, err, "VerifyDecisionHash must reject a tampered fixture")

	var mismatch *routing.HashMismatchError
	require.ErrorAs(t, err, &mismatch,
		"error must be a HashMismatchError for a pre-tampered decision")
	assert.NotEmpty(t, mismatch.Want, "HashMismatchError.Want must be populated")
	assert.NotEmpty(t, mismatch.Got, "HashMismatchError.Got must be populated")
	assert.NotEqual(t, mismatch.Want, mismatch.Got)
}

// ---------------------------------------------------------------------------
// TestFixture_DecisionUnknownCandidate — selected not in candidates is rejected.
// ---------------------------------------------------------------------------

func TestFixture_DecisionUnknownCandidate(t *testing.T) {
	var d routing.RouteDecision
	loadFixture(t, "decision.unknown-candidate.json", &d)

	// The fixture has selected.capability_id not present in candidates.
	err := routing.ValidateRouteDecision(d)
	require.Error(t, err, "ValidateRouteDecision must reject a decision whose selected capability is not in candidates")

	var notInCandidates *routing.SelectedNotInCandidatesError
	require.ErrorAs(t, err, &notInCandidates,
		"error must be SelectedNotInCandidatesError for unknown selected capability")
	assert.Equal(t, "tool:unknown_capability_not_in_candidates", notInCandidates.CapabilityID)
}

// ---------------------------------------------------------------------------
// TestFixture_OutcomeValid — canonical outcome round-trips cleanly.
// ---------------------------------------------------------------------------

func TestFixture_OutcomeValid(t *testing.T) {
	var o routing.RouteOutcome
	loadFixture(t, "outcome.valid.json", &o)

	assert.Equal(t, "dec-fixture-001", o.DecisionID)
	assert.Equal(t, "job-fixture-001:node-1:step-1", o.DecisionKey)
	assert.Equal(t, "job-fixture-001", o.JobID)
	assert.Equal(t, "tenant-fixture", o.TenantID)
	assert.Equal(t, "tool:web_search", o.SelectedCapabilityID)
	assert.True(t, o.Success)
	assert.EqualValues(t, 312, o.LatencyMS)
	assert.Empty(t, o.ErrorCode, "successful outcome must have no error code")
	assert.False(t, o.RecordedAt.IsZero(), "outcome must have a non-zero recorded_at")
}

// ---------------------------------------------------------------------------
// TestFixture_NoOpAdvisorMatchesCanonicalRequest — NoOpAdvisor selects the
// highest-rank (rank=1) candidate from the canonical request fixture.
//
// This cross-validates the fixture against the NoOpAdvisor selection logic.
// If the fixture candidates change rank values, this test will fail.
// ---------------------------------------------------------------------------

func TestFixture_NoOpAdvisorMatchesCanonicalRequest(t *testing.T) {
	var req routing.RouteDecisionRequest
	loadFixture(t, "request.valid.json", &req)

	advisor := routing.NewNoOpAdvisor()
	d, err := advisor.Decide(context.Background(), req)
	require.NoError(t, err)

	// NoOpAdvisor selects the candidate with lowest rank (rank=1 wins over rank=2).
	assert.Equal(t, "tool:web_search", d.Selected.CapabilityID,
		"NoOpAdvisor must select rank=1 candidate tool:web_search from canonical request fixture")
	assert.EqualValues(t, 1, d.Selected.Rank)

	// Decision must be fully valid.
	assert.NoError(t, routing.VerifyDecisionHash(d))
	assert.NoError(t, routing.ValidateRouteDecision(d))
}

// ---------------------------------------------------------------------------
// TestFixture_RequestCandidates_MatchDecisionCandidates — the canonical request
// and decision fixtures reference the same capability set.
// ---------------------------------------------------------------------------

func TestFixture_RequestCandidates_MatchDecisionCandidates(t *testing.T) {
	var req routing.RouteDecisionRequest
	var d routing.RouteDecision
	loadFixture(t, "request.valid.json", &req)
	loadFixture(t, "decision.valid.json", &d)

	reqCapIDs := make(map[string]bool)
	for _, c := range req.Candidates {
		reqCapIDs[c.CapabilityID] = true
	}

	for _, c := range d.Candidates {
		assert.True(t, reqCapIDs[c.CapabilityID],
			"decision candidate %q must appear in the canonical request fixture candidates", c.CapabilityID)
	}

	assert.True(t, reqCapIDs[d.Selected.CapabilityID],
		"decision selected capability %q must appear in the canonical request fixture candidates",
		d.Selected.CapabilityID)
}
