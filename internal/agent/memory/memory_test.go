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
	"errors"
	"testing"
	"time"
)

type mockMemory struct {
	recallItems []MemoryItem
	recallErr   error
	stored      []MemoryItem
}

func (m *mockMemory) Recall(ctx context.Context, query string) ([]MemoryItem, error) {
	if m.recallErr != nil {
		return nil, m.recallErr
	}
	return m.recallItems, nil
}
func (m *mockMemory) Store(ctx context.Context, item MemoryItem) error {
	m.stored = append(m.stored, item)
	return nil
}

func TestCompositeMemory_Recall_Merge(t *testing.T) {
	ctx := context.Background()
	m1 := &mockMemory{recallItems: []MemoryItem{{Type: "a", Content: "c1"}}}
	m2 := &mockMemory{recallItems: []MemoryItem{{Type: "b", Content: "c2"}}}
	c := NewCompositeMemory(m1, m2)
	items, err := c.Recall(ctx, "q")
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Content != "c1" || items[1].Content != "c2" {
		t.Errorf("items: %+v", items)
	}
}

func TestCompositeMemory_Recall_SkipErrorBackend(t *testing.T) {
	ctx := context.Background()
	m1 := &mockMemory{recallErr: errors.New("fail")}
	m2 := &mockMemory{recallItems: []MemoryItem{{Content: "ok"}}}
	c := NewCompositeMemory(m1, m2)
	items, err := c.Recall(ctx, "q")
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(items) != 1 || items[0].Content != "ok" {
		t.Errorf("expected one item from m2, got %+v", items)
	}
}

func TestCompositeMemory_Store_AllBackends(t *testing.T) {
	ctx := context.Background()
	m1 := &mockMemory{}
	m2 := &mockMemory{}
	c := NewCompositeMemory(m1, m2)
	item := MemoryItem{Type: "working", Content: "x", At: time.Now()}
	if err := c.Store(ctx, item); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if len(m1.stored) != 1 || len(m2.stored) != 1 {
		t.Errorf("Store should write to all backends: m1=%d m2=%d", len(m1.stored), len(m2.stored))
	}
}

func TestNewCompositeMemory_NoBackends(t *testing.T) {
	c := NewCompositeMemory()
	ctx := context.Background()
	items, err := c.Recall(ctx, "q")
	if err != nil || len(items) != 0 {
		t.Errorf("Recall with no backends: err=%v items=%d", err, len(items))
	}
	if err := c.Store(ctx, MemoryItem{}); err != nil {
		t.Errorf("Store with no backends: %v", err)
	}
}

// TestShortTerm_NewShortTerm tests short term memory creation
func TestShortTerm_NewShortTerm(t *testing.T) {
	st := NewShortTerm(0)
	if st.maxPer != 50 {
		t.Errorf("expected default 50, got %d", st.maxPer)
	}

	st2 := NewShortTerm(100)
	if st2.maxPer != 100 {
		t.Errorf("expected 100, got %d", st2.maxPer)
	}
}

// TestShortTerm_GetMessages tests getting messages
func TestShortTerm_GetMessages(t *testing.T) {
	st := NewShortTerm(10)

	// Empty session
	msgs := st.GetMessages("nonexistent")
	if len(msgs) != 0 {
		t.Errorf("expected empty, got %d", len(msgs))
	}

	// Add messages
	st.Append("session1", "user", "hello")
	st.Append("session1", "assistant", "hi")

	msgs = st.GetMessages("session1")
	if len(msgs) != 2 {
		t.Errorf("expected 2, got %d", len(msgs))
	}
}

// TestShortTerm_Append tests appending messages with truncation
func TestShortTerm_Append(t *testing.T) {
	st := NewShortTerm(3)

	// Add more than max
	st.Append("s1", "user", "1")
	st.Append("s1", "user", "2")
	st.Append("s1", "user", "3")
	st.Append("s1", "user", "4") // Should trigger truncation

	msgs := st.GetMessages("s1")
	if len(msgs) != 3 {
		t.Errorf("expected 3, got %d", len(msgs))
	}
	if msgs[0].Content != "2" {
		t.Errorf("expected first to be '2', got '%s'", msgs[0].Content)
	}
}

// TestShortTerm_Clear tests clearing session
func TestShortTerm_Clear(t *testing.T) {
	st := NewShortTerm(10)
	st.Append("s1", "user", "hello")

	st.Clear("s1")
	msgs := st.GetMessages("s1")
	if len(msgs) != 0 {
		t.Errorf("expected 0 after clear, got %d", len(msgs))
	}
}

// TestEpisodic_NewEpisodic tests episodic memory creation
func TestEpisodic_NewEpisodic(t *testing.T) {
	ep := NewEpisodic(0)
	if ep.limit != 1000 {
		t.Errorf("expected default 1000, got %d", ep.limit)
	}

	ep2 := NewEpisodic(50)
	if ep2.limit != 50 {
		t.Errorf("expected 50, got %d", ep2.limit)
	}
}
