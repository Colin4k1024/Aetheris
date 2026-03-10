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

package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// VerificationStatus 验证状态
type VerificationStatus string

const (
	VerificationStatusMatch    VerificationStatus = "match"    // 状态匹配，可以恢复结果跳过执行
	VerificationStatusMismatch VerificationStatus = "mismatch" // 状态不匹配，需要标记失败或回滚
	VerificationStatusPending  VerificationStatus = "pending"  // 待验证
	VerificationStatusSkipped  VerificationStatus = "skipped"  // 跳过验证（无状态变更）
	VerificationStatusError    VerificationStatus = "error"    // 验证过程出错
)

// ReplayDecision Replay 决策结果
type ReplayDecision string

const (
	ReplayDecisionRestoreAndSkip ReplayDecision = "restore_and_skip" // 恢复结果，跳过执行
	ReplayDecisionExecute        ReplayDecision = "execute"          // 正常执行
	ReplayDecisionFail           ReplayDecision = "fail"             // 标记失败
	ReplayDecisionNeedsReview    ReplayDecision = "needs_review"     // 需要人工审查
)

// Note: StateChangeRecord is defined in replay.go and reused here

// VerificationResult 单条状态变更的验证结果
type VerificationResult struct {
	Record       StateChangeRecord  `json:"record"`        // 原始状态变更记录 (从 replay.go 导入)
	Status       VerificationStatus `json:"status"`        // 验证状态
	Message      string             `json:"message"`       // 验证消息
	CurrentValue string             `json:"current_value"` // 当前外部状态值（可选）
	MatchDetails string             `json:"match_details"` // 匹配详情 JSON
}

// ReplayVerificationResult Replay 验证的完整结果
type ReplayVerificationResult struct {
	JobID            string               `json:"job_id"`            // Job ID
	VerificationTime string               `json:"verification_time"` // 验证时间 (RFC3339)
	OverallStatus    VerificationStatus   `json:"overall_status"`    // 整体验证状态
	Decision         ReplayDecision       `json:"decision"`          // Replay 决策
	Results          []VerificationResult `json:"results"`           // 各状态变更的验证结果
	SkippedCount     int                  `json:"skipped_count"`     // 跳过的验证数
	MatchedCount     int                  `json:"matched_count"`     // 匹配的数量
	MismatchCount    int                  `json:"mismatch_count"`    // 不匹配的数量
	ErrorCount       int                  `json:"error_count"`       // 错误的数量
	Summary          string               `json:"summary"`           // 验证摘要
}

// ExternalStateVerifier 外部状态验证器接口
type ExternalStateVerifier interface {
	// VerifyStateChange 验证单条状态变更是否与外部状态匹配
	VerifyStateChange(ctx context.Context, record StateChangeRecord) (*VerificationResult, error)
	// Name 返回验证器名称
	Name() string
}

// ToolLedgerVerifier ToolLedger 验证器 - 验证工具调用的幂等性
type ToolLedgerVerifier struct {
	// TODO: 注入 ToolLedger store 进行验证
}

// Name 返回验证器名称
func (v *ToolLedgerVerifier) Name() string {
	return "ToolLedgerVerifier"
}

// VerifyStateChange 验证工具调用状态变更
func (v *ToolLedgerVerifier) VerifyStateChange(ctx context.Context, record StateChangeRecord) (*VerificationResult, error) {
	result := &VerificationResult{
		Record: record,
		Status: VerificationStatusPending,
	}

	// 如果记录中没有外部引用，跳过验证
	if record.ExternalRef == "" && record.ResourceID == "" {
		result.Status = VerificationStatusSkipped
		result.Message = "no external reference to verify"
		return result, nil
	}

	// ToolLedger 验证：检查 idempotency_key 是否已存在
	// TODO: 实际查询 ToolLedger store
	// 这里返回已匹配作为占位实现
	result.Status = VerificationStatusMatch
	result.Message = "ToolLedger verification passed (stub)"
	result.MatchDetails = `{"verified": true, "store": "tool_ledger"}`

	return result, nil
}

// DatabaseStateVerifier 数据库状态验证器 - 验证数据库记录的版本/ETag
type DatabaseStateVerifier struct {
	// TODO: 注入数据库连接进行验证
}

// Name 返回验证器名称
func (v *DatabaseStateVerifier) Name() string {
	return "DatabaseStateVerifier"
}

// VerifyStateChange 验证数据库状态变更
func (v *DatabaseStateVerifier) VerifyStateChange(ctx context.Context, record StateChangeRecord) (*VerificationResult, error) {
	result := &VerificationResult{
		Record: record,
		Status: VerificationStatusPending,
	}

	if record.ResourceType != "database" {
		result.Status = VerificationStatusSkipped
		result.Message = "not a database resource"
		return result, nil
	}

	// 数据库验证：检查记录版本/ETag 是否匹配
	// TODO: 实际查询数据库
	// 这里返回已匹配作为占位实现
	result.Status = VerificationStatusMatch
	result.Message = "Database verification passed (stub)"
	result.MatchDetails = fmt.Sprintf(`{"resource_id": "%s", "verified": true}`, record.ResourceID)

	return result, nil
}

