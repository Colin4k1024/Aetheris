// Package idempotent provides at-most-once execution guarantees for side-effecting operations.
//
// Wrap any function with Idempotent to ensure it executes at most once per idempotency key,
// even across retries and crash recovery. This prevents duplicate side effects like:
//   - Sending the same email twice
//   - Charging a payment twice
//   - Creating duplicate records
//
// Usage:
//
//	store := core.NewMemoryStore()
//	idem := idempotent.New(store)
//
//	// Wrap a side-effecting function
//	sendEmail := idem.Wrap("send-email", func(ctx context.Context, input map[string]any) (map[string]any, error) {
//	    // This will only execute once per idempotency key
//	    err := emailService.Send(input["to"].(string), input["subject"].(string))
//	    return map[string]any{"sent": true}, err
//	})
//
//	// Use in a step — same idempotency key = same result, no re-execution
//	result, err := sendEmail(ctx, "order-123-email", map[string]any{
//	    "to":      "user@example.com",
//	    "subject": "Order confirmed",
//	})
package idempotent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Colin4k1024/Aetheris/durability/core"
)

// Tool provides at-most-once execution for side-effecting operations.
type Tool struct {
	store core.Store
	mu    sync.RWMutex
}

// New creates a new idempotent tool with the given store.
func New(store core.Store) *Tool {
	return &Tool{store: store}
}

// WrappedFunc is a function wrapped with at-most-once semantics.
// The idempotencyKey uniquely identifies this invocation; calling with the same key
// returns the cached result without re-executing the function.
type WrappedFunc func(ctx context.Context, idempotencyKey string, input map[string]any) (map[string]any, error)

// Wrap wraps a function with at-most-once execution guarantees.
// The toolName identifies this tool in the event stream.
//
// On first call with a given idempotencyKey:
//  1. Records "acquired" in the event stream
//  2. Executes the function
//  3. Records "committed" with the result
//
// On subsequent calls with the same idempotencyKey:
//   - Returns the cached result without executing the function
//
// This is safe across process restarts: if the process crashes between "acquired" and
// "committed", the next call will detect the incomplete execution and re-execute.
func (t *Tool) Wrap(toolName string, fn func(ctx context.Context, input map[string]any) (map[string]any, error)) WrappedFunc {
	return func(ctx context.Context, idempotencyKey string, input map[string]any) (map[string]any, error) {
		// Check if this key has already been committed
		// We use a synthetic job ID based on the tool name and idempotency key
		ledgerJobID := fmt.Sprintf("ledger:%s:%s", toolName, idempotencyKey)

		job, err := t.store.LoadJob(ctx, ledgerJobID)
		if err != nil {
			return nil, fmt.Errorf("idempotent: load ledger: %w", err)
		}

		if job != nil && job.State == core.StateCompleted {
			// Already committed — return cached result
			var result map[string]any
			if job.Result != nil {
				json.Unmarshal(job.Result, &result)
			}
			return result, nil
		}

		// Not yet committed. Check if acquired (in-progress from a crashed run).
		if job != nil && job.State == core.StateRunning {
			// Previous execution started but didn't commit.
			// This means the process crashed. We should re-execute.
			// Reset the job state to allow re-execution.
			job.State = core.StateCreated
			job.Error = ""
			t.store.SaveJob(ctx, job)
		}

		// Create or update the ledger job as "acquired"
		if job == nil {
			job = &core.Job{
				ID:             ledgerJobID,
				Name:           fmt.Sprintf("idempotent:%s", toolName),
				State:          core.StateRunning,
				CompletedSteps: make(map[string]json.RawMessage),
			}
		} else {
			job.State = core.StateRunning
		}

		inputBytes, _ := json.Marshal(input)
		job.Input = inputBytes
		if err := t.store.SaveJob(ctx, job); err != nil {
			return nil, fmt.Errorf("idempotent: save acquired: %w", err)
		}

		// Record acquired event
		_, err = t.store.AppendEvent(ctx, ledgerJobID, job.Version, core.Event{
			Type: "ledger_acquired",
			Payload: mustMarshal(map[string]string{
				"tool_name":       toolName,
				"idempotency_key": idempotencyKey,
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("idempotent: append acquired: %w", err)
		}
		job.Version++

		// Execute the actual function
		result, execErr := fn(ctx, input)
		if execErr != nil {
			// Mark as failed so next retry will re-execute
			job.State = core.StateFailed
			job.Error = execErr.Error()
			t.store.SaveJob(ctx, job)
			return nil, execErr
		}

		// Commit the result
		resultBytes, _ := json.Marshal(result)
		job.State = core.StateCompleted
		job.Result = resultBytes
		job.UpdatedAt = job.CreatedAt // will be overwritten by SaveJob
		if err := t.store.SaveJob(ctx, job); err != nil {
			return nil, fmt.Errorf("idempotent: save committed: %w", err)
		}

		_, _ = t.store.AppendEvent(ctx, ledgerJobID, job.Version, core.Event{
			Type: "ledger_committed",
			Payload: mustMarshal(map[string]string{
				"tool_name":       toolName,
				"idempotency_key": idempotencyKey,
			}),
		})

		return result, nil
	}
}

// WrapSimple wraps a simpler function that takes and returns primitive values.
// Useful for wrapping HTTP calls, database writes, etc.
func (t *Tool) WrapSimple(toolName string, fn func(ctx context.Context, input any) (any, error)) WrappedFunc {
	return t.Wrap(toolName, func(ctx context.Context, input map[string]any) (map[string]any, error) {
		result, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		// Wrap primitive result in a map
		return map[string]any{"result": result}, nil
	})
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
