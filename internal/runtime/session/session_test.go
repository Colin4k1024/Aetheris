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

package session

import (
	"testing"

	"rag-platform/internal/model/llm"
)

func TestNew(t *testing.T) {
	s := New("sid1")
	if s == nil || s.ID != "sid1" {
		t.Errorf("New: %+v", s)
	}
	if s.WorkingState == nil || s.Metadata == nil {
		t.Error("WorkingState and Metadata should be initialized")
	}
	s2 := New("")
	if s2.ID == "" {
		t.Error("empty id should generate id")
	}
}

func TestSession_AddMessage_CopyMessages(t *testing.T) {
	s := New("s1")
	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "hi")
	msgs := s.CopyMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("first message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi" {
		t.Errorf("second message: %+v", msgs[1])
	}
}

func TestSession_AddObservation_CopyToolCalls(t *testing.T) {
	s := New("s1")
	s.AddObservation("tool1", map[string]any{"q": "x"}, "out", "")
	calls := s.CopyToolCalls()
	if len(calls) != 1 || calls[0].Tool != "tool1" || calls[0].Output != "out" {
		t.Errorf("CopyToolCalls: %+v", calls)
	}
}

func TestSession_WorkingStateGet_WorkingStateSet(t *testing.T) {
	s := New("s1")
	s.WorkingStateSet("k1", "v1")
	v, ok := s.WorkingStateGet("k1")
	if !ok || v != "v1" {
		t.Errorf("WorkingStateGet: v=%v ok=%v", v, ok)
	}
	_, ok = s.WorkingStateGet("missing")
	if ok {
		t.Error("WorkingStateGet missing should be false")
	}
}

func TestNewManager(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store)
	if m == nil {
		t.Fatal("NewManager should not return nil")
	}
}

func TestMemoryStore_PutAndGet(t *testing.T) {
	store := NewMemoryStore()
	s := New("test-id")

	err := store.Put(nil, s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	got, err := store.Get(nil, "test-id")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got == nil || got.ID != "test-id" {
		t.Error("expected to get the session back")
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store := NewMemoryStore()
	got, err := store.Get(nil, "nonexistent")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent key")
	}
}

func TestMessage_FromLLM(t *testing.T) {
	llmMsg := llm.Message{
		Role:    "user",
		Content: "hello",
	}
	msg := FromLLM(llmMsg)
	if msg.Role != "user" || msg.Content != "hello" {
		t.Errorf("expected user/hello, got %s/%s", msg.Role, msg.Content)
	}
}

func TestMessage_MessagesToLLM(t *testing.T) {
	msgs := []*Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	llmMsgs := MessagesToLLM(msgs)
	if len(llmMsgs) != 2 {
		t.Errorf("expected 2, got %d", len(llmMsgs))
	}
}

func TestSession_MemoryBlocks(t *testing.T) {
	s := New("s1")

	// Add memory block
	block := &MemoryBlock{
		ID:    "block1",
		Type:  "working",
		Key:   "test-key",
		Value: []byte("test-value"),
	}
	s.AddMemoryBlock(block)

	// Get memory block
	got := s.GetMemoryBlock("block1")
	if got == nil || got.ID != "block1" {
		t.Errorf("GetMemoryBlock: %+v", got)
	}

	// Get non-existent
	got = s.GetMemoryBlock("missing")
	if got != nil {
		t.Error("expected nil for missing block")
	}

	// List by type
	list := s.ListMemoryBlocksByType("working")
	if len(list) != 1 || list[0].ID != "block1" {
		t.Errorf("ListMemoryBlocksByType: %d", len(list))
	}

	// List by non-existent type
	list = s.ListMemoryBlocksByType("missing")
	if len(list) != 0 {
		t.Errorf("expected 0 for missing type, got %d", len(list))
	}

	// Copy memory blocks
	copy := s.CopyMemoryBlocks()
	if len(copy) != 1 || copy[0].ID != "block1" {
		t.Errorf("CopyMemoryBlocks: %d", len(copy))
	}

	// Remove memory block
	removed := s.RemoveMemoryBlock("block1")
	if !removed {
		t.Error("expected true for remove")
	}

	// Verify removed
	got = s.GetMemoryBlock("block1")
	if got != nil {
		t.Error("expected nil after remove")
	}
}

func TestSession_TenantID(t *testing.T) {
	s := New("s1")
	if s.TenantID() != "" {
		t.Error("expected empty tenant ID")
	}

	s.SetTenantID("tenant-1")
	if s.TenantID() != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", s.TenantID())
	}
}

func TestSession_AgentID(t *testing.T) {
	s := New("s1")
	if s.AgentID() != "" {
		t.Error("expected empty agent ID")
	}

	s.SetAgentID("agent-1")
	if s.AgentID() != "agent-1" {
		t.Errorf("expected agent-1, got %s", s.AgentID())
	}
}
