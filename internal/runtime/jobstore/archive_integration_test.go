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
	"context"
	"errors"
	"testing"
	"time"
)

// newArchiveTestStore 创建带归档表的 pgStore，并清空 tool_invocations 与归档副本，保证测试独立。
func newArchiveTestStore(t *testing.T, ctx context.Context, opts ...pgStoreOption) (*pgStore, func()) {
	t.Helper()
	store, err := NewPostgresStoreWithOptions(ctx, testDSN(t), 2*time.Second, opts...)
	if err != nil {
		t.Fatalf("NewPostgresStoreWithOptions: %v", err)
	}
	pg, ok := store.(*pgStore)
	if !ok {
		t.Fatal("expected *pgStore")
	}
	if _, err := pg.pool.Exec(ctx, `DELETE FROM tool_invocations_archive`); err != nil {
		t.Fatalf("clear tool_invocations_archive: %v", err)
	}
	if _, err := pg.pool.Exec(ctx, `DELETE FROM tool_invocations`); err != nil {
		t.Fatalf("clear tool_invocations: %v", err)
	}
	return pg, func() { pg.Close() }
}

// seedInvocation 写入一条 tool_invocations 记录，createdAt 用于控制 TTL 过期。
func seedInvocation(t *testing.T, ctx context.Context, pg *pgStore, jobID, idemKey string, createdAt time.Time) {
	t.Helper()
	confirmed := createdAt.Add(time.Minute)
	tag, err := pg.pool.Exec(ctx,
		`INSERT INTO tool_invocations
		   (job_id, idempotency_key, invocation_id, step_id, tool_name, args_hash,
		    status, result, committed, created_at, updated_at, confirmed_at, external_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,true,$9,$9,$10,$11)`,
		jobID, idemKey, "inv-"+idemKey, "step-1", "http_request", "hash-"+idemKey,
		"confirmed", []byte(`{"ok":true,"key":"`+idemKey+`"}`), createdAt, confirmed, "ext-"+idemKey)
	if err != nil {
		t.Fatalf("seed invocation %s/%s: %v", jobID, idemKey, err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("seed invocation %s/%s: rows affected = %d, want 1", jobID, idemKey, tag.RowsAffected())
	}
}

// TestPgStore_ArchiveToolInvocations_PersistsFullCopy 验证归档写入完整字段副本，而非空操作。
func TestPgStore_ArchiveToolInvocations_PersistsFullCopy(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	created := time.Now().UTC().Add(-100 * 24 * time.Hour).Truncate(time.Microsecond)
	seedInvocation(t, ctx, pg, "job-arch", "key-1", created)

	refs := []ToolInvocationRef{{JobID: "job-arch", IdempotencyKey: "key-1"}}
	if err := pg.ArchiveToolInvocations(ctx, refs); err != nil {
		t.Fatalf("ArchiveToolInvocations: %v", err)
	}

	archived, err := pg.ListArchivedToolInvocations(ctx, refs)
	if err != nil {
		t.Fatalf("ListArchivedToolInvocations: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("archived copies = %d, want 1", len(archived))
	}

	got := archived[0]
	if got.JobID != "job-arch" || got.IdempotencyKey != "key-1" {
		t.Errorf("primary key = %s/%s, want job-arch/key-1", got.JobID, got.IdempotencyKey)
	}
	if got.InvocationID != "inv-key-1" {
		t.Errorf("InvocationID = %q, want %q", got.InvocationID, "inv-key-1")
	}
	if got.StepID != "step-1" {
		t.Errorf("StepID = %q, want %q", got.StepID, "step-1")
	}
	if got.ToolName != "http_request" {
		t.Errorf("ToolName = %q, want %q", got.ToolName, "http_request")
	}
	if got.ArgsHash != "hash-key-1" {
		t.Errorf("ArgsHash = %q, want %q", got.ArgsHash, "hash-key-1")
	}
	if got.Status != "confirmed" {
		t.Errorf("Status = %q, want %q", got.Status, "confirmed")
	}
	if string(got.Result) != `{"ok":true,"key":"key-1"}` {
		t.Errorf("Result = %q, want %q", string(got.Result), `{"ok":true,"key":"key-1"}`)
	}
	if !got.Committed {
		t.Error("Committed = false, want true")
	}
	if got.ExternalID == nil || *got.ExternalID != "ext-key-1" {
		t.Errorf("ExternalID = %v, want %q", got.ExternalID, "ext-key-1")
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created)
	}
	if got.ConfirmedAt == nil {
		t.Error("ConfirmedAt = nil, want non-nil")
	}
	if got.ArchivedAt.IsZero() {
		t.Error("ArchivedAt is zero, want archive timestamp")
	}
}

// TestPgStore_ArchiveToolInvocations_NotConfigured 验证未配置归档目标时明确失败，不再静默返回 nil。
func TestPgStore_ArchiveToolInvocations_NotConfigured(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx, WithoutToolInvocationArchive())
	defer cleanup()

	seedInvocation(t, ctx, pg, "job-arch", "key-1", time.Now().UTC().Add(-time.Hour))

	err := pg.ArchiveToolInvocations(ctx, []ToolInvocationRef{{JobID: "job-arch", IdempotencyKey: "key-1"}})
	if err == nil {
		t.Fatal("ArchiveToolInvocations = nil, want ErrArchiveNotConfigured")
	}
	if !errors.Is(err, ErrArchiveNotConfigured) {
		t.Errorf("error = %v, want wrapped ErrArchiveNotConfigured", err)
	}
}

