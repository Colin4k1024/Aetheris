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

// archiveOrderStore 记录 GC 对归档/删除的调用顺序与失败注入点。
type archiveOrderStore struct {
	JobStore
	invocations  []ToolInvocationRef
	archiveCalls int
	deleteCalls  int
	archived     []ToolInvocationRef
	deleted      []ToolInvocationRef
	archiveErr   error
}

func (s *archiveOrderStore) ListExpiredToolInvocations(context.Context, time.Time, int) ([]ToolInvocationRef, error) {
	if len(s.invocations) == 0 {
		return nil, nil
	}
	refs := s.invocations
	s.invocations = nil
	return refs, nil
}

func (s *archiveOrderStore) ArchiveToolInvocations(_ context.Context, refs []ToolInvocationRef) error {
	s.archiveCalls++
	if s.archiveErr != nil {
		return s.archiveErr
	}
	s.archived = append(s.archived, refs...)
	return nil
}

func (s *archiveOrderStore) DeleteToolInvocations(_ context.Context, refs []ToolInvocationRef) error {
	s.deleteCalls++
	s.deleted = append(s.deleted, refs...)
	return nil
}

// TestGC_ArchiveError_StopsBeforeDelete 验证归档返回错误时 GC 立即中止，绝不调用删除。
func TestGC_ArchiveError_StopsBeforeDelete(t *testing.T) {
	archiveErr := errors.New("archive write failed")
	store := &archiveOrderStore{
		JobStore:   NewMemoryStore(),
		archiveErr: archiveErr,
	}
	store.invocations = []ToolInvocationRef{{JobID: "job-1", IdempotencyKey: "key-1"}}

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveEnabled: true, BatchSize: 10}
	err := GC(context.Background(), store, cfg)
	if err == nil {
		t.Fatal("GC = nil, want archive error")
	}
	if !errors.Is(err, archiveErr) {
		t.Errorf("GC error = %v, want wrapped %v", err, archiveErr)
	}
	if store.deleteCalls != 0 {
		t.Errorf("DeleteToolInvocations calls = %d, want 0 (归档失败不得删除)", store.deleteCalls)
	}
	if len(store.deleted) != 0 {
		t.Errorf("deleted refs = %v, want none", store.deleted)
	}
}

// TestGC_ArchiveEnabled_ArchivesBeforeDeleting 验证开启归档时先归档后删除，且两者覆盖同一批 refs。
func TestGC_ArchiveEnabled_ArchivesBeforeDeleting(t *testing.T) {
	store := &archiveOrderStore{JobStore: NewMemoryStore()}
	refs := []ToolInvocationRef{
		{JobID: "job-1", IdempotencyKey: "key-1"},
		{JobID: "job-1", IdempotencyKey: "key-2"},
	}
	store.invocations = append([]ToolInvocationRef(nil), refs...)

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveEnabled: true, BatchSize: 10}
	if err := GC(context.Background(), store, cfg); err != nil {
		t.Fatalf("GC: %v", err)
	}

	if store.archiveCalls != 1 {
		t.Errorf("archive calls = %d, want 1", store.archiveCalls)
	}
	if len(store.archived) != 2 {
		t.Fatalf("archived refs = %d, want 2", len(store.archived))
	}
	if len(store.deleted) != 2 {
		t.Fatalf("deleted refs = %d, want 2", len(store.deleted))
	}
	for i := range refs {
		if store.archived[i] != refs[i] {
			t.Errorf("archived[%d] = %+v, want %+v", i, store.archived[i], refs[i])
		}
		if store.deleted[i] != refs[i] {
			t.Errorf("deleted[%d] = %+v, want %+v", i, store.deleted[i], refs[i])
		}
	}
}

// TestGC_ArchiveDisabled_SkipsArchive 验证未开启归档时不调用归档，保持原有直接清理语义。
func TestGC_ArchiveDisabled_SkipsArchive(t *testing.T) {
	store := &archiveOrderStore{JobStore: NewMemoryStore()}
	store.invocations = []ToolInvocationRef{{JobID: "job-1", IdempotencyKey: "key-1"}}

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveEnabled: false, BatchSize: 10}
	if err := GC(context.Background(), store, cfg); err != nil {
		t.Fatalf("GC: %v", err)
	}
	if store.archiveCalls != 0 {
		t.Errorf("archive calls = %d, want 0", store.archiveCalls)
	}
	if store.deleteCalls != 1 {
		t.Errorf("delete calls = %d, want 1", store.deleteCalls)
	}
}

