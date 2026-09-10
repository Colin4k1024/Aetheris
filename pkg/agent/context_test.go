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
	"context"
	"testing"
	"time"
)

func TestContextWithMaxSteps_Zero_NoOverride(t *testing.T) {
	ctx := ContextWithMaxSteps(context.Background(), 0)
	if v := MaxStepsFromContext(ctx); v != 0 {
		t.Errorf("expected 0 for zero maxSteps, got %d", v)
	}
}

func TestContextWithMaxSteps_Negative_NoOverride(t *testing.T) {
	ctx := ContextWithMaxSteps(context.Background(), -1)
	if v := MaxStepsFromContext(ctx); v != 0 {
		t.Errorf("expected 0 for negative maxSteps, got %d", v)
	}
}

func TestContextWithMaxSteps_Positive_Overrides(t *testing.T) {
	ctx := ContextWithMaxSteps(context.Background(), 5)
	if v := MaxStepsFromContext(ctx); v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
}

func TestMaxStepsFromContext_NoKey_ReturnsZero(t *testing.T) {
	v := MaxStepsFromContext(context.Background())
	if v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
}

func TestWithTimeout_SetsOption(t *testing.T) {
	o := applyRunOptions([]RunOption{WithTimeout(30 * time.Second)})
	if o.Timeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", o.Timeout)
	}
}

func TestWithRunMaxSteps_SetsOption(t *testing.T) {
	o := applyRunOptions([]RunOption{WithRunMaxSteps(10)})
	if o.MaxSteps != 10 {
		t.Errorf("expected 10, got %d", o.MaxSteps)
	}
}

func TestApplyRunOptions_Defaults(t *testing.T) {
	o := applyRunOptions(nil)
	if o.MaxSteps != 20 {
		t.Errorf("expected default 20, got %d", o.MaxSteps)
	}
	if o.Timeout != 0 {
		t.Errorf("expected default 0 timeout, got %v", o.Timeout)
	}
}

func TestWithSessionID_SetsOption(t *testing.T) {
	o := applyRunOptions([]RunOption{WithSessionID("test-session")})
	if o.SessionID != "test-session" {
		t.Errorf("expected test-session, got %s", o.SessionID)
	}
}

// Test that options don't pollute each other across calls
func TestRunOptions_NoCrossContamination(t *testing.T) {
	o1 := applyRunOptions([]RunOption{WithRunMaxSteps(5)})
	o2 := applyRunOptions([]RunOption{WithRunMaxSteps(10)})
	if o1.MaxSteps != 5 {
		t.Errorf("expected 5 for first call, got %d", o1.MaxSteps)
	}
	if o2.MaxSteps != 10 {
		t.Errorf("expected 10 for second call, got %d", o2.MaxSteps)
	}
}
