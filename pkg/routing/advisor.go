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

// Package routing provides model-tier routing and experimental capability routing.
//
// RoutingAdvisor (this file) is an experimental evidence-first boundary for
// capability routing. It is designed for integrations such as WisePick where an
// external or local advisor recommends which capability, tool, agent, model,
// adapter, or workflow path should execute a planned step.
//
// Core invariant: the advisor may only be called during original execution.
// Replay MUST reuse the recorded route decision evidence and MUST NOT call the
// advisor again.
package routing

import (
	"context"
	"time"
)

// CapabilityKind enumerates the types of capability a route decision may select.
type CapabilityKind string

const (
	KindTool     CapabilityKind = "tool"
	KindAgent    CapabilityKind = "agent"
	KindAdapter  CapabilityKind = "adapter"
	KindModel    CapabilityKind = "model"
	KindWorkflow CapabilityKind = "workflow"
)

// FallbackPolicy declares how the runtime should behave when the advisor fails
// or is unavailable.
type FallbackPolicy string

const (
	// FallbackOpen uses a deterministic local fallback when the advisor fails.
	// Must only be used when the local fallback is itself deterministic and
	// evidence-recorded.
	FallbackOpen FallbackPolicy = "fail_open"

	// FallbackClosed fails the job before executing an unadvised route.
	FallbackClosed FallbackPolicy = "fail_closed"

	// FallbackCached reuses a previously recorded decision with matching
	// decision_key when the advisor fails.
	FallbackCached FallbackPolicy = "cached_decision"
)

