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

package effects

import (
	"testing"
	"time"
)

func TestKind_String(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindLLM, "llm"},
		{KindTool, "tool"},
		{KindHTTP, "http"},
		{KindTime, "time"},
		{KindRandom, "random"},
		{KindSleep, "sleep"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("Kind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEffect(t *testing.T) {
	e := NewEffect(KindLLM, map[string]string{"model": "gpt-4"})
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.Kind != KindLLM {
		t.Errorf("expected KindLLM, got %s", e.Kind)
	}
	if e.Payload == nil {
		t.Error("expected non-nil Payload")
	}
}

func TestEffect_WithIdempotencyKey(t *testing.T) {
	e := NewEffect(KindTool, nil).WithIdempotencyKey("key-123")
	if e.IdempotencyKey != "key-123" {
		t.Errorf("expected key-123, got %s", e.IdempotencyKey)
	}
}

func TestEffect_WithDescription(t *testing.T) {
	e := NewEffect(KindLLM, nil).WithDescription("test effect")
	if e.Description != "test effect" {
		t.Errorf("expected test effect, got %s", e.Description)
	}
}

func TestEffect_WithJobID(t *testing.T) {
	e := NewEffect(KindLLM, nil).WithJobID("job-123")
	if e.JobID != "job-123" {
		t.Errorf("expected job-123, got %s", e.JobID)
	}
}

func TestEffect_WithAttemptID(t *testing.T) {
	e := NewEffect(KindLLM, nil).WithAttemptID("attempt-123")
	if e.AttemptID != "attempt-123" {
		t.Errorf("expected attempt-123, got %s", e.AttemptID)
	}
}

func TestEffect_SuccessResult(t *testing.T) {
	result := SuccessResult("effect-1", KindLLM, "response", 100*time.Millisecond)
	if result.ID != "effect-1" {
		t.Errorf("expected effect-1, got %s", result.ID)
	}
	if result.Kind != KindLLM {
		t.Errorf("expected KindLLM, got %s", result.Kind)
	}
	if result.Data != "response" {
		t.Errorf("expected response, got %v", result.Data)
	}
	if result.DurationMs != 100 {
		t.Errorf("expected 100, got %d", result.DurationMs)
	}
	if result.Error != nil {
		t.Error("expected nil error")
	}
}

func TestEffect_FailedResult(t *testing.T) {
	err := Error{
		Type:    "llm",
		Message: "rate limited",
		Code:    429,
	}
	result := FailedResult("effect-1", KindLLM, err, 50*time.Millisecond)
	if result.ID != "effect-1" {
		t.Errorf("expected effect-1, got %s", result.ID)
	}
	if result.Error == nil {
		t.Error("expected error")
	}
	if result.Error.Type != "llm" {
		t.Errorf("expected llm, got %s", result.Error.Type)
	}
}

func TestCachedResult(t *testing.T) {
	original := SuccessResult("original-1", KindLLM, "response", 100*time.Millisecond)
	cached := CachedResult("original-1", original)
	if !cached.Cached {
		t.Error("expected Cached to be true")
	}
	if cached.ReplayFromID != "original-1" {
		t.Errorf("expected original-1, got %s", cached.ReplayFromID)
	}
}

func TestEffect_MarshalPayload(t *testing.T) {
	e := NewEffect(KindLLM, map[string]string{"model": "gpt-4"})
	data, err := e.MarshalPayload()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestEffect_MarshalPayload_Nil(t *testing.T) {
	e := Effect{Payload: nil}
	data, err := e.MarshalPayload()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("expected null, got %s", string(data))
	}
}