// TestPgStore_ArchiveToolInvocations_Idempotent 验证重复归档同一批 refs 不产生副本重复也不报错。
func TestPgStore_ArchiveToolInvocations_Idempotent(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	seedInvocation(t, ctx, pg, "job-arch", "key-1", time.Now().UTC().Add(-time.Hour))
	refs := []ToolInvocationRef{{JobID: "job-arch", IdempotencyKey: "key-1"}}

	for i := 0; i < 3; i++ {
		if err := pg.ArchiveToolInvocations(ctx, refs); err != nil {
			t.Fatalf("ArchiveToolInvocations round %d: %v", i+1, err)
		}
	}

	var count int
	if err := pg.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tool_invocations_archive`).Scan(&count); err != nil {
		t.Fatalf("count archive rows: %v", err)
	}
	if count != 1 {
		t.Errorf("archive rows = %d, want 1 (重复归档必须幂等)", count)
	}
}

// TestPgStore_ArchiveToolInvocations_MissingSourceRows 验证 refs 指向已不存在的源记录时跳过且不报错，
// 避免归档永久阻塞 GC。
func TestPgStore_ArchiveToolInvocations_MissingSourceRows(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	refs := []ToolInvocationRef{{JobID: "ghost-job", IdempotencyKey: "ghost-key"}}
	if err := pg.ArchiveToolInvocations(ctx, refs); err != nil {
		t.Fatalf("ArchiveToolInvocations on missing rows: %v", err)
	}

	var count int
	if err := pg.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tool_invocations_archive`).Scan(&count); err != nil {
		t.Fatalf("count archive rows: %v", err)
	}
	if count != 0 {
		t.Errorf("archive rows = %d, want 0", count)
	}
}

// TestPgStore_ArchiveToolInvocations_EmptyRefs 验证空输入是无操作成功。
func TestPgStore_ArchiveToolInvocations_EmptyRefs(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	if err := pg.ArchiveToolInvocations(ctx, nil); err != nil {
		t.Fatalf("ArchiveToolInvocations(nil): %v", err)
	}
}

// failingArchiveSink 写入永远失败的归档目标，用于验证失败时不删除源记录。
type failingArchiveSink struct{ err error }

func (s failingArchiveSink) Write(context.Context, []ArchivedToolInvocation) error { return s.err }

func (s failingArchiveSink) CountPersisted(context.Context, []ToolInvocationRef) (int, error) {
	return 0, s.err
}

