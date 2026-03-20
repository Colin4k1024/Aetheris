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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/ed25519"
)

// SignEvidence signs the evidence package contents and returns a signature.
func SignEvidence(privateKey ed25519.PrivateKey, manifest Manifest, events []Event, ledger []ToolInvocation) (string, error) {
	manifestBytes, _ := json.Marshal(manifest)
	eventsBytes, _ := eventsToNDJSON(events)
	ledgerBytes, _ := ledgerToNDJSON(ledger)

	payload := append(manifestBytes, eventsBytes...)
	payload = append(payload, ledgerBytes...)

	sig := ed25519.Sign(privateKey, payload)
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifySignature verifies an Ed25519 signature against the evidence package contents.
func VerifySignature(publicKey ed25519.PublicKey, manifest Manifest, events []Event, ledger []ToolInvocation, signature string) error {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	manifestBytes, _ := json.Marshal(manifest)
	eventsBytes, _ := eventsToNDJSON(events)
	ledgerBytes, _ := ledgerToNDJSON(ledger)

	payload := append(manifestBytes, eventsBytes...)
	payload = append(payload, ledgerBytes...)

	if !ed25519.Verify(publicKey, payload, sigBytes) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// SignedAt returns current time in ISO 8601 format.
func SignedAt() string {
	return time.Now().UTC().Format(time.RFC3339)
}
