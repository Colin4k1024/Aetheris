package eino

import "time"

// RunStatus 运行状态。
type RunStatus string

const (
	RunStatusPending   RunStatus = "PENDING"
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusPaused    RunStatus = "PAUSED"
	RunStatusSucceeded RunStatus = "SUCCEEDED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusCanceled  RunStatus = "CANCELED"
)

// ResumeMode 恢复模式。
type ResumeMode string

const (
	ResumeModeFromToolCall ResumeMode = "FROM_TOOL_CALL"
)

// ResumeStrategy 恢复策略。
type ResumeStrategy string

const (
	ResumeStrategyReuseSuccessfulEffects ResumeStrategy = "REUSE_SUCCESSFUL_EFFECTS"
	ResumeStrategyReexecuteFromPoint     ResumeStrategy = "REEXECUTE_FROM_POINT"
)

// BudgetPolicy 成本预算策略。
type BudgetPolicy struct {
	MaxTokens    int `json:"max_tokens"`
	MaxToolCalls int `json:"max_tool_calls"`
	MaxRetries   int `json:"max_retries"`
}

// Run 运行实例。
type Run struct {
	ID             string                 `json:"id"`
	WorkflowID     string                 `json:"workflow_id"`
	Status         RunStatus              `json:"status"`
	Input          map[string]interface{} `json:"input"`
	Output         map[string]interface{} `json:"output,omitempty"`
	Budget         BudgetPolicy           `json:"budget"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Step 执行步骤。
type Step struct {
	ID        string                 `json:"id"`
	RunID     string                 `json:"run_id"`
	NodeName  string                 `json:"node_name"`
	Status    string                 `json:"status"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	StartedAt *time.Time             `json:"started_at,omitempty"`
	EndedAt   *time.Time             `json:"ended_at,omitempty"`
}

// ToolCall 工具调用。
type ToolCall struct {
	ID              string                 `json:"id"`
	RunID           string                 `json:"run_id"`
	StepID          string                 `json:"step_id"`
	ToolName        string                 `json:"tool_name"`
	Status          string                 `json:"status"`
	RequestPayload  map[string]interface{} `json:"request_payload,omitempty"`
	ResponsePayload map[string]interface{} `json:"response_payload,omitempty"`
	IdempotencyKey  string                 `json:"idempotency_key,omitempty"`
	SideEffectSafe  bool                   `json:"side_effect_safe"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	EndedAt         *time.Time             `json:"ended_at,omitempty"`
}

// EventType 事件类型。
type EventType string

const (
	EventTypeRunCreated      EventType = "RUN_CREATED"
	EventTypeRunPaused       EventType = "RUN_PAUSED"
	EventTypeRunResumed      EventType = "RUN_RESUMED"
	EventTypeStepStarted     EventType = "STEP_STARTED"
	EventTypeStepCompleted   EventType = "STEP_COMPLETED"
	EventTypeToolCallStarted EventType = "TOOL_CALL_STARTED"
	EventTypeToolCallEnded   EventType = "TOOL_CALL_ENDED"
	EventTypeRunFailed       EventType = "RUN_FAILED"
	EventTypeRunSucceeded    EventType = "RUN_SUCCEEDED"
	EventTypeHumanInjected   EventType = "HUMAN_INJECTED"
)

// RuntimeEvent 运行时事件（事实源）。
type RuntimeEvent struct {
	ID         string                 `json:"id"`
	RunID      string                 `json:"run_id"`
	StepID     string                 `json:"step_id,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	Type       EventType              `json:"type"`
	Seq        int64                  `json:"seq"`
	Actor      string                 `json:"actor,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
}

// ResumeRunRequest 恢复请求。
type ResumeRunRequest struct {
	Mode            ResumeMode     `json:"mode"`
	FromToolCallID  string         `json:"from_tool_call_id"`
	Strategy        ResumeStrategy `json:"strategy"`
	Operator        string         `json:"operator"`
	Reason          string         `json:"reason"`
	ResumeRequestID string         `json:"resume_request_id,omitempty"`
}

// HumanDecision 人工决策注入。
type HumanDecision struct {
	TargetStepID string                 `json:"target_step_id"`
	Patch        map[string]interface{} `json:"patch"`
	Operator     string                 `json:"operator"`
	Comment      string                 `json:"comment"`
}
