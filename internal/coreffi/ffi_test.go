package coreffi

import (
	"testing"
)

func TestJobStoreCreateAndClose(t *testing.T) {
	// Test that we can create and close a JobStore handle
	// Note: This requires the Rust library to be linked
	store, err := NewJobStore("postgres://localhost/test")
	if err != nil {
		t.Fatalf("NewJobStore failed: %v", err)
	}
	defer store.Close()

	if store.handle == 0 {
		t.Fatal("expected non-zero handle")
	}
}

func TestEffectStoreCreateAndClose(t *testing.T) {
	store, err := NewEffectStore()
	if err != nil {
		t.Fatalf("NewEffectStore failed: %v", err)
	}
	defer store.Close()

	if store.handle == 0 {
		t.Fatal("expected non-zero handle")
	}
}

func TestExecutorCreateAndClose(t *testing.T) {
	jobStore, err := NewJobStore("postgres://localhost/test")
	if err != nil {
		t.Fatalf("NewJobStore failed: %v", err)
	}
	defer jobStore.Close()

	executor, err := NewExecutor(jobStore)
	if err != nil {
		t.Fatalf("NewExecutor failed: %v", err)
	}
	defer executor.Close()

	if executor.handle == 0 {
		t.Fatal("expected non-zero handle")
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify error code constants match expected values
	if ErrSuccess != 0 {
		t.Errorf("ErrSuccess = %d, want 0", ErrSuccess)
	}
	if ErrInvalidInput != -1 {
		t.Errorf("ErrInvalidInput = %d, want -1", ErrInvalidInput)
	}
	if ErrVersionConflict != -2 {
		t.Errorf("ErrVersionConflict = %d, want -2", ErrVersionConflict)
	}
}
