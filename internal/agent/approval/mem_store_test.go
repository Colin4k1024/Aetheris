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

package approval

import (
	"context"
	"testing"
	"time"
)

func TestMemStore_Create(t *testing.T) {
	store := NewMemStore()
	ctx := context.Background()
	req := &ApprovalRequest{
		ID:             "apr-1",
		JobID:          "job-1",
		NodeID:         "node-1",
		CorrelationKey: "corr-1",
		ApproverType:   ApproverTypeAnyone,
		Title:          "Test Approval",
		Description:    "Test description",
		Status:         DecisionPending,
	}
	if err := store.Create(ctx, req); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	got, err := store.GetByID(ctx, "apr-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected approval request, got nil")
	}
	if got.Title != "Test Approval" {
		t.Errorf("title got %s, want Test Approval", got.Title)
	}
	if got.Status != DecisionPending {
		t.Errorf("status got %s, want pending", got.Status)
	}
}

func TestMemStore_Complete(t *testing.T) {
	store := NewMemStore()
	ctx := context.Background()
	req := &ApprovalRequest{
		ID:             "apr-2",
		JobID:          "job-1",
		NodeID:         "node-1",
		CorrelationKey: "corr-2",
		ApproverType:   ApproverTypeSpecific,
		ApproverID:     "user-1",
		Title:          "Approve Payment",
		Status:         DecisionPending,
	}
	_ = store.Create(ctx, req)
	resp := &ApprovalResponse{
		Decision:     DecisionApproved,
		ApproverID:   "user-1",
		ApproverName: "John Doe",
		Comment:      "Looks good",
		RespondedAt:  time.Now(),
	}
	if err := store.Complete(ctx, "apr-2", resp); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	got, _ := store.GetByID(ctx, "apr-2")
	if got.Status != DecisionApproved {
		t.Errorf("status got %s, want approved", got.Status)
	}
	if got.ApproverResp == nil {
		t.Fatal("expected approver response, got nil")
	}
	if got.ApproverResp.Comment != "Looks good" {
		t.Errorf("comment got %s, want Looks good", got.ApproverResp.Comment)
	}
}

func TestMemStore_GetPending(t *testing.T) {
	store := NewMemStore()
	ctx := context.Background()
	store.Create(ctx, &ApprovalRequest{
		ID:             "apr-3",
		JobID:          "job-1",
		CorrelationKey: "corr-3",
		Title:          "Pending 1",
		Status:         DecisionPending,
	})
	store.Create(ctx, &ApprovalRequest{
		ID:             "apr-4",
		JobID:          "job-2",
		CorrelationKey: "corr-4",
		Title:          "Pending 2",
		Status:         DecisionPending,
	})
	store.Create(ctx, &ApprovalRequest{
		ID:             "apr-5",
		JobID:          "job-3",
		CorrelationKey: "corr-5",
		Title:          "Already Approved",
		Status:         DecisionApproved,
	})
	store.Create(ctx, &ApprovalRequest{
		ID:             "apr-6",
		JobID:          "job-4",
		CorrelationKey: "corr-6",
		Title:          "Expired",
		Status:         DecisionPending,
		ExpiresAt:      &time.Time{}, // zero time = already expired
	})
	pending, err := store.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("pending count got %d, want 2", len(pending))
	}
}

func TestMemStore_GetByJobID(t *testing.T) {
	store := NewMemStore()
	ctx := context.Background()
	store.Create(ctx, &ApprovalRequest{ID: "apr-7", JobID: "job-x", Title: "First", Status: DecisionPending})
	store.Create(ctx, &ApprovalRequest{ID: "apr-8", JobID: "job-x", Title: "Second", Status: DecisionPending})
	store.Create(ctx, &ApprovalRequest{ID: "apr-9", JobID: "job-y", Title: "Other", Status: DecisionPending})
	items, err := store.GetByJobID(ctx, "job-x")
	if err != nil {
		t.Fatalf("GetByJobID failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("count got %d, want 2", len(items))
	}
}
