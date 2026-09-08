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

// tool_invocations 归档：GC 删除源记录前必须先在归档目标持久化完整副本并校验，
// 避免 ArchiveEnabled=true 时出现"归档假成功 + 源记录被删除"的数据丢失。

package jobstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrArchiveNotConfigured 归档已启用但未配置归档目标；此时必须失败，不得静默返回成功。
	ErrArchiveNotConfigured = errors.New("jobstore: tool invocation archive target not configured")

	// ErrArchiveVerificationFailed 归档目标声称写入成功，但删除前校验查不到对应副本。
	ErrArchiveVerificationFailed = errors.New("jobstore: archived tool invocation copy not verified")
)

// ArchivedToolInvocation tool_invocations 的归档副本（全字段快照 + 归档时间）。
type ArchivedToolInvocation struct {
	JobID          string
	IdempotencyKey string
	InvocationID   string
	StepID         string
	ToolName       string
	ArgsHash       string
	Status         string
	Result         []byte
	Committed      bool
	ExternalID     *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ConfirmedAt    *time.Time
	ArchivedAt     time.Time
}

// Ref 返回副本对应的源记录引用（复合主键）。
func (a ArchivedToolInvocation) Ref() ToolInvocationRef {
	return ToolInvocationRef{JobID: a.JobID, IdempotencyKey: a.IdempotencyKey}
}

// ToolInvocationArchiveSink 归档目标抽象：可插拔同库归档表、独立冷存储或对象存储。
//
// 契约：Write 返回 nil 必须意味着副本已持久化且可被 CountPersisted 读到；
// 无法保证该契约的实现不得用于生产，否则 GC 会在没有副本的情况下删除源记录。
type ToolInvocationArchiveSink interface {
	// Write 幂等写入归档副本；重复写入同一 (job_id, idempotency_key) 不得产生重复副本或报错。
	Write(ctx context.Context, records []ArchivedToolInvocation) error
	// CountPersisted 返回 refs 中已确认持久化的副本数量，供删除前校验使用。
	CountPersisted(ctx context.Context, refs []ToolInvocationRef) (int, error)
}

// pgToolInvocationArchive 默认归档目标：同库 tool_invocations_archive 表。
type pgToolInvocationArchive struct {
	pool *pgxpool.Pool
}

// NewPostgresToolInvocationArchive 基于已有连接池创建同库归档目标。
func NewPostgresToolInvocationArchive(pool *pgxpool.Pool) ToolInvocationArchiveSink {
	return &pgToolInvocationArchive{pool: pool}
}

// Write 幂等写入归档副本：主键冲突时保留首次归档内容（DO NOTHING），保证崩溃重跑安全。
func (a *pgToolInvocationArchive) Write(ctx context.Context, records []ArchivedToolInvocation) error {
	if len(records) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, r := range records {
		batch.Queue(
			`INSERT INTO tool_invocations_archive
			   (job_id, idempotency_key, invocation_id, step_id, tool_name, args_hash,
			    status, result, committed, external_id, created_at, updated_at, confirmed_at, archived_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			 ON CONFLICT (job_id, idempotency_key) DO NOTHING`,
			r.JobID, r.IdempotencyKey, r.InvocationID, r.StepID, r.ToolName, r.ArgsHash,
			r.Status, r.Result, r.Committed, r.ExternalID, r.CreatedAt, r.UpdatedAt, r.ConfirmedAt,
			time.Now().UTC(),
		)
	}
	results := a.pool.SendBatch(ctx, batch)
	defer results.Close()
	for range records {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("write tool invocation archive: %w", err)
		}
	}
	return nil
}

