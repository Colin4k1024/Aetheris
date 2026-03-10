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

package jobstore

import (
	"encoding/json"
	"time"
)

// EventType 任务事件类型（事件流语义，用于重放与审计）
type EventType string

const (
	JobCreated             EventType = "job_created"
	PlanGenerated          EventType = "plan_generated"
	NodeStarted            EventType = "node_started"
	NodeFinished           EventType = "node_finished"
	CommandEmitted         EventType = "command_emitted"
	CommandCommitted       EventType = "command_committed"
	ToolCalled             EventType = "tool_called"
	ToolReturned           EventType = "tool_returned"
	ToolInvocationStarted  EventType = "tool_invocation_started"
	ToolInvocationFinished EventType = "tool_invocation_finished"
	StepCommitted          EventType = "step_committed" // 显式 step 提交屏障；写入顺序：command_committed → node_finished → step_committed（2.0 Exactly-Once）
	JobCompleted           EventType = "job_completed"
	JobFailed              EventType = "job_failed"
	JobCancelled           EventType = "job_cancelled"

	// Job 状态机事件（见 design/job-state-machine.md）：驱动状态迁移，写入后应更新 metadata status
	JobQueued     EventType = "job_queued"
	JobLeased     EventType = "job_leased"
	JobRunning    EventType = "job_running"
	JobWaiting    EventType = "job_waiting" // Job 在 Wait 节点挂起
	JobRequeued   EventType = "job_requeued"
	WaitCompleted EventType = "wait_completed"

	// DDD 领域事件补充：Job 生命周期事件
	JobParked  EventType = "job_parked"  // Job 进入 Parked 状态（长时间等待）
	JobResumed EventType = "job_resumed" // Job 从 Parked/Waiting 恢复执行

	// DDD 领域事件补充：执行过程事件（Step 级别）
	StepStarted     EventType = "step_started"     // Step 开始执行
	StepFinished    EventType = "step_finished"    // Step 执行完成
	StepFailed      EventType = "step_failed"      // Step 执行失败
	StepRetried     EventType = "step_retried"     // Step 重试
	CheckpointSaved EventType = "checkpoint_saved" // Checkpoint 已保存

	// 以上事件中参与 Replay 的 Effect 事件（见 design/effect-system.md）：
	// PlanGenerated, CommandCommitted, ToolInvocationFinished, NodeFinished 用于重建 ReplayContext；
	// Replay 时forbidden真实调用 LLM/Tool，只读这些事件注入结果。
	// TimerFired、RandomRecorded、UUIDRecorded：Replay 时仅从事件注入时间/随机/UID，不重新执行（2.0 确定性）
	TimerFired     EventType = "timer_fired"
	RandomRecorded EventType = "random_recorded"
	UUIDRecorded   EventType = "uuid_recorded"
	HTTPRecorded   EventType = "http_recorded"

	// AgentMessage 信箱消息：外部向 Job 投递的消息，Wait 节点 wait_type=message 时可根据 channel/correlation_key 消费（design/agent-process-model.md Mailbox）
	AgentMessage EventType = "agent_message"

	// Semantic events for Trace narrative (v0.9); see design/trace-event-schema-v0.9.md
	StateCheckpointed    EventType = "state_checkpointed"
	AgentThoughtRecorded EventType = "agent_thought_recorded"
	DecisionMade         EventType = "decision_made"
	ToolSelected         EventType = "tool_selected"
	ToolResultSummarized EventType = "tool_result_summarized"
	RecoveryStarted      EventType = "recovery_started"
	RecoveryCompleted    EventType = "recovery_completed"
	StepCompensated      EventType = "step_compensated"
	StateChanged         EventType = "state_changed" // 外部资源变更（resource_type, resource_id, operation）供审计
	// ReasoningSnapshot 推理快照：每步完成后的决策上下文，供因果调试（哪个计划步骤、哪次 LLM 输出导致该步）
	ReasoningSnapshot EventType = "reasoning_snapshot"
	// DecisionSnapshot Planner 决策快照：PlanGoal 返回后写入，含 goal、memory 摘要、reasoning 摘要、decision（TaskGraph），供可追责与 Trace 展示（design/execution-forensics.md）
	DecisionSnapshot EventType = "decision_snapshot"

	// Trace 2.0 Cognition（design/trace-2.0-cognition.md）：不参与 Replay，仅用于 Trace 叙事
	MemoryRead    EventType = "memory_read"
	MemoryWrite   EventType = "memory_write"
	PlanEvolution EventType = "plan_evolution"

	// 2.0-M2: Retention & Audit events
	JobArchived   EventType = "job_archived"   // Job 已归档到冷存储
	JobDeleted    EventType = "job_deleted"    // Job 已删除（tombstone）
	AccessAudited EventType = "access_audited" // 访问审计（导出/查看）

	// 2.0-M3: Critical decision markers
	CriticalDecisionMade EventType = "critical_decision_made" // 关键决策（approve/deny/escalate）
	HumanApprovalGiven   EventType = "human_approval_given"   // 人类审批
	PaymentExecuted      EventType = "payment_executed"       // 支付执行
	EmailSent            EventType = "email_sent"             // 邮件发送

	// 2.1: Evidence Export audit events
	EvidenceExportRequested EventType = "evidence_export_requested" // 证据导出请求
	EvidenceExportCompleted EventType = "evidence_export_completed" // 证据导出完成
)

