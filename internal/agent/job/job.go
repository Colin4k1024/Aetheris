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

package job

import (
	"errors"
	"time"

	"rag-platform/internal/runtime/jobstore"
)

// JobStatus 任务状态；与 design/job-state-machine.md 一致，可由事件流推导（DeriveStatusFromEvents）
type JobStatus int

const (
	StatusPending JobStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusCancelled
	// StatusWaiting 短暂等待（<1 min），scheduler 仍扫描（防止 signal 丢失）
	StatusWaiting
	// StatusParked 长时间等待（>1 min），scheduler 跳过；仅由 signal 通过 WakeupQueue 唤醒（见 design/agent-process-model.md）
	StatusParked
	// StatusRetrying failed后等待重试（可选显式状态）
	StatusRetrying
)

// InvalidStateTransitionError 无效状态转换错误
type InvalidStateTransitionError struct {
	From   JobStatus
	To     JobStatus
	Reason string
}

func (e *InvalidStateTransitionError) Error() string {
	return "invalid state transition: cannot transition from " + e.From.String() + " to " + e.To.String() + ": " + e.Reason
}

// IsInvalidStateTransition 判断错误是否为无效状态转换
func IsInvalidStateTransition(err error) bool {
	var e *InvalidStateTransitionError
	return errors.As(err, &e)
}

func (s JobStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "cancelled"
	case StatusWaiting:
		return "waiting"
	case StatusParked:
		return "parked"
	case StatusRetrying:
		return "retrying"
	default:
		return "unknown"
	}
}

// Job Agent 任务实体：message 创建 Job，由 JobRunner 拉取并执行
type Job struct {
	ID        string
	AgentID   string
	TenantID  string // 租户 ID，默认 "default"；API/CLI 全链路传递，查询按租户隔离
	Goal      string
	Status    JobStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	// Cursor 恢复游标（Checkpoint ID），恢复时从下一节点继续
	Cursor string
	// RetryCount 已重试次数，供 Scheduler 重试与 backoff
	RetryCount int
	// SessionID 关联会话，Worker 恢复时 LoadAgentState(AgentID, SessionID)；空时用 AgentID 作为 sessionID
	SessionID string
	// CancelRequestedAt 非零表示已请求取消，Worker 应取消 runCtx 并将状态置为 Cancelled
	CancelRequestedAt time.Time
	// IdempotencyKey 幂等键：POST message 时可选 Idempotency-Key header，同 Agent 下相同 key 在有效窗口内只创建一次 Job
	IdempotencyKey string
	// Priority 优先级，数值越大越先被调度；空/0 为默认
	Priority int
	// QueueClass 队列类型（realtime / default / background / heavy），Scheduler 可按队列拉取
	QueueClass string
	// RequiredCapabilities 执行该 Job 所需能力（如 llm, tool, rag）；空表示任意 Worker 可执行；Scheduler 按能力派发
	RequiredCapabilities []string
	// ExecutionVersion 执行代码版本（如 git tag v1.2.0）；用于跨版本 replay 检测（design/versioning.md）
	ExecutionVersion string
	// PlannerVersion Planner 版本（可选）；记录生成 Plan 时的 Planner 版本
	PlannerVersion string

	// pendingEvent 待发出的领域事件（在状态转换时生成）
	pendingEvent *jobstore.JobEvent
}

// JobDomainEvent Job 领域事件，用于在聚合根内部传递状态变更信息
type JobDomainEvent struct {
	Type    jobstore.EventType
	Payload []byte
}

// CanTransitionTo 检查当前状态是否可以转换到目标状态
func (j *Job) CanTransitionTo(target JobStatus) error {
	validTransitions := map[JobStatus][]JobStatus{
		StatusPending:   {StatusRunning, StatusCancelled, StatusWaiting, StatusParked},
		StatusRunning:   {StatusCompleted, StatusFailed, StatusWaiting, StatusParked, StatusRetrying, StatusCancelled},
		StatusWaiting:   {StatusRunning, StatusFailed, StatusCancelled, StatusParked},
		StatusParked:    {StatusWaiting, StatusRunning, StatusCancelled},
		StatusRetrying:  {StatusRunning, StatusFailed, StatusCancelled},
		StatusCompleted: {}, // 终态
		StatusFailed:    {}, // 终态
		StatusCancelled: {}, // 终态
	}

	allowed, ok := validTransitions[j.Status]
	if !ok {
		return &InvalidStateTransitionError{From: j.Status, To: target, Reason: "unknown source status"}
	}

	for _, s := range allowed {
		if s == target {
			return nil
		}
	}

	return &InvalidStateTransitionError{From: j.Status, To: target, Reason: "target status not in allowed transitions"}
}

