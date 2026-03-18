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

package distributed

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLedgerSyncRequest(t *testing.T) {
	req := LedgerSyncRequest{
		OrgID:     "org-1",
		JobID:     "job-1",
		Events:    []Event{},
		Signature: "test-signature",
	}

	if req.OrgID != "org-1" {
		t.Errorf("expected org-1, got %s", req.OrgID)
	}

	if req.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", req.JobID)
	}
}

func TestEvent(t *testing.T) {
	event := Event{
		ID:        "event-1",
		Type:      "test",
		Payload:   []byte(`{"key": "value"}`),
		Hash:      "abc123",
		CreatedAt: time.Now(),
	}

	if event.ID != "event-1" {
		t.Errorf("expected event-1, got %s", event.ID)
	}

	if event.Type != "test" {
		t.Errorf("expected test, got %s", event.Type)
	}

	if string(event.Payload) != `{"key": "value"}` {
		t.Errorf("expected payload, got %s", event.Payload)
	}
}

func TestLedgerSyncResponse(t *testing.T) {
	resp := LedgerSyncResponse{
		Accepted:    true,
		ConflictIDs: []string{"conflict-1"},
		LocalHash:   "hash-123",
	}

	if !resp.Accepted {
		t.Error("expected accepted")
	}

	if len(resp.ConflictIDs) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(resp.ConflictIDs))
	}
}

func TestDistributedVerifier_New(t *testing.T) {
	verifier := NewDistributedVerifier()
	if verifier == nil {
		t.Error("expected verifier, got nil")
	}
}

func TestDistributedVerifier_VerifyAcrossOrgs_EmptyJobID(t *testing.T) {
	verifier := NewDistributedVerifier()

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "", []string{"org-1"})
	if err == nil {
		t.Error("expected error for empty job ID")
	}

	if result != nil {
		t.Error("expected nil result for empty job ID")
	}
}

func TestDistributedVerifier_VerifyAcrossOrgs_EmptyOrgs(t *testing.T) {
	verifier := NewDistributedVerifier()

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "job-1", []string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Consensus {
		t.Error("expected consensus true for empty orgs")
	}

	if result.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", result.JobID)
	}
}

func TestDistributedVerifier_VerifyAcrossOrgs_NoSource(t *testing.T) {
	verifier := NewDistributedVerifier()

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "job-1", []string{"org-1", "org-2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Consensus {
		t.Error("expected consensus true when no source")
	}

	if len(result.Divergences) != 0 {
		t.Errorf("expected no divergences, got %d", len(result.Divergences))
	}
}

// MockOrgEventSource implements OrgEventSource for testing
type mockOrgEventSource struct {
	events map[string][]Event
	err    error
}

func (m *mockOrgEventSource) PullOrgEvents(ctx context.Context, orgID string, jobID string) ([]Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := orgID + "/" + jobID
	if events, ok := m.events[key]; ok {
		return events, nil
	}
	return []Event{}, nil
}

func TestDistributedVerifier_VerifyAcrossOrgs_WithMockSource(t *testing.T) {
	verifier := NewDistributedVerifier()

	mockSource := &mockOrgEventSource{
		events: map[string][]Event{
			"org-1/job-1": {
				{ID: "e1", Type: "test", Hash: "hash1"},
				{ID: "e2", Type: "test", Hash: "root-hash"},
			},
			"org-2/job-1": {
				{ID: "e1", Type: "test", Hash: "hash1"},
				{ID: "e2", Type: "test", Hash: "root-hash"},
			},
		},
	}

	verifier.WithEventSource(mockSource)

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "job-1", []string{"org-1", "org-2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Consensus {
		t.Error("expected consensus true for matching hashes")
	}
}

func TestDistributedVerifier_VerifyAcrossOrgs_HashMismatch(t *testing.T) {
	verifier := NewDistributedVerifier()

	mockSource := &mockOrgEventSource{
		events: map[string][]Event{
			"org-1/job-1": {
				{ID: "e1", Type: "test", Hash: "hash1"},
				{ID: "e2", Type: "test", Hash: "root-hash-1"},
			},
			"org-2/job-1": {
				{ID: "e1", Type: "test", Hash: "hash1"},
				{ID: "e2", Type: "test", Hash: "root-hash-2"},
			},
		},
	}

	verifier.WithEventSource(mockSource)

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "job-1", []string{"org-1", "org-2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Consensus {
		t.Error("expected consensus false for hash mismatch")
	}

	if len(result.Divergences) == 0 {
		t.Error("expected divergences for hash mismatch")
	}
}

func TestDistributedVerifier_VerifyAcrossOrgs_PullError(t *testing.T) {
	verifier := NewDistributedVerifier()

	mockSource := &mockOrgEventSource{
		err: errors.New("pull failed"),
	}

	verifier.WithEventSource(mockSource)

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "job-1", []string{"org-1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Consensus {
		t.Error("expected consensus false for pull error")
	}

	if len(result.Divergences) == 0 {
		t.Error("expected divergences for pull error")
	}
}

func TestDistributedVerifier_VerifyAcrossOrgs_EmptyEvents(t *testing.T) {
	verifier := NewDistributedVerifier()

	mockSource := &mockOrgEventSource{
		events: map[string][]Event{
			"org-1/job-1": {},
		},
	}

	verifier.WithEventSource(mockSource)

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "job-1", []string{"org-1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Consensus {
		t.Error("expected consensus false for empty events")
	}
}

func TestDistributedVerifier_VerifyAcrossOrgs_MissingHash(t *testing.T) {
	verifier := NewDistributedVerifier()

	mockSource := &mockOrgEventSource{
		events: map[string][]Event{
			"org-1/job-1": {
				{ID: "e1", Type: "test", Hash: ""},
			},
		},
	}

	verifier.WithEventSource(mockSource)

	result, err := verifier.VerifyAcrossOrgs(context.Background(), "job-1", []string{"org-1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Consensus {
		t.Error("expected consensus false for missing hash")
	}
}

func TestDistributedVerifier_WithSyncProtocol(t *testing.T) {
	verifier := NewDistributedVerifier()

	// Test with nil protocol
	result := verifier.WithSyncProtocol(nil)
	if result.source != nil {
		t.Error("expected nil source for nil protocol")
	}
}

type mockSyncProtocolV2 struct {
	pullEvents map[string][]Event
	err        error
}

func (m *mockSyncProtocolV2) Push(ctx context.Context, targetOrg string, req LedgerSyncRequest) (*LedgerSyncResponse, error) {
	return nil, nil
}

func (m *mockSyncProtocolV2) Pull(ctx context.Context, sourceOrg string, jobID string) ([]Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := sourceOrg + "/" + jobID
	return m.pullEvents[key], nil
}

func (m *mockSyncProtocolV2) Resolve(ctx context.Context, conflicts []string) error {
	return nil
}

func TestProtocolEventSource(t *testing.T) {
	mockProtocol := &mockSyncProtocolV2{
		pullEvents: map[string][]Event{
			"org-1/job-1": {
				{ID: "e1", Type: "test", Hash: "hash1"},
			},
		},
	}

	source := &protocolEventSource{protocol: mockProtocol}
	events, err := source.PullOrgEvents(context.Background(), "org-1", "job-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}
