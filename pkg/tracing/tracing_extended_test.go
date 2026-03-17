// Copyright 2026 fanjia1024
// OpenTelemetry tracing tests

package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartIngestStepSpan(t *testing.T) {
	ctx := context.Background()
	ingestID := "test-ingest-123"
	step := "loader"

	span, ctx := StartIngestStepSpan(ctx, ingestID, step)
	assert.NotNil(t, span)
	assert.NotNil(t, ctx)

	// End span without error
	span.End(nil)
}

func TestStartQueryStepSpan(t *testing.T) {
	ctx := context.Background()
	queryID := "test-query-456"
	step := "retrieve"

	span, ctx := StartQueryStepSpan(ctx, queryID, step)
	assert.NotNil(t, span)
	assert.NotNil(t, ctx)

	span.End(nil)
}

func TestLLMSpan(t *testing.T) {
	ctx := context.Background()
	model := "gpt-4"
	prompt := "Hello, world!"

	span, ctx := StartLLMCallSpan(ctx, model, prompt)
	assert.NotNil(t, span)
	assert.NotNil(t, ctx)

	// Set tokens
	span.SetTokens(10, 20)

	// Set latency
	span.SetLatency(150 * time.Millisecond)

	// Set retries
	span.SetRetries(0)

	// End span without error
	span.End(nil)
}

func TestDAGNodeSpan(t *testing.T) {
	ctx := context.Background()
	nodeID := "node-001"
	nodeType := "llm"

	span, ctx := StartDAGNodeSpan(ctx, nodeID, nodeType)
	assert.NotNil(t, span)
	assert.NotNil(t, ctx)

	// Set input/output size
	span.SetInputSize(1000)
	span.SetOutputSize(500)

	span.End(nil)
}

func TestDAGNodeSpanWithError(t *testing.T) {
	ctx := context.Background()
	nodeID := "node-002"
	nodeType := "tool"

	span, _ := StartDAGNodeSpan(ctx, nodeID, nodeType)

	// End span with error
	testErr := assert.AnError
	span.End(testErr)
}

func TestIngestPipelineSpan(t *testing.T) {
	ctx := context.Background()
	ingestID := "test-ingest-789"
	step := "parser"

	span, ctx := StartIngestStepSpan(ctx, ingestID, step)
	assert.NotNil(t, span)

	// Simulate work
	time.Sleep(10 * time.Millisecond)

	// End with error
	span.End(assert.AnError)
}

func TestQueryPipelineSpan(t *testing.T) {
	ctx := context.Background()
	queryID := "test-query-101"
	step := "generate"

	span, ctx := StartQueryStepSpan(ctx, queryID, step)
	assert.NotNil(t, span)

	// Simulate work
	time.Sleep(10 * time.Millisecond)

	// End without error
	span.End(nil)
}

func TestGetTraceID(t *testing.T) {
	ctx := context.Background()

	// Without span, should return empty
	traceID := GetTraceID(ctx)
	assert.Equal(t, "", traceID)

	// With span
	span, ctx := StartIngestStepSpan(ctx, "test", "loader")
	traceID = GetTraceID(ctx)
	// Should have trace ID now (though it's implementation specific)
	_ = traceID
	span.End(nil)
}
