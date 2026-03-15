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

package agent

import (
	"testing"
	"time"
)

func TestRunOptions_Default(t *testing.T) {
	o := &RunOptions{}
	if o.MaxSteps != 0 {
		t.Errorf("expected MaxSteps=0, got %d", o.MaxSteps)
	}
	if o.Timeout != 0 {
		t.Errorf("expected Timeout=0, got %v", o.Timeout)
	}
	if o.SessionID != "" {
		t.Errorf("expected empty SessionID, got %s", o.SessionID)
	}
}

func TestWithSessionID(t *testing.T) {
	opt := WithSessionID("session-123")
	o := &RunOptions{}
	opt(o)
	if o.SessionID != "session-123" {
		t.Errorf("expected session-123, got %s", o.SessionID)
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 30 * time.Second
	opt := WithTimeout(timeout)
	o := &RunOptions{}
	opt(o)
	if o.Timeout != timeout {
		t.Errorf("expected %v, got %v", timeout, o.Timeout)
	}
}

func TestWithRunMaxSteps(t *testing.T) {
	opt := WithRunMaxSteps(50)
	o := &RunOptions{}
	opt(o)
	if o.MaxSteps != 50 {
		t.Errorf("expected 50, got %d", o.MaxSteps)
	}
}

func TestApplyRunOptions_Default(t *testing.T) {
	o := applyRunOptions(nil)
	if o.MaxSteps != 20 {
		t.Errorf("expected default MaxSteps=20, got %d", o.MaxSteps)
	}
}

func TestApplyRunOptions_WithOptions(t *testing.T) {
	o := applyRunOptions([]RunOption{
		WithSessionID("test-session"),
		WithTimeout(time.Minute),
		WithRunMaxSteps(100),
	})
	if o.SessionID != "test-session" {
		t.Errorf("expected test-session, got %s", o.SessionID)
	}
	if o.Timeout != time.Minute {
		t.Errorf("expected 1m, got %v", o.Timeout)
	}
	if o.MaxSteps != 100 {
		t.Errorf("expected 100, got %d", o.MaxSteps)
	}
}
