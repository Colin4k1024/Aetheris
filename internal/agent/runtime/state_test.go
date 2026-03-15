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

package runtime

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestToolCallRecord(t *testing.T) {
	now := time.Now()
	record := ToolCallRecord{
		ToolName: "tool1",
		Input:    `{"key":"value"}`,
		Output:   `{"result":"ok"}`,
		At:       now,
	}
	if record.ToolName != "tool1" {
		t.Errorf("expected tool1, got %s", record.ToolName)
	}
	if record.Input != `{"key":"value"}` {
		t.Errorf("expected input, got %s", record.Input)
	}
}

func TestAgentState(t *testing.T) {
	now := time.Now()
	state := AgentState{
		AgentID:        "agent-1",
		SessionID:      "session-1",
		LastCheckpoint: "checkpoint-1",
		UpdatedAt:      now,
	}
	if state.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", state.AgentID)
	}
	if state.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", state.SessionID)
	}
}

func TestAgentState_Variables(t *testing.T) {
	state := AgentState{
		AgentID:   "agent-1",
		Variables: map[string]any{"key": "value"},
	}
	if state.Variables == nil {
		t.Error("expected non-nil variables")
	}
	if state.Variables["key"] != "value" {
		t.Errorf("expected value, got %v", state.Variables["key"])
	}
}

func TestSessionToAgentState(t *testing.T) {
	s := NewSession("session-1", "agent-1")
	s.LastCheckpoint = "checkpoint-1"
	s.SetVariable("key", "value")
	s.AddMessage("user", "hello")

	state := SessionToAgentState(s)
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", state.AgentID)
	}
	if state.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", state.SessionID)
	}
	if state.LastCheckpoint != "checkpoint-1" {
		t.Errorf("expected checkpoint-1, got %s", state.LastCheckpoint)
	}
}

func TestSessionToAgentState_Nil(t *testing.T) {
	state := SessionToAgentState(nil)
	if state != nil {
		t.Error("expected nil for nil session")
	}
}

// MockAgentStateStore is a mock implementation for testing
type MockAgentStateStore struct {
	states map[string]*AgentState
	mu     sync.RWMutex
}

func NewMockAgentStateStore() *MockAgentStateStore {
	return &MockAgentStateStore{
		states: make(map[string]*AgentState),
	}
}

func (m *MockAgentStateStore) SaveAgentState(ctx context.Context, agentID, sessionID string, state *AgentState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := agentID + "/" + sessionID
	m.states[key] = state
	return nil
}

func (m *MockAgentStateStore) LoadAgentState(ctx context.Context, agentID, sessionID string) (*AgentState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := agentID + "/" + sessionID
	state, ok := m.states[key]
	if !ok {
		return nil, nil
	}
	return state, nil
}