// RouteCandidate describes one capability that the advisor may select.
type RouteCandidate struct {
	// CapabilityID is the unique identifier of the capability (e.g. "tool:web_search").
	CapabilityID string `json:"capability_id"`

	// Kind is the capability type.
	Kind CapabilityKind `json:"kind"`

	// Provider identifies who owns this capability (e.g. "builtin", "aetheris", "external").
	Provider string `json:"provider"`

	// Score is an optional relevance or confidence score from the advisor.
	Score float64 `json:"score,omitempty"`

	// Rank is an optional 1-based ordering returned by the advisor.
	Rank int `json:"rank,omitempty"`

	// ReasonCodes are optional advisor-provided tokens explaining the score/rank
	// (e.g. "lowest_expected_latency", "historical_success").
	ReasonCodes []string `json:"reason_codes,omitempty"`

	// Metadata carries capability-specific fields (e.g. {"tool_name": "web_search"}).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RouteConstraints captures the runtime constraints that the advisor must
// respect when ranking candidates.
type RouteConstraints struct {
	MaxLatencyMS         int      `json:"max_latency_ms,omitempty"`
	MaxCostUSD           float64  `json:"max_cost_usd,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	RiskLevel            string   `json:"risk_level,omitempty"`
}

// RouteDecisionRequest is the payload sent to a RoutingAdvisor.Decide call.
type RouteDecisionRequest struct {
	// SchemaVersion identifies the request schema for forward compatibility.
	SchemaVersion string `json:"schema_version"`

	// JobID, RunID, TenantID, PlanID, NodeID, StepID identify the execution context.
	JobID    string `json:"job_id"`
	RunID    string `json:"run_id,omitempty"`
	TenantID string `json:"tenant_id"`
	PlanID   string `json:"plan_id,omitempty"`
	NodeID   string `json:"node_id"`
	StepID   string `json:"step_id,omitempty"`

	// DecisionKey is a stable, deterministic key that uniquely identifies this
	// routing decision point within the job. Used for cached_decision lookups.
	DecisionKey string `json:"decision_key"`

	// Goal is a concise description of what the step is trying to accomplish.
	Goal string `json:"goal,omitempty"`

	// Constraints are optional runtime constraints the advisor must respect.
	Constraints RouteConstraints `json:"constraints,omitempty"`

	// Candidates is the ranked list of capabilities the advisor may choose from.
	// The runtime provides this list; the advisor MUST select from it.
	Candidates []RouteCandidate `json:"candidates"`
}

// RouteDecision is the evidence payload stored as route_decision_recorded.
// It is replay-relevant: recovery must read this event and must not call the
// advisor again.
type RouteDecision struct {
	// SchemaVersion identifies the evidence schema.
	SchemaVersion string `json:"schema_version"`

	// DecisionID is a unique, opaque identifier for this decision instance.
	DecisionID string `json:"decision_id"`

	// DecisionKey matches the key in the originating RouteDecisionRequest.
	DecisionKey string `json:"decision_key"`

	// DecisionHash is a deterministic SHA-256 hex digest of the canonical
	// decision payload (excluding itself). Used for tamper detection.
	DecisionHash string `json:"decision_hash"`

	// Job context fields mirror the request.
	JobID    string `json:"job_id"`
	RunID    string `json:"run_id,omitempty"`
	TenantID string `json:"tenant_id"`
	PlanID   string `json:"plan_id,omitempty"`
	NodeID   string `json:"node_id"`
	StepID   string `json:"step_id,omitempty"`

	// Advisor identifies the advisor that produced this decision.
	Advisor AdvisorInfo `json:"advisor"`

	// Selected is the capability chosen by the advisor.
	Selected RouteCandidate `json:"selected"`

	// Candidates is the full candidate set as presented to the advisor.
	Candidates []RouteCandidate `json:"candidates"`

	// FallbackPolicy is the policy declared by the advisor.
	FallbackPolicy FallbackPolicy `json:"fallback_policy"`

	// FallbackReasonCodes records why a fallback was used (empty on normal path).
	FallbackReasonCodes []string `json:"fallback_reason_codes,omitempty"`

	// CreatedAt is the wall-clock time when the decision was recorded.
	CreatedAt time.Time `json:"created_at"`
}

// AdvisorInfo identifies the advisor that produced a route decision.
type AdvisorInfo struct {
	Name           string `json:"name"`
	Version        string `json:"version,omitempty"`
	Adapter        string `json:"adapter,omitempty"`
	AdapterVersion string `json:"adapter_version,omitempty"`
}

// RouteOutcome records the result of executing the selected capability.
// It is used by RoutingAdvisor.RecordOutcome to let the advisor learn from
// execution results.
type RouteOutcome struct {
	// DecisionID references the RouteDecision this outcome is associated with.
	DecisionID string `json:"decision_id"`

	// DecisionKey matches the key in the originating decision.
	DecisionKey string `json:"decision_key"`

	// JobID, TenantID mirror the original decision for routing.
	JobID    string `json:"job_id"`
	TenantID string `json:"tenant_id"`

	// SelectedCapabilityID is the capability that was actually executed.
	SelectedCapabilityID string `json:"selected_capability_id"`

	// Success indicates whether the capability execution succeeded.
	Success bool `json:"success"`

	// LatencyMS is the observed latency in milliseconds.
	LatencyMS int64 `json:"latency_ms,omitempty"`

	// ErrorCode is set when Success is false.
	ErrorCode string `json:"error_code,omitempty"`

	// RecordedAt is the wall-clock time when the outcome was recorded.
	RecordedAt time.Time `json:"recorded_at"`
}

// RoutingAdvisor is the experimental capability routing interface.
//
// Implementations MUST guarantee:
//   - Decide is called only during original execution, never during replay.
//   - RecordOutcome does not affect replay paths.
//   - If FallbackPolicy is FallbackClosed and Decide fails, the job must fail
//     before any unadvised capability is executed.
type RoutingAdvisor interface {
	// Decide selects one capability from req.Candidates for the given execution
	// context. The returned RouteDecision MUST be persisted as a
	// route_decision_recorded event before the selected capability is executed.
	Decide(ctx context.Context, req RouteDecisionRequest) (RouteDecision, error)

	// RecordOutcome sends execution feedback to the advisor. Errors here MUST
	// NOT affect job execution. RecordOutcome MUST NOT be called during replay.
	RecordOutcome(ctx context.Context, outcome RouteOutcome) error
}

// RouteDecisionRequestSchemaV1 is the canonical schema_version for requests.
const RouteDecisionRequestSchemaV1 = "routing.advisor.request.v1alpha1"

// RouteDecisionSchemaV1 is the canonical schema_version for decision evidence.
const RouteDecisionSchemaV1 = "routing.advisor.decision.v1alpha1"

// ValidateFallbackPolicy returns an error if the policy value is not one of the
// three accepted values.
func ValidateFallbackPolicy(p FallbackPolicy) error {
	switch p {
	case FallbackOpen, FallbackClosed, FallbackCached:
		return nil
	default:
		return &InvalidFallbackPolicyError{Policy: string(p)}
	}
}

// InvalidFallbackPolicyError is returned when an unknown fallback policy is used.
type InvalidFallbackPolicyError struct {
	Policy string
}

func (e *InvalidFallbackPolicyError) Error() string {
	return "routing: unknown fallback policy: " + e.Policy
}

// ValidateRouteDecision returns an error if required fields are missing or
// if the selected candidate is not in the candidates list.
func ValidateRouteDecision(d RouteDecision) error {
	if d.SchemaVersion == "" {
		return missingField("schema_version")
	}
	if d.DecisionID == "" {
		return missingField("decision_id")
	}
	if d.DecisionKey == "" {
		return missingField("decision_key")
	}
	if d.DecisionHash == "" {
		return missingField("decision_hash")
	}
	if d.JobID == "" {
		return missingField("job_id")
	}
	if d.TenantID == "" {
		return missingField("tenant_id")
	}
	if d.Advisor.Name == "" {
		return missingField("advisor.name")
	}
	if d.Selected.CapabilityID == "" {
		return missingField("selected.capability_id")
	}
	if d.Selected.Kind == "" {
		return missingField("selected.kind")
	}
	if err := ValidateFallbackPolicy(d.FallbackPolicy); err != nil {
		return err
	}
	if d.CreatedAt.IsZero() {
		return missingField("created_at")
	}

	// Selected candidate must appear in the candidate list.
	if len(d.Candidates) > 0 {
		found := false
		for _, c := range d.Candidates {
			if c.CapabilityID == d.Selected.CapabilityID {
				found = true
				break
			}
		}
		if !found {
			return &SelectedNotInCandidatesError{CapabilityID: d.Selected.CapabilityID}
		}
	}

	return nil
}

// SelectedNotInCandidatesError is returned when the selected capability does not
// appear in the provided candidate list.
type SelectedNotInCandidatesError struct {
	CapabilityID string
}

func (e *SelectedNotInCandidatesError) Error() string {
	return "routing: selected capability_id " + e.CapabilityID + " is not in the candidate list"
}

// missingField is a convenience constructor for missing required field errors.
func missingField(name string) error {
	return &MissingFieldError{Field: name}
}

// MissingFieldError is returned when a required field is absent.
type MissingFieldError struct {
	Field string
}

func (e *MissingFieldError) Error() string {
	return "routing: required field missing: " + e.Field
}
