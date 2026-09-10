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
	"errors"
	"testing"
)

// --- mock implementations ---

type mockToolLedgerLookup struct {
	entries map[string]*ToolLedgerEntry
	err     error
}

func (m *mockToolLedgerLookup) LookupByExternalRef(_ context.Context, ref string) (*ToolLedgerEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	if e, ok := m.entries[ref]; ok {
		return e, nil
	}
	return nil, ErrToolLedgerNotFound
}

type mockDatabaseLookup struct {
	versions map[string]string
	etags    map[string]string
	err      error
}

func (m *mockDatabaseLookup) Lookup(_ context.Context, resourceID string) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	v, ok := m.versions[resourceID]
	if !ok {
		return "", "", ErrDatabaseResourceNotFound
	}
	return v, m.etags[resourceID], nil
}

// --- ReplayVerifier tests ---

func TestReplayVerifier_Verify_NilChanges(t *testing.T) {
	ctx := context.Background()
	verifier := NewReplayVerifier()

	result, err := verifier.Verify(ctx, "job-1", nil)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Decision != ReplayDecisionExecute {
		t.Errorf("expected Execute for nil state changes, got %v", result.Decision)
	}
}

func TestReplayVerifier_Verify_EmptyChanges(t *testing.T) {
	ctx := context.Background()
	verifier := NewReplayVerifier()

	result, err := verifier.Verify(ctx, "job-1", map[string][]StateChangeRecord{})
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Decision != ReplayDecisionExecute {
		t.Errorf("expected Execute for empty state changes, got %v", result.Decision)
	}
}

func TestReplayVerifier_NoVerifiers_AllSkipped(t *testing.T) {
	ctx := context.Background()
	verifier := NewReplayVerifier()

	stateChanges := map[string][]StateChangeRecord{
		"step-1": {
			{ResourceType: "database", ResourceID: "user-123", Operation: "update", Version: "v2"},
		},
	}
	result, err := verifier.Verify(ctx, "job-1", stateChanges)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	// Without verifiers, all records are skipped → Execute
	if result.Decision != ReplayDecisionExecute {
		t.Errorf("expected Execute, got %v", result.Decision)
	}
	if result.SkippedCount != 1 {
		t.Errorf("expected 1 skipped, got %d", result.SkippedCount)
	}
}

// --- ToolLedgerVerifier tests ---

func TestToolLedgerVerifier_Name(t *testing.T) {
	v := &ToolLedgerVerifier{}
	if v.Name() != "ToolLedgerVerifier" {
		t.Errorf("expected ToolLedgerVerifier, got %s", v.Name())
	}
}

func TestToolLedgerVerifier_NoStore_ReturnsError(t *testing.T) {
	v := &ToolLedgerVerifier{}
	record := StateChangeRecord{ExternalRef: "ref-1", ResourceType: "tool"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusError {
		t.Errorf("expected Error, got %v", result.Status)
	}
}

func TestToolLedgerVerifier_NoExternalRef_Skipped(t *testing.T) {
	v := NewToolLedgerVerifier(&mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{}})
	record := StateChangeRecord{}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusSkipped {
		t.Errorf("expected Skipped, got %v", result.Status)
	}
}

func TestToolLedgerVerifier_Confirmed_Match(t *testing.T) {
	lookup := &mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{
		"ref-1": {Status: "confirmed", ResultHash: "abc", ExternalRef: "ref-1"},
	}}
	v := NewToolLedgerVerifier(lookup)
	record := StateChangeRecord{ExternalRef: "ref-1", ResourceType: "tool"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMatch {
		t.Errorf("expected Match, got %v: %s", result.Status, result.Message)
	}
}

func TestToolLedgerVerifier_Success_Match(t *testing.T) {
	lookup := &mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{
		"ref-1": {Status: "success", ResultHash: "abc"},
	}}
	v := NewToolLedgerVerifier(lookup)
	record := StateChangeRecord{ExternalRef: "ref-1", ToolName: "web_search"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMatch {
		t.Errorf("expected Match, got %v", result.Status)
	}
}

func TestToolLedgerVerifier_Started_Pending(t *testing.T) {
	lookup := &mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{
		"ref-1": {Status: "started"},
	}}
	v := NewToolLedgerVerifier(lookup)
	record := StateChangeRecord{ExternalRef: "ref-1", ToolName: "web_search"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusPending {
		t.Errorf("expected Pending, got %v", result.Status)
	}
}

