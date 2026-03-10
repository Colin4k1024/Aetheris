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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// ToolInvocation 工具调用聚合 - 表示一次工具调用的完整生命周期
type ToolInvocation struct {
	ID             string                 `json:"id"`                    // 唯一标识
	JobID          string                 `json:"job_id"`                // 关联的 Job ID
	IdempotencyKey string                 `json:"idempotency_key"`       // 幂等键
	ToolName       string                 `json:"tool_name"`             // 工具名称
	Arguments      map[string]interface{} `json:"arguments"`             // 调用参数
	Result         map[string]interface{} `json:"result,omitempty"`      // 调用结果
	Error          string                 `json:"error,omitempty"`       // 错误信息
	Status         ToolInvocationStatus   `json:"status"`                // 调用状态
	StartedAt      time.Time              `json:"started_at"`            // 开始时间
	FinishedAt     *time.Time             `json:"finished_at,omitempty"` // 完成时间
	RetryCount     int                    `json:"retry_count"`           // 重试次数
}

// ToolInvocationStatus 工具调用状态
type ToolInvocationStatus string

const (
	ToolInvocationStatusPending   ToolInvocationStatus = "pending"   // 待执行
	ToolInvocationStatusRunning   ToolInvocationStatus = "running"   // 执行中
	ToolInvocationStatusCompleted ToolInvocationStatus = "completed" // 已完成
	ToolInvocationStatusFailed    ToolInvocationStatus = "failed"    // 执行失败
)

// ToolLedger 工具调用账本 - 领域服务，负责记录和查询工具调用的幂等性
type ToolLedger interface {
	// Record 开始记录一次工具调用
	Record(ctx context.Context, invocation *ToolInvocation) error
	// Complete 完成工具调用，记录结果
	Complete(ctx context.Context, idempotencyKey string, result map[string]interface{}, err error) error
	// Lookup 查询幂等键对应的已有结果
	Lookup(ctx context.Context, idempotencyKey string) (*ToolInvocation, error)
	// Verify 验证给定工具调用是否已执行过（幂等性检查）
	Verify(ctx context.Context, idempotencyKey string) (bool, *ToolInvocation)
	// ListByJob 列出 Job 关联的所有工具调用
	ListByJob(ctx context.Context, jobID string) ([]*ToolInvocation, error)
}

// ToolLedgerInMemory 内存实现的 ToolLedger
type ToolLedgerInMemory struct {
	byKey   map[string]*ToolInvocation
	byJobID map[string][]*ToolInvocation
}

// NewToolLedgerInMemory 创建内存实现的 ToolLedger
func NewToolLedgerInMemory() *ToolLedgerInMemory {
	return &ToolLedgerInMemory{
		byKey:   make(map[string]*ToolInvocation),
		byJobID: make(map[string][]*ToolInvocation),
	}
}

// Record 开始记录一次工具调用
func (l *ToolLedgerInMemory) Record(ctx context.Context, invocation *ToolInvocation) error {
	if invocation == nil {
		return nil
	}
	invocation.Status = ToolInvocationStatusPending
	invocation.StartedAt = time.Now()

	l.byKey[invocation.IdempotencyKey] = invocation
	l.byJobID[invocation.JobID] = append(l.byJobID[invocation.JobID], invocation)
	return nil
}

// Complete 完成工具调用，记录结果
func (l *ToolLedgerInMemory) Complete(ctx context.Context, idempotencyKey string, result map[string]interface{}, err error) error {
	invocation, ok := l.byKey[idempotencyKey]
	if !ok {
		return nil // 没有记录，忽略
	}

	now := time.Now()
	invocation.FinishedAt = &now
	if err != nil {
		invocation.Status = ToolInvocationStatusFailed
		invocation.Error = err.Error()
	} else {
		invocation.Status = ToolInvocationStatusCompleted
		invocation.Result = result
	}
	return nil
}

// Lookup 查询幂等键对应的已有结果
func (l *ToolLedgerInMemory) Lookup(ctx context.Context, idempotencyKey string) (*ToolInvocation, error) {
	invocation, ok := l.byKey[idempotencyKey]
	if !ok {
		return nil, nil
	}
	return invocation, nil
}

// Verify 验证给定工具调用是否已执行过
func (l *ToolLedgerInMemory) Verify(ctx context.Context, idempotencyKey string) (bool, *ToolInvocation) {
	invocation, ok := l.byKey[idempotencyKey]
	if !ok {
		return false, nil
	}
	// 如果已完成或有结果，认为已执行过
	if invocation.Status == ToolInvocationStatusCompleted || invocation.Result != nil {
		return true, invocation
	}
	// 如果失败，也认为已执行过（避免重复执行失败的调用）
	if invocation.Status == ToolInvocationStatusFailed {
		return true, invocation
	}
	return false, nil
}

// ListByJob 列出 Job 关联的所有工具调用
func (l *ToolLedgerInMemory) ListByJob(ctx context.Context, jobID string) ([]*ToolInvocation, error) {
	return l.byJobID[jobID], nil
}

// NewToolInvocation 创建新的 ToolInvocation
func NewToolInvocation(jobID, toolName string, args map[string]interface{}) *ToolInvocation {
	return &ToolInvocation{
		JobID:          jobID,
		ToolName:       toolName,
		Arguments:      args,
		IdempotencyKey: computeIdempotencyKey(toolName, args),
		Status:         ToolInvocationStatusPending,
	}
}

// computeIdempotencyKey 计算幂等键
func computeIdempotencyKey(toolName string, args map[string]interface{}) string {
	// 使用与 pkg/effects/tool.go 相同的算法
	data, err := json.Marshal(map[string]interface{}{
		"name": toolName,
		"args": args,
	})
	if err != nil {
		return "tool:" + toolName
	}
	return "tool:" + hashSHA256(data)
}

// hashSHA256 计算 SHA256 哈希
func hashSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
