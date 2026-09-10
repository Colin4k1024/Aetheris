// Copyright 2026 Aetheris
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

package compliance

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- mock probes ---

type mockAtRestProbe struct {
	evidence *EncryptionEvidence
	err      error
}

func (m *mockAtRestProbe) Probe(_ context.Context) (*EncryptionEvidence, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.evidence, nil
}

type mockTLSProbe struct {
	evidence *TLSEvidence
	err      error
}

func (m *mockTLSProbe) Probe(_ context.Context) (*TLSEvidence, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.evidence, nil
}

// --- CheckEncryptionAtRest tests ---

func TestCheckEncryptionAtRest_NotRequired(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionAtRest: false})
	if err := h.CheckEncryptionAtRest(context.Background()); err != nil {
		t.Fatalf("expected nil when not required, got %v", err)
	}
}

func TestCheckEncryptionAtRest_NoProbe_ReturnsError(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionAtRest: true})
	err := h.CheckEncryptionAtRest(context.Background())
	if err == nil {
		t.Fatal("expected error when no probe configured")
	}
}

func TestCheckEncryptionAtRest_Encrypted_Passes(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionAtRest: true})
	h.SetEncryptionAtRestProbe(&mockAtRestProbe{evidence: &EncryptionEvidence{
		Encrypted:  true,
		Algorithm:  "AES-256",
		Provider:   "KMS",
		VerifiedAt: time.Now(),
	}})
	if err := h.CheckEncryptionAtRest(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckEncryptionAtRest_NotEncrypted_Fails(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionAtRest: true})
	h.SetEncryptionAtRestProbe(&mockAtRestProbe{evidence: &EncryptionEvidence{
		Encrypted: false,
	}})
	err := h.CheckEncryptionAtRest(context.Background())
	if err == nil {
		t.Fatal("expected error when storage not encrypted")
	}
}

func TestCheckEncryptionAtRest_NilEvidence_Fails(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionAtRest: true})
	h.SetEncryptionAtRestProbe(&mockAtRestProbe{evidence: nil})
	err := h.CheckEncryptionAtRest(context.Background())
	if err == nil {
		t.Fatal("expected error when evidence is nil")
	}
}

func TestCheckEncryptionAtRest_ProbeError_Fails(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionAtRest: true})
	h.SetEncryptionAtRestProbe(&mockAtRestProbe{err: errors.New("storage unreachable")})
	err := h.CheckEncryptionAtRest(context.Background())
	if err == nil {
		t.Fatal("expected error when probe fails")
	}
}

// --- CheckEncryptionInTransit tests ---

func TestCheckEncryptionInTransit_NotRequired(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionInTransit: false})
	if err := h.CheckEncryptionInTransit(context.Background()); err != nil {
		t.Fatalf("expected nil when not required, got %v", err)
	}
}

func TestCheckEncryptionInTransit_NoProbe_ReturnsError(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionInTransit: true})
	err := h.CheckEncryptionInTransit(context.Background())
	if err == nil {
		t.Fatal("expected error when no TLS probe configured")
	}
}

func TestCheckEncryptionInTransit_TLSEnabled_Passes(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionInTransit: true})
	h.SetTLSConnectionProbe(&mockTLSProbe{evidence: &TLSEvidence{
		TLSEnabled:  true,
		Version:     "TLS 1.3",
		CipherSuite: "TLS_AES_256_GCM_SHA384",
		VerifiedAt:  time.Now(),
	}})
	if err := h.CheckEncryptionInTransit(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckEncryptionInTransit_TLSMissing_Fails(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionInTransit: true})
	h.SetTLSConnectionProbe(&mockTLSProbe{evidence: &TLSEvidence{
		TLSEnabled: false,
	}})
	err := h.CheckEncryptionInTransit(context.Background())
	if err == nil {
		t.Fatal("expected error when TLS not enabled")
	}
}

func TestCheckEncryptionInTransit_NilEvidence_Fails(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionInTransit: true})
	h.SetTLSConnectionProbe(&mockTLSProbe{evidence: nil})
	err := h.CheckEncryptionInTransit(context.Background())
	if err == nil {
		t.Fatal("expected error when evidence is nil")
	}
}

func TestCheckEncryptionInTransit_ProbeError_Fails(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{RequireEncryptionInTransit: true})
	h.SetTLSConnectionProbe(&mockTLSProbe{err: errors.New("connection refused")})
	err := h.CheckEncryptionInTransit(context.Background())
	if err == nil {
		t.Fatal("expected error when probe fails")
	}
}

// --- ValidatePHIData regression ---

func TestValidatePHIData_NotEncrypted_Fails(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{EnablePHIEncryption: true})
	err := h.ValidatePHIData(context.Background(), PHIData{
		Type:      "name",
		Value:     "test",
		Encrypted: false,
	})
	if err == nil {
		t.Fatal("expected error for unencrypted PHI")
	}
}

func TestValidatePHIData_Encrypted_Passes(t *testing.T) {
	h := NewHIPAACompliance(&HIPAAConfig{EnablePHIEncryption: true})
	err := h.ValidatePHIData(context.Background(), PHIData{
		Type:      "name",
		Value:     "test",
		Encrypted: true,
		ExpiredAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
