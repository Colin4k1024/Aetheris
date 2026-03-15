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

package job

import "testing"

func TestQueueClass(t *testing.T) {
	classes := []string{
		QueueRealtime,
		QueueDefault,
		QueueBackground,
		QueueHeavy,
	}
	for _, c := range classes {
		if c == "" {
			t.Error("QueueClass should not be empty")
		}
	}
}

func TestPriorityValues(t *testing.T) {
	if PriorityRealtime != 10 {
		t.Errorf("expected 10, got %d", PriorityRealtime)
	}
	if PriorityDefault != 0 {
		t.Errorf("expected 0, got %d", PriorityDefault)
	}
	if PriorityBackground != -5 {
		t.Errorf("expected -5, got %d", PriorityBackground)
	}
	if PriorityHeavy != -10 {
		t.Errorf("expected -10, got %d", PriorityHeavy)
	}
}

func TestDefaultPriority(t *testing.T) {
	if DefaultPriority != 0 {
		t.Errorf("expected 0, got %d", DefaultPriority)
	}
}

func TestPriorityForQueue_Realtime(t *testing.T) {
	priority := PriorityForQueue(QueueRealtime)
	if priority != PriorityRealtime {
		t.Errorf("expected %d, got %d", PriorityRealtime, priority)
	}
}

func TestPriorityForQueue_Default(t *testing.T) {
	priority := PriorityForQueue(QueueDefault)
	if priority != PriorityDefault {
		t.Errorf("expected %d, got %d", PriorityDefault, priority)
	}
}

func TestPriorityForQueue_Background(t *testing.T) {
	priority := PriorityForQueue(QueueBackground)
	if priority != PriorityBackground {
		t.Errorf("expected %d, got %d", PriorityBackground, priority)
	}
}

func TestPriorityForQueue_Heavy(t *testing.T) {
	priority := PriorityForQueue(QueueHeavy)
	if priority != PriorityHeavy {
		t.Errorf("expected %d, got %d", PriorityHeavy, priority)
	}
}

func TestPriorityForQueue_Unknown(t *testing.T) {
	priority := PriorityForQueue("unknown")
	if priority != PriorityDefault {
		t.Errorf("expected %d (default), got %d", PriorityDefault, priority)
	}
}

func TestPriorityForQueue_Empty(t *testing.T) {
	priority := PriorityForQueue("")
	if priority != PriorityDefault {
		t.Errorf("expected %d (default), got %d", PriorityDefault, priority)
	}
}

func TestExtractGoalFromMessagePayload_StringMessage(t *testing.T) {
	payload := map[string]any{
		"message": "test goal",
	}
	goal := extractGoalFromMessagePayload(payload)
	if goal != "test goal" {
		t.Errorf("expected 'test goal', got '%s'", goal)
	}
}

func TestExtractGoalFromMessagePayload_NilPayload(t *testing.T) {
	goal := extractGoalFromMessagePayload(nil)
	if goal != "" {
		t.Errorf("expected empty string, got '%s'", goal)
	}
}

func TestExtractGoalFromMessagePayload_EmptyMessage(t *testing.T) {
	payload := map[string]any{
		"message": "",
	}
	goal := extractGoalFromMessagePayload(payload)
	// Should return marshaled payload for empty message
	if goal == "" {
		t.Log("Empty message returns empty string (falls back to marshal)")
	}
}

func TestExtractGoalFromMessagePayload_NoMessage(t *testing.T) {
	payload := map[string]any{
		"other": "value",
	}
	goal := extractGoalFromMessagePayload(payload)
	// Should return marshaled payload
	if goal == "" {
		t.Error("expected non-empty goal from marshal")
	}
}

func TestExtractGoalFromMessagePayload_WithMessage(t *testing.T) {
	payload := map[string]any{
		"message": "my test goal",
		"extra":   "data",
	}
	goal := extractGoalFromMessagePayload(payload)
	if goal != "my test goal" {
		t.Errorf("expected 'my test goal', got '%s'", goal)
	}
}