func TestToolLedgerVerifier_NotFound_Mismatch(t *testing.T) {
	lookup := &mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{}}
	v := NewToolLedgerVerifier(lookup)
	record := StateChangeRecord{ExternalRef: "ref-1", ToolName: "web_search"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMismatch {
		t.Errorf("expected Mismatch, got %v", result.Status)
	}
}

func TestToolLedgerVerifier_LookupError_Error(t *testing.T) {
	lookup := &mockToolLedgerLookup{err: errors.New("db down")}
	v := NewToolLedgerVerifier(lookup)
	record := StateChangeRecord{ExternalRef: "ref-1", ToolName: "web_search"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusError {
		t.Errorf("expected Error, got %v", result.Status)
	}
}

func TestToolLedgerVerifier_FailureStatus_Mismatch(t *testing.T) {
	lookup := &mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{
		"ref-1": {Status: "failure"},
	}}
	v := NewToolLedgerVerifier(lookup)
	record := StateChangeRecord{ExternalRef: "ref-1", ToolName: "web_search"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMismatch {
		t.Errorf("expected Mismatch, got %v", result.Status)
	}
}

func TestToolLedgerVerifier_ResourceIDFallback(t *testing.T) {
	lookup := &mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{
		"res-123": {Status: "confirmed"},
	}}
	v := NewToolLedgerVerifier(lookup)
	record := StateChangeRecord{ResourceID: "res-123", ToolName: "file_writer"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMatch {
		t.Errorf("expected Match, got %v", result.Status)
	}
}

// --- DatabaseStateVerifier tests ---

func TestDatabaseStateVerifier_Name(t *testing.T) {
	v := &DatabaseStateVerifier{}
	if v.Name() != "DatabaseStateVerifier" {
		t.Errorf("expected DatabaseStateVerifier, got %s", v.Name())
	}
}

func TestDatabaseStateVerifier_NoLookup_ReturnsError(t *testing.T) {
	v := &DatabaseStateVerifier{}
	record := StateChangeRecord{ResourceType: "database", ResourceID: "user-1"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusError {
		t.Errorf("expected Error, got %v", result.Status)
	}
}

func TestDatabaseStateVerifier_NotDatabase_Skipped(t *testing.T) {
	v := NewDatabaseStateVerifier(&mockDatabaseLookup{})
	record := StateChangeRecord{ResourceType: "file", ResourceID: "x"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusSkipped {
		t.Errorf("expected Skipped, got %v", result.Status)
	}
}

func TestDatabaseStateVerifier_EmptyResourceID_Skipped(t *testing.T) {
	v := NewDatabaseStateVerifier(&mockDatabaseLookup{})
	record := StateChangeRecord{ResourceType: "database"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusSkipped {
		t.Errorf("expected Skipped, got %v", result.Status)
	}
}

func TestDatabaseStateVerifier_VersionMatch(t *testing.T) {
	lookup := &mockDatabaseLookup{
		versions: map[string]string{"user-1": "v2"},
		etags:    map[string]string{"user-1": "abc123"},
	}
	v := NewDatabaseStateVerifier(lookup)
	record := StateChangeRecord{ResourceType: "database", ResourceID: "user-1", Version: "v2", Etag: "abc123"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMatch {
		t.Errorf("expected Match, got %v: %s", result.Status, result.Message)
	}
}

func TestDatabaseStateVerifier_VersionMismatch(t *testing.T) {
	lookup := &mockDatabaseLookup{
		versions: map[string]string{"user-1": "v3"},
		etags:    map[string]string{"user-1": "abc123"},
	}
	v := NewDatabaseStateVerifier(lookup)
	record := StateChangeRecord{ResourceType: "database", ResourceID: "user-1", Version: "v2", Etag: "abc123"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMismatch {
		t.Errorf("expected Mismatch, got %v", result.Status)
	}
}

func TestDatabaseStateVerifier_EtagMismatch(t *testing.T) {
	lookup := &mockDatabaseLookup{
		versions: map[string]string{"user-1": "v2"},
		etags:    map[string]string{"user-1": "xyz789"},
	}
	v := NewDatabaseStateVerifier(lookup)
	record := StateChangeRecord{ResourceType: "database", ResourceID: "user-1", Version: "v2", Etag: "abc123"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMismatch {
		t.Errorf("expected Mismatch, got %v", result.Status)
	}
}

func TestDatabaseStateVerifier_NotFound_Mismatch(t *testing.T) {
	lookup := &mockDatabaseLookup{versions: map[string]string{}, etags: map[string]string{}}
	v := NewDatabaseStateVerifier(lookup)
	record := StateChangeRecord{ResourceType: "database", ResourceID: "user-1"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMismatch {
		t.Errorf("expected Mismatch, got %v", result.Status)
	}
}

func TestDatabaseStateVerifier_LookupError_Error(t *testing.T) {
	lookup := &mockDatabaseLookup{err: errors.New("connection refused")}
	v := NewDatabaseStateVerifier(lookup)
	record := StateChangeRecord{ResourceType: "database", ResourceID: "user-1"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusError {
		t.Errorf("expected Error, got %v", result.Status)
	}
}

func TestDatabaseStateVerifier_NoVersionEtag_Match(t *testing.T) {
	// record 没有声明 version/etag，只要资源存在即 match
	lookup := &mockDatabaseLookup{
		versions: map[string]string{"user-1": "v2"},
		etags:    map[string]string{"user-1": "abc123"},
	}
	v := NewDatabaseStateVerifier(lookup)
	record := StateChangeRecord{ResourceType: "database", ResourceID: "user-1"}
	result, err := v.VerifyStateChange(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != VerificationStatusMatch {
		t.Errorf("expected Match, got %v", result.Status)
	}
}

// --- integration: ReplayVerifier with real verifiers ---

func TestReplayVerifier_WithRealVerifiers_Match(t *testing.T) {
	ctx := context.Background()
	ledgerLookup := &mockToolLedgerLookup{entries: map[string]*ToolLedgerEntry{
		"ref-1": {Status: "confirmed", ResultHash: "abc"},
	}}
	dbLookup := &mockDatabaseLookup{
		versions: map[string]string{"user-123": "v2"},
		etags:    map[string]string{"user-123": "abc123"},
	}
	verifier := NewReplayVerifier(
		NewToolLedgerVerifier(ledgerLookup),
		NewDatabaseStateVerifier(dbLookup),
	)
	stateChanges := map[string][]StateChangeRecord{
		"step-1": {
			{ResourceType: "database", ResourceID: "user-123", Operation: "update", Version: "v2", Etag: "abc123"},
		},
		"step-2": {
			{ExternalRef: "ref-1", ResourceType: "tool", Operation: "created"},
		},
	}
	result, err := verifier.Verify(ctx, "job-1", stateChanges)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.OverallStatus != VerificationStatusMatch {
		t.Errorf("expected Match, got %v", result.OverallStatus)
	}
	if result.Decision != ReplayDecisionRestoreAndSkip {
		t.Errorf("expected RestoreAndSkip, got %v", result.Decision)
	}
	if result.MatchedCount != 2 {
		t.Errorf("expected 2 matched, got %d", result.MatchedCount)
	}
}

func TestReplayVerifier_WithMismatch_Fails(t *testing.T) {
	ctx := context.Background()
	dbLookup := &mockDatabaseLookup{
		versions: map[string]string{"user-123": "v3"},
		etags:    map[string]string{"user-123": "abc123"},
	}
	verifier := NewReplayVerifier(NewDatabaseStateVerifier(dbLookup))
	stateChanges := map[string][]StateChangeRecord{
		"step-1": {
			{ResourceType: "database", ResourceID: "user-123", Version: "v2", Etag: "abc123"},
		},
	}
	result, err := verifier.Verify(ctx, "job-1", stateChanges)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.OverallStatus != VerificationStatusMismatch {
		t.Errorf("expected Mismatch, got %v", result.OverallStatus)
	}
	if result.Decision != ReplayDecisionFail {
		t.Errorf("expected Fail, got %v", result.Decision)
	}
}

func TestReplayVerifier_WithError_NeedsReview(t *testing.T) {
	ctx := context.Background()
	// DatabaseStateVerifier without lookup → error
	verifier := NewReplayVerifier(&DatabaseStateVerifier{})
	stateChanges := map[string][]StateChangeRecord{
		"step-1": {
			{ResourceType: "database", ResourceID: "user-123", Version: "v2"},
		},
	}
	result, err := verifier.Verify(ctx, "job-1", stateChanges)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.OverallStatus != VerificationStatusError {
		t.Errorf("expected Error, got %v", result.OverallStatus)
	}
	if result.Decision != ReplayDecisionNeedsReview {
		t.Errorf("expected NeedsReview, got %v", result.Decision)
	}
}

// --- status/decision enum tests ---

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
