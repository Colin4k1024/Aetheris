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

package approval

import (
	"context"
	"time"
)

// Decision 审批决定
type Decision string

const (
	DecisionPending   Decision = "pending"
	DecisionApproved  Decision = "approved"
	DecisionRejected  Decision = "rejected"
	DecisionExpired   Decision = "expired"
	DecisionDelegated Decision = "delegated"
)

// ApproverType 指定审批人的类型
type ApproverType string

const (
	ApproverTypeAnyone   ApproverType = "anyone"   // 任何人都可以审批
	ApproverTypeSpecific ApproverType = "specific" // 指定用户审批
	ApproverTypeRole     ApproverType = "role"     // 指定角色审批
)

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	ID             string                 `json:"id"`
	JobID          string                 `json:"job_id"`
	NodeID         string                 `json:"node_id"`
	CorrelationKey string                 `json:"correlation_key"`
	ApproverType   ApproverType           `json:"approver_type"`
	ApproverID     string                 `json:"approver_id,omitempty"`
	Role           string                 `json:"role,omitempty"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	Status         Decision               `json:"status"`
	ApproverResp   *ApprovalResponse      `json:"approver_response,omitempty"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ApprovalResponse 审批响应
type ApprovalResponse struct {
	Decision     Decision  `json:"decision"`
	ApproverID   string    `json:"approver_id"`
	ApproverName string    `json:"approver_name"`
	Comment      string    `json:"comment,omitempty"`
	DelegatedTo  string    `json:"delegated_to,omitempty"`
	RespondedAt  time.Time `json:"responded_at"`
}

// ApprovalStore 审批存储接口
type ApprovalStore interface {
	// Create 创建审批请求
	Create(ctx context.Context, req *ApprovalRequest) error
	// GetByID 根据 ID 获取审批请求
	GetByID(ctx context.Context, id string) (*ApprovalRequest, error)
	// GetByJobID 根据 Job ID 获取所有审批请求
	GetByJobID(ctx context.Context, jobID string) ([]*ApprovalRequest, error)
	// GetPending 获取所有待审批请求
	GetPending(ctx context.Context) ([]*ApprovalRequest, error)
	// GetPendingByApprover 根据审批人获取待审批请求
	GetPendingByApprover(ctx context.Context, approverID string) ([]*ApprovalRequest, error)
	// Complete 完成审批请求
	Complete(ctx context.Context, id string, resp *ApprovalResponse) error
	// Expire 标记审批请求已过期
	Expire(ctx context.Context, id string) error
}