// Start 将 Job 状态转换为 Running
// 有效转换: Pending -> Running
func (j *Job) Start() error {
	if err := j.CanTransitionTo(StatusRunning); err != nil {
		return err
	}
	j.Status = StatusRunning
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobRunning,
	}
	return nil
}

// Complete 将 Job 状态转换为 Completed
// 有效转换: Running -> Completed
func (j *Job) Complete() error {
	if err := j.CanTransitionTo(StatusCompleted); err != nil {
		return err
	}
	j.Status = StatusCompleted
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobCompleted,
	}
	return nil
}

// Fail 将 Job 状态转换为 Failed
// 有效转换: Running -> Failed, Waiting -> Failed, Retrying -> Failed
func (j *Job) Fail() error {
	if err := j.CanTransitionTo(StatusFailed); err != nil {
		return err
	}
	j.Status = StatusFailed
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobFailed,
	}
	return nil
}

// Cancel 将 Job 状态转换为 Cancelled
// 有效转换: Pending -> Cancelled, Running -> Cancelled, Waiting -> Cancelled, Parked -> Cancelled, Retrying -> Cancelled
func (j *Job) Cancel() error {
	if err := j.CanTransitionTo(StatusCancelled); err != nil {
		return err
	}
	j.Status = StatusCancelled
	j.UpdatedAt = time.Now()
	j.CancelRequestedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobCancelled,
	}
	return nil
}

// Park 将 Job 状态转换为 Parked（长时间等待）
// 有效转换: Running -> Parked, Waiting -> Parked
func (j *Job) Park(reason string) error {
	if err := j.CanTransitionTo(StatusParked); err != nil {
		return err
	}
	j.Status = StatusParked
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobWaiting, // Parked 使用 JobWaiting 事件，payload 中标记为 parked
	}
	return nil
}

// Resume 将 Job 状态从 Parked/Waiting 转换回 Running
// 有效转换: Parked -> Running, Waiting -> Running
func (j *Job) Resume() error {
	if err := j.CanTransitionTo(StatusRunning); err != nil {
		return err
	}
	j.Status = StatusRunning
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobRunning,
	}
	return nil
}

// Wait 将 Job 状态转换为 Waiting（短暂等待）
// 有效转换: Pending -> Waiting, Running -> Waiting
func (j *Job) Wait() error {
	if err := j.CanTransitionTo(StatusWaiting); err != nil {
		return err
	}
	j.Status = StatusWaiting
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobWaiting,
	}
	return nil
}

// Retry 将 Job 状态转换为 Retrying
// 有效转换: Running -> Retrying, Failed -> Retrying
func (j *Job) Retry() error {
	if err := j.CanTransitionTo(StatusRetrying); err != nil {
		return err
	}
	j.Status = StatusRetrying
	j.RetryCount++
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobRequeued,
	}
	return nil
}

// Requeue 将 Job 状态重新转换回 Pending（用于重试）
// 有效转换: Retrying -> Pending, Waiting -> Pending
func (j *Job) Requeue() error {
	if j.Status != StatusRetrying && j.Status != StatusWaiting {
		return &InvalidStateTransitionError{From: j.Status, To: StatusPending, Reason: "can only requeue from Retrying or Waiting"}
	}
	j.Status = StatusPending
	j.UpdatedAt = time.Now()
	j.pendingEvent = &jobstore.JobEvent{
		JobID: j.ID,
		Type:  jobstore.JobRequeued,
	}
	return nil
}

// PendingEvent 返回待发出的领域事件，并清除pending状态
func (j *Job) PendingEvent() *jobstore.JobEvent {
	event := j.pendingEvent
	j.pendingEvent = nil
	return event
}

// NewJob 创建新 Job（工厂方法）
func NewJob(id, agentID, tenantID, goal string) *Job {
	now := time.Now()
	return &Job{
		ID:        id,
		AgentID:   agentID,
		TenantID:  tenantID,
		Goal:      goal,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
