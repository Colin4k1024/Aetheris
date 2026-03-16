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
	"context"
	"testing"
)

// TestSessionManagerIntegration_CreateAndGet 测试 Session Manager 的基本创建和获取
func TestSessionManagerIntegration_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	// 创建新 Session
	s1, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s1.ID == "" {
		t.Error("session ID should not be empty")
	}

	// 获取已存在的 Session
	s1Fetched, err := manager.Get(ctx, s1.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s1Fetched.ID != s1.ID {
		t.Errorf("fetched ID: got %s want %s", s1Fetched.ID, s1.ID)
	}
}

// TestSessionManagerIntegration_GetOrCreate 测试 GetOrCreate 逻辑
func TestSessionManagerIntegration_GetOrCreate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	// 使用已知 ID 获取不存在的 Session，应该自动创建
	existingID := "session-known-id"
	s1, err := manager.GetOrCreate(ctx, existingID)
	if err != nil {
		t.Fatalf("GetOrCreate new: %v", err)
	}
	if s1.ID != existingID {
		t.Errorf("created session ID: got %s want %s", s1.ID, existingID)
	}

	// 再次获取同一 ID，应该返回已存在的
	s2, err := manager.GetOrCreate(ctx, existingID)
	if err != nil {
		t.Fatalf("GetOrCreate existing: %v", err)
	}
	if s2.ID != existingID {
		t.Errorf("existing session ID: got %s want %s", s2.ID, existingID)
	}

	// 两个 Session 应该是同一个
	if s1 != s2 {
		t.Error("GetOrCreate should return same instance")
	}
}

// TestSessionManagerIntegration_EmptyIDCreate 测试空 ID 自动创建
func TestSessionManagerIntegration_EmptyIDCreate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	// 空 ID 应该创建新 Session
	s, err := manager.GetOrCreate(ctx, "")
	if err != nil {
		t.Fatalf("GetOrCreate empty: %v", err)
	}
	if s.ID == "" {
		t.Error("empty ID should generate new session ID")
	}
}

// TestSessionManagerIntegration_SaveAndPersist 测试 Save 持久化
func TestSessionManagerIntegration_SaveAndPersist(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	// 创建 Session 并添加消息
	s, _ := manager.Create(ctx)
	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "hi there")

	// Save
	err := manager.Save(ctx, s)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 获取并验证
	fetched, _ := manager.Get(ctx, s.ID)
	if fetched == nil {
		t.Fatal("session should be persisted")
	}
	msgs := fetched.CopyMessages()
	if len(msgs) != 2 {
		t.Errorf("messages count: got %d want 2", len(msgs))
	}
}

// TestSessionManagerIntegration_MultipleSessions 测试多个 Session 隔离
func TestSessionManagerIntegration_MultipleSessions(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	// 创建多个 Session
	s1, _ := manager.Create(ctx)
	s2, _ := manager.Create(ctx)

	s1.AddMessage("user", "message for s1")
	s2.AddMessage("user", "message for s2")

	// 验证隔离
	s1Fetched, _ := manager.Get(ctx, s1.ID)
	s2Fetched, _ := manager.Get(ctx, s2.ID)

	msgs1 := s1Fetched.CopyMessages()
	msgs2 := s2Fetched.CopyMessages()

	if len(msgs1) != 1 || msgs1[0].Content != "message for s1" {
		t.Errorf("s1 messages: %+v", msgs1)
	}
	if len(msgs2) != 1 || msgs2[0].Content != "message for s2" {
		t.Errorf("s2 messages: %+v", msgs2)
	}
}

// TestSessionManagerIntegration_ConcurrentAccess 测试并发访问
func TestSessionManagerIntegration_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	s, _ := manager.Create(ctx)

	// 并发添加消息
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			session, err := manager.Get(ctx, s.ID)
			if err != nil {
				errs <- err
				return
			}
			session.AddMessage("user", "message-"+string(rune('0'+idx)))
			err = manager.Save(ctx, session)
			errs <- err
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < 10; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}
	close(errs)

	// 最终应该有 10 条消息
	final, _ := manager.Get(ctx, s.ID)
	msgs := final.CopyMessages()
	if len(msgs) != 10 {
		t.Errorf("final messages: got %d want 10", len(msgs))
	}
}

