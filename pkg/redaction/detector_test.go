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
	"testing"
)

func TestNewPIIDetector(t *testing.T) {
	d := NewPIIDetector()
	if d == nil {
		t.Fatal("NewPIIDetector should not return nil")
	}
}

func TestPIIDetector_Detect_Email(t *testing.T) {
	d := NewPIIDetector()
	detections := d.Detect("Contact me at test@example.com")
	if len(detections) == 0 {
		t.Error("expected to detect email")
	}
}

func TestPIIDetector_Detect_Phone(t *testing.T) {
	d := NewPIIDetector()
	detections := d.Detect("Call me at 123-456-7890")
	if len(detections) == 0 {
		t.Error("expected to detect phone")
	}
}

func TestPIIDetector_Detect_NoPII(t *testing.T) {
	d := NewPIIDetector()
	detections := d.Detect("This is a normal text")
	if len(detections) != 0 {
		t.Error("expected no detections")
	}
}

func TestPIIDetector_DetectInMap(t *testing.T) {
	d := NewPIIDetector()
	m := map[string]interface{}{
		"email": "test@example.com",
		"name":  "John",
	}
	result := d.DetectInMap(m)
	if len(result) == 0 {
		t.Error("expected to detect PII in map")
	}
}

func TestGetBuiltInFields(t *testing.T) {
	fields := GetBuiltInFields()
	if len(fields) == 0 {
		t.Error("expected built-in fields")
	}
}

func TestNewAutoDetectPolicy(t *testing.T) {
	policy := NewAutoDetectPolicy(RedactionModeRedact)
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
}