// TestGCConfig_ArchiveTTLDaysDefault 验证归档保留策略默认永久保留（不清理归档副本）。
func TestGCConfig_ArchiveTTLDaysDefault(t *testing.T) {
	cfg := DefaultGCConfig()
	if cfg.ArchiveTTLDays != 0 {
		t.Errorf("DefaultGCConfig().ArchiveTTLDays = %d, want 0 (永久保留)", cfg.ArchiveTTLDays)
	}
}

// archiveRetentionStore 验证 GC 对归档保留策略的接线。
type archiveRetentionStore struct {
	JobStore
	retentionCalls int
	lastCutoff     time.Time
	deleted        int
	retentionErr   error
}

func (s *archiveRetentionStore) DeleteArchivedToolInvocationsBefore(_ context.Context, cutoff time.Time) (int, error) {
	s.retentionCalls++
	s.lastCutoff = cutoff
	if s.retentionErr != nil {
		return 0, s.retentionErr
	}
	return s.deleted, nil
}

// TestGC_ArchiveRetention_ZeroKeepsCopies 验证 ArchiveTTLDays<=0 时不清理归档副本（默认永久保留）。
func TestGC_ArchiveRetention_ZeroKeepsCopies(t *testing.T) {
	store := &archiveRetentionStore{JobStore: NewMemoryStore()}

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveTTLDays: 0, BatchSize: 10}
	if err := GC(context.Background(), store, cfg); err != nil {
		t.Fatalf("GC: %v", err)
	}
	if store.retentionCalls != 0 {
		t.Errorf("DeleteArchivedToolInvocationsBefore calls = %d, want 0 (默认永久保留)", store.retentionCalls)
	}
}

// TestGC_ArchiveRetention_CleansExpiredCopies 验证 ArchiveTTLDays>0 时按归档时间清理副本。
func TestGC_ArchiveRetention_CleansExpiredCopies(t *testing.T) {
	store := &archiveRetentionStore{JobStore: NewMemoryStore(), deleted: 3}

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveTTLDays: 365, BatchSize: 10}
	if err := GC(context.Background(), store, cfg); err != nil {
		t.Fatalf("GC: %v", err)
	}
	if store.retentionCalls != 1 {
		t.Fatalf("DeleteArchivedToolInvocationsBefore calls = %d, want 1", store.retentionCalls)
	}
	wantCutoff := time.Now().UTC().AddDate(0, 0, -365)
	if store.lastCutoff.Before(wantCutoff.Add(-time.Minute)) || store.lastCutoff.After(wantCutoff.Add(time.Minute)) {
		t.Errorf("cutoff = %v, want ~%v", store.lastCutoff, wantCutoff)
	}
}

// TestGC_ArchiveRetention_FailurePropagates 验证归档清理失败时 GC 返回错误。
func TestGC_ArchiveRetention_FailurePropagates(t *testing.T) {
	retentionErr := errors.New("archive table unreachable")
	store := &archiveRetentionStore{JobStore: NewMemoryStore(), retentionErr: retentionErr}

	cfg := GCConfig{Enable: true, TTLDays: 90, ArchiveTTLDays: 365, BatchSize: 10}
	err := GC(context.Background(), store, cfg)
	if err == nil {
		t.Fatal("GC = nil, want retention error")
	}
	if !errors.Is(err, retentionErr) {
		t.Errorf("GC error = %v, want wrapped %v", err, retentionErr)
	}
}

// TestErrArchiveNotConfigured_IsSentinel 验证归档相关哨兵错误可被 errors.Is 识别。
func TestErrArchiveNotConfigured_IsSentinel(t *testing.T) {
	if ErrArchiveNotConfigured == nil {
		t.Fatal("ErrArchiveNotConfigured is nil")
	}
	if ErrArchiveVerificationFailed == nil {
		t.Fatal("ErrArchiveVerificationFailed is nil")
	}
	if !errors.Is(errors.Join(ErrArchiveNotConfigured, errors.New("ctx")), ErrArchiveNotConfigured) {
		t.Error("ErrArchiveNotConfigured not matchable via errors.Is")
	}
}
