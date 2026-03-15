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

package signal

import (
	"context"
	"testing"
)

func TestNewInboxMem(t *testing.T) {
	inbox := NewInboxMem()
	if inbox == nil {
		t.Fatal("expected non-nil inbox")
	}
}

func TestInboxMem_Append(t *testing.T) {
	inbox := NewInboxMem()
	ctx := context.Background()

	id, err := inbox.Append(ctx, "job-1", "correlation-key", []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty id")
	}
}

func TestInboxMem_MarkAcked(t *testing.T) {
	inbox := NewInboxMem()
	ctx := context.Background()

	// Append a signal
	id, err := inbox.Append(ctx, "job-1", "correlation-key", []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mark as acked
	err = inbox.MarkAcked(ctx, "job-1", id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInboxMem_MarkAcked_NonExistent(t *testing.T) {
	inbox := NewInboxMem()
	ctx := context.Background()

	// Mark as acked for non-existent id should not error
	err := inbox.MarkAcked(ctx, "job-1", "non-existent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSignalRecord(t *testing.T) {
	record := SignalRecord{
		ID:             "test-id",
		JobID:          "job-1",
		CorrelationKey: "key-1",
		Payload:        []byte("payload"),
	}
	if record.ID != "test-id" {
		t.Errorf("expected test-id, got %s", record.ID)
	}
	if record.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", record.JobID)
	}
	if string(record.Payload) != "payload" {
		t.Errorf("expected payload, got %s", string(record.Payload))
	}
}