// ReplayVerifier Confirmation Replay 验证器
type ReplayVerifier struct {
	verifiers []ExternalStateVerifier
}

// NewReplayVerifier 创建 ReplayVerifier
func NewReplayVerifier(verifiers ...ExternalStateVerifier) *ReplayVerifier {
	if len(verifiers) == 0 {
		// 默认添加 ToolLedger 和 Database 验证器
		verifiers = append(verifiers, &ToolLedgerVerifier{}, &DatabaseStateVerifier{})
	}
	return &ReplayVerifier{
		verifiers: verifiers,
	}
}

// Verify 执行 Confirmation Replay 验证
func (v *ReplayVerifier) Verify(ctx context.Context, jobID string, stateChangesByStep map[string][]StateChangeRecord) (*ReplayVerificationResult, error) {
	result := &ReplayVerificationResult{
		JobID:            jobID,
		OverallStatus:    VerificationStatusMatch,
		Decision:         ReplayDecisionRestoreAndSkip,
		Results:          []VerificationResult{},
		VerificationTime: "TODO", // 使用 time.Now().Format(time.RFC3339)
	}

	// 统计
	totalChanges := 0

	// 遍历所有状态变更
	for stepID, changes := range stateChangesByStep {
		for i := range changes {
			// 设置记录的 stepID
			changes[i].StepID = stepID
			record := changes[i]
			totalChanges++
			vr := v.verifySingleRecord(ctx, record)
			result.Results = append(result.Results, *vr)

			// 更新统计
			switch vr.Status {
			case VerificationStatusMatch:
				result.MatchedCount++
			case VerificationStatusMismatch:
				result.MismatchCount++
			case VerificationStatusSkipped:
				result.SkippedCount++
			case VerificationStatusError:
				result.ErrorCount++
			case VerificationStatusPending:
				// 保持 pending
			}
		}
	}

	// 确定整体状态和决策
	if result.ErrorCount > 0 {
		result.OverallStatus = VerificationStatusError
		result.Decision = ReplayDecisionNeedsReview
		result.Summary = fmt.Sprintf("verification errors: %d/%d", result.ErrorCount, totalChanges)
	} else if result.MismatchCount > 0 {
		result.OverallStatus = VerificationStatusMismatch
		result.Decision = ReplayDecisionFail
		result.Summary = fmt.Sprintf("state mismatch: %d/%d", result.MismatchCount, totalChanges)
	} else if result.MatchedCount == 0 && result.SkippedCount == totalChanges {
		result.OverallStatus = VerificationStatusSkipped
		result.Decision = ReplayDecisionExecute
		result.Summary = "no state changes to verify, will execute normally"
	} else {
		result.OverallStatus = VerificationStatusMatch
		result.Decision = ReplayDecisionRestoreAndSkip
		result.Summary = fmt.Sprintf("all %d state changes verified match", result.MatchedCount)
	}

	return result, nil
}

// verifySingleRecord 验证单条状态变更记录
func (v *ReplayVerifier) verifySingleRecord(ctx context.Context, record StateChangeRecord) *VerificationResult {
	// 尝试每个验证器
	for _, verifier := range v.verifiers {
		result, err := verifier.VerifyStateChange(ctx, record)
		if err != nil {
			result.Status = VerificationStatusError
			result.Message = fmt.Sprintf("verifier %s error: %v", verifier.Name(), err)
			return result
		}

		// 如果验证器不跳过（skipped），返回结果
		if result.Status != VerificationStatusSkipped {
			return result
		}
	}

	// 所有验证器都跳过
	return &VerificationResult{
		Record:       record,
		Status:       VerificationStatusSkipped,
		Message:      "no suitable verifier found",
		MatchDetails: "{}",
	}
}

// ConfirmationReplay 执行 Confirmation Replay 验证并返回 ReplayContext
func ConfirmationReplay(ctx context.Context, rc *ReplayVerifier, jobID string, stateChangesByStep map[string][]StateChangeRecord) (*ReplayContext, *ReplayVerificationResult, error) {
	// 验证外部状态
	verificationResult, err := rc.Verify(ctx, jobID, stateChangesByStep)
	if err != nil {
		return nil, nil, fmt.Errorf("verification failed: %w", err)
	}

	// 根据验证结果决定如何处理
	switch verificationResult.Decision {
	case ReplayDecisionRestoreAndSkip:
		// 状态匹配，返回 ReplayContext 让 Runner 跳过已完成的步骤
		// ReplayContext 已包含 CompletedNodeIDs 等信息
		return &ReplayContext{
			StateChangesByStep: stateChangesByStep,
		}, verificationResult, nil

	case ReplayDecisionExecute:
		// 无状态变更或跳过验证，正常执行
		return nil, verificationResult, nil

	case ReplayDecisionFail:
		// 状态不匹配，标记失败
		return nil, verificationResult, errors.New("confirmation replay failed: external state mismatch")

	case ReplayDecisionNeedsReview:
		// 需要人工审查
		return nil, verificationResult, errors.New("confirmation replay needs manual review")
	}

	return nil, verificationResult, nil
}

// MarshalJSON 自定义 ReplayVerificationResult 的 JSON 序列化
func (r *ReplayVerificationResult) MarshalJSON() ([]byte, error) {
	type Alias ReplayVerificationResult
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}
