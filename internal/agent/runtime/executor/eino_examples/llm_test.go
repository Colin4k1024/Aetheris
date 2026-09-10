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

package eino_examples

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

// --- MockChatModel.Stream Tests ---

func TestMockChatModel_Stream(t *testing.T) {
	model := MockChatModelWithResponse("hello world streaming")
	sr, err := model.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []string
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("recv error: %v", err)
		}
		chunks = append(chunks, msg.Content)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	combined := ""
	for _, c := range chunks {
		combined += c
	}
	if combined != "hello world streaming " {
		t.Errorf("expected 'hello world streaming ', got %q", combined)
	}
}

func TestMockChatModel_Stream_Error(t *testing.T) {
	model := &MockChatModel{err: fmt.Errorf("test error")}
	sr, err := model.Stream(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, recvErr := sr.Recv()
	if recvErr == nil {
		t.Fatal("expected error from stream")
	}
}

func TestMockChatModel_Stream_Cancelled(t *testing.T) {
	model := MockChatModelWithResponse("one two three four five")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sr, err := model.Stream(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should eventually get an error from cancelled context
	for {
		_, err := sr.Recv()
		if err != nil {
			break
		}
	}
}

// --- OllamaChatModel.Stream Tests ---

func TestOllamaChatModel_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		// Simulate Ollama NDJSON streaming
		fmt.Fprintf(w, `{"message":{"content":"hello"},"done":false}`+"\n")
		fmt.Fprintf(w, `{"message":{"content":" world"},"done":false}`+"\n")
		fmt.Fprintf(w, `{"message":{},"done":true}`+"\n")
	}))
	defer server.Close()

	// Create OllamaChatModel with a mock client pointing to the test server
	// Since we can't easily create a real OllamaClient, test the streaming via streamHTTP
	sr, err := streamHTTP(context.Background(), server.URL+"/api/chat", map[string]any{
		"model":  "test",
		"stream": true,
	}, nil, parseOllamaStreamChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []string
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("recv error: %v", err)
		}
		chunks = append(chunks, msg.Content)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0] != "hello" {
		t.Errorf("expected 'hello', got %q", chunks[0])
	}
	if chunks[1] != " world" {
		t.Errorf("expected ' world', got %q", chunks[1])
	}
}

func TestOllamaChatModel_Stream_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "server error")
	}))
	defer server.Close()

	_, err := streamHTTP(context.Background(), server.URL, map[string]any{}, nil, parseOllamaStreamChunk)
	if err == nil {
		t.Fatal("expected error for server failure")
	}
}

// --- OpenAIChatModel.Stream Tests ---

func TestOpenAIChatModel_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Simulate OpenAI SSE streaming
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n")
		fmt.Fprintf(w, "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n")
	}))
	defer server.Close()

	sr, err := streamHTTP(context.Background(), server.URL, map[string]any{
		"model":  "gpt-4",
		"stream": true,
	}, nil, parseOpenAIStreamChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []string
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("recv error: %v", err)
		}
		chunks = append(chunks, msg.Content)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0] != "hello" {
		t.Errorf("expected 'hello', got %q", chunks[0])
	}
}

func TestOpenAIChatModel_Stream_Done(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	sr, err := streamHTTP(context.Background(), server.URL, map[string]any{}, nil, parseOpenAIStreamChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for {
		_, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestOpenAIChatModel_Stream_EmptyLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "\n\n")
		fmt.Fprintf(w, ": comment\n\n")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	sr, err := streamHTTP(context.Background(), server.URL, map[string]any{}, nil, parseOpenAIStreamChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []string
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("recv error: %v", err)
		}
		chunks = append(chunks, msg.Content)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 content chunk (comments skipped), got %d", len(chunks))
	}
}

// --- CreateToolsFromFuncs Tests ---

func TestCreateToolsFromFuncs(t *testing.T) {
	funcs := map[string]func(ctx context.Context, args map[string]any) (string, error){
		"tool_a": func(ctx context.Context, args map[string]any) (string, error) {
			return "result_a", nil
		},
		"tool_b": func(ctx context.Context, args map[string]any) (string, error) {
			return "result_b", nil
		},
	}

	tools := CreateToolsFromFuncs(funcs)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	for _, t_ := range tools {
		executor, ok := t_.(ToolExecutor)
		if !ok {
			t.Fatal("expected tool to implement ToolExecutor")
		}
		result, err := executor.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("execute error: %v", err)
		}
		if !startsWith(result, "result_") {
			t.Errorf("expected result_ prefix, got %s", result)
		}
	}
}

// --- SimpleTool Tests ---

func TestSimpleTool_Name(t *testing.T) {
	tool := NewSimpleTool("my_tool", "A test tool", nil)
	if tool.Name() != "my_tool" {
		t.Errorf("expected 'my_tool', got %s", tool.Name())
	}
}

func TestSimpleTool_Description(t *testing.T) {
	tool := NewSimpleTool("test", "A description", nil)
	if tool.Description() != "A description" {
		t.Errorf("expected 'A description', got %s", tool.Description())
	}
}

