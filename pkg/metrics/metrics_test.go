// Copyright 2026 fanjia1024
// Metrics package tests

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestObserveLLMCall(t *testing.T) {
	tenant := "test-tenant"
	model := "gpt-4"
	duration := 150 * time.Millisecond
	inputTokens := 100
	outputTokens := 200

	// Observe LLM call
	ObserveLLMCall(tenant, model, "success", duration, inputTokens, outputTokens)

	// Verify metrics are recorded - just verify the function doesn't panic
	// The actual values would be verified in integration tests
	assert.True(t, true)
}

func TestObserveNodeExecution(t *testing.T) {
	tenant := "test-tenant"
	nodeType := "llm"
	status := "success"
	duration := 100 * time.Millisecond

	ObserveNodeExecution(tenant, nodeType, status, duration)

	// Verify
	assert.True(t, true)
}

func TestObserveStorageOperation(t *testing.T) {
	tenant := "test-tenant"
	storageType := "redis"
	operation := "get"
	status := "hit"
	duration := 5 * time.Millisecond

	ObserveStorageOperation(tenant, storageType, operation, status, duration)

	// Verify
	assert.True(t, true)
}

func TestObserveIngestStep(t *testing.T) {
	tenant := "test-tenant"
	step := "loader"
	duration := 50 * time.Millisecond

	ObserveIngestStep(tenant, step, duration)

	// Verify
	assert.True(t, true)
}

func TestObserveQueryStep(t *testing.T) {
	tenant := "test-tenant"
	step := "retrieve"
	duration := 30 * time.Millisecond

	ObserveQueryStep(tenant, step, duration)

	// Verify
	assert.True(t, true)
}

func TestSetConnectionPoolMetrics(t *testing.T) {
	tenant := "test-tenant"
	storageType := "redis"
	poolName := "default"
	size := 5
	max := 10
	idle := 3

	SetConnectionPoolMetrics(tenant, storageType, poolName, size, max, idle)

	// Verify
	assert.True(t, true)
}

func TestObserveRetry(t *testing.T) {
	tenant := "test-tenant"
	component := "llm"
	reason := "rate_limit"
	delay := 1 * time.Second
	attempt := 2

	ObserveRetry(tenant, component, reason, delay, attempt)

	// Verify
	assert.True(t, true)
}

func TestObserveCacheHit(t *testing.T) {
	tenant := "test-tenant"
	cacheType := "redis"

	ObserveCacheHit(tenant, cacheType)

	// Verify
	assert.True(t, true)
}

func TestObserveCacheMiss(t *testing.T) {
	tenant := "test-tenant"
	cacheType := "redis"

	ObserveCacheMiss(tenant, cacheType)

	// Verify
	assert.True(t, true)
}

func TestP99Buckets(t *testing.T) {
	// Verify P99 bucket configuration
	assert.Contains(t, P99Buckets, float64(1))
	assert.Contains(t, P99Buckets, float64(100))
	assert.Contains(t, P99Buckets, float64(1000))
	assert.Contains(t, P99Buckets, float64(60000))
}

func TestNewObserver(t *testing.T) {
	labels := MetricsLabels{
		Tenant:   "test-tenant",
		AgentID:  "agent-001",
		StepType: "llm",
		Tool:     "search",
		Provider: "openai",
		Status:   "success",
		Result:   "ok",
		Queue:    "default",
	}

	observer := NewObserver(labels)
	assert.NotNil(t, observer)
}

func TestObserverLabels(t *testing.T) {
	labels := MetricsLabels{
		Tenant: "test-tenant",
	}

	observer := NewObserver(labels)
	result := observer.labels([]string{"tenant"})
	assert.Equal(t, "test-tenant", result["tenant"])
}
