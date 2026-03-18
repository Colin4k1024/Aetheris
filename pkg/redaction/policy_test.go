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

func TestLoadPolicyFromConfig_Disabled(t *testing.T) {
	config := PolicyConfig{
		Enable: false,
	}
	policy := LoadPolicyFromConfig(config)
	if policy != nil {
		t.Error("expected nil policy when disabled")
	}
}

func TestLoadPolicyFromConfig_Empty(t *testing.T) {
	config := PolicyConfig{
		Enable:   true,
		Policies: []EventPolicyConfig{},
	}
	policy := LoadPolicyFromConfig(config)
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	if len(policy.EventRules) != 0 {
		t.Errorf("expected 0 event rules, got %d", len(policy.EventRules))
	}
}

func TestLoadPolicyFromConfig_SingleEvent(t *testing.T) {
	config := PolicyConfig{
		Enable: true,
		Policies: []EventPolicyConfig{
			{
				EventType: "job_created",
				Fields: []FieldMaskConfig{
					{Path: "payload.email", Mode: RedactionModeRedact},
					{Path: "payload.phone", Mode: RedactionModeHash, Salt: "salt123"},
				},
			},
		},
	}
	policy := LoadPolicyFromConfig(config)
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}

	rules, ok := policy.EventRules["job_created"]
	if !ok {
		t.Error("expected job_created rules")
	}
	if len(rules) != 2 {
		t.Errorf("expected 2 field masks, got %d", len(rules))
	}

	// Check first rule
	if rules[0].FieldPath != "payload.email" {
		t.Errorf("expected payload.email, got %s", rules[0].FieldPath)
	}
	if rules[0].Mode != RedactionModeRedact {
		t.Errorf("expected redact mode, got %s", rules[0].Mode)
	}

	// Check second rule
	if rules[1].FieldPath != "payload.phone" {
		t.Errorf("expected payload.phone, got %s", rules[1].FieldPath)
	}
	if rules[1].Mode != RedactionModeHash {
		t.Errorf("expected hash mode, got %s", rules[1].Mode)
	}
	if rules[1].Salt != "salt123" {
		t.Errorf("expected salt123, got %s", rules[1].Salt)
	}
}

func TestLoadPolicyFromConfig_MultipleEvents(t *testing.T) {
	config := PolicyConfig{
		Enable: true,
		Policies: []EventPolicyConfig{
			{
				EventType: "job_created",
				Fields: []FieldMaskConfig{
					{Path: "payload.email", Mode: RedactionModeRedact},
				},
			},
			{
				EventType: "tool_called",
				Fields: []FieldMaskConfig{
					{Path: "result.api_key", Mode: RedactionModeEncrypt},
				},
			},
		},
	}
	policy := LoadPolicyFromConfig(config)
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}

	if len(policy.EventRules) != 2 {
		t.Errorf("expected 2 event rules, got %d", len(policy.EventRules))
	}

	// Check job_created rules
	rules, ok := policy.EventRules["job_created"]
	if !ok {
		t.Error("expected job_created rules")
	}
	if len(rules) != 1 || rules[0].FieldPath != "payload.email" {
		t.Error("job_created rules mismatch")
	}

	// Check tool_called rules
	rules, ok = policy.EventRules["tool_called"]
	if !ok {
		t.Error("expected tool_called rules")
	}
	if len(rules) != 1 || rules[0].FieldPath != "result.api_key" || rules[0].Mode != RedactionModeEncrypt {
		t.Error("tool_called rules mismatch")
	}
}

func TestRedactionModeConstants(t *testing.T) {
	tests := []struct {
		mode     RedactionMode
		expected string
	}{
		{RedactionModeRedact, "redact"},
		{RedactionModeHash, "hash"},
		{RedactionModeEncrypt, "encrypt"},
		{RedactionModeRemove, "remove"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.mode)
		}
	}
}

func TestFieldMask(t *testing.T) {
	mask := FieldMask{
		FieldPath: "payload.ssn",
		Mode:      RedactionModeRedact,
		Salt:      "mysalt",
	}

	if mask.FieldPath != "payload.ssn" {
		t.Errorf("expected payload.ssn, got %s", mask.FieldPath)
	}
	if mask.Mode != RedactionModeRedact {
		t.Errorf("expected redact mode, got %s", mask.Mode)
	}
	if mask.Salt != "mysalt" {
		t.Errorf("expected mysalt, got %s", mask.Salt)
	}
}

func TestRedactionPolicy(t *testing.T) {
	policy := RedactionPolicy{
		EventRules: map[string][]FieldMask{
			"job_created": {
				{FieldPath: "payload.email", Mode: RedactionModeRedact},
			},
		},
		GlobalRules: []FieldMask{
			{FieldPath: "metadata.password", Mode: RedactionModeRemove},
		},
	}

	if len(policy.EventRules) != 1 {
		t.Errorf("expected 1 event rule, got %d", len(policy.EventRules))
	}
	if len(policy.GlobalRules) != 1 {
		t.Errorf("expected 1 global rule, got %d", len(policy.GlobalRules))
	}
}
