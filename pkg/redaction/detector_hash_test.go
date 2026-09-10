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

package redaction

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// --- Hash mode tests ---

func TestHash_DifferentInputsDifferentDigests(t *testing.T) {
	d := NewPIIDetector()
	h1 := d.hashValue("test@example.com")
	h2 := d.hashValue("other@example.com")
	if h1 == h2 {
		t.Error("expected different hashes for different inputs")
	}
}

func TestHash_SameInputSameDigest(t *testing.T) {
	d := NewPIIDetector()
	h1 := d.hashValue("test@example.com")
	h2 := d.hashValue("test@example.com")
	if h1 != h2 {
		t.Error("expected same hash for same input")
	}
}

func TestHash_WithSalt(t *testing.T) {
	d1 := NewPIIDetector()
	d1.SetHashSalt("salt-1")
	d2 := NewPIIDetector()
	d2.SetHashSalt("salt-2")
	h1 := d1.hashValue("test@example.com")
	h2 := d2.hashValue("test@example.com")
	if h1 == h2 {
		t.Error("expected different hashes with different salts")
	}
}

func TestHash_MatchesManualSHA256(t *testing.T) {
	salt := "mysalt"
	d := NewPIIDetector()
	d.SetHashSalt(salt)
	result := d.hashValue("secret")
	// Verify it matches manual computation
	manual := sha256.New()
	manual.Write([]byte("secret"))
	manual.Write([]byte(salt))
	expected := "hash:" + hex.EncodeToString(manual.Sum(nil))
	if result != expected {
		t.Errorf("hash mismatch: got %s, expected %s", result, expected)
	}
}

func TestHash_PrefixFormat(t *testing.T) {
	d := NewPIIDetector()
	result := d.hashValue("test")
	if !strings.HasPrefix(result, "hash:") {
		t.Errorf("expected hash: prefix, got %s", result)
	}
}

// --- Encrypt mode tests ---

var testEncryptKey = []byte("0123456789abcdef0123456789abcdef") // 32 bytes

func TestEncrypt_RoundTrip(t *testing.T) {
	d := NewPIIDetectorWithKeys("", testEncryptKey)
	original := "test@example.com"
	encrypted, err := d.encryptValue(original)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	decrypted, err := d.DecryptValue(encrypted)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != original {
		t.Errorf("round-trip mismatch: got %s, expected %s", decrypted, original)
	}
}

func TestEncrypt_PrefixFormat(t *testing.T) {
	d := NewPIIDetectorWithKeys("", testEncryptKey)
	encrypted, err := d.encryptValue("secret")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if !strings.HasPrefix(encrypted, "enc:") {
		t.Errorf("expected enc: prefix, got %s", encrypted)
	}
}

func TestEncrypt_DifferentNoncesDifferentCiphertext(t *testing.T) {
	d := NewPIIDetectorWithKeys("", testEncryptKey)
	e1, _ := d.encryptValue("same-value")
	e2, _ := d.encryptValue("same-value")
	// AES-GCM uses random nonce, so same plaintext → different ciphertext
	if e1 == e2 {
		t.Error("expected different ciphertext for same input (random nonce)")
	}
	// But both should decrypt to same value
	d1, _ := d.DecryptValue(e1)
	d2, _ := d.DecryptValue(e2)
	if d1 != d2 || d1 != "same-value" {
		t.Error("decryption mismatch")
	}
}

func TestEncrypt_NoKey_FallsBackToRedact(t *testing.T) {
	d := NewPIIDetector() // no key
	result := d.getReplacement("secret", PIITypeEmail, RedactionModeEncrypt)
	if result != "***REDACTED***" {
		t.Errorf("expected REDACTED fallback, got %s", result)
	}
}

func TestEncrypt_NoKey_ReturnsError(t *testing.T) {
	d := NewPIIDetector()
	_, err := d.encryptValue("secret")
	if err == nil {
		t.Fatal("expected error when no key configured")
	}
}

func TestDecrypt_TamperedCiphertext_Rejected(t *testing.T) {
	d := NewPIIDetectorWithKeys("", testEncryptKey)
	encrypted, _ := d.encryptValue("secret")
	// Tamper with the hex string
	tampered := encrypted[:len(encrypted)-1]
	if tampered[len(tampered)-1] == '0' {
		tampered = tampered[:len(tampered)-1] + "1"
	} else {
		tampered = tampered[:len(tampered)-1] + "0"
	}
	_, err := d.DecryptValue(tampered)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecrypt_InvalidPrefix_Rejected(t *testing.T) {
	d := NewPIIDetectorWithKeys("", testEncryptKey)
	_, err := d.DecryptValue("notencrypted")
	if err == nil {
		t.Fatal("expected error for non-enc value")
	}
}

func TestDecrypt_NoKey_ReturnsError(t *testing.T) {
	d := NewPIIDetector()
	_, err := d.DecryptValue("enc:abcd")
	if err == nil {
		t.Fatal("expected error when no key configured")
	}
}

// --- Integration: RedactInText with real hash/encrypt ---

func TestRedactInText_Hash_RealHash(t *testing.T) {
	d := NewPIIDetector()
	d.SetHashSalt("salt")
	text := "Email: test@example.com"
	result := d.RedactInText(text, RedactionModeHash)
	// Should contain hash: prefix, not ***HASH***
	if strings.Contains(result, "***HASH***") {
		t.Error("expected real hash, not fixed placeholder")
	}
	if !strings.Contains(result, "hash:") {
		t.Error("expected hash: prefix in result")
	}
	// Original email should not be present
	if strings.Contains(result, "test@example.com") {
		t.Error("original PII should not be present in output")
	}
}

func TestRedactInText_Encrypt_RealEncrypt(t *testing.T) {
	d := NewPIIDetectorWithKeys("", testEncryptKey)
	text := "Email: test@example.com"
	result := d.RedactInText(text, RedactionModeEncrypt)
	if strings.Contains(result, "***ENCRYPTED***") {
		t.Error("expected real encryption, not fixed placeholder")
	}
	if !strings.Contains(result, "enc:") {
		t.Error("expected enc: prefix in result")
	}
	if strings.Contains(result, "test@example.com") {
		t.Error("original PII should not be present in output")
	}
	// Decrypt should recover original value
	for _, word := range strings.Fields(result) {
		if strings.HasPrefix(word, "enc:") {
			decrypted, err := d.DecryptValue(word)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}
			if decrypted != "test@example.com" {
				t.Errorf("expected test@example.com, got %s", decrypted)
			}
		}
	}
}

func TestRedactInText_Encrypt_NoKey_FallsBack(t *testing.T) {
	d := NewPIIDetector() // no key
	text := "Email: test@example.com"
	result := d.RedactInText(text, RedactionModeEncrypt)
	// Should fall back to redaction
	if !strings.Contains(result, "***REDACTED***") {
		t.Error("expected REDACTED fallback when no key")
	}
}

func TestRedactInText_MultipleMatches_AllTransformed(t *testing.T) {
	d := NewPIIDetectorWithKeys("salt", testEncryptKey)
	text := "Contact a@x.com and b@x.com"
	// Hash mode: all emails should be hashed
	hashed := d.RedactInText(text, RedactionModeHash)
	if strings.Contains(hashed, "a@x.com") || strings.Contains(hashed, "b@x.com") {
		t.Error("emails should be replaced")
	}
	// Both should have hash: prefix
	hashCount := strings.Count(hashed, "hash:")
	if hashCount < 2 {
		t.Errorf("expected at least 2 hashes, got %d", hashCount)
	}
}
