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

package domain

import (
	"context"
	"testing"
)

func TestToolLedgerInMemory_RecordAndLookup(t *testing.T) {
	ctx := context.Background()
	ledger := NewToolLedgerInMemory()

	// 记录一次工具调用
	invocation := NewToolInvocation("job-1", "get_user", map[string]interface{}{"id": 123})
	err := ledger.Record(ctx, invocation)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// 查询
	result, err := ledger.Lookup(ctx, invocation.IdempotencyKey)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected to find invocation")
	}
	if result.ToolName != "get_user" {
		t.Errorf("expected tool name get_user, got %s", result.ToolName)
	}
}

func TestToolLedgerInMemory_Verify_Idempotency(t *testing.T) {
	ctx := context.Background()
	ledger := NewToolLedgerInMemory()

	// 记录并完成一次工具调用
	invocation := NewToolInvocation("job-1", "create_order", map[string]interface{}{"item": "book"})
	_ = ledger.Record(ctx, invocation)
	_ = ledger.Complete(ctx, invocation.IdempotencyKey, map[string]interface{}{"order_id": "123"}, nil)

	// 验证幂等性
	found, result := ledger.Verify(ctx, invocation.IdempotencyKey)
	if !found {
		t.Error("expected to find invocation")
	}
	if result == nil || result.Status != ToolInvocationStatusCompleted {
		t.Errorf("expected completed status, got %v", result.Status)
	}
}

func TestToolLedgerInMemory_Complete(t *testing.T) {
	ctx := context.Background()
	ledger := NewToolLedgerInMemory()

	// 记录工具调用
	invocation := NewToolInvocation("job-1", "send_email", map[string]interface{}{"to": "test@example.com"})
	_ = ledger.Record(ctx, invocation)

	// 完成调用
	err := ledger.Complete(ctx, invocation.IdempotencyKey, map[string]interface{}{"message_id": "msg-123"}, nil)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// 查询结果
	result, err := ledger.Lookup(ctx, invocation.IdempotencyKey)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if result.Status != ToolInvocationStatusCompleted {
		t.Errorf("expected completed status, got %v", result.Status)
	}
	if result.Result == nil {
		t.Error("expected result to be set")
	}
}

func TestToolLedgerInMemory_CompleteWithError(t *testing.T) {
	ctx := context.Background()
	ledger := NewToolLedgerInMemory()

	// 记录工具调用
	invocation := NewToolInvocation("job-1", "api_call", map[string]interface{}{"url": "http://example.com"})
	_ = ledger.Record(ctx, invocation)

	// 完成调用（失败）
	err := ledger.Complete(ctx, invocation.IdempotencyKey, nil, &testError{Message: "timeout"})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// 查询结果
	result, err := ledger.Lookup(ctx, invocation.IdempotencyKey)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if result.Status != ToolInvocationStatusFailed {
		t.Errorf("expected failed status, got %v", result.Status)
	}
	if result.Error != "timeout" {
		t.Errorf("expected error message 'timeout', got %s", result.Error)
	}
}

func TestToolLedgerInMemory_ListByJob(t *testing.T) {
	ctx := context.Background()
	ledger := NewToolLedgerInMemory()

	// 记录多个工具调用
	_ = ledger.Record(ctx, NewToolInvocation("job-1", "tool_a", map[string]interface{}{"a": 1}))
	_ = ledger.Record(ctx, NewToolInvocation("job-1", "tool_b", map[string]interface{}{"b": 2}))
	_ = ledger.Record(ctx, NewToolInvocation("job-2", "tool_c", map[string]interface{}{"c": 3}))

	// 查询 job-1 的工具调用
	list, err := ledger.ListByJob(ctx, "job-1")
	if err != nil {
		t.Fatalf("ListByJob failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 invocations for job-1, got %d", len(list))
	}

	// 查询 job-2 的工具调用
	list2, _ := ledger.ListByJob(ctx, "job-2")
	if len(list2) != 1 {
		t.Errorf("expected 1 invocation for job-2, got %d", len(list2))
	}
}

func TestToolLedgerInMemory_Verify_NotFound(t *testing.T) {
	ctx := context.Background()
	ledger := NewToolLedgerInMemory()

	// 验证不存在的幂等键
	found, result := ledger.Verify(ctx, "nonexistent-key")
	if found {
		t.Error("expected not to find invocation")
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestNewToolInvocation(t *testing.T) {
	invocation := NewToolInvocation("job-1", "test_tool", map[string]interface{}{"key": "value"})
	if invocation.JobID != "job-1" {
		t.Errorf("expected job ID job-1, got %s", invocation.JobID)
	}
	if invocation.ToolName != "test_tool" {
		t.Errorf("expected tool name test_tool, got %s", invocation.ToolName)
	}
	if invocation.IdempotencyKey == "" {
		t.Error("expected idempotency key to be set")
	}
	if invocation.Status != ToolInvocationStatusPending {
		t.Errorf("expected pending status, got %v", invocation.Status)
	}
}

// testError 用于测试的错误类型
type testError struct {
	Message string
}

func (e *testError) Error() string {
	return e.Message
}
