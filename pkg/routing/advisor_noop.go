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

package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NoOpAdvisor is a deterministic, local RoutingAdvisor implementation that
// selects the first candidate by rank (then by capability_id for stability) and
// requires no external calls.
//
// It is intended as:
//   - The default advisor when no external advisor is configured.
//   - A fallback during fail_open scenarios.
//   - A baseline for replay safety tests.
//
// NoOpAdvisor is replay-safe by definition because it is deterministic; all
// decisions are evidence-recorded exactly as with any other advisor.
type NoOpAdvisor struct {
	// Name overrides the advisor name in produced decisions. Defaults to "noop".
	Name string

	// Version overrides the advisor version. Defaults to "v1".
	Version string

	// FallbackPolicy is embedded in every produced RouteDecision.
	// Defaults to FallbackOpen.
	FallbackPolicy FallbackPolicy
}

// NewNoOpAdvisor returns a NoOpAdvisor with default settings.
func NewNoOpAdvisor() *NoOpAdvisor {
	return &NoOpAdvisor{
		Name:           "noop",
		Version:        "v1",
		FallbackPolicy: FallbackOpen,
	}
}

// Decide implements RoutingAdvisor.Decide. It selects the candidate with the
// lowest rank value (i.e. rank == 1 wins); ties are broken alphabetically by
// capability_id to ensure determinism.
//
// If all candidates have rank == 0 (unset), the first element is selected.
func (a *NoOpAdvisor) Decide(ctx context.Context, req RouteDecisionRequest) (RouteDecision, error) {
	if len(req.Candidates) == 0 {
		return RouteDecision{}, fmt.Errorf("routing: noop advisor: no candidates provided for decision_key=%s", req.DecisionKey)
	}

	selected := selectBestCandidate(req.Candidates)

	name := a.Name
	if name == "" {
		name = "noop"
	}
	version := a.Version
	if version == "" {
		version = "v1"
	}
	policy := a.FallbackPolicy
	if err := ValidateFallbackPolicy(policy); err != nil {
		policy = FallbackOpen
	}

	now := time.Now().UTC()
	decisionID := uuid.New().String()

	d := RouteDecision{
		SchemaVersion:  RouteDecisionSchemaV1,
		DecisionID:     decisionID,
		DecisionKey:    req.DecisionKey,
		JobID:          req.JobID,
		RunID:          req.RunID,
		TenantID:       req.TenantID,
		PlanID:         req.PlanID,
		NodeID:         req.NodeID,
		StepID:         req.StepID,
		Advisor:        AdvisorInfo{Name: name, Version: version, Adapter: "noop"},
		Selected:       selected,
		Candidates:     req.Candidates,
		FallbackPolicy: policy,
		CreatedAt:      now,
	}

	hash, err := HashRouteDecision(d)
	if err != nil {
		return RouteDecision{}, fmt.Errorf("routing: noop advisor: failed to hash decision: %w", err)
	}
	d.DecisionHash = hash

	return d, nil
}

// RecordOutcome implements RoutingAdvisor.RecordOutcome. NoOpAdvisor discards
// outcomes; errors are never returned.
func (a *NoOpAdvisor) RecordOutcome(_ context.Context, _ RouteOutcome) error {
	return nil
}

// selectBestCandidate chooses the candidate with the lowest non-zero rank.
// If all ranks are zero, returns candidates[0]. Ties are broken alphabetically
// by capability_id.
func selectBestCandidate(candidates []RouteCandidate) RouteCandidate {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if isBetter(c, best) {
			best = c
		}
	}
	return best
}

// isBetter returns true if candidate a should be preferred over b.
func isBetter(a, b RouteCandidate) bool {
	aRanked := a.Rank > 0
	bRanked := b.Rank > 0

	switch {
	case aRanked && bRanked:
		if a.Rank != b.Rank {
			return a.Rank < b.Rank
		}
		return a.CapabilityID < b.CapabilityID
	case aRanked && !bRanked:
		return true
	case !aRanked && bRanked:
		return false
	default:
		// Neither ranked — use score descending, then capability_id for determinism.
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return a.CapabilityID < b.CapabilityID
	}
}