func TestSimpleTool_Execute(t *testing.T) {
	tool := NewSimpleTool("test", "desc", func(ctx context.Context, args map[string]any) (string, error) {
		return "executed with: " + args["input"].(string), nil
	})
	result, err := tool.Execute(context.Background(), map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "executed with: hello" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestSimpleTool_NilHandler(t *testing.T) {
	tool := NewSimpleTool("test", "desc", nil)
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestSimpleTool_ImplementsToolExecutor(t *testing.T) {
	tool := NewSimpleTool("test", "desc", func(ctx context.Context, args map[string]any) (string, error) {
		return "ok", nil
	})
	var _ ToolExecutor = tool
}

// --- ParseToolArguments Tests ---

func TestParseToolArguments_ValidJSON(t *testing.T) {
	args, err := ParseToolArguments(`{"name":"test","count":42,"nested":{"key":"value"}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args["name"] != "test" {
		t.Errorf("name mismatch: %v", args["name"])
	}
	if args["count"] != float64(42) {
		t.Errorf("count mismatch: %v", args["count"])
	}
	nested, ok := args["nested"].(map[string]any)
	if !ok {
		t.Fatal("expected nested object")
	}
	if nested["key"] != "value" {
		t.Errorf("nested key mismatch: %v", nested["key"])
	}
}

func TestParseToolArguments_Empty(t *testing.T) {
	args, err := ParseToolArguments("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 0 {
		t.Errorf("expected empty map, got %d items", len(args))
	}
}

func TestParseToolArguments_InvalidJSON(t *testing.T) {
	_, err := ParseToolArguments(`{bad json}`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseToolArguments_NoLongerKeyEqualsValue(t *testing.T) {
	// Verify the old key=value parsing is gone
	args, err := ParseToolArguments(`key=value`)
	if err == nil {
		t.Error("expected error for key=value (should require JSON)")
	}
	_ = args
}

// --- isOllamaAvailable Tests ---

func TestIsOllamaAvailable_NotRunning(t *testing.T) {
	// Without OLLAMA_BASE_URL set to a running server, should return false
	// (default localhost:11434 is unlikely to be running in CI)
	// Save and restore env
	orig := http.DefaultClient.Timeout
	http.DefaultClient.Timeout = 500 * time.Millisecond
	defer func() { http.DefaultClient.Timeout = orig }()

	_ = isOllamaAvailable() // Just verify it doesn't panic; result depends on environment
}

func TestIsOllamaAvailable_Running(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"models":[]}`)
	}))
	defer server.Close()

	orig := os.Getenv("OLLAMA_BASE_URL")
	os.Setenv("OLLAMA_BASE_URL", server.URL)
	defer os.Setenv("OLLAMA_BASE_URL", orig)

	if !isOllamaAvailable() {
		t.Error("expected isOllamaAvailable to return true for running server")
	}
}

func TestIsOllamaAvailable_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	orig := os.Getenv("OLLAMA_BASE_URL")
	os.Setenv("OLLAMA_BASE_URL", server.URL)
	defer os.Setenv("OLLAMA_BASE_URL", orig)

	if isOllamaAvailable() {
		t.Error("expected isOllamaAvailable to return false for 404")
	}
}

// --- Stream Parser Tests ---

func TestParseOllamaStreamChunk(t *testing.T) {
	content, done, err := parseOllamaStreamChunk(`{"message":{"content":"hello"},"done":false}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "hello" {
		t.Errorf("expected 'hello', got %s", content)
	}
	if done {
		t.Error("expected done=false")
	}

	content, done, _ = parseOllamaStreamChunk(`{"message":{},"done":true}`)
	if !done {
		t.Error("expected done=true")
	}
}

func TestParseOpenAIStreamChunk(t *testing.T) {
	content, done, err := parseOpenAIStreamChunk(`data: {"choices":[{"delta":{"content":"hi"}}]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "hi" {
		t.Errorf("expected 'hi', got %s", content)
	}
	if done {
		t.Error("expected done=false")
	}

	_, done, _ = parseOpenAIStreamChunk(`data: [DONE]`)
	if !done {
		t.Error("expected done=true for [DONE]")
	}

	_, done, _ = parseOpenAIStreamChunk(`data: {"choices":[{"finish_reason":"stop"}]}`)
	if !done {
		t.Error("expected done=true for finish_reason")
	}
}

func TestParseOpenAIStreamChunk_NonDataLine(t *testing.T) {
	content, done, _ := parseOpenAIStreamChunk(`: comment`)
	if content != "" {
		t.Error("expected empty content for comment line")
	}
	if done {
		t.Error("expected done=false for comment line")
	}
}

// --- Integration: MockChatModel streaming with adapter ---

func TestMockChatModel_StreamViaAdapter(t *testing.T) {
	model := MockChatModelWithResponse("The quick brown fox")
	sr, err := model.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	combined, err := schema.ConcatMessageStream(sr)
	if err != nil {
		t.Fatalf("concat error: %v", err)
	}
	if combined.Content != "The quick brown fox " {
		t.Errorf("expected 'The quick brown fox ', got %q", combined.Content)
	}
}

// --- Helper ---

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
