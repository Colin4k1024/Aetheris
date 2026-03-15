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

package runtime

import (
	"context"
	"testing"
	"time"
)

func TestRunContext_Deadline(t *testing.T) {
	baseCtx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	deadline := time.Now().Add(30 * time.Minute)
	rc := WithRunContext(baseCtx, "agent-1", "session-1", "checkpoint-1", deadline)

	d, ok := rc.Deadline()
	if !ok {
		t.Error("expected deadline to be set")
	}
	// Should return the deadline from RunContext, not from base context
	if d.Unix() != deadline.Unix() {
		t.Errorf("expected %v, got %v", deadline, d)
	}
}

func TestRunContext_Deadline_Zero(t *testing.T) {
	baseCtx := context.Background()
	rc := WithRunContext(baseCtx, "agent-1", "session-1", "", time.Time{})

	_, ok := rc.Deadline()
	if ok {
		t.Error("expected no deadline")
	}
}

func TestRunContext_Done(t *testing.T) {
	baseCtx, cancel := context.WithCancel(context.Background())
	rc := WithRunContext(baseCtx, "agent-1", "session-1", "", time.Time{})

	done := rc.Done()
	if done == nil {
		t.Error("expected non-nil done channel")
	}

	cancel()
	<-done
}

func TestRunContext_Err(t *testing.T) {
	baseCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately to trigger cancellation
	rc := WithRunContext(baseCtx, "agent-1", "session-1", "", time.Time{})

	err := rc.Err()
	if err != context.Canceled {
		t.Errorf("expected Canceled, got %v", err)
	}
}

func TestRunContext_Value(t *testing.T) {
	baseCtx := context.WithValue(context.Background(), "key", "value")
	rc := WithRunContext(baseCtx, "agent-1", "session-1", "", time.Time{})

	val := rc.Value("key")
	if val != "value" {
		t.Errorf("expected value, got %v", val)
	}
}

func TestRunContext_WithDeadline(t *testing.T) {
	baseCtx := context.Background()
	originalDeadline := time.Now().Add(time.Hour)
	rc := WithRunContext(baseCtx, "agent-1", "session-1", "checkpoint-1", originalDeadline)

	newDeadline := time.Now().Add(2 * time.Hour)
	rc2 := rc.WithDeadline(newDeadline)

	if rc2.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", rc2.AgentID)
	}
	if rc2.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", rc2.SessionID)
	}
	if rc2.Checkpoint != "checkpoint-1" {
		t.Errorf("expected checkpoint-1, got %s", rc2.Checkpoint)
	}

	d, ok := rc2.Deadline()
	if !ok {
		t.Error("expected deadline to be set")
	}
	// Should be approximately 2 hours from now
	if d.Sub(newDeadline).Abs() > time.Second {
		t.Errorf("expected %v, got %v", newDeadline, d)
	}
}

func TestWithRunContext(t *testing.T) {
	baseCtx := context.Background()
	deadline := time.Now().Add(time.Hour)

	rc := WithRunContext(baseCtx, "agent-123", "session-456", "checkpoint-789", deadline)

	if rc.AgentID != "agent-123" {
		t.Errorf("expected agent-123, got %s", rc.AgentID)
	}
	if rc.SessionID != "session-456" {
		t.Errorf("expected session-456, got %s", rc.SessionID)
	}
	if rc.Checkpoint != "checkpoint-789" {
		t.Errorf("expected checkpoint-789, got %s", rc.Checkpoint)
	}
	if rc.DeadlineAt != deadline {
		t.Errorf("expected %v, got %v", deadline, rc.DeadlineAt)
	}
	if rc.Context != baseCtx {
		t.Error("context should be the same")
	}
}
