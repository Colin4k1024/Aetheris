package jobstore

import (
	"context"
	"testing"
	"time"
)

func TestDefaultGCConfig(t *testing.T) {
	cfg := DefaultGCConfig()

	if cfg.Enable != false {
		t.Errorf("expected Enable=false, got %v", cfg.Enable)
	}
	if cfg.TTLDays != 90 {
		t.Errorf("expected TTLDays=90, got %d", cfg.TTLDays)
	}
	if cfg.ArchiveEnabled != false {
		t.Errorf("expected ArchiveEnabled=false, got %v", cfg.ArchiveEnabled)
	}
	if cfg.RunInterval != 24*time.Hour {
		t.Errorf("expected RunInterval=24h, got %v", cfg.RunInterval)
	}
	if cfg.BatchSize != 1000 {
		t.Errorf("expected BatchSize=1000, got %d", cfg.BatchSize)
	}
}

func TestToolInvocationRef(t *testing.T) {
	ref := ToolInvocationRef{
		JobID:          "job-1",
		IdempotencyKey: "idem-1",
	}

	if ref.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", ref.JobID)
	}
	if ref.IdempotencyKey != "idem-1" {
		t.Errorf("expected idem-1, got %s", ref.IdempotencyKey)
	}
}

func TestGCConfig(t *testing.T) {
	cfg := GCConfig{
		Enable:         true,
		TTLDays:        30,
		ArchiveEnabled: true,
		RunInterval:    time.Hour,
		BatchSize:      500,
	}

	if !cfg.Enable {
		t.Error("expected Enable=true")
	}
	if cfg.TTLDays != 30 {
		t.Errorf("expected 30, got %d", cfg.TTLDays)
	}
	if !cfg.ArchiveEnabled {
		t.Error("expected ArchiveEnabled=true")
	}
	if cfg.RunInterval != time.Hour {
		t.Errorf("expected 1h, got %v", cfg.RunInterval)
	}
	if cfg.BatchSize != 500 {
		t.Errorf("expected 500, got %d", cfg.BatchSize)
	}
}

type fakeLifecycleStore struct {
	JobStore
	expiredBatches [][]ToolInvocationRef
	archiveCalls   int
	deleteCalls    int
}

func (f *fakeLifecycleStore) ListExpiredToolInvocations(ctx context.Context, cutoff time.Time, limit int) ([]ToolInvocationRef, error) {
	if len(f.expiredBatches) == 0 {
		return nil, nil
	}
	batch := f.expiredBatches[0]
	f.expiredBatches = f.expiredBatches[1:]
	return append([]ToolInvocationRef(nil), batch...), nil
}

func (f *fakeLifecycleStore) ArchiveToolInvocations(ctx context.Context, refs []ToolInvocationRef) error {
	f.archiveCalls++
	return nil
}

func (f *fakeLifecycleStore) DeleteToolInvocations(ctx context.Context, refs []ToolInvocationRef) error {
	f.deleteCalls++
	return nil
}

func TestGC_NoopWhenDisabled(t *testing.T) {
	store := &fakeLifecycleStore{JobStore: NewMemoryStore()}
	err := GC(context.Background(), store, GCConfig{Enable: false})
	if err != nil {
		t.Fatalf("GC disabled should return nil, got: %v", err)
	}
	if store.archiveCalls != 0 || store.deleteCalls != 0 {
		t.Fatalf("expected no lifecycle calls, archive=%d delete=%d", store.archiveCalls, store.deleteCalls)
	}
}

func TestGC_ArchiveAndDelete(t *testing.T) {
	store := &fakeLifecycleStore{
		JobStore: NewMemoryStore(),
		expiredBatches: [][]ToolInvocationRef{
			{{JobID: "job_1", IdempotencyKey: "inv_1"}, {JobID: "job_1", IdempotencyKey: "inv_2"}},
		},
	}
	cfg := GCConfig{
		Enable:         true,
		TTLDays:        90,
		ArchiveEnabled: true,
		BatchSize:      1000,
	}
	if err := GC(context.Background(), store, cfg); err != nil {
		t.Fatalf("GC failed: %v", err)
	}
	if store.archiveCalls != 1 {
		t.Fatalf("archive calls = %d, want 1", store.archiveCalls)
	}
	if store.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", store.deleteCalls)
	}
}

func TestGC_DeleteOnly(t *testing.T) {
	store := &fakeLifecycleStore{
		JobStore: NewMemoryStore(),
		expiredBatches: [][]ToolInvocationRef{
			{{JobID: "job_1", IdempotencyKey: "inv_1"}},
		},
	}
	cfg := GCConfig{
		Enable:         true,
		ArchiveEnabled: false,
		BatchSize:      10,
	}
	if err := GC(context.Background(), store, cfg); err != nil {
		t.Fatalf("GC failed: %v", err)
	}
	if store.archiveCalls != 0 {
		t.Fatalf("archive calls = %d, want 0", store.archiveCalls)
	}
	if store.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", store.deleteCalls)
	}
}
