package distributed

import (
	"context"
	"errors"
	"testing"
)

type fakeOrgEventSource struct {
	events map[string][]Event
	errs   map[string]error
}

func (f *fakeOrgEventSource) PullOrgEvents(ctx context.Context, orgID string, jobID string) ([]Event, error) {
	if err, ok := f.errs[orgID]; ok {
		return nil, err
	}
	return append([]Event(nil), f.events[orgID]...), nil
}

func TestVerifyAcrossOrgs_Consensus(t *testing.T) {
	v := NewDistributedVerifier().WithEventSource(&fakeOrgEventSource{
		events: map[string][]Event{
			"org_a": {{Hash: "h1"}, {Hash: "root"}},
			"org_b": {{Hash: "h1"}, {Hash: "root"}},
		},
	})
	res, err := v.VerifyAcrossOrgs(context.Background(), "job_1", []string{"org_a", "org_b"})
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !res.Consensus {
		t.Fatalf("expected consensus=true, divergences=%v", res.Divergences)
	}
	if res.RootByOrg["org_a"] != "root" || res.RootByOrg["org_b"] != "root" {
		t.Fatalf("expected root_by_org to expose accepted roots: %+v", res.RootByOrg)
	}
}

func TestVerifyAcrossOrgs_Divergence(t *testing.T) {
	v := NewDistributedVerifier().WithEventSource(&fakeOrgEventSource{
		events: map[string][]Event{
			"org_a": {{Hash: "root_a"}},
			"org_b": {{Hash: "root_b"}},
		},
		errs: map[string]error{
			"org_c": errors.New("timeout"),
		},
	})
	res, err := v.VerifyAcrossOrgs(context.Background(), "job_2", []string{"org_a", "org_b", "org_c"})
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if res.Consensus {
		t.Fatal("expected consensus=false")
	}
	if len(res.Divergences) == 0 {
		t.Fatal("expected divergences")
	}
}

func TestVerifyAcrossOrgs_EmptyJobID(t *testing.T) {
	v := NewDistributedVerifier()
	_, err := v.VerifyAcrossOrgs(context.Background(), "", []string{"org_a"})
	// Empty jobID should return error
	if err == nil {
		t.Fatal("expected error for empty jobID")
	}
}

func TestVerifyAcrossOrgs_EmptyOrgs(t *testing.T) {
	v := NewDistributedVerifier()
	res, err := v.VerifyAcrossOrgs(context.Background(), "job_1", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Consensus {
		t.Error("expected consensus=true for empty orgs")
	}
}

func TestVerifyAcrossOrgs_NilSource(t *testing.T) {
	v := NewDistributedVerifier()
	res, err := v.VerifyAcrossOrgs(context.Background(), "job_1", []string{"org_a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Consensus {
		t.Error("expected consensus=true for nil source")
	}
}

func TestVerifyAcrossOrgs_EmptyEventStream(t *testing.T) {
	v := NewDistributedVerifier().WithEventSource(&fakeOrgEventSource{
		events: map[string][]Event{
			"org_a": {},
		},
	})
	res, err := v.VerifyAcrossOrgs(context.Background(), "job_1", []string{"org_a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Consensus {
		t.Error("expected consensus=false for empty event stream")
	}
}

func TestVerifyAcrossOrgs_MissingHash(t *testing.T) {
	v := NewDistributedVerifier().WithEventSource(&fakeOrgEventSource{
		events: map[string][]Event{
			"org_a": {{ID: "e1", Type: "test"}},
		},
	})
	res, err := v.VerifyAcrossOrgs(context.Background(), "job_1", []string{"org_a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Consensus {
		t.Error("expected consensus=false for missing hash")
	}
}

func TestVerifyAcrossOrgs_WithSyncProtocol(t *testing.T) {
	mockProtocol := &mockSyncProtocol{
		events: map[string][]Event{
			"org_a": {{Hash: "root"}},
			"org_b": {{Hash: "root"}},
		},
	}
	v := NewDistributedVerifier().WithSyncProtocol(mockProtocol)
	res, err := v.VerifyAcrossOrgs(context.Background(), "job_1", []string{"org_a", "org_b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Consensus {
		t.Errorf("expected consensus=true, got %v", res.Divergences)
	}
}

func TestMultiOrgVerifyResult(t *testing.T) {
	result := MultiOrgVerifyResult{
		JobID:         "job-1",
		Organizations: []string{"org_a", "org_b"},
		Consensus:     true,
		Divergences:   nil,
	}

	if result.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", result.JobID)
	}
	if len(result.Organizations) != 2 {
		t.Errorf("expected 2 orgs, got %d", len(result.Organizations))
	}
	if !result.Consensus {
		t.Error("expected consensus=true")
	}
}

func TestDistributedVerifierReleaseDrill_AcceptedAndDivergentRoots(t *testing.T) {
	accepted := NewDistributedVerifier().WithEventSource(&fakeOrgEventSource{
		events: map[string][]Event{
			"org_a": {{Hash: "root"}},
			"org_b": {{Hash: "root"}},
		},
	})
	acceptedResult, err := accepted.VerifyAcrossOrgs(context.Background(), "job_release", []string{"org_a", "org_b"})
	if err != nil {
		t.Fatalf("accepted verify failed: %v", err)
	}
	if !acceptedResult.Consensus {
		t.Fatalf("expected accepted roots to reach consensus: %+v", acceptedResult)
	}

	divergent := NewDistributedVerifier().WithEventSource(&fakeOrgEventSource{
		events: map[string][]Event{
			"org_a": {{Hash: "root_a"}},
			"org_b": {{Hash: "root_b"}},
		},
	})
	divergentResult, err := divergent.VerifyAcrossOrgs(context.Background(), "job_release", []string{"org_a", "org_b"})
	if err != nil {
		t.Fatalf("divergent verify failed: %v", err)
	}
	if divergentResult.Consensus {
		t.Fatalf("expected divergent roots to fail consensus")
	}
	if len(divergentResult.Divergences) == 0 {
		t.Fatalf("expected divergence details")
	}
}

func TestDistributedVerifierReleaseDrill_ReadinessRequiresOperationalEvidence(t *testing.T) {
	notReady := AssessPromotionReadiness(PromotionEvidence{
		RootHashDrillCovered: true,
	})
	if notReady.Ready {
		t.Fatalf("distributed verifier should stay prototype without saturation, lease, and recovery evidence")
	}
	if len(notReady.Missing) != 3 {
		t.Fatalf("missing evidence count = %d, want 3: %+v", len(notReady.Missing), notReady.Missing)
	}

	ready := AssessPromotionReadiness(PromotionEvidence{
		SingleNodeSaturationObserved: true,
		LeaseFailureModesCovered:     true,
		RecoveryFailureModesCovered:  true,
		RootHashDrillCovered:         true,
	})
	if !ready.Ready {
		t.Fatalf("expected readiness when all evidence is present: %+v", ready)
	}
}

type mockSyncProtocol struct {
	events map[string][]Event
	err    error
}

func (m *mockSyncProtocol) Push(ctx context.Context, targetOrg string, req LedgerSyncRequest) (*LedgerSyncResponse, error) {
	return nil, nil
}

func (m *mockSyncProtocol) Pull(ctx context.Context, sourceOrg string, jobID string) ([]Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events[sourceOrg], nil
}

func (m *mockSyncProtocol) Resolve(ctx context.Context, conflicts []string) error {
	return nil
}
