// Copyright 2026 fanjia1024
// OpenTelemetry integration for distributed tracing

package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func TestStartJobSpan(t *testing.T) {
	ctx := context.Background()
	jobID := "job-123"
	agentID := "agent-456"

	// Should not panic
	ctx, span := StartJobSpan(ctx, jobID, agentID)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
	_ = ctx
}

func TestStartNodeSpan(t *testing.T) {
	ctx := context.Background()
	nodeID := "node-123"
	nodeType := "llm"

	// Should not panic
	ctx, span := StartNodeSpan(ctx, nodeID, nodeType)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
	_ = ctx
}

func TestStartToolSpan(t *testing.T) {
	ctx := context.Background()
	toolName := "search"
	idempotencyKey := "key-123"

	// Should not panic
	ctx, span := StartToolSpan(ctx, toolName, idempotencyKey)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
	_ = ctx
}

func TestStartPlanSpan(t *testing.T) {
	ctx := context.Background()
	goal := "Find information about X"

	// Should not panic
	ctx, span := StartPlanSpan(ctx, goal)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
	_ = ctx
}

func TestStartCompileSpan(t *testing.T) {
	ctx := context.Background()

	// Should not panic
	ctx, span := StartCompileSpan(ctx)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
	_ = ctx
}

func TestStartInvokeSpan(t *testing.T) {
	ctx := context.Background()

	// Should not panic
	ctx, span := StartInvokeSpan(ctx)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
	_ = ctx
}

func TestStartLLMSpan(t *testing.T) {
	ctx := context.Background()
	model := "gpt-4"

	// Should not panic
	ctx, span := StartLLMSpan(ctx, model)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
	_ = ctx
}

func TestTracerName(t *testing.T) {
	// Verify that we can get a tracer
	tracer := otel.Tracer("aetheris")
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}

	// Create a span to verify tracer works (should not panic)
	_, span := tracer.Start(context.Background(), "test")
	span.End()
}

func TestSpanAttributes(t *testing.T) {
	// Test attribute creation
	attrs := []attribute.KeyValue{
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
		attribute.Bool("key3", true),
	}

	if len(attrs) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(attrs))
	}
}

func TestMultipleSpanCreation(t *testing.T) {
	ctx := context.Background()

	// Create multiple spans in sequence
	ctx, span1 := StartJobSpan(ctx, "job-1", "agent-1")
	span1.End()

	ctx, span2 := StartNodeSpan(ctx, "node-1", "llm")
	span2.End()

	ctx, span3 := StartToolSpan(ctx, "tool-1", "key-1")
	span3.End()

	// All should complete without panic
	_ = ctx
}

func TestOTelConfig(t *testing.T) {
	cfg := OTelConfig{
		ServiceName:    "test-service",
		ExportEndpoint: "localhost:4318",
		Insecure:      true,
	}

	if cfg.ServiceName != "test-service" {
		t.Errorf("expected test-service, got %s", cfg.ServiceName)
	}
	if cfg.ExportEndpoint != "localhost:4318" {
		t.Errorf("expected localhost:4318, got %s", cfg.ExportEndpoint)
	}
	if !cfg.Insecure {
		t.Error("expected Insecure to be true")
	}
}
