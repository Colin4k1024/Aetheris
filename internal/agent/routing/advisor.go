package routing

import (
	"context"
	"time"
)

// RoutingAdvisor selects capabilities for planned steps.
// Every decision is recorded as durable evidence before execution.
// During replay, the advisor is NEVER called — recorded decisions are reused.
type RoutingAdvisor interface {
	Decide(ctx context.Context, req RouteDecisionRequest) (RouteDecision, error)
	RecordOutcome(ctx context.Context, outcome RouteOutcome) error
}

// RouteDecisionRequest contains candidates and constraints for a routing decision.
type RouteDecisionRequest struct {
	JobID       string           `json:"job_id"`
	StepID      string           `json:"step_id"`
	Candidates  []Capability     `json:"candidates"`
	Constraints RouteConstraints `json:"constraints"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
}

// Capability represents a selectable execution target.
type Capability struct {
	ID           string         `json:"id"`
	Type         CapabilityType `json:"type"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	CostEstimate float64        `json:"cost_estimate,omitempty"`
	LatencyEst   time.Duration  `json:"latency_estimate,omitempty"`
}

// CapabilityType enumerates the types of capabilities.
type CapabilityType string

const (
	CapabilityTool     CapabilityType = "tool"
	CapabilityAgent    CapabilityType = "agent"
	CapabilityModel    CapabilityType = "model"
	CapabilityAdapter  CapabilityType = "adapter"
	CapabilityWorkflow CapabilityType = "workflow"
)

// RouteConstraints defines constraints for routing decisions.
type RouteConstraints struct {
	MaxCost      float64       `json:"max_cost,omitempty"`
	MaxLatency   time.Duration `json:"max_latency,omitempty"`
	RequiredTags []string      `json:"required_tags,omitempty"`
	ExcludedIDs  []string      `json:"excluded_ids,omitempty"`
}

// RouteDecision is the advisor's recommendation.
type RouteDecision struct {
	SelectedID   string         `json:"selected_id"`
	ReasonCodes  []string       `json:"reason_codes"`
	DecisionHash string         `json:"decision_hash"`
	Timestamp    time.Time      `json:"timestamp"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// RouteOutcome records the result of executing a routed capability.
type RouteOutcome struct {
	JobID      string        `json:"job_id"`
	StepID     string        `json:"step_id"`
	SelectedID string        `json:"selected_id"`
	Success    bool          `json:"success"`
	Duration   time.Duration `json:"duration"`
	Error      string        `json:"error,omitempty"`
	TokensUsed int           `json:"tokens_used,omitempty"`
}
