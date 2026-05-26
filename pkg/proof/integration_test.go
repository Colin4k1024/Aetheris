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

package proof

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestEndToEnd_ExportAndVerify 端到端测试：导出并验证证据包
func TestEndToEnd_ExportAndVerify(t *testing.T) {
	jobID := "job_e2e_1"

	// 1. 创建测试数据（基础事件）
	baseEvents := makeTestEvents(jobID, 18)

	// 添加 tool_invocation_finished 事件（对应 ledger 中的记录）
	finishedEvent := Event{
		ID:        "19",
		JobID:     jobID,
		Type:      "tool_invocation_finished",
		Payload:   `{"idempotency_key":"key_1","outcome":"success","tool_name":"github_create_issue"}`,
		CreatedAt: time.Now().UTC().Add(19 * time.Second),
		PrevHash:  baseEvents[len(baseEvents)-1].Hash,
	}
	finishedEvent.Hash = ComputeEventHash(finishedEvent)

	completedEvent := Event{
		ID:        "20",
		JobID:     jobID,
		Type:      "job_completed",
		Payload:   `{}`,
		CreatedAt: time.Now().UTC().Add(20 * time.Second),
		PrevHash:  finishedEvent.Hash,
	}
	completedEvent.Hash = ComputeEventHash(completedEvent)

	events := append(baseEvents, finishedEvent, completedEvent)

	toolInvocations := []ToolInvocation{
		{
			ID:             "inv_1",
			JobID:          jobID,
			IdempotencyKey: "key_1",
			StepID:         "step_1",
			ToolName:       "github_create_issue",
			Status:         "success",
			Committed:      true,
			Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		},
	}

	// 2. 导出证据包
	zipBytes, err := ExportEvidenceZip(
		context.Background(),
		jobID,
		memJobStore{events: events},
		memLedger{invocations: toolInvocations},
		ExportOptions{
			RuntimeVersion: "2.0.0-test",
			SchemaVersion:  "2.0",
		},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// 3. 验证证据包
	result := VerifyEvidenceZip(zipBytes, nil)
	if !result.OK {
		t.Errorf("verification should pass, errors: %v", result.Errors)
	}

	// 4. 检查验证结果详情
	if !result.HashChainValid {
		t.Error("hash chain should be valid")
	}
	if !result.ManifestValid {
		t.Error("manifest should be valid")
	}
	if !result.EventsValid {
		t.Error("events should be valid")
	}
	if !result.LedgerValid {
		t.Error("ledger should be valid")
	}
	if len(result.Events) != 20 {
		t.Errorf("expected 20 events, got %d", len(result.Events))
	}
}

// TestEndToEnd_TamperDetection 端到端测试：篡改检测
func TestEndToEnd_TamperDetection(t *testing.T) {
	jobID := "job_e2e_2"
	events := makeTestEvents(jobID, 10)

	zipBytes, err := ExportEvidenceZip(
		context.Background(),
		jobID,
		memJobStore{events: events},
		memLedger{invocations: nil},
		ExportOptions{RuntimeVersion: "test"},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// 验证原始证据包（应该通过）
	result := VerifyEvidenceZip(zipBytes, nil)
	if !result.OK {
		t.Fatalf("original package should verify, errors: %v", result.Errors)
	}

	// 篡改证据包
	tamperedZip := tamperZipFile(zipBytes, "events.ndjson", func(b []byte) []byte {
		if len(b) > 50 {
			b[50] ^= 0xAA
		}
		return b
	})

	// 验证篡改后的证据包（应该失败）
	tamperedResult := VerifyEvidenceZip(tamperedZip, nil)
	if tamperedResult.OK {
		t.Error("tampered package should fail verification")
	}
	if len(tamperedResult.Errors) == 0 {
		t.Error("expected errors in tampered result")
	}
}

// TestEndToEnd_ChainIntegrity 端到端测试：哈希链完整性
func TestEndToEnd_ChainIntegrity(t *testing.T) {
	jobID := "job_e2e_3"

	// 创建一个有明确哈希链的事件序列
	events := []Event{
		{
			ID:        "1",
			JobID:     jobID,
			Type:      "job_created",
			Payload:   `{"goal":"test"}`,
			CreatedAt: time.Now().UTC(),
			PrevHash:  "",
		},
	}
	events[0].Hash = ComputeEventHash(events[0])

	// 添加第二个事件
	event2 := Event{
		ID:        "2",
		JobID:     jobID,
		Type:      "plan_generated",
		Payload:   `{"task_graph":{}}`,
		CreatedAt: time.Now().UTC().Add(time.Second),
		PrevHash:  events[0].Hash,
	}
	event2.Hash = ComputeEventHash(event2)
	events = append(events, event2)

	// 验证链
	if err := ValidateChain(events); err != nil {
		t.Errorf("chain should be valid: %v", err)
	}

	// 导出并验证
	zipBytes, err := ExportEvidenceZip(
		context.Background(),
		jobID,
		memJobStore{events: events},
		nil,
		ExportOptions{RuntimeVersion: "test"},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	result := VerifyEvidenceZip(zipBytes, nil)
	if !result.OK {
		t.Errorf("verification failed: %v", result.Errors)
	}
}

func TestEndToEnd_RedactedExportRemovesPIIAndVerifies(t *testing.T) {
	jobID := "job_e2e_redacted"
	event := Event{
		ID:        "1",
		JobID:     jobID,
		Type:      "tool_invocation_finished",
		Payload:   `{"email":"alice@example.com","phone":"+1-555-0100","ssn":"123-45-6789","api_key":"secret-key","safe":"keep"}`,
		CreatedAt: time.Now().UTC(),
	}
	event.Hash = ComputeEventHash(event)

	zipBytes, err := ExportEvidenceZip(
		context.Background(),
		jobID,
		memJobStore{events: []Event{event}},
		nil,
		ExportOptions{
			RuntimeVersion:   "test",
			RedactionEnabled: true,
			RedactionSalt:    "fixture-salt",
		},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	result := VerifyEvidenceZip(zipBytes, nil)
	if !result.OK {
		t.Fatalf("redacted package should verify, errors: %v", result.Errors)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected one event, got %d", len(result.Events))
	}
	payload := result.Events[0].Payload
	for _, forbidden := range []string{"alice@example.com", "+1-555-0100", "123-45-6789", "secret-key", "api_key"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("redacted payload leaked %q: %s", forbidden, payload)
		}
	}
	if !strings.Contains(payload, `"safe":"keep"`) {
		t.Fatalf("redacted payload should retain safe field: %s", payload)
	}
	if !strings.Contains(payload, `"email":"hash:`) {
		t.Fatalf("email should be hashed for correlation: %s", payload)
	}
}

func TestEndToEnd_RedactedExportRequiresSalt(t *testing.T) {
	jobID := "job_e2e_redacted_no_salt"
	event := Event{
		ID:        "1",
		JobID:     jobID,
		Type:      "tool_invocation_finished",
		Payload:   `{"email":"alice@example.com"}`,
		CreatedAt: time.Now().UTC(),
	}
	event.Hash = ComputeEventHash(event)

	_, err := ExportEvidenceZip(
		context.Background(),
		jobID,
		memJobStore{events: []Event{event}},
		nil,
		ExportOptions{
			RuntimeVersion:   "test",
			RedactionEnabled: true,
		},
	)
	if err == nil {
		t.Fatal("expected error when redaction is enabled without salt")
	}
	if !strings.Contains(err.Error(), "redaction_salt is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