// TestPgStore_ArchiveToolInvocations_SinkWriteFailure 验证归档目标写入失败时返回错误。
func TestPgStore_ArchiveToolInvocations_SinkWriteFailure(t *testing.T) {
	ctx := context.Background()
	sinkErr := errors.New("cold storage unavailable")
	pg, cleanup := newArchiveTestStore(t, ctx, WithToolInvocationArchiveSink(failingArchiveSink{err: sinkErr}))
	defer cleanup()

	seedInvocation(t, ctx, pg, "job-arch", "key-1", time.Now().UTC().Add(-time.Hour))

	err := pg.ArchiveToolInvocations(ctx, []ToolInvocationRef{{JobID: "job-arch", IdempotencyKey: "key-1"}})
	if err == nil {
		t.Fatal("ArchiveToolInvocations = nil, want sink error")
	}
	if !errors.Is(err, sinkErr) {
		t.Errorf("error = %v, want wrapped %v", err, sinkErr)
	}
}

// unverifiedArchiveSink 声称写入成功但校验时查不到副本，模拟"假成功"归档目标。
type unverifiedArchiveSink struct{}

func (unverifiedArchiveSink) Write(context.Context, []ArchivedToolInvocation) error { return nil }

func (unverifiedArchiveSink) CountPersisted(context.Context, []ToolInvocationRef) (int, error) {
	return 0, nil
}

// TestPgStore_ArchiveToolInvocations_VerifiesPersistedCopy 验证写入后必须校验副本已持久化，
// 归档目标假成功时返回错误，从而阻止 GC 删除源记录。
func TestPgStore_ArchiveToolInvocations_VerifiesPersistedCopy(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx, WithToolInvocationArchiveSink(unverifiedArchiveSink{}))
	defer cleanup()

	seedInvocation(t, ctx, pg, "job-arch", "key-1", time.Now().UTC().Add(-time.Hour))

	err := pg.ArchiveToolInvocations(ctx, []ToolInvocationRef{{JobID: "job-arch", IdempotencyKey: "key-1"}})
	if err == nil {
		t.Fatal("ArchiveToolInvocations = nil, want verification error for unverified copy")
	}
	if !errors.Is(err, ErrArchiveVerificationFailed) {
		t.Errorf("error = %v, want wrapped ErrArchiveVerificationFailed", err)
	}
}

// TestGC_ArchiveEnabled_DeletesOnlyAfterVerifiedArchive 端到端验证：GC 开启归档时，
// 源记录被删除且归档副本完整保留。
func TestGC_ArchiveEnabled_DeletesOnlyAfterVerifiedArchive(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	expired := time.Now().UTC().Add(-200 * 24 * time.Hour)
	seedInvocation(t, ctx, pg, "job-gc", "key-expired", expired)
	fresh := time.Now().UTC()
	seedInvocation(t, ctx, pg, "job-gc", "key-fresh", fresh)

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveEnabled: true, BatchSize: 100}
	if err := GC(ctx, pg, cfg); err != nil {
		t.Fatalf("GC: %v", err)
	}

	// 过期记录已从源表删除
	var srcCount int
	if err := pg.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tool_invocations WHERE job_id = 'job-gc' AND idempotency_key = 'key-expired'`).
		Scan(&srcCount); err != nil {
		t.Fatalf("count source rows: %v", err)
	}
	if srcCount != 0 {
		t.Errorf("expired source rows = %d, want 0", srcCount)
	}

	// 未过期记录保留
	var freshCount int
	if err := pg.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tool_invocations WHERE job_id = 'job-gc' AND idempotency_key = 'key-fresh'`).
		Scan(&freshCount); err != nil {
		t.Fatalf("count fresh rows: %v", err)
	}
	if freshCount != 1 {
		t.Errorf("fresh source rows = %d, want 1", freshCount)
	}

	// 归档副本存在且内容完整
	archived, err := pg.ListArchivedToolInvocations(ctx,
		[]ToolInvocationRef{{JobID: "job-gc", IdempotencyKey: "key-expired"}})
	if err != nil {
		t.Fatalf("ListArchivedToolInvocations: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("archived copies = %d, want 1 (删除前必须已归档)", len(archived))
	}
	if archived[0].ToolName != "http_request" || string(archived[0].Result) != `{"ok":true,"key":"key-expired"}` {
		t.Errorf("archived copy content = %q/%q, want full snapshot",
			archived[0].ToolName, string(archived[0].Result))
	}
}

