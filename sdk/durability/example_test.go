package durability_test

import (
	"context"
	"fmt"
	"log"

	"github.com/Colin4k1024/Aetheris/durability/core"
	"github.com/Colin4k1024/Aetheris/durability/idempotent"
)

// Example demonstrates the basic usage of the Aetheris durability SDK.
func Example() {
	ctx := context.Background()

	// 1. Create a store (in-memory for demo; use postgres.NewStore for production)
	store := core.NewMemoryStore()

	// 2. Create a runner
	runner := core.NewRunner(store)

	// 3. Start a job
	job, err := runner.Start(ctx, "process-order", map[string]any{
		"order_id": "ORD-123",
		"amount":   float64(99.99),
	})
	if err != nil {
		log.Fatal(err)
	}

	// 4. Define steps (each step receives state and returns updated state)
	steps := []core.Step{
		{
			ID:   "validate",
			Name: "Validate Order",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				orderID := state["order_id"].(string)
				fmt.Printf("Validating order %s...\n", orderID)
				return map[string]any{"validated": true}, nil
			},
		},
		{
			ID:         "charge",
			Name:       "Charge Payment",
			MaxRetries: 3,
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				amount := state["amount"].(float64)
				fmt.Printf("Charging $%.2f...\n", amount)
				return map[string]any{"charged": true, "transaction_id": "TXN-456"}, nil
			},
		},
		{
			ID:   "ship",
			Name: "Create Shipment",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				txnID := state["transaction_id"].(string)
				fmt.Printf("Creating shipment for txn %s...\n", txnID)
				return map[string]any{"shipped": true, "tracking": "TRACK-789"}, nil
			},
		},
	}

	// 5. Execute the job
	result, err := runner.Execute(ctx, job.ID, steps)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Validated: %v\n", result["validated"])
	fmt.Printf("Charged: %v\n", result["charged"])
	fmt.Printf("Shipped: %v\n", result["shipped"])
	fmt.Printf("Tracking: %v\n", result["tracking"])

	// Output:
	// Validating order ORD-123...
	// Charging $99.99...
	// Creating shipment for txn TXN-456...
	// Validated: true
	// Charged: true
	// Shipped: true
	// Tracking: TRACK-789
}

// Example_crashRecovery demonstrates crash recovery with checkpointing.
func Example_crashRecovery() {
	ctx := context.Background()
	store := core.NewMemoryStore()
	runner := core.NewRunner(store)

	job, _ := runner.Start(ctx, "batch-process", map[string]any{"processed": float64(0)})

	crashed := false
	steps := []core.Step{
		{
			ID: "batch-1",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				return map[string]any{"processed": float64(100)}, nil
			},
		},
		{
			ID: "batch-2",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				if !crashed {
					crashed = true
					return nil, fmt.Errorf("process killed!")
				}
				return map[string]any{"processed": float64(200)}, nil
			},
		},
		{
			ID: "batch-3",
			Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
				return map[string]any{"processed": float64(300)}, nil
			},
		},
	}

	// First attempt: crashes at batch-2
	_, err := runner.Execute(ctx, job.ID, steps)
	fmt.Printf("First attempt error: %v\n", err)

	// Resume: batch-1 is skipped (checkpointed), batch-2 retries successfully
	result, err := runner.Execute(ctx, job.ID, steps)
	fmt.Printf("Resume result: processed=%.0f\n", result["processed"])

	// Output:
	// First attempt error: durability: step batch-2 failed: process killed!
	// Resume result: processed=300
}

// Example_idempotentTool demonstrates at-most-once tool execution.
func Example_idempotentTool() {
	ctx := context.Background()
	store := core.NewMemoryStore()
	idem := idempotent.New(store)

	callCount := 0

	// Wrap a side-effecting function (e.g., sending an email)
	sendEmail := idem.Wrap("send-email", func(ctx context.Context, input map[string]any) (map[string]any, error) {
		callCount++
		return map[string]any{"sent": true, "message_id": "MSG-001"}, nil
	})

	// First call: executes the function
	result1, _ := sendEmail(ctx, "order-123-confirmation", map[string]any{
		"to": "user@example.com",
	})

	// Second call with same key: returns cached result, doesn't re-execute
	result2, _ := sendEmail(ctx, "order-123-confirmation", map[string]any{
		"to": "user@example.com",
	})

	fmt.Printf("Call count: %d\n", callCount)
	fmt.Printf("Result 1: %v\n", result1["sent"])
	fmt.Printf("Result 2: %v\n", result2["sent"])
	fmt.Printf("Same result: %v\n", result1["message_id"] == result2["message_id"])

	// Output:
	// Call count: 1
	// Result 1: true
	// Result 2: true
	// Same result: true
}
