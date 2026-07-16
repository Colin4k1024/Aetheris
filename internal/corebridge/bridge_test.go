package corebridge

import (
	"os"
	"testing"
)

func TestUseRustCore_DefaultFalse(t *testing.T) {
	os.Unsetenv("AETHERIS_USE_RUST_CORE")
	if UseRustCore() {
		t.Error("expected UseRustCore() to be false by default")
	}
}

func TestUseRustCore_Enabled(t *testing.T) {
	os.Setenv("AETHERIS_USE_RUST_CORE", "true")
	defer os.Unsetenv("AETHERIS_USE_RUST_CORE")

	if !UseRustCore() {
		t.Error("expected UseRustCore() to be true when env is set")
	}
}

func TestNewJobStore_GoNative(t *testing.T) {
	os.Unsetenv("AETHERIS_USE_RUST_CORE")

	store, err := NewJobStore("postgres://localhost/test")
	if err != nil {
		t.Fatalf("NewJobStore failed: %v", err)
	}
	defer store.Close()

	// Should be GoNativeJobStore
	if _, ok := store.(*GoNativeJobStore); !ok {
		t.Errorf("expected GoNativeJobStore, got %T", store)
	}
}

func TestNewJobStore_RustCore(t *testing.T) {
	os.Setenv("AETHERIS_USE_RUST_CORE", "true")
	defer os.Unsetenv("AETHERIS_USE_RUST_CORE")

	store, err := NewJobStore("postgres://localhost/test")
	if err != nil {
		t.Fatalf("NewJobStore failed: %v", err)
	}
	defer store.Close()

	// Should be RustJobStore
	if _, ok := store.(*RustJobStore); !ok {
		t.Errorf("expected RustJobStore, got %T", store)
	}
}

func TestNewEffectStore_GoNative(t *testing.T) {
	os.Unsetenv("AETHERIS_USE_RUST_CORE")

	store, err := NewEffectStore()
	if err != nil {
		t.Fatalf("NewEffectStore failed: %v", err)
	}
	defer store.Close()

	if _, ok := store.(*GoNativeEffectStore); !ok {
		t.Errorf("expected GoNativeEffectStore, got %T", store)
	}
}

func TestNewEffectStore_RustCore(t *testing.T) {
	os.Setenv("AETHERIS_USE_RUST_CORE", "true")
	defer os.Unsetenv("AETHERIS_USE_RUST_CORE")

	store, err := NewEffectStore()
	if err != nil {
		t.Fatalf("NewEffectStore failed: %v", err)
	}
	defer store.Close()

	if _, ok := store.(*RustEffectStore); !ok {
		t.Errorf("expected RustEffectStore, got %T", store)
	}
}