// CountPersisted 统计 refs 中已存在于归档表的副本数量（单次查询，避免批量回表）。
func (a *pgToolInvocationArchive) CountPersisted(ctx context.Context, refs []ToolInvocationRef) (int, error) {
	if len(refs) == 0 {
		return 0, nil
	}
	jobIDs := make([]string, 0, len(refs))
	keys := make([]string, 0, len(refs))
	for _, r := range refs {
		jobIDs = append(jobIDs, r.JobID)
		keys = append(keys, r.IdempotencyKey)
	}
	var count int
	err := a.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tool_invocations_archive a
		 JOIN unnest($1::text[], $2::text[]) AS r(job_id, idempotency_key)
		   ON a.job_id = r.job_id AND a.idempotency_key = r.idempotency_key`,
		jobIDs, keys).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count persisted tool invocation archive: %w", err)
	}
	return count, nil
}

// listArchived 按 refs 读取归档副本，refs 为空时返回 nil。
func listArchived(ctx context.Context, pool *pgxpool.Pool, refs []ToolInvocationRef) ([]ArchivedToolInvocation, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	batch := &pgx.Batch{}
	const q = `SELECT job_id, idempotency_key, invocation_id, step_id, tool_name, args_hash,
	                  status, result, committed, external_id, created_at, updated_at, confirmed_at, archived_at
	           FROM tool_invocations_archive WHERE job_id = $1 AND idempotency_key = $2`
	for _, r := range refs {
		batch.Queue(q, r.JobID, r.IdempotencyKey)
	}
	results := pool.SendBatch(ctx, batch)
	defer results.Close()

	var out []ArchivedToolInvocation
	for range refs {
		var rec ArchivedToolInvocation
		err := results.QueryRow().Scan(
			&rec.JobID, &rec.IdempotencyKey, &rec.InvocationID, &rec.StepID, &rec.ToolName, &rec.ArgsHash,
			&rec.Status, &rec.Result, &rec.Committed, &rec.ExternalID,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ConfirmedAt, &rec.ArchivedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("read tool invocation archive: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

// pgStoreOption 配置 pgStore 的可选能力。
type pgStoreOption func(*pgStore)

// WithToolInvocationArchiveSink 覆盖默认的同库归档目标（例如接入独立冷存储）。
// 传入 nil 等价于 WithoutToolInvocationArchive。
func WithToolInvocationArchiveSink(sink ToolInvocationArchiveSink) pgStoreOption {
	return func(s *pgStore) {
		if sink == nil {
			s.archive = nil
			return
		}
		s.archive = sink
	}
}

// WithoutToolInvocationArchive 显式声明该 JobStore 没有归档目标。
// 此时 ArchiveToolInvocations 返回 ErrArchiveNotConfigured，GC 在 ArchiveEnabled=true 时会中止且不删除源记录。
func WithoutToolInvocationArchive() pgStoreOption {
	return func(s *pgStore) { s.archive = nil }
}

// ArchiveToolInvocations 归档调用记录：读取源记录全字段快照 → 写入归档目标 → 校验副本已持久化。
//
// 仅当副本确认可读且数量一致时返回 nil；调用方（GC）据此才允许删除源记录。
// 未配置归档目标时返回 ErrArchiveNotConfigured，绝不静默成功。
func (s *pgStore) ArchiveToolInvocations(ctx context.Context, refs []ToolInvocationRef) error {
	if len(refs) == 0 {
		return nil
	}
	if s.archive == nil {
		return fmt.Errorf("%w: GCConfig.ArchiveEnabled requires a configured archive sink", ErrArchiveNotConfigured)
	}

	records, err := s.loadToolInvocationsForArchive(ctx, refs)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		// refs 指向的源记录已不存在（先前 GC 已处理或崩溃后重跑）：无内容可归档，不得阻塞 GC
		return nil
	}

	if err := s.archive.Write(ctx, records); err != nil {
		return err
	}

	// 删除前校验：副本必须确实可读到，否则宁可保留源记录也不能丢数据
	persisted := make([]ToolInvocationRef, 0, len(records))
	for _, r := range records {
		persisted = append(persisted, r.Ref())
	}
	count, err := s.archive.CountPersisted(ctx, persisted)
	if err != nil {
		return fmt.Errorf("verify archived tool invocations: %w", err)
	}
	if count != len(records) {
		return fmt.Errorf("%w: verified %d of %d copies", ErrArchiveVerificationFailed, count, len(records))
	}
	return nil
}

// loadToolInvocationsForArchive 读取 refs 对应的源记录全字段快照；已不存在的 ref 被跳过。
func (s *pgStore) loadToolInvocationsForArchive(ctx context.Context, refs []ToolInvocationRef) ([]ArchivedToolInvocation, error) {
	batch := &pgx.Batch{}
	const q = `SELECT job_id, idempotency_key, invocation_id, step_id, tool_name, args_hash,
	                  status, result, committed, external_id, created_at, updated_at, confirmed_at
	           FROM tool_invocations WHERE job_id = $1 AND idempotency_key = $2`
	for _, r := range refs {
		batch.Queue(q, r.JobID, r.IdempotencyKey)
	}
	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	var out []ArchivedToolInvocation
	for range refs {
		var rec ArchivedToolInvocation
		err := results.QueryRow().Scan(
			&rec.JobID, &rec.IdempotencyKey, &rec.InvocationID, &rec.StepID, &rec.ToolName, &rec.ArgsHash,
			&rec.Status, &rec.Result, &rec.Committed, &rec.ExternalID,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ConfirmedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("load tool invocations for archive: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

// ListArchivedToolInvocations 读取归档副本，用于校验、审计与恢复。
func (s *pgStore) ListArchivedToolInvocations(ctx context.Context, refs []ToolInvocationRef) ([]ArchivedToolInvocation, error) {
	return listArchived(ctx, s.pool, refs)
}

// DeleteArchivedToolInvocationsBefore 按归档保留策略清理 archived_at < cutoff 的副本，返回删除行数。
// cutoff 为零值表示永久保留，不删除任何副本。
func (s *pgStore) DeleteArchivedToolInvocationsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	if cutoff.IsZero() {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM tool_invocations_archive WHERE archived_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete archived tool invocations: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
