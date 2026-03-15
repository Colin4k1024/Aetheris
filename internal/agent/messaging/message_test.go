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

package messaging

import (
	"testing"
	"time"
)

func TestMessageKindConstants(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		{KindUser, "user"},
		{KindSignal, "signal"},
		{KindTimer, "timer"},
		{KindWebhook, "webhook"},
		{KindAgent, "agent"},
	}

	for _, tt := range tests {
		if tt.kind != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.kind)
		}
	}
}

func TestMessage(t *testing.T) {
	now := time.Now()
	msg := Message{
		ID:          "msg-1",
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Channel:     "test",
		Kind:        KindUser,
		Payload:     map[string]any{"key": "value"},
		CreatedAt:   now,
	}

	if msg.ID != "msg-1" {
		t.Errorf("expected msg-1, got %s", msg.ID)
	}
	if msg.FromAgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", msg.FromAgentID)
	}
	if msg.ToAgentID != "agent-2" {
		t.Errorf("expected agent-2, got %s", msg.ToAgentID)
	}
	if msg.Kind != KindUser {
		t.Errorf("expected user, got %s", msg.Kind)
	}
}

func TestSendOptions(t *testing.T) {
	scheduledAt := time.Now().Add(time.Hour)
	expiresAt := time.Now().Add(time.Hour * 24)

	opts := SendOptions{
		Channel:        "test-channel",
		Kind:           KindSignal,
		CausationID:    "cause-1",
		ScheduledAt:    &scheduledAt,
		ExpiresAt:      &expiresAt,
		IdempotencyKey: "idem-1",
	}

	if opts.Channel != "test-channel" {
		t.Errorf("expected test-channel, got %s", opts.Channel)
	}
	if opts.Kind != KindSignal {
		t.Errorf("expected signal, got %s", opts.Kind)
	}
	if opts.CausationID != "cause-1" {
		t.Errorf("expected cause-1, got %s", opts.CausationID)
	}
}
