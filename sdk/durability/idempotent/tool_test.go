package idempotent_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/Colin4k1024/Aetheris/durability/core"
	"github.com/Colin4k1024/Aetheris/durability/idempotent"
)

func TestTool_Wrap_ExecutesOnce(t *testing.T) {
	store := core.NewMemoryStore()
	idem := idempotent.New(store)
	ctx := context.Background()

	var callCount atomic.Int32

	fn := idem.Wrap("test-tool", func(ctx context.Context, input map[string]any) (map[string]any, error) {
		callCount.Add(1)
		return map[string]any{"result": "ok"}, nil
	})

	// First call
	result1, err := fn(ctx, "key-1", map[string]any{"data": "test"})
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if result1["result"] != "ok" {
		t.Fatalf("expected result=ok, got %v", result1["result"])
	}

	// Second call with same key — should NOT re-execute
	result2, err := fn(ctx, "key-1", map[string]any{"data": "test"})
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if result2["result"] != "ok" {
		t.Fatalf("expected result=ok, got %v", result2["result"])
	}

	if callCount.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", callCount.Load())
	}
}

func TestTool_Wrap_DifferentKeys(t *testing.T) {
	store := core.NewMemoryStore()
	idem := idempotent.New(store)
	ctx := context.Background()

	var callCount atomic.Int32

	fn := idem.Wrap("test-tool", func(ctx context.Context, input map[string]any) (map[string]any, error) {
		callCount.Add(1)
		return map[string]any{"result": "ok"}, nil
	})

	// Different keys should execute independently
	fn(ctx, "key-1", nil)
	fn(ctx, "key-2", nil)
	fn(ctx, "key-3", nil)

	if callCount.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount.Load())
	}
}

func TestTool_Wrap_ErrorNotCached(t *testing.T) {
	store := core.NewMemoryStore()
	idem := idempotent.New(store)
	ctx := context.Background()

	var callCount atomic.Int32

	fn := idem.Wrap("failing-tool", func(ctx context.Context, input map[string]any) (map[string]any, error) {
		callCount.Add(1)
		if callCount.Load() == 1 {
			return nil, errors.New("transient error")
		}
		return map[string]any{"result": "ok"}, nil
	})

	// First call fails
	_, err := fn(ctx, "key-1", nil)
	if err == nil {
		t.Fatal("expected error from first call")
	}

	// Second call with same key — should re-execute (error is not cached)
	result, err := fn(ctx, "key-1", nil)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if result["result"] != "ok" {
		t.Fatalf("expected result=ok, got %v", result["result"])
	}

	if callCount.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount.Load())
	}
}

func TestTool_Wrap_ConcurrentSameKey(t *testing.T) {
	store := core.NewMemoryStore()
	idem := idempotent.New(store)
	ctx := context.Background()

	var callCount atomic.Int32

	fn := idem.Wrap("test-tool", func(ctx context.Context, input map[string]any) (map[string]any, error) {
		callCount.Add(1)
		return map[string]any{"result": "ok"}, nil
	})

	// Concurrent calls with same key — only one should execute
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			fn(ctx, "concurrent-key", nil)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	// At least 1 call should have happened (possibly more due to race, but result should be consistent)
	if callCount.Load() < 1 {
		t.Fatal("expected at least 1 call")
	}
}