// TestSessionManagerIntegration_TenantIsolation 测试租户隔离
func TestSessionManagerIntegration_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	// 创建不同租户的 Session
	s1, _ := manager.Create(ctx)
	s1.SetTenantID("tenant-a")
	s1.AddMessage("user", "secret data")

	s2, _ := manager.Create(ctx)
	s2.SetTenantID("tenant-b")
	s2.AddMessage("user", "other data")

	// Save
	_ = manager.Save(ctx, s1)
	_ = manager.Save(ctx, s2)

	// 验证租户 ID 隔离
	s1Fetched, _ := manager.Get(ctx, s1.ID)
	s2Fetched, _ := manager.Get(ctx, s2.ID)

	if s1Fetched.TenantID() != "tenant-a" {
		t.Errorf("s1 tenant: got %s want tenant-a", s1Fetched.TenantID())
	}
	if s2Fetched.TenantID() != "tenant-b" {
		t.Errorf("s2 tenant: got %s want tenant-b", s2Fetched.TenantID())
	}
}

// TestSessionManagerIntegration_AgentIsolation 测试 Agent 隔离
func TestSessionManagerIntegration_AgentIsolation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	s, _ := manager.Create(ctx)
	s.SetAgentID("agent-001")

	// 验证 Agent ID
	if s.AgentID() != "agent-001" {
		t.Errorf("agent ID: got %s want agent-001", s.AgentID())
	}

	// 修改 Agent ID
	s.SetAgentID("agent-002")
	if s.AgentID() != "agent-002" {
		t.Errorf("agent ID after set: got %s want agent-002", s.AgentID())
	}
}

// TestSessionIntegration_MemoryBlocks 测试 Memory Blocks 的完整流程
func TestSessionIntegration_MemoryBlocks(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	s, _ := manager.Create(ctx)

	// 添加多个 Memory Block
	blocks := []*MemoryBlock{
		{ID: "block-1", Type: "working", Key: "k1", Value: []byte("v1")},
		{ID: "block-2", Type: "working", Key: "k2", Value: []byte("v2")},
		{ID: "block-3", Type: "history", Key: "history", Value: []byte("h1")},
	}

	for _, b := range blocks {
		s.AddMemoryBlock(b)
	}

	// 验证添加
	if len(s.ListMemoryBlocksByType("working")) != 2 {
		t.Error("working blocks count mismatch")
	}
	if len(s.ListMemoryBlocksByType("history")) != 1 {
		t.Error("history blocks count mismatch")
	}

	// 删除一个 block
	s.RemoveMemoryBlock("block-2")

	// 验证删除
	if s.GetMemoryBlock("block-2") != nil {
		t.Error("block-2 should be deleted")
	}
	if s.GetMemoryBlock("block-1") == nil {
		t.Error("block-1 should still exist")
	}

	// 验证 CopyMemoryBlocks
	copy := s.CopyMemoryBlocks()
	if len(copy) != 2 {
		t.Errorf("copied blocks: got %d want 2", len(copy))
	}
}

// TestSessionIntegration_ToolCalls 测试 Tool Calls 的添加和复制
func TestSessionIntegration_ToolCalls(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	s, _ := manager.Create(ctx)

	// 添加 Tool Observations
	s.AddObservation("search", map[string]any{"query": "golang"}, "results", "")
	s.AddObservation("calculator", map[string]any{"expr": "2+2"}, "4", "")

	// 复制 Tool Calls
	calls := s.CopyToolCalls()
	if len(calls) != 2 {
		t.Errorf("tool calls count: got %d want 2", len(calls))
	}

	// 验证内容
	if calls[0].Tool != "search" {
		t.Errorf("first tool: got %s want search", calls[0].Tool)
	}
	if calls[1].Tool != "calculator" {
		t.Errorf("second tool: got %s want calculator", calls[1].Tool)
	}
}

// TestSessionIntegration_WorkingState 测试 WorkingState 的完整操作
func TestSessionIntegration_WorkingState(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	manager := NewManager(store)

	s, _ := manager.Create(ctx)

	// 设置多个值
	s.WorkingStateSet("key1", "value1")
	s.WorkingStateSet("key2", "value2")
	s.WorkingStateSet("counter", 100)

	// 获取值
	v1, ok := s.WorkingStateGet("key1")
	if !ok || v1 != "value1" {
		t.Errorf("key1: v=%v ok=%v", v1, ok)
	}

	v2, ok := s.WorkingStateGet("key2")
	if !ok || v2 != "value2" {
		t.Errorf("key2: v=%v ok=%v", v2, ok)
	}

	vCounter, ok := s.WorkingStateGet("counter")
	if !ok || vCounter != 100 {
		t.Errorf("counter: v=%v ok=%v", vCounter, ok)
	}

	// 不存在的 key
	_, ok = s.WorkingStateGet("nonexistent")
	if ok {
		t.Error("nonexistent key should return ok=false")
	}
}
