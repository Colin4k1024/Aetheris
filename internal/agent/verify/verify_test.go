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

package verify

import (
	"context"
	"testing"

	"rag-platform/internal/runtime/jobstore"
)

func TestCompute_EmptyEvents(t *testing.T) {
	result, err := Compute(context.Background(), nil, "job-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExecutionHash != "" {
		t.Errorf("expected empty hash, got %s", result.ExecutionHash)
	}
	if !result.ToolInvocationLedgerProof.OK {
		t.Error("expected ledger proof to be OK for empty events")
	}
	if result.ReplayProofResult.OK {
		t.Error("expected replay proof to be false for empty events")
	}
}

func TestEventChainRoot_Empty(t *testing.T) {
	hash := EventChainRoot([]jobstore.JobEvent{})
	if hash != "" {
		t.Errorf("expected empty hash, got %s", hash)
	}
}

func TestEventChainRoot_SingleEvent(t *testing.T) {
	events := []jobstore.JobEvent{
		{
			ID:      "event-1",
			Type:    jobstore.JobCreated,
			Payload: []byte(`{"key":"value"}`),
		},
	}
	hash := EventChainRoot(events)
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestExecutionHash_Empty(t *testing.T) {
	hash := ExecutionHash([]jobstore.JobEvent{})
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestExecutionHash_WithPlanAndNodes(t *testing.T) {
	events := []jobstore.JobEvent{
		{
			ID:      "event-1",
			Type:    jobstore.PlanGenerated,
			Payload: []byte(`{"plan_hash":"abc123"}`),
		},
		{
			ID:      "event-2",
			Type:    jobstore.NodeFinished,
			Payload: []byte(`{"node_id":"node-1","result_type":"success"}`),
		},
	}
	hash := ExecutionHash(events)
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestLedgerProof_Empty(t *testing.T) {
	result := LedgerProof([]jobstore.JobEvent{})
	if !result.OK {
		t.Error("expected OK for empty events")
	}
}

func TestLedgerProof_AllMatched(t *testing.T) {
	events := []jobstore.JobEvent{
		{
			ID:      "event-1",
			Type:    jobstore.ToolInvocationStarted,
			Payload: []byte(`{"idempotency_key":"key-1"}`),
		},
		{
			ID:      "event-2",
			Type:    jobstore.ToolInvocationFinished,
			Payload: []byte(`{"idempotency_key":"key-1"}`),
		},
	}
	result := LedgerProof(events)
	if !result.OK {
		t.Error("expected OK when all matched")
	}
}

func TestLedgerProof_Pending(t *testing.T) {
	events := []jobstore.JobEvent{
		{
			ID:      "event-1",
			Type:    jobstore.ToolInvocationStarted,
			Payload: []byte(`{"idempotency_key":"key-1"}`),
		},
	}
	result := LedgerProof(events)
	if result.OK {
		t.Error("expected not OK when pending")
	}
	if len(result.PendingIdempotencyKeys) != 1 {
		t.Errorf("expected 1 pending key, got %d", len(result.PendingIdempotencyKeys))
	}
}
