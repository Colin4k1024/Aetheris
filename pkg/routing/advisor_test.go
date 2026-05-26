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

package routing_test

import (
	"context"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/pkg/routing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeRequest(candidates ...routing.RouteCandidate) routing.RouteDecisionRequest {
	return routing.RouteDecisionRequest{
		SchemaVersion: routing.RouteDecisionRequestSchemaV1,
		JobID:         "job-1",
		TenantID:      "tenant-A",
		NodeID:        "node-1",
		DecisionKey:   "job-1:node-1:step-1",
		Candidates:    candidates,
	}
}

func tool(id string, rank int) routing.RouteCandidate {
	return routing.RouteCandidate{CapabilityID: id, Kind: routing.KindTool, Provider: "builtin", Rank: rank}
}

// ---------------------------------------------------------------------------
// ValidateFallbackPolicy
// ---------------------------------------------------------------------------

func TestValidateFallbackPolicy_Valid(t *testing.T) {
	for _, p := range []routing.FallbackPolicy{routing.FallbackOpen, routing.FallbackClosed, routing.FallbackCached} {
		if err := routing.ValidateFallbackPolicy(p); err != nil {
			t.Errorf("expected nil for %q, got %v", p, err)
		}
	}
}

func TestValidateFallbackPolicy_Invalid(t *testing.T) {
	err := routing.ValidateFallbackPolicy(routing.FallbackPolicy("bad_policy"))
	if err == nil {
		t.Fatal("expected error for unknown policy")
	}
}

// ---------------------------------------------------------------------------
// ValidateRouteDecision
// ---------------------------------------------------------------------------

func validDecision() routing.RouteDecision {
	return routing.RouteDecision{
		SchemaVersion:  routing.RouteDecisionSchemaV1,
		DecisionID:     "dec-1",
		DecisionKey:    "job-1:node-1:step-1",
		DecisionHash:   "deadbeef",
		JobID:          "job-1",
		TenantID:       "tenant-A",
		NodeID:         "node-1",
		Advisor:        routing.AdvisorInfo{Name: "noop"},
		Selected:       routing.RouteCandidate{CapabilityID: "tool:web_search", Kind: routing.KindTool},
		Candidates:     []routing.RouteCandidate{{CapabilityID: "tool:web_search", Kind: routing.KindTool}},
		FallbackPolicy: routing.FallbackOpen,
		CreatedAt:      time.Now(),
	}
}

func TestValidateRouteDecision_Valid(t *testing.T) {
	if err := routing.ValidateRouteDecision(validDecision()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRouteDecision_MissingSchemaVersion(t *testing.T) {
	d := validDecision()
	d.SchemaVersion = ""
	if err := routing.ValidateRouteDecision(d); err == nil {
		t.Fatal("expected error for missing schema_version")
	}
}

func TestValidateRouteDecision_MissingDecisionHash(t *testing.T) {
	d := validDecision()
	d.DecisionHash = ""
	if err := routing.ValidateRouteDecision(d); err == nil {
		t.Fatal("expected error for missing decision_hash")
	}
}

func TestValidateRouteDecision_MissingAdvisorName(t *testing.T) {
	d := validDecision()
	d.Advisor.Name = ""
	if err := routing.ValidateRouteDecision(d); err == nil {
		t.Fatal("expected error for missing advisor.name")
	}
}

func TestValidateRouteDecision_SelectedNotInCandidates(t *testing.T) {
	d := validDecision()
	d.Selected = routing.RouteCandidate{CapabilityID: "tool:unknown", Kind: routing.KindTool}
	if err := routing.ValidateRouteDecision(d); err == nil {
		t.Fatal("expected error when selected not in candidates")
	}
}

func TestValidateRouteDecision_InvalidFallbackPolicy(t *testing.T) {
	d := validDecision()
	d.FallbackPolicy = "nonsense"
	if err := routing.ValidateRouteDecision(d); err == nil {
		t.Fatal("expected error for invalid fallback policy")
	}
}

// ---------------------------------------------------------------------------
// HashRouteDecision / VerifyDecisionHash
// ---------------------------------------------------------------------------

func TestHashRouteDecision_Deterministic(t *testing.T) {
	d := validDecision()
	d.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	h1, err := routing.HashRouteDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := routing.HashRouteDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("hashes differ across two calls: %s vs %s", h1, h2)
	}
}

func TestHashRouteDecision_ChangeSensitive(t *testing.T) {
	d := validDecision()
	d.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	h1, _ := routing.HashRouteDecision(d)

	d.Selected.CapabilityID = "tool:other_tool"
	h2, _ := routing.HashRouteDecision(d)

	if h1 == h2 {
		t.Error("hash should change when selected capability changes")
	}
}

func TestHashRouteDecision_ReasonCodesOrderIndependent(t *testing.T) {
	base := validDecision()
	base.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	base.Candidates = []routing.RouteCandidate{
		{CapabilityID: "tool:web_search", Kind: routing.KindTool, ReasonCodes: []string{"fast", "cheap"}},
	}
	base.Selected = base.Candidates[0]

	reordered := base
	reordered.Candidates = []routing.RouteCandidate{
		{CapabilityID: "tool:web_search", Kind: routing.KindTool, ReasonCodes: []string{"cheap", "fast"}},
	}
	reordered.Selected = reordered.Candidates[0]

	h1, _ := routing.HashRouteDecision(base)
	h2, _ := routing.HashRouteDecision(reordered)
	if h1 != h2 {
		t.Errorf("hash should be invariant to reason_codes ordering: %s vs %s", h1, h2)
	}
}

func TestVerifyDecisionHash_Pass(t *testing.T) {
	d := validDecision()
	d.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	h, err := routing.HashRouteDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	d.DecisionHash = h

	if err := routing.VerifyDecisionHash(d); err != nil {
		t.Fatalf("unexpected verify error: %v", err)
	}
}

func TestVerifyDecisionHash_Mismatch(t *testing.T) {
	d := validDecision()
	d.DecisionHash = "not-a-real-hash"
	if err := routing.VerifyDecisionHash(d); err == nil {
		t.Fatal("expected hash mismatch error")
	}
}

// ---------------------------------------------------------------------------
// NoOpAdvisor
// ---------------------------------------------------------------------------

func TestNoOpAdvisor_SelectsLowestRank(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	req := makeRequest(
		tool("tool:b", 2),
		tool("tool:a", 1),
		tool("tool:c", 3),
	)
	dec, err := advisor.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Selected.CapabilityID != "tool:a" {
		t.Errorf("expected tool:a (rank 1), got %s", dec.Selected.CapabilityID)
	}
}

func TestNoOpAdvisor_TieBreakByCapabilityID(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	req := makeRequest(
		tool("tool:z", 1),
		tool("tool:a", 1),
	)
	dec, err := advisor.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Selected.CapabilityID != "tool:a" {
		t.Errorf("expected tool:a (alpha tie-break), got %s", dec.Selected.CapabilityID)
	}
}

func TestNoOpAdvisor_NoRankUsesScore(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	req := makeRequest(
		routing.RouteCandidate{CapabilityID: "tool:low", Kind: routing.KindTool, Score: 0.3},
		routing.RouteCandidate{CapabilityID: "tool:high", Kind: routing.KindTool, Score: 0.9},
	)
	dec, err := advisor.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Selected.CapabilityID != "tool:high" {
		t.Errorf("expected tool:high (highest score), got %s", dec.Selected.CapabilityID)
	}
}

func TestNoOpAdvisor_NoCandidatesReturnsError(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	req := makeRequest()
	_, err := advisor.Decide(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
}

func TestNoOpAdvisor_DecisionHashVerifiable(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	req := makeRequest(tool("tool:web_search", 1))
	dec, err := advisor.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if err := routing.VerifyDecisionHash(dec); err != nil {
		t.Fatalf("noop decision hash failed verification: %v", err)
	}
}

func TestNoOpAdvisor_DecisionPassesValidation(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	req := makeRequest(tool("tool:web_search", 1))
	dec, err := advisor.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if err := routing.ValidateRouteDecision(dec); err != nil {
		t.Fatalf("noop decision failed structural validation: %v", err)
	}
}

func TestNoOpAdvisor_RecordOutcomeIsNoop(t *testing.T) {
	advisor := routing.NewNoOpAdvisor()
	err := advisor.RecordOutcome(context.Background(), routing.RouteOutcome{
		DecisionID:           "dec-1",
		DecisionKey:          "k",
		JobID:                "j",
		TenantID:             "t",
		SelectedCapabilityID: "tool:x",
		Success:              true,
		RecordedAt:           time.Now(),
	})
	if err != nil {
		t.Fatalf("RecordOutcome should never fail: %v", err)
	}
}

func TestNoOpAdvisor_ImplementsInterface(t *testing.T) {
	var _ routing.RoutingAdvisor = routing.NewNoOpAdvisor()
}
