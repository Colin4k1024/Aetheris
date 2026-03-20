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

package proof

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func TestSignAndVerifyEvidence(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	events := []Event{
		{ID: "1", JobID: "job_test", Type: "job_created", Hash: "abc", CreatedAt: time.Now()},
	}
	ledger := []ToolInvocation{
		{ID: "inv1", JobID: "job_test", IdempotencyKey: "key1"},
	}
	manifest := Manifest{JobID: "job_test", Version: "2.0"}

	sig, err := SignEvidence(privateKey, manifest, events, ledger)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	err = VerifySignature(publicKey, manifest, events, ledger, sig)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestSignAndVerifyEvidence_TamperedContent(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	events := []Event{
		{ID: "1", JobID: "job_test", Type: "job_created", Hash: "abc", CreatedAt: time.Now()},
	}
	ledger := []ToolInvocation{
		{ID: "inv1", JobID: "job_test", IdempotencyKey: "key1"},
	}
	manifest := Manifest{JobID: "job_test", Version: "2.0"}

	sig, err := SignEvidence(privateKey, manifest, events, ledger)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Tamper with events
	tamperedEvents := []Event{
		{ID: "1", JobID: "job_test", Type: "job_created", Hash: "tampered", CreatedAt: time.Now()},
	}

	err = VerifySignature(publicKey, manifest, tamperedEvents, ledger, sig)
	if err == nil {
		t.Fatal("expected verification to fail with tampered content, but it passed")
	}
}

func TestSignAndVerifyEvidence_WrongKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	_, anotherPrivateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	events := []Event{
		{ID: "1", JobID: "job_test", Type: "job_created", Hash: "abc", CreatedAt: time.Now()},
	}
	ledger := []ToolInvocation{
		{ID: "inv1", JobID: "job_test", IdempotencyKey: "key1"},
	}
	manifest := Manifest{JobID: "job_test", Version: "2.0"}

	sig, err := SignEvidence(privateKey, manifest, events, ledger)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Verify with wrong public key (from different key pair)
	err = VerifySignature(anotherPrivateKey.Public().(ed25519.PublicKey), manifest, events, ledger, sig)
	if err == nil {
		t.Fatal("expected verification to fail with wrong public key, but it passed")
	}
}

func TestSignEvidence_InvalidSignatureEncoding(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	events := []Event{
		{ID: "1", JobID: "job_test", Type: "job_created", Hash: "abc", CreatedAt: time.Now()},
	}
	ledger := []ToolInvocation{}
	manifest := Manifest{JobID: "job_test", Version: "2.0"}

	// Sign with valid key
	sig, err := SignEvidence(privateKey, manifest, events, ledger)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Corrupt the signature
	corruptedSig := sig[:len(sig)-4] + "XXXX"

	// Verify should fail
	err = VerifySignature(publicKey, manifest, events, ledger, corruptedSig)
	if err == nil {
		t.Fatal("expected verification to fail with corrupted signature, but it passed")
	}
}
