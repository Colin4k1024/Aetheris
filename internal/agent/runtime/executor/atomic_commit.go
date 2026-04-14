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

package executor

import (
	"context"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

// LedgerEventSink 写入 Ledger 原子性事件：ledger_acquired（执行权获取）和 ledger_committed（结果提交）
// 用于检测 crash window 中的孤岛事件，实现 at-most-once 保证的原子性协议。
type LedgerEventSink interface {
	// AppendLedgerAcquired 写入 ledger_acquired 事件（在 Acquire 返回 AllowExecute 之前调用）
	AppendLedgerAcquired(ctx context.Context, jobID string, ver int64, payload *LedgerAcquiredPayload) error
	// AppendLedgerCommitted 写入 ledger_committed 事件（在 Commit 成功之后调用）
	AppendLedgerCommitted(ctx context.Context, jobID string, ver int64, payload *LedgerCommittedPayload) error
	// ListEvents 列出 job 所有事件（用于检测孤岛 ledger_acquired）
	ListEvents(ctx context.Context, jobID string) ([]jobstore.JobEvent, int64, error)
}

// AtomicCommitFunc 执行 tool 并原子性写入 ledger_acquired → commit → ledger_committed
// 崩溃恢复时：孤岛的 ledger_acquired 无 ledger_committed → 视为 in-progress
type AtomicCommitFunc func(ctx context.Context, ledger InvocationLedger, sink LedgerEventSink, jobID, stepID, toolName, idempotencyKey string, ver int64, argsHash string, toolFn func() (string, error)) error

// LedgerAcquiredPayload ledger_acquired 事件 payload
type LedgerAcquiredPayload = jobstore.LedgerAcquiredPayload

// LedgerCommittedPayload ledger_committed 事件 payload
type LedgerCommittedPayload = jobstore.LedgerCommittedPayload

// DefaultAtomicCommit 标准的原子提交实现：ledger_acquired → tool → ledger_committed
// 步骤：
// 1. 写入 ledger_acquired 事件
// 2. 执行 toolFn（返回结果字符串和可能的错误）
// 3. Ledger.Commit()
// 4. 写入 ledger_committed 事件
// 崩溃恢复时：若事件流有 ledger_acquired 无 ledger_committed，视为 in-progress。
func DefaultAtomicCommit(ctx context.Context, ledger InvocationLedger, sink LedgerEventSink, jobID, stepID, toolName, idempotencyKey string, ver int64, argsHash string, toolFn func() (string, error)) error {
	invocationID := newInvocationID()
	// Step 1: 写入 ledger_acquired（标识执行权已获取，tool 尚未执行）
	{
		payload := &LedgerAcquiredPayload{
			InvocationID:   invocationID,
			JobID:          jobID,
			StepID:         stepID,
			ToolName:       toolName,
			IdempotencyKey: idempotencyKey,
		}
		if err := sink.AppendLedgerAcquired(ctx, jobID, ver, payload); err != nil {
			return err
		}
		ver++
	}
	// Step 2: 执行 tool
	resultStr, err := toolFn()
	if err != nil {
		// 执行失败时不写 ledger_committed，等下次重试
		return err
	}
	// Step 3: Ledger.Commit
	if err := ledger.Commit(ctx, invocationID, idempotencyKey, []byte(resultStr)); err != nil {
		return err
	}
	// Step 4: 写入 ledger_committed（标识结果已提交，tool 执行完成）
	{
		payload := &LedgerCommittedPayload{
			InvocationID:   invocationID,
			JobID:          jobID,
			StepID:         stepID,
			ToolName:       toolName,
			IdempotencyKey: idempotencyKey,
		}
		if err := sink.AppendLedgerCommitted(ctx, jobID, ver, payload); err != nil {
			// Commit 已成功，ledger_committed 写入失败仅记录日志，不返回错误
			// 下次 replay 时会通过 store 中的 committed=true 判断而非事件
			return nil
		}
	}
	return nil
}

func newInvocationID() string {
	return "inv-" + time.Now().Format("20060102150405.000000000")
}
