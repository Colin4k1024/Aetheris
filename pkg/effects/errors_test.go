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
)

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		err    Error
		errMsg string
	}{
		{ErrLLMGenerationFailed, "LLM generation failed"},
		{ErrLLMTimeout, "LLM request timed out"},
		{ErrLLMRateLimited, "LLM rate limit exceeded"},
		{ErrToolExecutionFailed, "tool execution failed"},
		{ErrToolNotFound, "tool not found"},
		{ErrToolTimeout, "tool execution timed out"},
		{ErrHTTPRequestFailed, "HTTP request failed"},
		{ErrHTTPNotFound, "HTTP resource not found"},
		{ErrHTTPServerError, "HTTP server error"},
	}

	for _, tt := range tests {
		if tt.errMsg != "" && tt.err.Message != tt.errMsg {
			t.Errorf("expected %s, got %s", tt.errMsg, tt.err.Message)
		}
	}
}

func TestNewError(t *testing.T) {
	err := NewError("llm", "rate limited: %s", "429")
	if err.Type != "llm" {
		t.Errorf("expected llm, got %s", err.Type)
	}
	if err.Message != "rate limited: 429" {
		t.Errorf("expected 'rate limited: 429', got %s", err.Message)
	}
}

func TestNewError_NoArgs(t *testing.T) {
	err := NewError("tool", "execution failed")
	if err.Type != "tool" {
		t.Errorf("expected tool, got %s", err.Type)
	}
	if err.Message != "execution failed" {
		t.Errorf("expected 'execution failed', got %s", err.Message)
	}
}

func TestError_IsRetriable(t *testing.T) {
	tests := []struct {
		err      Error
		expected bool
	}{
		{ErrLLMGenerationFailed, true},
		{ErrLLMTimeout, true},
		{ErrLLMRateLimited, true},
		{ErrToolExecutionFailed, true},
		{ErrToolNotFound, false},
		{ErrToolTimeout, true},
		{ErrHTTPRequestFailed, true},
		{ErrHTTPNotFound, false},
		{ErrHTTPServerError, true},
	}

	for _, tt := range tests {
		if tt.err.Retriable != tt.expected {
			t.Errorf("expected Retriable=%v for %s, got %v", tt.expected, tt.err.Message, tt.err.Retriable)
		}
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrReplayingForbidden.Error() != "effects: real execution forbidden during replay" {
		t.Errorf("unexpected ErrReplayingForbidden message")
	}
	if ErrNotFound.Error() != "effects: effect not found in history" {
		t.Errorf("unexpected ErrNotFound message")
	}
	if ErrAlreadyExists.Error() != "effects: idempotency key already exists" {
		t.Errorf("unexpected ErrAlreadyExists message")
	}
	if ErrNoRecorder.Error() != "effects: no event recorder configured" {
		t.Errorf("unexpected ErrNoRecorder message")
	}
	if ErrNoSystem.Error() != "effects: no effect system configured" {
		t.Errorf("unexpected ErrNoSystem message")
	}
}
