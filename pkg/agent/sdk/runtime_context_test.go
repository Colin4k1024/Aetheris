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

package sdk

import (
	"context"
	"testing"
	"time"
)

type mockRuntimeContext struct{}

func (m *mockRuntimeContext) Now(ctx context.Context) time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func (m *mockRuntimeContext) UUID(ctx context.Context) string {
	return "test-uuid-123"
}

func (m *mockRuntimeContext) HTTP(ctx context.Context, effectID string, doRequest func() (reqJSON, respJSON []byte, err error)) (reqJSON, respJSON []byte, err error) {
	return []byte(`{"req":1}`), []byte(`{"resp":2}`), nil
}

func (m *mockRuntimeContext) JobID(ctx context.Context) string {
	return "test-job-id"
}

func (m *mockRuntimeContext) StepID(ctx context.Context) string {
	return "test-step-id"
}

func TestWithRuntimeContext(t *testing.T) {
	ctx := context.Background()
	rc := &mockRuntimeContext{}

	ctx = WithRuntimeContext(ctx, rc)
	got := FromRuntimeContext(ctx)
	if got == nil {
		t.Fatal("FromRuntimeContext should not return nil")
	}
}

func TestFromRuntimeContext_NilContext(t *testing.T) {
	got := FromRuntimeContext(nil)
	if got != nil {
		t.Error("FromRuntimeContext(nil) should return nil")
	}
}

func TestFromRuntimeContext_NotSet(t *testing.T) {
	ctx := context.Background()
	got := FromRuntimeContext(ctx)
	if got != nil {
		t.Error("FromRuntimeContext without WithRuntimeContext should return nil")
	}
}

func TestNow_WithContext(t *testing.T) {
	ctx := context.Background()
	rc := &mockRuntimeContext{}
	ctx = WithRuntimeContext(ctx, rc)

	now := Now(ctx)
	expected := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !now.Equal(expected) {
		t.Errorf("Now() = %v, want %v", now, expected)
	}
}

func TestNow_WithoutContext(t *testing.T) {
	ctx := context.Background()
	now := Now(ctx)
	// Should fall back to time.Now()
	_ = now
}

func TestUUID_WithContext(t *testing.T) {
	ctx := context.Background()
	rc := &mockRuntimeContext{}
	ctx = WithRuntimeContext(ctx, rc)

	id := UUID(ctx)
	if id != "test-uuid-123" {
		t.Errorf("UUID() = %v, want test-uuid-123", id)
	}
}

func TestUUID_WithoutContext(t *testing.T) {
	ctx := context.Background()
	id := UUID(ctx)
	// Should fall back to uuid.New()
	if id == "" {
		t.Error("UUID() should not be empty without context")
	}
}

func TestHTTP_WithContext(t *testing.T) {
	ctx := context.Background()
	rc := &mockRuntimeContext{}
	ctx = WithRuntimeContext(ctx, rc)

	req, resp, err := HTTP(ctx, "effect-1", func() (reqJSON, respJSON []byte, err error) {
		return []byte(`{"test":1}`), []byte(`{"test":2}`), nil
	})
	if err != nil {
		t.Errorf("HTTP() error = %v", err)
	}
	if string(req) != `{"req":1}` {
		t.Errorf("HTTP() req = %v", string(req))
	}
	if string(resp) != `{"resp":2}` {
		t.Errorf("HTTP() resp = %v", string(resp))
	}
}

func TestHTTP_WithoutContext(t *testing.T) {
	ctx := context.Background()
	_, _, err := HTTP(ctx, "effect-1", func() (reqJSON, respJSON []byte, err error) {
		return nil, nil, nil
	})
	if err != ErrNoRuntimeContext {
		t.Errorf("HTTP() without context error = %v, want ErrNoRuntimeContext", err)
	}
}

func TestJobID_WithContext(t *testing.T) {
	ctx := context.Background()
	rc := &mockRuntimeContext{}
	ctx = WithRuntimeContext(ctx, rc)

	jobID := JobID(ctx)
	if jobID != "test-job-id" {
		t.Errorf("JobID() = %v, want test-job-id", jobID)
	}
}

func TestJobID_WithoutContext(t *testing.T) {
	ctx := context.Background()
	jobID := JobID(ctx)
	if jobID != "" {
		t.Errorf("JobID() without context = %v, want empty string", jobID)
	}
}

func TestStepID_WithContext(t *testing.T) {
	ctx := context.Background()
	rc := &mockRuntimeContext{}
	ctx = WithRuntimeContext(ctx, rc)

	stepID := StepID(ctx)
	if stepID != "test-step-id" {
		t.Errorf("StepID() = %v, want test-step-id", stepID)
	}
}

func TestStepID_WithoutContext(t *testing.T) {
	ctx := context.Background()
	stepID := StepID(ctx)
	if stepID != "" {
		t.Errorf("StepID() without context = %v, want empty string", stepID)
	}
}

func TestErrNoRuntimeContext(t *testing.T) {
	if ErrNoRuntimeContext.Error() != "sdk: runtime context not set" {
		t.Errorf("ErrNoRuntimeContext.Error() = %v", ErrNoRuntimeContext.Error())
	}
}

// Ensure mockRuntimeContext implements RuntimeContext
var _ RuntimeContext = (*mockRuntimeContext)(nil)
