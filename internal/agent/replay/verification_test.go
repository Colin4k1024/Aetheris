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

package replay

import (
	"context"
	"testing"
)

func TestReplayVerifier_Verify(t *testing.T) {
	ctx := context.Background()
	verifier := NewReplayVerifier()

	// 测试无状态变更
	result, err := verifier.Verify(ctx, "job-1", nil)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Decision != ReplayDecisionExecute {
		t.Errorf("expected decision Execute for nil state changes, got %v", result.Decision)
	}

	// 测试空状态变更
	result2, err := verifier.Verify(ctx, "job-1", map[string][]StateChangeRecord{})
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result2.Decision != ReplayDecisionExecute {
		t.Errorf("expected decision Execute for empty state changes, got %v", result2.Decision)
	}
}

func TestReplayVerifier_WithStateChanges(t *testing.T) {
	ctx := context.Background()
	verifier := NewReplayVerifier()

	// 测试有状态变更
	stateChanges := map[string][]StateChangeRecord{
		"step-1": {
			{
				ResourceType: "database",
				ResourceID:   "user-123",
				Operation:    "update",
				Version:      "v2",
				Etag:         "abc123",
			},
		},
	}

	result, err := verifier.Verify(ctx, "job-1", stateChanges)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.OverallStatus != VerificationStatusMatch {
		t.Errorf("expected overall status Match, got %v", result.OverallStatus)
	}
	if result.Decision != ReplayDecisionRestoreAndSkip {
		t.Errorf("expected decision RestoreAndSkip, got %v", result.Decision)
	}
	if result.MatchedCount != 1 {
		t.Errorf("expected 1 matched count, got %d", result.MatchedCount)
	}
}

func TestToolLedgerVerifier_Name(t *testing.T) {
	verifier := &ToolLedgerVerifier{}
	if verifier.Name() != "ToolLedgerVerifier" {
		t.Errorf("expected name ToolLedgerVerifier, got %s", verifier.Name())
	}
}

func TestDatabaseStateVerifier_Name(t *testing.T) {
	verifier := &DatabaseStateVerifier{}
	if verifier.Name() != "DatabaseStateVerifier" {
		t.Errorf("expected name DatabaseStateVerifier, got %s", verifier.Name())
	}
}

func TestVerificationResult_Statuses(t *testing.T) {
	tests := []struct {
		status   VerificationStatus
		expected string
	}{
		{VerificationStatusMatch, "match"},
		{VerificationStatusMismatch, "mismatch"},
		{VerificationStatusPending, "pending"},
		{VerificationStatusSkipped, "skipped"},
		{VerificationStatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.status)
		}
	}
}

func TestReplayDecision_Decisions(t *testing.T) {
	tests := []struct {
		decision ReplayDecision
		expected string
	}{
		{ReplayDecisionRestoreAndSkip, "restore_and_skip"},
		{ReplayDecisionExecute, "execute"},
		{ReplayDecisionFail, "fail"},
		{ReplayDecisionNeedsReview, "needs_review"},
	}

	for _, tt := range tests {
		if string(tt.decision) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.decision)
		}
	}
}
