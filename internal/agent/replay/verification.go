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
	"time"
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

// ToolLedgerEntry Tool 账本查询返回的只读记录
type ToolLedgerEntry struct {
	IdempotencyKey string
	ExternalRef    string
	Status         string // started | success | failure | confirmed
	ResultHash     string // 结果摘要，供比较
}

// ErrToolLedgerNotFound 查询无对应记录
var ErrToolLedgerNotFound = errors.New("tool ledger entry not found")

// ToolLedgerLookup 只读账本查询接口，供 ToolLedgerVerifier 注入。
// 生产实现委托 executor.ToolInvocationStore（适配器转换）；测试可 mock。
type ToolLedgerLookup interface {
	// LookupByExternalRef 按 external_ref 或 resource_id 查询工具调用记录
	LookupByExternalRef(ctx context.Context, externalRef string) (*ToolLedgerEntry, error)
}

// ToolLedgerVerifier ToolLedger 验证器 - 验证工具调用的幂等性。
// 必须注入 ToolLedgerLookup；未注入时 VerifyStateChange 返回 error（禁止静默 match）。
type ToolLedgerVerifier struct {
	lookup ToolLedgerLookup
}

// NewToolLedgerVerifier 创建带 store 的 ToolLedgerVerifier
func NewToolLedgerVerifier(lookup ToolLedgerLookup) *ToolLedgerVerifier {
	return &ToolLedgerVerifier{lookup: lookup}
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

	// 无外部引用时跳过
	if record.ExternalRef == "" && record.ResourceID == "" {
		result.Status = VerificationStatusSkipped
		result.Message = "no external reference to verify"
		return result, nil
	}

	// 非工具相关记录跳过（交给 DatabaseStateVerifier 等处理）
	if record.ToolName == "" && record.ResourceType != "tool" && record.ResourceType != "tool_invocation" {
		result.Status = VerificationStatusSkipped
		result.Message = "not a tool invocation record"
		return result, nil
	}

	// 未注入 store 时禁止静默 match
	if v.lookup == nil {
		result.Status = VerificationStatusError
		result.Message = "ToolLedgerVerifier: no store configured, cannot verify"
		return result, nil
	}

	// 确定查询键：优先 external_ref，回退 resource_id
	ref := record.ExternalRef
	if ref == "" {
		ref = record.ResourceID
	}

	entry, err := v.lookup.LookupByExternalRef(ctx, ref)
	if err != nil {
		if errors.Is(err, ErrToolLedgerNotFound) {
			result.Status = VerificationStatusMismatch
			result.Message = fmt.Sprintf("tool ledger entry not found for ref %q", ref)
			return result, nil
		}
		result.Status = VerificationStatusError
		result.Message = fmt.Sprintf("tool ledger lookup error: %v", err)
		return result, nil
	}

	// 检查状态：仅 confirmed/success 表示外部世界已变更
	switch entry.Status {
	case "confirmed", "success":
		result.Status = VerificationStatusMatch
		result.Message = "tool ledger entry confirmed"
		result.MatchDetails = fmt.Sprintf(`{"status": "%s", "result_hash": "%s"}`, entry.Status, entry.ResultHash)
	case "started":
		result.Status = VerificationStatusPending
		result.Message = "tool ledger entry is in-flight"
	default:
		result.Status = VerificationStatusMismatch
		result.Message = fmt.Sprintf("tool ledger entry status %q does not indicate committed state", entry.Status)
	}

	return result, nil
}

// DatabaseStateLookup 只读数据库资源查询接口，供 DatabaseStateVerifier 注入。
type DatabaseStateLookup interface {
	// Lookup 查询资源当前的 version 和 etag
	Lookup(ctx context.Context, resourceID string) (version, etag string, err error)
}

// ErrDatabaseResourceNotFound 查询无对应资源
var ErrDatabaseResourceNotFound = errors.New("database resource not found")

// DatabaseStateVerifier 数据库状态验证器 - 验证数据库记录的版本/ETag。
// 必须注入 DatabaseStateLookup；未注入时返回 error（禁止静默 match）。
type DatabaseStateVerifier struct {
	lookup DatabaseStateLookup
}

// NewDatabaseStateVerifier 创建带 lookup 的 DatabaseStateVerifier
func NewDatabaseStateVerifier(lookup DatabaseStateLookup) *DatabaseStateVerifier {
	return &DatabaseStateVerifier{lookup: lookup}
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

	if record.ResourceID == "" {
		result.Status = VerificationStatusSkipped
		result.Message = "no resource ID to verify"
		return result, nil
	}

	// 未注入 lookup 时禁止静默 match
	if v.lookup == nil {
		result.Status = VerificationStatusError
		result.Message = "DatabaseStateVerifier: no lookup configured, cannot verify"
		return result, nil
	}

	currentVersion, currentEtag, err := v.lookup.Lookup(ctx, record.ResourceID)
	if err != nil {
		if errors.Is(err, ErrDatabaseResourceNotFound) {
			result.Status = VerificationStatusMismatch
			result.Message = fmt.Sprintf("database resource %q not found", record.ResourceID)
			return result, nil
		}
		result.Status = VerificationStatusError
		result.Message = fmt.Sprintf("database lookup error: %v", err)
		return result, nil
	}

	// 比较 version 和 etag
	mismatchMsg := ""
	if record.Version != "" && currentVersion != "" && record.Version != currentVersion {
		mismatchMsg = fmt.Sprintf("version mismatch: expected %s, got %s", record.Version, currentVersion)
	}
	if record.Etag != "" && currentEtag != "" && record.Etag != currentEtag {
		if mismatchMsg != "" {
			mismatchMsg += "; "
		}
		mismatchMsg += fmt.Sprintf("etag mismatch: expected %s, got %s", record.Etag, currentEtag)
	}

	if mismatchMsg != "" {
		result.Status = VerificationStatusMismatch
		result.Message = mismatchMsg
		return result, nil
	}

	result.Status = VerificationStatusMatch
	result.Message = "database state verified"
	result.MatchDetails = fmt.Sprintf(`{"resource_id": "%s", "version": "%s", "etag": "%s"}`, record.ResourceID, currentVersion, currentEtag)

	return result, nil
}

// ReplayVerifier Confirmation Replay 验证器
type ReplayVerifier struct {
	verifiers []ExternalStateVerifier
}

// NewReplayVerifier 创建 ReplayVerifier。
// 不传 verifiers 时使用空列表，所有状态变更将被 skip → decision=Execute（正常执行）。
// 生产环境应显式注入带 store 的 ToolLedgerVerifier / DatabaseStateVerifier。
func NewReplayVerifier(verifiers ...ExternalStateVerifier) *ReplayVerifier {
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
		VerificationTime: time.Now().UTC().Format(time.RFC3339),
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
