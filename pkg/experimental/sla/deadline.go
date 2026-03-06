// Copyright 2026 Aetheris
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

package sla

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobDeadlineManager Job 截止日期管理器
type JobDeadlineManager struct {
	mu            sync.RWMutex
	deadlines     map[string]JobDeadline // jobID -> deadline
	breachChan    chan BreachEvent
	checkInterval time.Duration
}

// JobDeadline Job 截止日期
type JobDeadline struct {
	JobID       string          `json:"job_id"`
	TenantID    string          `json:"tenant_id"`
	Deadline    time.Time       `json:"deadline"`
	StepIDs     []string        `json:"step_ids"` // 需要追踪的步骤
	CreatedAt   time.Time       `json:"created_at"`
	Enforcement EnforcementMode `json:"enforcement"`
}

// EnforcementMode 强制执行模式
type EnforcementMode string

const (
	EnforcementModeNone     EnforcementMode = "none"     // 不强制
	EnforcementModeWarn     EnforcementMode = "warn"     // 仅警告
	EnforcementModeCancel   EnforcementMode = "cancel"   // 取消任务
	EnforcementModeFailover EnforcementMode = "failover" // 故障转移
)

// BreachEvent SLA 违约事件
type BreachEvent struct {
	JobID      string          `json:"job_id"`
	TenantID   string          `json:"tenant_id"`
	Type       string          `json:"type"`    // job_deadline, step_deadline
	StepID     string          `json:"step_id"` // 如果是步骤级别
	Deadline   time.Time       `json:"deadline"`
	ActualTime time.Time       `json:"actual_time"`
	Overdue    time.Duration   `json:"overdue"`
	Action     EnforcementMode `json:"action"`
	Timestamp  time.Time       `json:"timestamp"`
}

// NewJobDeadlineManager 创建 Job 截止日期管理器
func NewJobDeadlineManager() *JobDeadlineManager {
	return &JobDeadlineManager{
		deadlines:     make(map[string]JobDeadline),
		breachChan:    make(chan BreachEvent, 1000),
		checkInterval: 1 * time.Second,
	}
}

// SetDeadline 设置任务截止日期
func (m *JobDeadlineManager) SetDeadline(ctx context.Context, deadline JobDeadline) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if deadline.Deadline.Before(time.Now()) {
		return fmt.Errorf("deadline must be in the future")
	}

	m.deadlines[deadline.JobID] = deadline
	return nil
}

// GetDeadline 获取任务截止日期
func (m *JobDeadlineManager) GetDeadline(jobID string) (JobDeadline, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deadline, exists := m.deadlines[jobID]
	return deadline, exists
}

// RemoveDeadline 移除任务截止日期
func (m *JobDeadlineManager) RemoveDeadline(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.deadlines, jobID)
}

// CheckDeadlines 检查所有截止日期
func (m *JobDeadlineManager) CheckDeadlines(ctx context.Context) []BreachEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var breaches []BreachEvent
	now := time.Now()

	for jobID, deadline := range m.deadlines {
		if now.After(deadline.Deadline) {
			overdue := now.Sub(deadline.Deadline)
			breach := BreachEvent{
				JobID:      jobID,
				TenantID:   deadline.TenantID,
				Type:       "job_deadline",
				Deadline:   deadline.Deadline,
				ActualTime: now,
				Overdue:    overdue,
				Action:     deadline.Enforcement,
				Timestamp:  now,
			}
			breaches = append(breaches, breach)
		}
	}

	return breaches
}

// Subscribe 订阅违约事件
func (m *JobDeadlineManager) Subscribe() <-chan BreachEvent {
	return m.breachChan
}

// StepSLATracker 步骤级别 SLA 追踪器
type StepSLATracker struct {
	mu            sync.RWMutex
	stepDeadlines map[string]StepDeadline // jobID:stepID -> deadline
	measurements  map[string]*StepMeasurement
	breachChan    chan BreachEvent
}

// StepDeadline 步骤截止日期
type StepDeadline struct {
	JobID     string    `json:"job_id"`
	StepID    string    `json:"step_id"`
	TenantID  string    `json:"tenant_id"`
	Deadline  time.Time `json:"deadline"`
	CreatedAt time.Time `json:"created_at"`
}

// StepMeasurement 步骤测量数据
type StepMeasurement struct {
	StepID    string        `json:"step_id"`
	JobID     string        `json:"job_id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Status    string        `json:"status"` // success, failure, timeout
}

// NewStepSLATracker 创建步骤 SLA 追踪器
func NewStepSLATracker() *StepSLATracker {
	return &StepSLATracker{
		stepDeadlines: make(map[string]StepDeadline),
		measurements:  make(map[string]*StepMeasurement),
		breachChan:    make(chan BreachEvent, 1000),
	}
}

// SetStepDeadline 设置步骤截止日期
func (t *StepSLATracker) SetStepDeadline(ctx context.Context, deadline StepDeadline) {
	key := fmt.Sprintf("%s:%s", deadline.JobID, deadline.StepID)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stepDeadlines[key] = deadline
}

// RecordStepStart 记录步骤开始
func (t *StepSLATracker) RecordStepStart(jobID, stepID string) {
	key := fmt.Sprintf("%s:%s", jobID, stepID)
	t.mu.Lock()
	defer t.mu.Unlock()

	t.measurements[key] = &StepMeasurement{
		StepID:    stepID,
		JobID:     jobID,
		StartTime: time.Now(),
	}
}

// RecordStepEnd 记录步骤结束
func (t *StepSLATracker) RecordStepEnd(jobID, stepID, status string) {
	key := fmt.Sprintf("%s:%s", jobID, stepID)
	t.mu.Lock()
	defer t.mu.Unlock()

	measurement, exists := t.measurements[key]
	if !exists {
		return
	}

	measurement.EndTime = time.Now()
	measurement.Duration = measurement.EndTime.Sub(measurement.StartTime)
	measurement.Status = status

	// 检查是否超时
	deadline, hasDeadline := t.stepDeadlines[key]
	if hasDeadline && measurement.EndTime.After(deadline.Deadline) {
		breach := BreachEvent{
			JobID:      jobID,
			TenantID:   deadline.TenantID,
			Type:       "step_deadline",
			StepID:     stepID,
			Deadline:   deadline.Deadline,
			ActualTime: measurement.EndTime,
			Overdue:    measurement.EndTime.Sub(deadline.Deadline),
			Timestamp:  time.Now(),
		}
		select {
		case t.breachChan <- breach:
		default:
		}
	}
}

// GetStepMeasurement 获取步骤测量数据
func (t *StepSLATracker) GetStepMeasurement(jobID, stepID string) (*StepMeasurement, bool) {
	key := fmt.Sprintf("%s:%s", jobID, stepID)
	t.mu.RLock()
	defer t.mu.RUnlock()

	measurement, exists := t.measurements[key]
	return measurement, exists
}

// SubscribeStepBreaches 订阅步骤违约事件
func (t *StepSLATracker) SubscribeStepBreaches() <-chan BreachEvent {
	return t.breachChan
}