// JobWaitingPayload job_waiting 事件 payload 契约；只有携带相同 correlation_key 的 signal 才能解除该 block（design/runtime-contract.md）
type JobWaitingPayload struct {
	NodeID            string          `json:"node_id"`
	WaitType          string          `json:"wait_type"`       // webhook | human | timer | signal | message
	CorrelationKey    string          `json:"correlation_key"` // 唯一标识此次等待，signal 必须匹配
	WaitKind          string          `json:"wait_kind"`
	Reason            string          `json:"reason"`
	ExpiresAtRFC3339  string          `json:"expires_at"`
	ResumptionContext json.RawMessage `json:"resumption_context,omitempty"` // 恢复上下文：payload_results snapshot + plan_decision_id，保证等待后"同一思维"继续（design/agent-process-model.md § Continuation）
}

// ParseJobWaitingPayload 解析 job_waiting 事件的 payload；若缺少 correlation_key returned empty字符串
func ParseJobWaitingPayload(payload []byte) (p JobWaitingPayload, err error) {
	if len(payload) == 0 {
		return p, nil
	}
	err = json.Unmarshal(payload, &p)
	return p, err
}

// AgentMessagePayload agent_message 事件 payload；POST /api/jobs/:id/message 写入，Wait wait_type=message 时按 channel 或 correlation_key 匹配解除
type AgentMessagePayload struct {
	MessageID      string                 `json:"message_id"`
	Channel        string                 `json:"channel"`
	CorrelationKey string                 `json:"correlation_key"`
	Payload        map[string]interface{} `json:"payload"`
}

