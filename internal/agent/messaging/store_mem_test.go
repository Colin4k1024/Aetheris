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
	"context"
	"testing"
	"time"
)

func TestNewStoreMem(t *testing.T) {
	store := NewStoreMem()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestStoreMem_Send(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	id, err := store.Send(ctx, "agent-1", "agent-2", map[string]any{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty id")
	}
}

func TestStoreMem_Send_WithOptions(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	opts := &SendOptions{
		Kind:        KindUser,
		Channel:     "test-channel",
		CausationID: "cause-1",
	}
	id, err := store.Send(ctx, "agent-1", "agent-2", map[string]any{"key": "value"}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty id")
	}
}

func TestStoreMem_SendDelayed(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	future := time.Now().Add(time.Hour)
	id, err := store.SendDelayed(ctx, "agent-2", map[string]any{"key": "value"}, future, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty id")
	}
}

func TestStoreMem_PeekInbox(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	// Send message
	store.Send(ctx, "agent-1", "agent-2", map[string]any{"key": "value"}, nil)

	// Peek inbox
	messages, err := store.PeekInbox(ctx, "agent-2", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
}

func TestStoreMem_PeekInbox_Empty(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	messages, err := store.PeekInbox(ctx, "agent-2", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestStoreMem_ConsumeInbox(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	// Send message
	store.Send(ctx, "agent-1", "agent-2", map[string]any{"key": "value"}, nil)

	// Consume inbox
	messages, err := store.ConsumeInbox(ctx, "agent-2", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
}

func TestStoreMem_MarkConsumed(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	// Send and consume message
	id, _ := store.Send(ctx, "agent-1", "agent-2", map[string]any{"key": "value"}, nil)
	store.ConsumeInbox(ctx, "agent-2", 10)

	// Mark consumed
	err := store.MarkConsumed(ctx, id, "job-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Peek again should return empty
	messages, _ := store.PeekInbox(ctx, "agent-2", 10)
	if len(messages) != 0 {
		t.Errorf("expected 0 messages after consume, got %d", len(messages))
	}
}

func TestStoreMem_ListAgentIDsWithUnconsumedMessages(t *testing.T) {
	store := NewStoreMem()
	ctx := context.Background()

	// Send messages to different agents
	store.Send(ctx, "agent-1", "agent-2", map[string]any{"key": "value1"}, nil)
	store.Send(ctx, "agent-1", "agent-3", map[string]any{"key": "value2"}, nil)

	// List agents with unconsumed messages
	agents, err := store.ListAgentIDsWithUnconsumedMessages(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}
