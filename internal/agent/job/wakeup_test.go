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

import (
	"context"
	"testing"
	"time"
)

func TestNewWakeupQueueMem(t *testing.T) {
	q := NewWakeupQueueMem(100)
	if q == nil {
		t.Fatal("expected non-nil WakeupQueueMem")
	}
	if q.ch == nil {
		t.Error("expected non-nil channel")
	}
}

func TestNewWakeupQueueMem_ZeroBuffer(t *testing.T) {
	// Zero buffer should use default
	q := NewWakeupQueueMem(0)
	if q == nil {
		t.Fatal("expected non-nil WakeupQueueMem")
	}
	// Should have default buffer size
	if cap(q.ch) != 256 {
		t.Errorf("expected buffer size 256, got %d", cap(q.ch))
	}
}

func TestNewWakeupQueueMem_NegativeBuffer(t *testing.T) {
	// Negative buffer should use default
	q := NewWakeupQueueMem(-1)
	if q == nil {
		t.Fatal("expected non-nil WakeupQueueMem")
	}
	if cap(q.ch) != 256 {
		t.Errorf("expected buffer size 256, got %d", cap(q.ch))
	}
}

func TestWakeupQueueMem_NotifyReady(t *testing.T) {
	q := NewWakeupQueueMem(10)
	err := q.NotifyReady(context.Background(), "job-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWakeupQueueMem_NotifyReady_EmptyJobID(t *testing.T) {
	q := NewWakeupQueueMem(10)
	err := q.NotifyReady(context.Background(), "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWakeupQueueMem_Receive_WithNotify(t *testing.T) {
	q := NewWakeupQueueMem(10)

	// Notify first
	err := q.NotifyReady(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Receive should get the job
	ctx := context.Background()
	jobID, ok := q.Receive(ctx, time.Second)
	if !ok {
		t.Error("expected to receive job")
	}
	if jobID != "job-1" {
		t.Errorf("expected job-1, got %s", jobID)
	}
}

func TestWakeupQueueMem_Receive_Timeout(t *testing.T) {
	q := NewWakeupQueueMem(10)

	// Receive with short timeout should timeout
	ctx := context.Background()
	jobID, ok := q.Receive(ctx, 50*time.Millisecond)
	if ok {
		t.Error("expected timeout, got ok=true")
	}
	if jobID != "" {
		t.Errorf("expected empty jobID, got %s", jobID)
	}
}

func TestWakeupQueueMem_Receive_ContextCancel(t *testing.T) {
	q := NewWakeupQueueMem(10)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Receive should return immediately due to context cancellation
	jobID, ok := q.Receive(ctx, time.Second)
	if ok {
		t.Error("expected cancelled context to return ok=false")
	}
	if jobID != "" {
		t.Errorf("expected empty jobID, got %s", jobID)
	}
}
