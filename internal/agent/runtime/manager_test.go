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
	"testing"
)

func TestManager_Create_Get_List_Delete(t *testing.T) {
	ctx := context.Background()
	m := NewManager()
	agent, err := m.Create(ctx, "my-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if agent == nil || agent.Name != "my-agent" {
		t.Errorf("Create: %+v", agent)
	}
	if agent.Session == nil {
		t.Fatal("Create with nil session should set Session")
	}
	if agent.Session.AgentID != agent.ID {
		t.Errorf("Session.AgentID should be set to agent ID")
	}
	got, err := m.Get(ctx, agent.ID)
	if err != nil || got != agent {
		t.Errorf("Get: err=%v got=%v", err, got)
	}
	list, err := m.List(ctx)
	if err != nil || len(list) != 1 {
		t.Errorf("List: err=%v len=%d", err, len(list))
	}
	if err := m.Delete(ctx, agent.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got2, _ := m.Get(ctx, agent.ID)
	if got2 != nil {
		t.Error("Get after Delete should return nil")
	}
}

func TestManager_Create_WithSession(t *testing.T) {
	ctx := context.Background()
	m := NewManager()
	sess := NewSession("s1", "")
	agent, err := m.Create(ctx, "a", sess, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if agent.Session != sess {
		t.Error("Create with session should use that session")
	}
	if sess.AgentID != agent.ID {
		t.Errorf("session AgentID should be set to %q", agent.ID)
	}
}

// TestAgent_TakeRelease verifies RTN-05: atomic Take/Release for Agent concurrency
func TestAgent_TakeRelease(t *testing.T) {
	ctx := context.Background()
	m := NewManager()
	agent, err := m.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Initial status should be Idle
	if got := agent.GetStatus(); got != StatusIdle {
		t.Errorf("initial status: got %v, want StatusIdle", got)
	}

	// Take should succeed from Idle
	if !agent.Take() {
		t.Error("Take from Idle should succeed")
	}
	if got := agent.GetStatus(); got != StatusRunning {
		t.Errorf("after Take: got %v, want StatusRunning", got)
	}

	// Second Take should fail (already Running)
	if agent.Take() {
		t.Error("Take from Running should fail")
	}

	// Release should put back to Idle
	agent.Release()
	if got := agent.GetStatus(); got != StatusIdle {
		t.Errorf("after Release: got %v, want StatusIdle", got)
	}

	// Take again after Release should succeed
	if !agent.Take() {
		t.Error("Take after Release should succeed")
	}
}

// TestAgent_TakeFromSuspended verifies Take works from Suspended state (RTN-05)
func TestAgent_TakeFromSuspended(t *testing.T) {
	ctx := context.Background()
	m := NewManager()
	agent, err := m.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Force set to Suspended
	agent.SetStatus(StatusSuspended)
	if !agent.Take() {
		t.Error("Take from Suspended should succeed")
	}
	if got := agent.GetStatus(); got != StatusRunning {
		t.Errorf("after Take: got %v, want StatusRunning", got)
	}
	agent.Release()
}

// TestAgent_ReleaseFromWaiting verifies Release works from WaitingTool state (RTN-05)
func TestAgent_ReleaseFromWaiting(t *testing.T) {
	ctx := context.Background()
	m := NewManager()
	agent, err := m.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Take then set to WaitingTool
	agent.Take()
	agent.SetStatus(StatusWaitingTool)
	if got := agent.GetStatus(); got != StatusWaitingTool {
		t.Errorf("status: got %v, want StatusWaitingTool", got)
	}

	// Release should work from WaitingTool
	agent.Release()
	if got := agent.GetStatus(); got != StatusIdle {
		t.Errorf("after Release: got %v, want StatusIdle", got)
	}
}

// TestScheduler_WakeAgent_TakeRelease verifies RTN-05: Scheduler.WakeAgent uses atomic Take (RTN-05)
func TestScheduler_WakeAgent_TakeRelease(t *testing.T) {
	ctx := context.Background()
	m := NewManager()
	scheduler := NewScheduler(m, nil) // nil runFunc for testing

	agent, err := m.Create(ctx, "test-agent", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First WakeAgent should succeed (calls Take internally)
	err = scheduler.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Errorf("first WakeAgent: %v", err)
	}
	if got := agent.GetStatus(); got != StatusRunning {
		t.Errorf("after first WakeAgent: got %v, want StatusRunning", got)
	}

	// Second WakeAgent should be no-op (Take fails)
	err = scheduler.WakeAgent(ctx, agent.ID)
	if err != nil {
		t.Errorf("second WakeAgent: %v", err)
	}
	// Status should still be Running
	if got := agent.GetStatus(); got != StatusRunning {
		t.Errorf("after second WakeAgent: got %v, want StatusRunning", got)
	}

	// Manually release for cleanup
	agent.Release()
}
