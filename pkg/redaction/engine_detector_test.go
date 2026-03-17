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
	"bytes"
	"encoding/json"
	"testing"
)

func TestEngine_NewEngine(t *testing.T) {
	policy := &RedactionPolicy{
		EventRules: map[string][]FieldMask{
			"test": {
				{FieldPath: "email", Mode: RedactionModeRedact},
			},
		},
	}
	engine := NewEngine(policy, nil)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.policy != policy {
		t.Error("expected policy to be set")
	}
}

func TestEngine_NewEngine_NilPolicy(t *testing.T) {
	engine := NewEngine(nil, nil)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestEngine_RedactData_EmptyData(t *testing.T) {
	engine := NewEngine(nil, nil)
	data := []byte("{}")
	result, err := engine.RedactData("test", data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Error("expected unchanged data")
	}
}

func TestEngine_RedactData_NilPolicy(t *testing.T) {
	engine := NewEngine(nil, nil)
	data := []byte(`{"email": "test@example.com"}`)
	result, err := engine.RedactData("test", data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should return original data when policy is nil
	if string(result) != string(data) {
		t.Error("expected unchanged data with nil policy")
	}
}

func TestEngine_RedactData_WithPolicy(t *testing.T) {
	policy := &RedactionPolicy{
		EventRules: map[string][]FieldMask{
			"test_event": {
				{FieldPath: "email", Mode: RedactionModeRedact},
			},
		},
	}
	engine := NewEngine(policy, nil)
	data := []byte(`{"email": "test@example.com", "name": "John"}`)
	result, err := engine.RedactData("test_event", data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resultMap["email"] == "test@example.com" {
		t.Error("expected email to be redacted")
	}
}

func TestEngine_RedactData_GlobalRules(t *testing.T) {
	policy := &RedactionPolicy{
		EventRules:  map[string][]FieldMask{},
		GlobalRules: []FieldMask{{FieldPath: "ssn", Mode: RedactionModeRedact}},
	}
	engine := NewEngine(policy, nil)
	data := []byte(`{"ssn": "123-45-6789"}`)
	result, err := engine.RedactData("any_event", data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var resultMap map[string]interface{}
	json.Unmarshal(result, &resultMap)

	if resultMap["ssn"] == "123-45-6789" {
		t.Error("expected ssn to be redacted by global rule")
	}
}

func TestEngine_HashValue(t *testing.T) {
	engine := NewEngine(nil, nil)
	hash1 := engine.hashValue("test@example.com", "")
	hash2 := engine.hashValue("test@example.com", "")
	hash3 := engine.hashValue("test@example.com", "salt")

	if hash1 == "" {
		t.Error("expected non-empty hash")
	}
	if hash1 != hash2 {
		t.Error("expected deterministic hash")
	}
	if hash1 == hash3 {
		t.Error("expected different hash with salt")
	}
}

func TestEngine_HashValue_WithSalt(t *testing.T) {
	engine := NewEngine(nil, nil)
	hash := engine.hashValue("value", "mysalt")
	if len(hash) == 0 {
		t.Error("expected non-empty hash")
	}
}

func TestEngine_EncryptValue_NoKey(t *testing.T) {
	engine := NewEngine(nil, nil)
	_, err := engine.encryptValue("test")
	if err == nil {
		t.Error("expected error when encryption key not configured")
	}
}

func TestPIITypeConstants(t *testing.T) {
	tests := []struct {
		piiType  PIIType
		expected string
	}{
		{PIITypeEmail, "email"},
		{PIITypePhone, "phone"},
		{PIITypeSSN, "ssn"},
		{PIITypeCreditCard, "credit_card"},
		{PIITypeIPAddress, "ip_address"},
		{PIITypeAddress, "address"},
		{PIITypeDateOfBirth, "date_of_birth"},
		{PIITypePassport, "passport"},
		{PIITypeDriverLicense, "driver_license"},
	}

	for _, tt := range tests {
		if string(tt.piiType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.piiType)
		}
	}
}

func TestPIIDetector_Detect_SSN(t *testing.T) {
	detector := NewPIIDetector()
	text := "My SSN is 123-45-6789"
	detections := detector.Detect(text)

	if len(detections) == 0 {
		t.Error("expected to detect SSN")
	}
	if detections[0].Type != PIITypeSSN {
		t.Errorf("expected SSN type, got %s", detections[0].Type)
	}
}

func TestPIIDetector_Detect_CreditCard(t *testing.T) {
	detector := NewPIIDetector()
	text := "Credit card: 4111111111111111"
	detections := detector.Detect(text)

	if len(detections) == 0 {
		t.Error("expected to detect credit card")
	}
}

func TestPIIDetector_Detect_IPAddress(t *testing.T) {
	detector := NewPIIDetector()
	text := "IP: 192.168.1.1"
	detections := detector.Detect(text)

	if len(detections) == 0 {
		t.Error("expected to detect IP address")
	}
}

func TestPIIDetector_RedactInText_Email(t *testing.T) {
	detector := NewPIIDetector()
	text := "Contact me at test@example.com"
	result := detector.RedactInText(text, RedactionModeRedact)

	if result == text {
		t.Error("expected text to be redacted")
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestPIIDetector_RedactInText_Hash(t *testing.T) {
	detector := NewPIIDetector()
	text := "Email: test@example.com"
	result := detector.RedactInText(text, RedactionModeHash)

	if result == text {
		t.Error("expected text to be hashed")
	}
}

func TestPIIDetector_RedactInMap(t *testing.T) {
	detector := NewPIIDetector()
	m := map[string]interface{}{
		"email": "test@example.com",
		"name":  "John",
	}
	result := detector.RedactInMap(m, RedactionModeRedact)

	if result["email"] == "test@example.com" {
		t.Error("expected email to be redacted")
	}
	if result["name"] != "John" {
		t.Error("expected name to remain unchanged")
	}
}

func TestPIIDetector_RedactInMap_Nested(t *testing.T) {
	detector := NewPIIDetector()
	m := map[string]interface{}{
		"user": map[string]interface{}{
			"email": "test@example.com",
		},
	}
	result := detector.RedactInMap(m, RedactionModeRedact)

	userMap, ok := result["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested map")
	}
	if userMap["email"] == "test@example.com" {
		t.Error("expected nested email to be redacted")
	}
}

func TestPIIDetector_RedactInMap_Slice(t *testing.T) {
	detector := NewPIIDetector()
	m := map[string]interface{}{
		"emails": []interface{}{"a@test.com", "b@test.com"},
	}
	result := detector.RedactInMap(m, RedactionModeRedact)

	emails, ok := result["emails"].([]interface{})
	if !ok {
		t.Fatal("expected slice")
	}
	if emails[0] == "a@test.com" {
		t.Error("expected email in slice to be redacted")
	}
}

func TestPIIDetector_GetReplacement(t *testing.T) {
	detector := NewPIIDetector()

	// Test redact mode
	replacement := detector.getReplacement(PIITypeEmail, RedactionModeRedact)
	if replacement == "" {
		t.Error("expected non-empty replacement")
	}

	// Test hash mode
	replacement = detector.getReplacement(PIITypeEmail, RedactionModeHash)
	if replacement == "" {
		t.Error("expected non-empty replacement")
	}
}

func TestPIIDetector_GetConfidence(t *testing.T) {
	detector := NewPIIDetector()
	confidence := detector.getConfidence(PIITypeEmail)
	if confidence <= 0 || confidence > 1 {
		t.Errorf("expected confidence between 0 and 1, got %f", confidence)
	}
}

func TestPIIDetector_GetConfidence_Unknown(t *testing.T) {
	detector := NewPIIDetector()
	confidence := detector.getConfidence("unknown")
	// Unknown types should return default confidence
	if confidence <= 0 {
		t.Errorf("expected positive confidence for unknown type, got %f", confidence)
	}
}

func TestAutoDetectPolicy_Apply(t *testing.T) {
	policy := NewAutoDetectPolicy(RedactionModeRedact)
	// Use a phone format that matches the regex
	m := map[string]interface{}{
		"email": "test@example.com",
		"phone": "(555) 123-4567",
	}
	result := policy.Apply(m)

	if result["email"] == "test@example.com" {
		t.Error("expected email to be redacted")
	}
	if result["phone"] == "(555) 123-4567" {
		t.Error("expected phone to be redacted")
	}
}

func TestAutoDetectPolicy_IsEnabled(t *testing.T) {
	policy := NewAutoDetectPolicy(RedactionModeRedact)

	if !policy.isEnabled(PIITypeEmail) {
		t.Error("expected email to be enabled")
	}
}

func TestPIIDetection(t *testing.T) {
	detection := PIIDetection{
		Type:       PIITypeEmail,
		Start:      0,
		End:        5,
		Value:      "test",
		Confidence: 0.9,
	}

	if detection.Type != PIITypeEmail {
		t.Errorf("expected email type, got %s", detection.Type)
	}
	if detection.Confidence != 0.9 {
		t.Errorf("expected 0.9, got %f", detection.Confidence)
	}
}