// MemoryReadPayload memory_read 事件 payload（design/trace-2.0-cognition.md）
type MemoryReadPayload struct {
	JobID      string `json:"job_id,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
	StepIndex  int    `json:"step_index,omitempty"`
	MemoryType string `json:"memory_type"` // working | long_term | episodic
	KeyOrScope string `json:"key_or_scope,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

// MemoryWritePayload memory_write 事件 payload（design/trace-2.0-cognition.md）
type MemoryWritePayload struct {
	JobID      string `json:"job_id,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
	StepIndex  int    `json:"step_index,omitempty"`
	MemoryType string `json:"memory_type"` // working | long_term | episodic
	KeyOrScope string `json:"key_or_scope,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

// PlanEvolutionPayload plan_evolution 事件 payload（design/trace-2.0-cognition.md）；可选，Trace 也可直接用 plan_generated + decision_snapshot 序列
type PlanEvolutionPayload struct {
	PlanVersion int    `json:"plan_version,omitempty"`
	DiffSummary string `json:"diff_summary,omitempty"`
}

// JobEvent 单条不可变事件；Job 的真实形态是事件流
type JobEvent struct {
	ID        string    // 单条事件唯一 ID，用于排序/去重；Append 时为空可由实现生成
	JobID     string    // 所属任务流 ID
	Type      EventType // 事件类型
	Payload   []byte    // JSON，由各 EventType 语义定义
	CreatedAt time.Time

	// 2.0-M1: Proof chain for tamper detection
	PrevHash string // 上一个事件的 hash（SHA256）
	Hash     string // 当前事件 hash（SHA256(JobID|Type|Payload|Timestamp|PrevHash)）
}

// UnmarshalPayload 解码事件 Payload 到目标结构体
func (e *JobEvent) UnmarshalPayload(v any) error {
	if e.Payload == nil {
		return nil
	}
	return json.Unmarshal(e.Payload, v)
}

// DDD 领域事件 Payloads

// JobParkedPayload job_parked 事件 payload
type JobParkedPayload struct {
	Reason         string     `json:"reason"`               // 停放原因
	CorrelationKey string     `json:"correlation_key"`      // 用于唤醒的关联键
	WaitType       string     `json:"wait_type"`            // webhook, human, timer, signal, message
	ExpiresAt      *time.Time `json:"expires_at,omitempty"` // 过期时间（可选）
}

// JobResumedPayload job_resumed 事件 payload
type JobResumedPayload struct {
	CorrelationKey string `json:"correlation_key"` // 唤醒时使用的关联键
	ResumedBy      string `json:"resumed_by"`      // signal, webhook, timer, manual
}

// StepStartedPayload step_started 事件 payload
type StepStartedPayload struct {
	StepID    string `json:"step_id"`         // Step 唯一标识
	NodeID    string `json:"node_id"`         // 关联的 Node ID
	StepIndex int    `json:"step_index"`      // Step 在 TaskGraph 中的索引
	Input     string `json:"input,omitempty"` // Step 输入（可选，用于调试）
}

// StepFinishedPayload step_finished 事件 payload
type StepFinishedPayload struct {
	StepID     string `json:"step_id"`          // Step 唯一标识
	NodeID     string `json:"node_id"`          // 关联的 Node ID
	StepIndex  int    `json:"step_index"`       // Step 在 TaskGraph 中的索引
	Output     string `json:"output,omitempty"` // Step 输出（可选）
	DurationMs int64  `json:"duration_ms"`      // 执行耗时（毫秒）
}

// StepFailedPayload step_failed 事件 payload
type StepFailedPayload struct {
	StepID    string     `json:"step_id"`            // Step 唯一标识
	NodeID    string     `json:"node_id"`            // 关联的 Node ID
	StepIndex int        `json:"step_index"`         // Step 在 TaskGraph 中的索引
	Error     string     `json:"error"`              // 错误信息
	Retryable bool       `json:"retryable"`          // 是否可重试
	RetryAt   *time.Time `json:"retry_at,omitempty"` // 计划重试时间（可选）
}

// StepRetriedPayload step_retried 事件 payload
type StepRetriedPayload struct {
	StepID     string `json:"step_id"`     // Step 唯一标识
	NodeID     string `json:"node_id"`     // 关联的 Node ID
	StepIndex  int    `json:"step_index"`  // Step 在 TaskGraph 中的索引
	RetryCount int    `json:"retry_count"` // 当前重试次数
	PrevError  string `json:"prev_error"`  // 上次错误信息
	MaxRetries int    `json:"max_retries"` // 最大重试次数
}

// CheckpointSavedPayload checkpoint_saved 事件 payload
type CheckpointSavedPayload struct {
	CheckpointID string `json:"checkpoint_id"` // Checkpoint 唯一标识
	JobID        string `json:"job_id"`        // 所属 Job ID
	SessionID    string `json:"session_id"`    // 关联的 Session ID
	Cursor       string `json:"cursor"`        // 恢复游标
	StepIndex    int    `json:"step_index"`    // Checkpoint 时的 Step 索引
	SnapshotSize int    `json:"snapshot_size"` // 快照大小（字节）
}

// NewJobParkedEvent 创建 job_parked 事件
func NewJobParkedEvent(jobID, reason, correlationKey, waitType string) (*JobEvent, error) {
	payload := JobParkedPayload{
		Reason:         reason,
		CorrelationKey: correlationKey,
		WaitType:       waitType,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &JobEvent{
		JobID:     jobID,
		Type:      JobParked,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}

// NewJobResumedEvent 创建 job_resumed 事件
func NewJobResumedEvent(jobID, correlationKey, resumedBy string) (*JobEvent, error) {
	payload := JobResumedPayload{
		CorrelationKey: correlationKey,
		ResumedBy:      resumedBy,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &JobEvent{
		JobID:     jobID,
		Type:      JobResumed,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}

// NewStepStartedEvent 创建 step_started 事件
func NewStepStartedEvent(jobID, stepID, nodeID string, stepIndex int, input string) (*JobEvent, error) {
	payload := StepStartedPayload{
		StepID:    stepID,
		NodeID:    nodeID,
		StepIndex: stepIndex,
		Input:     input,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &JobEvent{
		JobID:     jobID,
		Type:      StepStarted,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}

// NewStepFinishedEvent 创建 step_finished 事件
func NewStepFinishedEvent(jobID, stepID, nodeID string, stepIndex int, output string, durationMs int64) (*JobEvent, error) {
	payload := StepFinishedPayload{
		StepID:     stepID,
		NodeID:     nodeID,
		StepIndex:  stepIndex,
		Output:     output,
		DurationMs: durationMs,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &JobEvent{
		JobID:     jobID,
		Type:      StepFinished,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}

// NewStepFailedEvent 创建 step_failed 事件
func NewStepFailedEvent(jobID, stepID, nodeID string, stepIndex int, errMsg string, retryable bool) (*JobEvent, error) {
	payload := StepFailedPayload{
		StepID:    stepID,
		NodeID:    nodeID,
		StepIndex: stepIndex,
		Error:     errMsg,
		Retryable: retryable,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &JobEvent{
		JobID:     jobID,
		Type:      StepFailed,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}

// NewStepRetriedEvent 创建 step_retried 事件
func NewStepRetriedEvent(jobID, stepID, nodeID string, stepIndex, retryCount, maxRetries int, prevError string) (*JobEvent, error) {
	payload := StepRetriedPayload{
		StepID:     stepID,
		NodeID:     nodeID,
		StepIndex:  stepIndex,
		RetryCount: retryCount,
		PrevError:  prevError,
		MaxRetries: maxRetries,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &JobEvent{
		JobID:     jobID,
		Type:      StepRetried,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}

// NewCheckpointSavedEvent 创建 checkpoint_saved 事件
func NewCheckpointSavedEvent(jobID, checkpointID, sessionID, cursor string, stepIndex, snapshotSize int) (*JobEvent, error) {
	payload := CheckpointSavedPayload{
		CheckpointID: checkpointID,
		JobID:        jobID,
		SessionID:    sessionID,
		Cursor:       cursor,
		StepIndex:    stepIndex,
		SnapshotSize: snapshotSize,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &JobEvent{
		JobID:     jobID,
		Type:      CheckpointSaved,
		Payload:   data,
		CreatedAt: time.Now(),
	}, nil
}
