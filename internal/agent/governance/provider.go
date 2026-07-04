// Package governance defines the integration contract between Aetheris (L1)
// and hermesx (L3). It contains two interface families:
//
//   - GovernanceProvider: what Aetheris expects from L3 (policy, compliance, audit)
//   - ExecutionProvider: what Aetheris exposes to L2/L3 (job submission, state query, events)
//
// See docs/architecture/hermesx-integration-contract.md for the full contract.
// See docs/adr/ADR-0002-aetheris-four-layer-architecture.md for architectural context.
package governance

import (
	"context"
	"io"
	"time"
)

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

// PolicyDecision is the outcome of a policy evaluation.
type PolicyDecision string

const (
	PolicyAllow PolicyDecision = "allow"
	PolicyDeny  PolicyDecision = "deny"
	PolicyAudit PolicyDecision = "audit"
)

// AuditLevel controls how much detail is captured in governance events.
type AuditLevel string

const (
	AuditBasic    AuditLevel = "basic"
	AuditDetailed AuditLevel = "detailed"
	AuditForensic AuditLevel = "forensic"
)

// FallbackPolicy controls behavior when hermesx is unreachable.
type FallbackPolicy string

const (
	FailOpen   FallbackPolicy = "fail_open"
	FailClosed FallbackPolicy = "fail_closed"
)

// EventType enumerates all governance events Aetheris reports to hermesx.
type EventType string

const (
	EventJobCreated              EventType = "job.created"
	EventJobCompleted            EventType = "job.completed"
	EventJobFailed               EventType = "job.failed"
	EventStepStarted             EventType = "step.started"
	EventStepCompleted           EventType = "step.completed"
	EventStepFailed              EventType = "step.failed"
	EventRouteDecisionRecorded   EventType = "route.decision_recorded"
	EventEvidenceExported        EventType = "evidence.exported"
	EventPolicyViolationDetected EventType = "policy.violation_detected"
	EventPolicyEvaluationTimeout EventType = "policy.evaluation_timeout"
)

// ActionClass categorizes actions for policy evaluation.
type ActionClass string

const (
	ActionToolInvocation ActionClass = "tool_invocation"
	ActionLLMGeneration  ActionClass = "llm_generation"
	ActionWorkflowExec   ActionClass = "workflow_execution"
	ActionDataExport     ActionClass = "data_export"
	ActionExternalAPI    ActionClass = "external_api_call"
)

// ---------------------------------------------------------------------------
// GovernanceEvent — L1 → L3 event reporting
// ---------------------------------------------------------------------------

// GovernanceEvent is a single event that Aetheris reports to hermesx.
type GovernanceEvent struct {
	EventID      string         `json:"event_id"`
	EventType    EventType      `json:"event_type"`
	Timestamp    time.Time      `json:"timestamp"`
	JobID        string         `json:"job_id"`
	StepID       string         `json:"step_id,omitempty"`
	TenantID     string         `json:"tenant_id"`
	EvidenceHash string         `json:"evidence_hash,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Policy Evaluation — L3 → L1 (synchronous, on execution path)
// ---------------------------------------------------------------------------

// PolicyAction describes the action being evaluated.
type PolicyAction struct {
	Type         ActionClass    `json:"type"`
	CapabilityID string         `json:"capability_id"`
	Target       string         `json:"target,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
}

// PolicyContext carries ambient context for policy evaluation.
type PolicyContext struct {
	UserID              string `json:"user_id,omitempty"`
	SessionID           string `json:"session_id,omitempty"`
	ParentJobID         string `json:"parent_job_id,omitempty"`
	RoutingDecisionHash string `json:"routing_decision_hash,omitempty"`
}

// PolicyEvaluationRequest is sent by Aetheris to hermesx before executing a step.
type PolicyEvaluationRequest struct {
	RequestID string        `json:"request_id"`
	TenantID  string        `json:"tenant_id"`
	JobID     string        `json:"job_id"`
	StepID    string        `json:"step_id"`
	Action    PolicyAction  `json:"action"`
	Context   PolicyContext `json:"context"`
}

// ComplianceConstraints are returned by hermesx and enforced by Aetheris.
type ComplianceConstraints struct {
	DataResidency       string            `json:"data_residency,omitempty"`
	RetentionDays       int               `json:"retention_days,omitempty"`
	PIIMasking          *PIIMaskingConfig `json:"pii_masking,omitempty"`
	AuditLevel          AuditLevel        `json:"audit_level,omitempty"`
	AllowedTools        []string          `json:"allowed_tools,omitempty"`
	DeniedTools         []string          `json:"denied_tools,omitempty"`
	MaxJobDurationHours int               `json:"max_job_duration_hours,omitempty"`
	RequireApprovalFor  []string          `json:"require_approval_for,omitempty"`
}

// PIIMaskingConfig defines how personally identifiable information is masked.
type PIIMaskingConfig struct {
	Enabled      bool     `json:"enabled"`
	Patterns     []string `json:"patterns,omitempty"`
	MaskStrategy string   `json:"mask_strategy,omitempty"`
}

// PolicyEvaluationResponse is the hermesx response to a policy evaluation request.
type PolicyEvaluationResponse struct {
	RequestID     string                 `json:"request_id"`
	Decision      PolicyDecision         `json:"decision"`
	Reason        string                 `json:"reason,omitempty"`
	Constraints   *ComplianceConstraints `json:"constraints,omitempty"`
	EvaluatedAt   time.Time              `json:"evaluated_at"`
	PolicyVersion string                 `json:"policy_version,omitempty"`
}