// TestGC_ArchiveEnabled_ArchiveFailureBlocksDelete 验证归档失败时 GC 返回错误且源记录不被删除。
func TestGC_ArchiveEnabled_ArchiveFailureBlocksDelete(t *testing.T) {
	ctx := context.Background()
	sinkErr := errors.New("archive target down")
	pg, cleanup := newArchiveTestStore(t, ctx, WithToolInvocationArchiveSink(failingArchiveSink{err: sinkErr}))
	defer cleanup()

	seedInvocation(t, ctx, pg, "job-gc", "key-expired", time.Now().UTC().Add(-200*24*time.Hour))

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveEnabled: true, BatchSize: 100}
	err := GC(ctx, pg, cfg)
	if err == nil {
		t.Fatal("GC = nil, want archive failure error")
	}
	if !errors.Is(err, sinkErr) {
		t.Errorf("GC error = %v, want wrapped %v", err, sinkErr)
	}

	var count int
	if err := pg.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tool_invocations`).Scan(&count); err != nil {
		t.Fatalf("count source rows: %v", err)
	}
	if count != 1 {
		t.Errorf("source rows after failed archive = %d, want 1 (归档失败不得删除)", count)
	}
}

// TestGC_ArchiveEnabled_NotConfiguredBlocksDelete 验证归档已启用但目标未配置时 GC 失败且不删除。
func TestGC_ArchiveEnabled_NotConfiguredBlocksDelete(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx, WithoutToolInvocationArchive())
	defer cleanup()

	seedInvocation(t, ctx, pg, "job-gc", "key-expired", time.Now().UTC().Add(-200*24*time.Hour))

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveEnabled: true, BatchSize: 100}
	err := GC(ctx, pg, cfg)
	if err == nil {
		t.Fatal("GC = nil, want ErrArchiveNotConfigured")
	}
	if !errors.Is(err, ErrArchiveNotConfigured) {
		t.Errorf("GC error = %v, want wrapped ErrArchiveNotConfigured", err)
	}

	var count int
	if err := pg.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tool_invocations`).Scan(&count); err != nil {
		t.Fatalf("count source rows: %v", err)
	}
	if count != 1 {
		t.Errorf("source rows = %d, want 1 (未配置归档目标不得删除)", count)
	}
}

// TestGC_ArchiveCrashBeforeDelete_NoDataLoss 模拟归档成功后进程崩溃（未执行删除），
// 重跑 GC 不得丢失数据：副本仍为单份，源记录最终被删除。
func TestGC_ArchiveCrashBeforeDelete_NoDataLoss(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	expired := time.Now().UTC().Add(-200 * 24 * time.Hour)
	seedInvocation(t, ctx, pg, "job-gc", "key-1", expired)
	refs := []ToolInvocationRef{{JobID: "job-gc", IdempotencyKey: "key-1"}}

	// 崩溃点：归档已持久化，删除未执行
	if err := pg.ArchiveToolInvocations(ctx, refs); err != nil {
		t.Fatalf("ArchiveToolInvocations before crash: %v", err)
	}

	// 恢复后重跑：重复归档必须幂等
	if err := pg.ArchiveToolInvocations(ctx, refs); err != nil {
		t.Fatalf("ArchiveToolInvocations after recovery: %v", err)
	}
	var archiveCount int
	if err := pg.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tool_invocations_archive`).Scan(&archiveCount); err != nil {
		t.Fatalf("count archive rows: %v", err)
	}
	if archiveCount != 1 {
		t.Errorf("archive rows after recovery = %d, want 1", archiveCount)
	}

	// 完整 GC 收敛：源删除、副本保留
	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveEnabled: true, BatchSize: 100}
	if err := GC(ctx, pg, cfg); err != nil {
		t.Fatalf("GC after recovery: %v", err)
	}

	var srcCount int
	if err := pg.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tool_invocations`).Scan(&srcCount); err != nil {
		t.Fatalf("count source rows: %v", err)
	}
	if srcCount != 0 {
		t.Errorf("source rows = %d, want 0", srcCount)
	}
	archived, err := pg.ListArchivedToolInvocations(ctx, refs)
	if err != nil {
		t.Fatalf("ListArchivedToolInvocations: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("archived copies = %d, want 1 (崩溃恢复不得丢失数据)", len(archived))
	}
}

