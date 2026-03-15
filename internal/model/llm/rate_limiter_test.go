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

package llm

import (
	"context"
	"testing"
	"time"
)

func TestNewLLMRateLimiter(t *testing.T) {
	configs := map[string]LLMLimitConfig{
		"openai": {
			TokensPerMinute:   90000,
			RequestsPerMinute: 60,
			MaxConcurrent:     10,
		},
	}
	limiter := NewLLMRateLimiter(configs, nil)
	if limiter == nil {
		t.Fatal("NewLLMRateLimiter should not return nil")
	}
}

func TestNewLLMRateLimiter_Defaults(t *testing.T) {
	limiter := NewLLMRateLimiter(nil, nil)
	if limiter == nil {
		t.Fatal("NewLLMRateLimiter should not return nil")
	}
	if limiter.defaults.TokensPerMinute != 90000 {
		t.Errorf("expected default tokens 90000, got %d", limiter.defaults.TokensPerMinute)
	}
}

func TestLLMRateLimiter_Wait(t *testing.T) {
	configs := map[string]LLMLimitConfig{
		"openai": {
			TokensPerMinute:   90000,
			RequestsPerMinute: 60,
			MaxConcurrent:     10,
		},
	}
	limiter := NewLLMRateLimiter(configs, nil)

	ctx := context.Background()
	err := limiter.Wait(ctx, "openai", 100)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLLMRateLimiter_Release(t *testing.T) {
	configs := map[string]LLMLimitConfig{
		"openai": {
			MaxConcurrent: 10,
		},
	}
	limiter := NewLLMRateLimiter(configs, nil)

	// Acquire and release
	ctx := context.Background()
	limiter.Wait(ctx, "openai", 100)
	limiter.Release("openai")
}

func TestLLMRateLimiter_GetStats(t *testing.T) {
	configs := map[string]LLMLimitConfig{
		"openai": {
			TokensPerMinute:   90000,
			RequestsPerMinute: 60,
			MaxConcurrent:     10,
		},
	}
	limiter := NewLLMRateLimiter(configs, nil)

	stats := limiter.GetStats("openai")
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	if stats["tokens_per_minute"] != 90000 {
		t.Errorf("expected 90000, got %v", stats["tokens_per_minute"])
	}
}

func TestLLMRateLimiter_GetStats_UnknownProvider(t *testing.T) {
	limiter := NewLLMRateLimiter(nil, nil)

	stats := limiter.GetStats("unknown")
	if stats != nil {
		t.Errorf("expected nil for unknown provider, got %v", stats)
	}
}

func TestLLMRateLimiter_Allow(t *testing.T) {
	configs := map[string]LLMLimitConfig{
		"openai": {
			RequestsPerMinute: 60,
		},
	}
	limiter := NewLLMRateLimiter(configs, nil)

	allowed := limiter.Allow("openai", 100)
	if !allowed {
		t.Error("expected to be allowed")
	}
}

func TestLLMRateLimiter_RecordTokenUsage(t *testing.T) {
	configs := map[string]LLMLimitConfig{
		"openai": {
			TokensPerMinute: 90000,
		},
	}
	limiter := NewLLMRateLimiter(configs, nil)

	limiter.RecordTokenUsage("openai", 100)

	stats := limiter.GetStats("openai")
	if stats["tokens_used_minute"] != 100 {
		t.Errorf("expected 100 tokens, got %v", stats["tokens_used_minute"])
	}
}
