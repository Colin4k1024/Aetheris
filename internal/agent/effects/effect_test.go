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

func TestEffectKind(t *testing.T) {
	tests := []struct {
		kind     EffectKind
		expected string
	}{
		{EffectKindLLMResponseRecorded, "llm_response_recorded"},
		{EffectKindToolResultRecorded, "tool_result_recorded"},
		{EffectKindExternalCallRecorded, "external_call_recorded"},
		{EffectKindTimerScheduled, "timer_scheduled"},
		{EffectKindRetryDecision, "retry_decision"},
	}

	for _, tt := range tests {
		if string(tt.kind) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.kind)
		}
	}
}

func TestPayloadForCommandCommitted(t *testing.T) {
	result := []byte(`{"key":"value"}`)
	payload, err := PayloadForCommandCommitted("node-1", "cmd-1", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestPayloadForCommandCommitted_EmptyCommandID(t *testing.T) {
	result := []byte(`{"key":"value"}`)
	payload, err := PayloadForCommandCommitted("node-1", "", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payload) == 0 {
		t.Error("expected non-empty payload")
	}
}
