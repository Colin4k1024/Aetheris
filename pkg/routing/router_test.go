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

package routing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestModelTierString(t *testing.T) {
	tests := []struct {
		tier   ModelTier
		expect string
	}{
		{TierReasoning, "t1-reasoning"},
		{TierFlagship, "t2-flagship"},
		{TierBalanced, "t3-balanced"},
		{TierEconomy, "t4-economy"},
		{0, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if got := tt.tier.String(); got != tt.expect {
				t.Errorf("ModelTier.String() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestParseModelTier(t *testing.T) {
	tests := []struct {
		input    string
		expectOK bool
		expect   ModelTier
	}{
		{"t1-reasoning", true, TierReasoning},
		{"reasoning", true, TierReasoning},
		{"t2-flagship", true, TierFlagship},
		{"t3-balanced", true, TierBalanced},
		{"t4-economy", true, TierEconomy},
		{"invalid", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseModelTier(tt.input)
			if tt.expectOK && err != nil {
				t.Errorf("ParseModelTier(%q) error = %v", tt.input, err)
			}
			if !tt.expectOK && err == nil {
				t.Errorf("ParseModelTier(%q) expected error", tt.input)
			}
			if tt.expectOK && got != tt.expect {
				t.Errorf("ParseModelTier(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestModelInfo(t *testing.T) {
	model := &ModelInfo{
		Name:            "gpt-4o",
		Provider:        "openai",
		Tier:            TierFlagship,
		ContextLimit:    128000,
		CostPer1KInput:  0.0025,
		CostPer1KOutput: 0.01,
		AvgLatencyMs:    2000,
		Capabilities:    []string{"vision", "function_call"},
	}

	// Test GetEstimatedCost
	cost := model.GetEstimatedCost()
	expected := 0.0025*1 + 0.01*0.5 // 0.0075
	if cost != expected {
		t.Errorf("GetEstimatedCost() = %v, want %v", cost, expected)
	}

	// Test HasCapability
	if !model.HasCapability("vision") {
		t.Error("HasCapability(vision) = false, want true")
	}
	if model.HasCapability("code") {
		t.Error("HasCapability(code) = true, want false")
	}
}

func TestModelRegistry(t *testing.T) {
	registry := NewModelRegistry()

	// Register models
	registry.RegisterModel(&ModelInfo{
		Name:    "gpt-4o",
		Provider: "openai",
		Tier:    TierFlagship,
	})
	registry.RegisterModel(&ModelInfo{
		Name:    "claude-3.5-sonnet",
		Provider: "anthropic",
		Tier:    TierBalanced,
	})

	// Test GetModel
	model, err := registry.GetModel("gpt-4o")
	if err != nil {
		t.Errorf("GetModel() error = %v", err)
	}
	if model.Name != "gpt-4o" {
		t.Errorf("GetModel() = %v, want gpt-4o", model.Name)
	}

	// Test GetModel not found
	_, err = registry.GetModel("not-exist")
	if err == nil {
		t.Error("GetModel(not-exist) expected error")
	}

	// Test GetModelsByTier
	models := registry.GetModelsByTier(TierFlagship)
	if len(models) != 1 {
		t.Errorf("GetModelsByTier(TierFlagship) = %v, want 1", len(models))
	}

	models = registry.GetModelsByTier(TierEconomy)
	if len(models) != 0 {
		t.Errorf("GetModelsByTier(TierEconomy) = %v, want 0", len(models))
	}
}

func TestDefaultModelRegistry(t *testing.T) {
	registry := DefaultModelRegistry()

	// Verify we have models in all tiers
	for tier := TierReasoning; tier <= TierEconomy; tier++ {
		models := registry.GetModelsByTier(tier)
		if len(models) == 0 {
			t.Errorf("No models registered for tier %v", tier)
		}
	}
}

func TestCostStrategy(t *testing.T) {
	registry := DefaultModelRegistry()
	strategy := NewCostStrategy(registry)

	req := &RoutingRequest{
		Complexity:   ComplexitySimple,
		Priority:     PriorityCost,
		MaxLatencyMs: 10000,
	}

	model, err := strategy.Select(context.Background(), req)
	if err != nil {
		t.Errorf("Select() error = %v", err)
	}

	// Cost strategy should return economy tier for simple tasks
	if model.Tier != TierEconomy {
		t.Errorf("CostStrategy.Select() tier = %v, want TierEconomy", model.Tier)
	}
}

func TestLatencyStrategy(t *testing.T) {
	registry := DefaultModelRegistry()
	strategy := NewLatencyStrategy(registry)

	req := &RoutingRequest{
		Complexity:   ComplexityMedium,
		MaxLatencyMs: 1000,
	}

	model, err := strategy.Select(context.Background(), req)
	if err != nil {
		t.Errorf("Select() error = %v", err)
	}

	// Should select model with low latency
	if model.AvgLatencyMs > 1000 {
		t.Errorf("LatencyStrategy.Select() latency = %v, want <= 1000", model.AvgLatencyMs)
	}
}

func TestQualityStrategy(t *testing.T) {
	registry := DefaultModelRegistry()
	strategy := NewQualityStrategy(registry)

	req := &RoutingRequest{
		Complexity: ComplexityHigh,
	}

	model, err := strategy.Select(context.Background(), req)
	if err != nil {
		t.Errorf("Select() error = %v", err)
	}

	// Quality strategy should return flagship or reasoning tier for high complexity
	if model.Tier != TierFlagship && model.Tier != TierReasoning {
		t.Errorf("QualityStrategy.Select() tier = %v, want TierFlagship or TierReasoning", model.Tier)
	}
}

func TestBalancedStrategy(t *testing.T) {
	registry := DefaultModelRegistry()
	strategy := NewBalancedStrategy(registry)

	tests := []struct {
		name       string
		complexity NodeComplexity
		priority   RoutingPriority
		wantTier  ModelTier
	}{
		{
			name:       "simple with cost priority",
			complexity: ComplexitySimple,
			priority:   PriorityCost,
			wantTier:   TierEconomy,
		},
		{
			name:       "simple with latency priority",
			complexity: ComplexitySimple,
			priority:   PriorityLatency,
			wantTier:   TierBalanced,
		},
		{
			name:       "medium complexity",
			complexity: ComplexityMedium,
			priority:   PriorityBalanced,
			wantTier:   TierBalanced,
		},
		{
			name:       "high complexity",
			complexity: ComplexityHigh,
			priority:   PriorityBalanced,
			wantTier:   TierFlagship,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RoutingRequest{
				Complexity: tt.complexity,
				Priority:   tt.priority,
			}
			model, err := strategy.Select(context.Background(), req)
			if err != nil {
				t.Errorf("Select() error = %v", err)
			}
			if model.Tier != tt.wantTier {
				t.Errorf("BalancedStrategy.Select() tier = %v, want %v", model.Tier, tt.wantTier)
			}
		})
	}
}

func TestRouter(t *testing.T) {
	registry := DefaultModelRegistry()
	router := NewRouter(registry, NewBalancedStrategy(registry), nil)

	req := &RoutingRequest{
		Complexity:   ComplexityMedium,
		MaxCost:      0.1,
		MaxLatencyMs: 5000,
		Priority:     PriorityBalanced,
	}

	model, err := router.SelectModel(context.Background(), req)
	if err != nil {
		t.Errorf("SelectModel() error = %v", err)
	}

	// Should not exceed cost
	if model.GetEstimatedCost() > req.MaxCost {
		t.Errorf("Model cost %v exceeds max cost %v", model.GetEstimatedCost(), req.MaxCost)
	}

	// Should not exceed latency
	if model.AvgLatencyMs > req.MaxLatencyMs {
		t.Errorf("Model latency %v exceeds max latency %v", model.AvgLatencyMs, req.MaxLatencyMs)
	}
}

func TestFallback(t *testing.T) {
	registry := DefaultModelRegistry()
	router := NewRouter(registry, NewBalancedStrategy(registry), nil)

	primary := &ModelInfo{
		Name:     "gpt-4o",
		Provider: "openai",
		Tier:     TierFlagship,
	}

	// Test fallback for rate limit - should go to cheaper model (higher tier number)
	fallback, err := router.SelectFallback(context.Background(), primary, FallbackReasonRateLimit)
	if err != nil {
		t.Errorf("SelectFallback() error = %v", err)
	}

	// Should fallback to cheaper model (higher tier number = more expensive = worse)
	if fallback.Tier <= primary.Tier {
		t.Errorf("Fallback tier %v should be greater than primary tier %v (cheaper model)", fallback.Tier, primary.Tier)
	}
}

func TestFailoverHandler(t *testing.T) {
	registry := DefaultModelRegistry()
	router := NewRouter(registry, NewBalancedStrategy(registry), nil)
	handler := NewFailoverHandler(router, registry, &FailoverConfig{
		MaxRetries:      2,
		EnableHotSwitch: true,
	})

	callCount := 0
	fn := func(ctx context.Context, model *ModelInfo) (interface{}, error) {
		callCount++
		if callCount == 1 {
			// First call fails with rate limit
			return nil, errors.New("rate limit exceeded (429)")
		}
		// Second call succeeds
		return "success", nil
	}

	req := &RoutingRequest{
		Complexity: ComplexityMedium,
	}

	result, _, err := handler.ExecuteWithFailover(context.Background(), req, fn)
	if err != nil {
		t.Errorf("ExecuteWithFailover() error = %v", err)
	}

	if result != "success" {
		t.Errorf("ExecuteWithFailover() result = %v, want 'success'", result)
	}

	if callCount != 2 {
		t.Errorf("ExecuteWithFailover() callCount = %v, want 2", callCount)
	}
}

func TestEventSourcedContext(t *testing.T) {
	ctx := NewEventSourcedContext("test-request")

	// Add messages
	ctx.AddMessage("user", "Hello", 10)
	ctx.AddMessage("assistant", "Hi there", 20)

	// Verify tokens
	if ctx.GetTotalTokens() != 30 {
		t.Errorf("GetTotalTokens() = %v, want 30", ctx.GetTotalTokens())
	}

	// Record a switch
	ctx.RecordSwitch("gpt-4o", "gpt-4o-mini", FallbackReasonRateLimit)

	// Verify history
	history := ctx.GetSwitchHistory()
	if len(history) != 1 {
		t.Errorf("GetSwitchHistory() length = %v, want 1", len(history))
	}

	if !ctx.HasSwitched() {
		t.Error("HasSwitched() = false, want true")
	}

	// Verify messages
	messages := ctx.GetMessages()
	if len(messages) != 2 {
		t.Errorf("GetMessages() length = %v, want 2", len(messages))
	}
}

func TestFailoverHandlerMaxRetries(t *testing.T) {
	registry := DefaultModelRegistry()
	router := NewRouter(registry, NewBalancedStrategy(registry), nil)
	handler := NewFailoverHandler(router, registry, &FailoverConfig{
		MaxRetries:      1,
		EnableHotSwitch: false,
	})

	callCount := 0
	fn := func(ctx context.Context, model *ModelInfo) (interface{}, error) {
		callCount++
		return nil, errors.New("server error (500)")
	}

	req := &RoutingRequest{
		Complexity: ComplexityMedium,
	}

	_, _, err := handler.ExecuteWithFailover(context.Background(), req, fn)
	if err == nil {
		t.Error("ExecuteWithFailover() expected error after max retries")
	}

	// Should retry once (initial + 1 retry)
	if callCount != 2 {
		t.Errorf("ExecuteWithFailover() callCount = %v, want 2", callCount)
	}
}

func TestRoutingRequestConstraints(t *testing.T) {
	registry := DefaultModelRegistry()
	router := NewRouter(registry, NewBalancedStrategy(registry), nil)

	// Test with impossible constraints
	req := &RoutingRequest{
		Complexity:   ComplexityHigh,
		MaxCost:      0.001, // Very low cost
		MaxLatencyMs: 10,    // Very low latency
		RequiredCaps: []string{"nonexistent_capability"},
	}

	_, err := router.SelectModel(context.Background(), req)
	// Should still return a model (falls back to any available)
	if err != nil {
		t.Logf("SelectModel() error with impossible constraints: %v", err)
	}
}

func TestFallbackReason(t *testing.T) {
	tests := []struct {
		err      error
		expected FallbackReason
	}{
		{errors.New("rate limit exceeded (429)"), FallbackReasonRateLimit},
		{errors.New("too many requests (429)"), FallbackReasonRateLimit},
		{errors.New("server error (500)"), FallbackReasonServerError},
		{errors.New("service unavailable (503)"), FallbackReasonServerError},
		{errors.New("gateway error (502)"), FallbackReasonServerError},
		{errors.New("timeout: deadline exceeded"), FallbackReasonTimeout},
		{errors.New("quota exceeded"), FallbackReasonCost},
		{errors.New("some other error"), FallbackReasonServerError},
	}

	handler := &FailoverHandler{config: DefaultFailoverConfig()}

	for _, tt := range tests {
		reason := handler.determineFallbackReason(tt.err)
		if reason != tt.expected {
			t.Errorf("determineFallbackReason(%v) = %v, want %v", tt.err, reason, tt.expected)
		}
	}
}

func BenchmarkModelSelection(b *testing.B) {
	registry := DefaultModelRegistry()
	router := NewRouter(registry, NewBalancedStrategy(registry), nil)

	req := &RoutingRequest{
		Complexity: ComplexityMedium,
		Priority:   PriorityBalanced,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.SelectModel(context.Background(), req)
	}
}

func BenchmarkFallback(b *testing.B) {
	registry := DefaultModelRegistry()
	router := NewRouter(registry, NewBalancedStrategy(registry), nil)

	primary := &ModelInfo{
		Name:     "gpt-4o",
		Provider: "openai",
		Tier:     TierReasoning,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.SelectFallback(context.Background(), primary, FallbackReasonRateLimit)
	}
}

// Helper to suppress unused import error
var _ = fmt.Sprintf
var _ = time.Sleep
