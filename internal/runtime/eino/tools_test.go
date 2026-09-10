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

package eino

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

// --- createUnavailableTool tests ---

func TestCreateUnavailableTool_ReturnsError(t *testing.T) {
	tt := createUnavailableTool("retriever", "检索相关文档")
	result, err := invokeTool(t, tt, "test input")
	if err == nil {
		t.Fatal("expected error from unavailable tool")
	}
	if result != "" {
		t.Errorf("expected empty result, got %s", result)
	}
}

func TestCreateUnavailableTool_DoesNotEchoInput(t *testing.T) {
	tt := createUnavailableTool("generator", "生成回答")
	result, err := invokeTool(t, tt, "test input")
	if err == nil {
		t.Fatal("expected error")
	}
	// Result should NOT contain the input as a fake success
	if strings.Contains(result, "test input") {
		t.Error("unavailable tool should not echo input as fake result")
	}
}

func TestCreateUnavailableTool_ErrorMentionsDependency(t *testing.T) {
	tt := createUnavailableTool("embedding", "文本向量化")
	_, err := invokeTool(t, tt, `"input"`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unavailable") && !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error should mention unavailability, got: %v", err)
	}
}

// --- Engine with nil deps returns unavailable tools ---

func TestCreateRetrieverTool_NilEngine_Unavailable(t *testing.T) {
	tt := CreateRetrieverTool(nil) // nil engine → unavailable
	_, err := invokeTool(t, tt, `{"query":"test"}`)
	if err == nil {
		t.Fatal("expected error when engine is nil")
	}
}

func TestCreateGeneratorTool_NilEngine_Unavailable(t *testing.T) {
	tt := CreateGeneratorTool(nil)
	_, err := invokeTool(t, tt, `{"prompt":"test"}`)
	if err == nil {
		t.Fatal("expected error when engine is nil")
	}
}

func TestCreateDocumentLoaderTool_NilEngine_Unavailable(t *testing.T) {
	tt := CreateDocumentLoaderTool(nil)
	_, err := invokeTool(t, tt, `{"path":"/tmp/test"}`)
	if err == nil {
		t.Fatal("expected error when engine is nil")
	}
}

// --- ContextManager.ExecuteTool ---

func TestContextManager_ExecuteTool_NilContext_ReturnsError(t *testing.T) {
	cm := NewContextManager()
	// No runners registered → GetRunner fails
	_, err := cm.ExecuteTool(context.Background(), "nonexistent", "tool", "input")
	if err == nil {
		t.Fatal("expected error for non-existent runner")
	}
}

func TestContextManager_ExecuteTool_DoesNotReturnMockResult(t *testing.T) {
	cm := NewContextManager()
	result, err := cm.ExecuteTool(context.Background(), "test", "mytool", "input")
	if err == nil {
		t.Fatal("expected error, not mock result")
	}
	// Result should NOT contain mock "执行结果" text
	if strings.Contains(result, "执行结果") {
		t.Error("should not return mock result")
	}
}

// --- Query workflow returns error when deps missing ---

func TestQueryWorkflow_NilRetriever_ReturnsError(t *testing.T) {
	qwf := NewQueryWorkflowExecutor(nil, nil, nil, nil)
	_, err := qwf.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error when retriever/generator is nil")
	}
}

// --- Ingest workflow states ---
// IngestWorkflow auto-creates loader/parser/splitter if nil; with nil embedding+indexer → parse_only mode

func TestIngestWorkflow_ParseOnly_WhenNoEmbeddingIndexer(t *testing.T) {
	iwf := NewIngestWorkflowExecutor(nil, nil, nil, nil, nil, nil)
	result, err := iwf.Execute(context.Background(), map[string]interface{}{
		"content": []byte("hello world test content"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["mode"] != "parse_only" {
		t.Errorf("expected parse_only mode, got %v", m["mode"])
	}
}

// --- helpers ---

// invokeTool invokes an InvokableTool and returns result/error
func invokeTool(t *testing.T, tt tool.BaseTool, input string) (string, error) {
	t.Helper()
	invokable, ok := tt.(tool.InvokableTool)
	if !ok {
		t.Fatalf("tool is not InvokableTool: %T", tt)
	}
	return invokable.InvokableRun(context.Background(), input)
}
