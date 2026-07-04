package routing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

// ComputeDecisionHash computes a deterministic SHA256 hash over a RouteDecision.
//
// Rules (from routing-advisor-contract.md "Deterministic Hash Rules"):
//  1. DecisionHash field is excluded from the hash input.
//  2. ReasonCodes are sorted alphabetically.
//  3. Marshal using encoding/json (no custom encoder).
//  4. Nil metadata serializes as null; empty reason_codes serializes as [].
//
// The hash is returned as a lowercase hex string prefixed with "sha256:".
func ComputeDecisionHash(d RouteDecision) string {
	canonical := canonicalDecision{
		SelectedID:  d.SelectedID,
		ReasonCodes: sortedCopy(d.ReasonCodes),
		Timestamp:   d.Timestamp,
		Metadata:    d.Metadata,
	}

	b, _ := json.Marshal(canonical)

	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

// VerifyDecisionHash recomputes the hash and compares it to the recorded value.
func VerifyDecisionHash(d RouteDecision) bool {
	expected := ComputeDecisionHash(d)
	return expected == d.DecisionHash
}

// canonicalDecision is the intermediate struct used for deterministic serialization.
// DecisionHash is intentionally omitted.
type canonicalDecision struct {
	SelectedID  string         `json:"selected_id"`
	ReasonCodes []string       `json:"reason_codes"`
	Timestamp   time.Time      `json:"timestamp"`
	Metadata    map[string]any `json:"metadata"`
}

// sortedCopy returns a sorted copy of the string slice.
// nil input returns nil (serialized as null by encoding/json).
func sortedCopy(s []string) []string {
	if s == nil {
		return nil
	}
	cp := make([]string, len(s))
	copy(cp, s)
	sort.Strings(cp)
	return cp
}
