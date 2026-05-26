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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// canonicalDecision is an internal, stable representation used for hashing.
// It intentionally omits decision_hash (to avoid circular dependency) and
// normalises all variable-length collections to sorted order.
type canonicalDecision struct {
	SchemaVersion  string               `json:"schema_version"`
	DecisionID     string               `json:"decision_id"`
	DecisionKey    string               `json:"decision_key"`
	JobID          string               `json:"job_id"`
	RunID          string               `json:"run_id"`
	TenantID       string               `json:"tenant_id"`
	PlanID         string               `json:"plan_id"`
	NodeID         string               `json:"node_id"`
	StepID         string               `json:"step_id"`
	Advisor        AdvisorInfo          `json:"advisor"`
	Selected       canonicalCandidate   `json:"selected"`
	Candidates     []canonicalCandidate `json:"candidates"`
	FallbackPolicy FallbackPolicy       `json:"fallback_policy"`
	CreatedAt      time.Time            `json:"created_at"`
}

// canonicalCandidate is a stable representation of RouteCandidate for hashing.
// reason_codes is sorted to ensure hash determinism regardless of input order.
type canonicalCandidate struct {
	CapabilityID string         `json:"capability_id"`
	Kind         CapabilityKind `json:"kind"`
	Provider     string         `json:"provider"`
	Score        float64        `json:"score"`
	Rank         int            `json:"rank"`
	ReasonCodes  []string       `json:"reason_codes"`
	Metadata     map[string]any `json:"metadata"`
}

// toCanonicalCandidate converts a RouteCandidate to the stable form used for
// hashing. reason_codes are sorted to eliminate order dependency.
func toCanonicalCandidate(c RouteCandidate) canonicalCandidate {
	codes := make([]string, len(c.ReasonCodes))
	copy(codes, c.ReasonCodes)
	sort.Strings(codes)

	return canonicalCandidate{
		CapabilityID: c.CapabilityID,
		Kind:         c.Kind,
		Provider:     c.Provider,
		Score:        c.Score,
		Rank:         c.Rank,
		ReasonCodes:  codes,
		Metadata:     c.Metadata,
	}
}

// HashRouteDecision computes a deterministic SHA-256 hex digest over the
// canonical form of d (excluding d.DecisionHash). The result should be stored
// in d.DecisionHash before persisting the event.
//
// Determinism guarantee: given identical input fields, this function always
// produces the same hash across Go versions by serialising to a stable JSON
// form with sorted keys.
func HashRouteDecision(d RouteDecision) (string, error) {
	candidates := make([]canonicalCandidate, len(d.Candidates))
	for i, c := range d.Candidates {
		candidates[i] = toCanonicalCandidate(c)
	}
	// Sort candidates by capability_id for stable ordering.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CapabilityID < candidates[j].CapabilityID
	})

	canon := canonicalDecision{
		SchemaVersion:  d.SchemaVersion,
		DecisionID:     d.DecisionID,
		DecisionKey:    d.DecisionKey,
		JobID:          d.JobID,
		RunID:          d.RunID,
		TenantID:       d.TenantID,
		PlanID:         d.PlanID,
		NodeID:         d.NodeID,
		StepID:         d.StepID,
		Advisor:        d.Advisor,
		Selected:       toCanonicalCandidate(d.Selected),
		Candidates:     candidates,
		FallbackPolicy: d.FallbackPolicy,
		CreatedAt:      d.CreatedAt,
	}

	data, err := json.Marshal(canon)
	if err != nil {
		return "", fmt.Errorf("routing: failed to marshal canonical decision: %w", err)
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// VerifyDecisionHash recomputes the hash of d and returns an error if it does
// not match d.DecisionHash. Used during replay to detect tampered evidence.
func VerifyDecisionHash(d RouteDecision) error {
	want := d.DecisionHash
	if want == "" {
		return &MissingFieldError{Field: "decision_hash"}
	}

	got, err := HashRouteDecision(d)
	if err != nil {
		return fmt.Errorf("routing: hash computation failed: %w", err)
	}
	if got != want {
		return &HashMismatchError{Want: want, Got: got}
	}
	return nil
}

// HashMismatchError is returned when the recomputed hash does not match the
// stored hash in a RouteDecision.
type HashMismatchError struct {
	Want string
	Got  string
}

func (e *HashMismatchError) Error() string {
	return fmt.Sprintf("routing: decision hash mismatch: stored=%s computed=%s", e.Want, e.Got)
}
