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

package memory

import (
	"context"
	"testing"
	"time"

	"rag-platform/internal/agent/runtime"
)

func TestNewWorkingSession(t *testing.T) {
	s := runtime.NewSession("test-id", "agent-1")
	ws := NewWorkingSession(s)
	if ws == nil {
		t.Fatal("expected non-nil WorkingSession")
	}
}

func TestWorkingSession_SetSession(t *testing.T) {
	s1 := runtime.NewSession("id1", "agent-1")
	s2 := runtime.NewSession("id2", "agent-1")
	ws := NewWorkingSession(s1)

	ws.SetSession(s2)
}

func TestWorkingSession_Recall_NilSession(t *testing.T) {
	ws := &WorkingSession{}
	ctx := context.Background()
	items, err := ws.Recall(ctx, "query")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if items != nil {
		t.Error("expected nil items for nil session")
	}
}

func TestWorkingSession_Recall_WithSession(t *testing.T) {
	s := runtime.NewSession("test", "agent-1")
	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "hi")
	ws := NewWorkingSession(s)

	ctx := context.Background()
	items, err := ws.Recall(ctx, "query")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestWorkingSession_Store(t *testing.T) {
	s := runtime.NewSession("test", "agent-1")
	ws := NewWorkingSession(s)

	ctx := context.Background()
	item := MemoryItem{
		Type:    "working",
		Content: "test content",
		At:      time.Now(),
	}
	err := ws.Store(ctx, item)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewWorking(t *testing.T) {
	w := NewWorking()
	if w == nil {
		t.Fatal("expected non-nil Working")
	}
}

func TestNewLongTermMemoryStoreMem(t *testing.T) {
	store := NewLongTermMemoryStoreMem()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewEpisodicMemoryStoreMem(t *testing.T) {
	store := NewEpisodicMemoryStoreMem()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestLongTermMem_SetAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewLongTermMemoryStoreMem()

	// Set a value
	err := store.Set(ctx, "agent-1", "namespace-1", "key-1", []byte("value-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get the value
	val, err := store.Get(ctx, "agent-1", "namespace-1", "key-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "value-1" {
		t.Errorf("expected value-1, got %s", string(val))
	}
}

func TestLongTermMem_Get_NonExistent(t *testing.T) {
	ctx := context.Background()
	store := NewLongTermMemoryStoreMem()

	val, err := store.Get(ctx, "agent-1", "namespace-1", "key-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil, got %s", string(val))
	}
}

func TestLongTermMem_Get_AgentNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewLongTermMemoryStoreMem()

	// Set value for different agent
	store.Set(ctx, "agent-1", "ns", "key", []byte("value"))

	// Try to get for different agent
	val, err := store.Get(ctx, "agent-2", "ns", "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil for non-existent agent")
	}
}

func TestLongTermMem_ListByAgent(t *testing.T) {
	ctx := context.Background()
	store := NewLongTermMemoryStoreMem()

	// Set multiple values
	store.Set(ctx, "agent-1", "ns1", "key1", []byte("val1"))
	store.Set(ctx, "agent-1", "ns1", "key2", []byte("val2"))
	store.Set(ctx, "agent-1", "ns2", "key3", []byte("val3"))
	store.Set(ctx, "agent-2", "ns1", "key4", []byte("val4"))

	// List all for agent-1 with ns1
	list, err := store.ListByAgent(ctx, "agent-1", "ns1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}

	// List all for agent-1 without namespace filter
	list, err = store.ListByAgent(ctx, "agent-1", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}

	// List with limit
	list, err = store.ListByAgent(ctx, "agent-1", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items with limit, got %d", len(list))
	}
}

func TestEpisodicMem_Append(t *testing.T) {
	ctx := context.Background()
	store := NewEpisodicMemoryStoreMem()

	entry := &EpisodicEntry{
		AgentID:   "agent-1",
		SessionID: "session-1",
		Summary:   "test summary",
		Payload:   map[string]any{"key": "value"},
	}

	err := store.Append(ctx, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Entry should have ID generated
	if entry.ID == "" {
		t.Error("expected ID to be generated")
	}
}

func TestEpisodicMem_Append_NilEntry(t *testing.T) {
	ctx := context.Background()
	store := NewEpisodicMemoryStoreMem()

	err := store.Append(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEpisodicMem_ListByAgent(t *testing.T) {
	ctx := context.Background()
	store := NewEpisodicMemoryStoreMem()

	// Add entries
	store.Append(ctx, &EpisodicEntry{AgentID: "agent-1", SessionID: "session-1", Summary: "test1"})
	store.Append(ctx, &EpisodicEntry{AgentID: "agent-1", SessionID: "session-2", Summary: "test2"})
	store.Append(ctx, &EpisodicEntry{AgentID: "agent-2", SessionID: "session-3", Summary: "test3"})

	// List by agent
	list, err := store.ListByAgent(ctx, "agent-1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 entries, got %d", len(list))
	}

	// List with limit
	list, err = store.ListByAgent(ctx, "agent-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 entry with limit, got %d", len(list))
	}
}

func TestEpisodicMem_ListBySession(t *testing.T) {
	ctx := context.Background()
	store := NewEpisodicMemoryStoreMem()

	// Add entries
	store.Append(ctx, &EpisodicEntry{AgentID: "agent-1", SessionID: "session-1", Summary: "test1"})
	store.Append(ctx, &EpisodicEntry{AgentID: "agent-1", SessionID: "session-1", Summary: "test2"})
	store.Append(ctx, &EpisodicEntry{AgentID: "agent-1", SessionID: "session-2", Summary: "test3"})

	// List by session
	list, err := store.ListBySession(ctx, "agent-1", "session-1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 entries, got %d", len(list))
	}

	// Non-existent session
	list, err = store.ListBySession(ctx, "agent-1", "session-99", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 entries for non-existent session, got %d", len(list))
	}
}