// ---------------------------------------------------------------------------
// Audit Query — L3 queries L1's audit trail
// ---------------------------------------------------------------------------

// EventFilter is used to subscribe to or query governance events.
type EventFilter struct {
	TenantID   string      `json:"tenant_id,omitempty"`
	JobID      string      `json:"job_id,omitempty"`
	EventTypes []EventType `json:"event_types,omitempty"`
	Since      *time.Time  `json:"since,omitempty"`
	Until      *time.Time  `json:"until,omitempty"`
	Limit      int         `json:"limit,omitempty"`
}

// AuditEvent is a single event returned in audit trail queries.
type AuditEvent struct {
	Event  GovernanceEvent `json:"event"`
	Source string          `json:"source"` // "internal" | "governance"
}

// AuditTrailResponse contains the event chain for a job.
type AuditTrailResponse struct {
	JobID  string       `json:"job_id"`
	Events []AuditEvent `json:"events"`
	Total  int          `json:"total"`
}

// AuditQueryByTimeResponse contains events within a time range for a tenant.
type AuditQueryByTimeResponse struct {
	TenantID string       `json:"tenant_id"`
	From     time.Time    `json:"from"`
	To       time.Time    `json:"to"`
	Events   []AuditEvent `json:"events"`
	Total    int          `json:"total"`
	HasMore  bool         `json:"has_more"`
}

// ---------------------------------------------------------------------------
// Evidence Export — L3 requests signed evidence package from L1
// ---------------------------------------------------------------------------

// EvidenceExportResponse is the signed evidence package for a job.
type EvidenceExportResponse struct {
	JobID         string    `json:"job_id"`
	ExportedAt    time.Time `json:"exported_at"`
	Format        string    `json:"format"`         // "zip"
	SignatureAlgo string    `json:"signature_algo"` // "hmac-sha256"
	ContentHash   string    `json:"content_hash"`   // SHA-256 of the archive
	Content       io.Reader `json:"-"`              // ZIP archive stream
}

// ---------------------------------------------------------------------------
// Job Submission — L2/L3 → L1
// ---------------------------------------------------------------------------

// JobSubmissionRequest is submitted by L2 (or L3) to create a new job.
type JobSubmissionRequest struct {
	TenantID  string         `json:"tenant_id"`
	Goal      string         `json:"goal"`
	Config    map[string]any `json:"config,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// JobSubmissionResponse is returned after a job is created.
type JobSubmissionResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"` // "pending"
	CreatedAt time.Time `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Job Status Query — L2/L3 → L1
// ---------------------------------------------------------------------------

// JobStatusResponse contains the current status of a job.
type JobStatusResponse struct {
	JobID          string     `json:"job_id"`
	TenantID       string     `json:"tenant_id"`
	Status         string     `json:"status"` // "pending" | "running" | "completed" | "failed" | "parked"
	Goal           string     `json:"goal"`
	TotalSteps     int        `json:"total_steps"`
	CompletedSteps int        `json:"completed_steps"`
	FailedSteps    int        `json:"failed_steps"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Error          string     `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Job Events Query — L2/L3 → L1
// ---------------------------------------------------------------------------

// JobEventsResponse contains the event stream for a job.
type JobEventsResponse struct {
	JobID  string       `json:"job_id"`
	Events []AuditEvent `json:"events"`
	Total  int          `json:"total"`
}

// ---------------------------------------------------------------------------
// Event Subscription — L2/L3 subscribe to real-time events
// ---------------------------------------------------------------------------

// EventSubscription represents an active subscription to governance events.
type EventSubscription struct {
	Events <-chan GovernanceEvent
	Errors <-chan error
	// Close stops the subscription. Implementations must close the Events channel.
	Close func()
}

// ---------------------------------------------------------------------------
// Provider Interfaces
// ---------------------------------------------------------------------------

// GovernanceProvider is what Aetheris expects from upper layers (L3/hermesx).
// L3 implements these to provide policy, compliance, and audit capabilities.
type GovernanceProvider interface {
	// EvaluatePolicy evaluates whether a job/step complies with governance policies.
	EvaluatePolicy(ctx context.Context, req PolicyEvaluationRequest) (PolicyEvaluationResponse, error)

	// ReportEvent reports a job event to the governance layer for audit/compliance.
	ReportEvent(ctx context.Context, event GovernanceEvent) error

	// GetComplianceConstraints returns compliance constraints for a tenant.
	GetComplianceConstraints(ctx context.Context, tenantID string) (ComplianceConstraints, error)
}

// ExecutionProvider is what Aetheris exposes to upper layers (L2/L3).
// Upper layers call these to submit work, query state, and receive events.
type ExecutionProvider interface {
	// SubmitJob submits a job for durable execution.
	SubmitJob(ctx context.Context, req JobSubmissionRequest) (JobSubmissionResponse, error)

	// GetJobStatus returns the current status of a job.
	GetJobStatus(ctx context.Context, jobID string) (JobStatusResponse, error)

	// GetJobEvents returns the event stream for a job.
	GetJobEvents(ctx context.Context, jobID string) (JobEventsResponse, error)

	// GetEvidenceExport returns a signed evidence ZIP for a job.
	GetEvidenceExport(ctx context.Context, jobID string) (EvidenceExportResponse, error)

	// SubscribeEvents subscribes to real-time job events matching the filter.
	SubscribeEvents(ctx context.Context, filter EventFilter) (EventSubscription, error)
}
