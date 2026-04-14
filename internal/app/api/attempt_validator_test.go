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

package api

import (
	"context"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

func TestNewAttemptValidator_NilStore(t *testing.T) {
	v := NewAttemptValidator(nil)
	if v != nil {
		t.Error("expected nil for nil store")
	}
}

func TestAttemptValidator_ValidateAttempt_NoAttemptID(t *testing.T) {
	// Create a mock store that returns empty attempt ID
	mockStore := &mockJobStore{
		getCurrentAttemptIDFunc: func(ctx context.Context, jobID string) (string, error) {
			return "", nil
		},
	}

	v := &attemptValidator{store: mockStore}
	err := v.ValidateAttempt(context.Background(), "job-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAttemptValidator_ValidateAttempt_MatchingAttemptID(t *testing.T) {
	mockStore := &mockJobStore{
		getCurrentAttemptIDFunc: func(ctx context.Context, jobID string) (string, error) {
			return "attempt-123", nil
		},
	}

	v := &attemptValidator{store: mockStore}
	ctx := jobstore.WithAttemptID(context.Background(), "attempt-123")
	err := v.ValidateAttempt(ctx, "job-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAttemptValidator_ValidateAttempt_StaleAttemptID(t *testing.T) {
	mockStore := &mockJobStore{
		getCurrentAttemptIDFunc: func(ctx context.Context, jobID string) (string, error) {
			return "attempt-123", nil
		},
	}

	v := &attemptValidator{store: mockStore}
	ctx := jobstore.WithAttemptID(context.Background(), "attempt-old")
	err := v.ValidateAttempt(ctx, "job-1")
	if err != jobstore.ErrStaleAttempt {
		t.Errorf("expected ErrStaleAttempt, got %v", err)
	}
}

func TestAttemptValidator_ValidateAttempt_StoreError(t *testing.T) {
	mockStore := &mockJobStore{
		getCurrentAttemptIDFunc: func(ctx context.Context, jobID string) (string, error) {
			return "", context.DeadlineExceeded
		},
	}

	v := &attemptValidator{store: mockStore}
	ctx := jobstore.WithAttemptID(context.Background(), "attempt-123")
	err := v.ValidateAttempt(ctx, "job-1")
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

// mockJobStore implements jobstore.JobStore for testing
type mockJobStore struct {
	jobstore.JobStore
	getCurrentAttemptIDFunc func(ctx context.Context, jobID string) (string, error)
}

func (m *mockJobStore) GetCurrentAttemptID(ctx context.Context, jobID string) (string, error) {
	if m.getCurrentAttemptIDFunc != nil {
		return m.getCurrentAttemptIDFunc(ctx, jobID)
	}
	return "", nil
}
