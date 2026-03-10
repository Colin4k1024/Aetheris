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

package domain

import (
	"time"
)

// Step 表示执行计划中的一个步骤
type Step struct {
	ID       string                 `json:"id"`        // Step 唯一标识
	NodeID   string                 `json:"node_id"`   // 关联的 Node ID
	NodeType string                 `json:"node_type"` // 节点类型
	Input    map[string]interface{} `json:"input"`     // 输入参数
	Output   map[string]interface{} `json:"output"`    // 输出结果
	Status   StepStatus             `json:"status"`    // 执行状态
	Error    string                 `json:"error"`     // 错误信息
}

// StepStatus Step 执行状态
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"   // 待执行
	StepStatusRunning   StepStatus = "running"   // 执行中
	StepStatusCompleted StepStatus = "completed" // 已完成
	StepStatusFailed    StepStatus = "failed"    // 执行失败
)

// Checkpoint 检查点 - 保存执行状态用于恢复
type Checkpoint struct {
	ID        string                 `json:"id"`         // Checkpoint 唯一标识
	JobID     string                 `json:"job_id"`     // 关联的 Job ID
	SessionID string                 `json:"session_id"` // 关联的 Session ID
	Cursor    string                 `json:"cursor"`     // 恢复游标
	Steps     []*Step                `json:"steps"`      // 已完成的 Steps
	State     map[string]interface{} `json:"state"`      // 执行状态快照
	StepIndex int                    `json:"step_index"` // 最后一个 Step 的索引
	CreatedAt time.Time              `json:"created_at"` // 创建时间
}

// Runner 聚合根 - 负责步骤执行、checkpoint 保存和 replay
// 根据 DDD 设计，Runner 是执行上下文的聚合根
type Runner struct {
	JobID     string `json:"job_id"`     // 关联的 Job ID
	SessionID string `json:"session_id"` // 关联的 Session ID

	Checkpoint *Checkpoint `json:"checkpoint"` // 当前 Checkpoint
	Steps      []*Step     `json:"steps"`      // 执行计划中的 Steps

	// 运行时状态
	CurrentStepIndex int          `json:"current_step_index"` // 当前执行的 Step 索引
	Status           RunnerStatus `json:"status"`             // Runner 状态

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RunnerStatus Runner 状态
type RunnerStatus string

const (
	RunnerStatusIdle      RunnerStatus = "idle"      // 空闲
	RunnerStatusRunning   RunnerStatus = "running"   // 执行中
	RunnerStatusPaused    RunnerStatus = "paused"    // 已暂停（等待）
	RunnerStatusCompleted RunnerStatus = "completed" // 已完成
	RunnerStatusFailed    RunnerStatus = "failed"    // 失败
)

// NewRunner 创建新的 Runner 聚合根
func NewRunner(jobID, sessionID string) *Runner {
	now := time.Now()
	return &Runner{
		JobID:            jobID,
		SessionID:        sessionID,
		Checkpoint:       nil,
		Steps:            nil,
		CurrentStepIndex: -1,
		Status:           RunnerStatusIdle,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// AddStep 添加执行步骤
func (r *Runner) AddStep(step *Step) {
	r.Steps = append(r.Steps, step)
	r.UpdatedAt = time.Now()
}

// CurrentStep 获取当前执行的 Step
func (r *Runner) CurrentStep() *Step {
	if r.CurrentStepIndex < 0 || r.CurrentStepIndex >= len(r.Steps) {
		return nil
	}
	return r.Steps[r.CurrentStepIndex]
}

// ExecuteStep 开始执行下一步
func (r *Runner) ExecuteStep() error {
	if r.Status == RunnerStatusRunning {
		return nil // 已经在执行
	}
	r.CurrentStepIndex++
	if r.CurrentStepIndex >= len(r.Steps) {
		r.Status = RunnerStatusCompleted
		r.UpdatedAt = time.Now()
		return nil
	}
	r.Status = RunnerStatusRunning
	r.UpdatedAt = time.Now()
	currentStep := r.Steps[r.CurrentStepIndex]
	currentStep.Status = StepStatusRunning
	return nil
}

// CompleteStep 完成当前 Step
func (r *Runner) CompleteStep(output map[string]interface{}) {
	if r.CurrentStepIndex < 0 || r.CurrentStepIndex >= len(r.Steps) {
		return
	}
	currentStep := r.Steps[r.CurrentStepIndex]
	currentStep.Status = StepStatusCompleted
	currentStep.Output = output
	r.UpdatedAt = time.Now()
}

// FailStep 标记当前 Step 失败
func (r *Runner) FailStep(errMsg string) {
	if r.CurrentStepIndex < 0 || r.CurrentStepIndex >= len(r.Steps) {
		return
	}
	currentStep := r.Steps[r.CurrentStepIndex]
	currentStep.Status = StepStatusFailed
	currentStep.Error = errMsg
	r.Status = RunnerStatusFailed
	r.UpdatedAt = time.Now()
}

// SaveCheckpoint 保存 Checkpoint
func (r *Runner) SaveCheckpoint(cursor string) *Checkpoint {
	now := time.Now()
	r.Checkpoint = &Checkpoint{
		ID:        "cp-" + r.JobID + "-" + now.Format("20060102150405"),
		JobID:     r.JobID,
		SessionID: r.SessionID,
		Cursor:    cursor,
		Steps:     r.Steps[:r.CurrentStepIndex+1],
		StepIndex: r.CurrentStepIndex,
		CreatedAt: now,
	}
	r.UpdatedAt = now
	return r.Checkpoint
}

// LoadCheckpoint 从 Checkpoint 恢复
func (r *Runner) LoadCheckpoint(cp *Checkpoint) error {
	if cp == nil {
		return nil
	}
	r.Checkpoint = cp
	r.Steps = cp.Steps
	r.CurrentStepIndex = cp.StepIndex
	r.Status = RunnerStatusPaused
	r.UpdatedAt = time.Now()
	return nil
}

// Reset 重置 Runner
func (r *Runner) Reset() {
	r.CurrentStepIndex = -1
	r.Status = RunnerStatusIdle
	r.Steps = nil
	r.Checkpoint = nil
	r.UpdatedAt = time.Now()
}

// IsCompleted 检查是否已完成所有 Steps
func (r *Runner) IsCompleted() bool {
	return r.CurrentStepIndex >= len(r.Steps)-1 && r.Status == RunnerStatusCompleted
}

// HasFailed 检查是否失败
func (r *Runner) HasFailed() bool {
	return r.Status == RunnerStatusFailed
}

// Progress 返回执行进度
func (r *Runner) Progress() (completed, total int) {
	completed = r.CurrentStepIndex + 1
	if r.Status == RunnerStatusCompleted {
		completed = len(r.Steps)
	}
	total = len(r.Steps)
	return
}
