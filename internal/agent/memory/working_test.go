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