// TestPgStore_ArchiveToolInvocations_PartialBatchFailure 验证批量归档中部分 refs 失败时整批报错，
// 不产生"部分归档 + 全部删除"的数据丢失。
func TestPgStore_ArchiveToolInvocations_PartialBatchFailure(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	expired := time.Now().UTC().Add(-200 * 24 * time.Hour)
	for i := 0; i < 5; i++ {
		seedInvocation(t, ctx, pg, "job-batch", "key-"+string(rune('a'+i)), expired)
	}

	refs := make([]ToolInvocationRef, 0, 6)
	for i := 0; i < 5; i++ {
		refs = append(refs, ToolInvocationRef{JobID: "job-batch", IdempotencyKey: "key-" + string(rune('a'+i))})
	}

	if err := pg.ArchiveToolInvocations(ctx, refs); err != nil {
		t.Fatalf("ArchiveToolInvocations: %v", err)
	}

	var count int
	if err := pg.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tool_invocations_archive`).Scan(&count); err != nil {
		t.Fatalf("count archive rows: %v", err)
	}
	if count != 5 {
		t.Errorf("archive rows = %d, want 5 (批量归档必须覆盖全部 refs)", count)
	}
}

// TestPgStore_ArchiveRetention_DeletesOnlyExpiredCopies 验证归档保留策略：
// ArchiveTTLDays<=0 表示永久保留，>0 时仅清理超期副本。
func TestPgStore_ArchiveRetention_DeletesOnlyExpiredCopies(t *testing.T) {
	ctx := context.Background()
	pg, cleanup := newArchiveTestStore(t, ctx)
	defer cleanup()

	old := time.Now().UTC().Add(-400 * 24 * time.Hour)
	seedInvocation(t, ctx, pg, "job-ret", "key-old", old)
	recent := time.Now().UTC().Add(-100 * 24 * time.Hour)
	seedInvocation(t, ctx, pg, "job-ret", "key-recent", recent)

	refs := []ToolInvocationRef{
		{JobID: "job-ret", IdempotencyKey: "key-old"},
		{JobID: "job-ret", IdempotencyKey: "key-recent"},
	}
	if err := pg.ArchiveToolInvocations(ctx, refs); err != nil {
		t.Fatalf("ArchiveToolInvocations: %v", err)
	}

	// 永久保留：cutoff 为零值时不删除任何副本
	deleted, err := pg.DeleteArchivedToolInvocationsBefore(ctx, time.Time{})
	if err != nil {
		t.Fatalf("DeleteArchivedToolInvocationsBefore(zero): %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted with zero cutoff = %d, want 0", deleted)
	}

	// 将一条副本的 archived_at 回拨，验证按归档时间清理
	if _, err := pg.pool.Exec(ctx,
		`UPDATE tool_invocations_archive SET archived_at = $1 WHERE idempotency_key = 'key-old'`,
		time.Now().UTC().Add(-400*24*time.Hour)); err != nil {
		t.Fatalf("backdate archived_at: %v", err)
	}

	deleted, err = pg.DeleteArchivedToolInvocationsBefore(ctx, time.Now().UTC().Add(-365*24*time.Hour))
	if err != nil {
		t.Fatalf("DeleteArchivedToolInvocationsBefore: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	remaining, err := pg.ListArchivedToolInvocations(ctx,
		[]ToolInvocationRef{{JobID: "job-ret", IdempotencyKey: "key-recent"}})
	if err != nil {
		t.Fatalf("ListArchivedToolInvocations: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("remaining archive copies = %d, want 1 (未超期副本必须保留)", len(remaining))
	}
}
