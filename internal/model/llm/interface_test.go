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

package llm

import (
	"context"
	"testing"
)

type mockInnerClient struct{}

func (m *mockInnerClient) Generate(prompt string, options GenerateOptions) (string, error) {
	return "response", nil
}

func (m *mockInnerClient) GenerateWithContext(ctx context.Context, prompt string, options GenerateOptions) (string, error) {
	return "response", nil
}

func (m *mockInnerClient) Chat(messages []Message, options GenerateOptions) (string, error) {
	return "response", nil
}

func (m *mockInnerClient) ChatWithContext(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	return "response", nil
}

func (m *mockInnerClient) Model() string { return "mock-model" }

func (m *mockInnerClient) Provider() string { return "mock" }

func (m *mockInnerClient) SetModel(model string) {}

func (m *mockInnerClient) SetAPIKey(apiKey string) {}

func TestNewClient_OpenAI(t *testing.T) {
	client, err := NewClient("openai", "gpt-4", "test-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Model() != "gpt-4" {
		t.Errorf("expected model gpt-4, got %s", client.Model())
	}
}

func TestNewClient_Claude(t *testing.T) {
	client, err := NewClient("claude", "claude-3", "test-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_Gemini(t *testing.T) {
	client, err := NewClient("gemini", "gemini-pro", "test-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_Ollama(t *testing.T) {
	client, err := NewClient("ollama", "llama2", "", "http://localhost:11434")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_Default(t *testing.T) {
	client, err := NewClient("unknown", "model", "key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestGenerateOptions_Defaults(t *testing.T) {
	opts := GenerateOptions{}
	if opts.Temperature != 0 {
		t.Errorf("expected default temperature 0, got %f", opts.Temperature)
	}
	if opts.MaxTokens != 0 {
		t.Errorf("expected default max_tokens 0, got %d", opts.MaxTokens)
	}
}

func TestMessage_Role(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "hello",
	}
	if msg.Role != "user" {
		t.Errorf("expected role user, got %s", msg.Role)
	}
	if msg.Content != "hello" {
		t.Errorf("expected content hello, got %s", msg.Content)
	}
}

func TestNewRateLimitedClient(t *testing.T) {
	inner := &mockInnerClient{}
	limiter := NewLLMRateLimiter(map[string]LLMLimitConfig{}, nil)
	client := NewRateLimitedClient(inner, limiter)

	if client.Model() != "mock-model" {
		t.Errorf("expected mock-model, got %s", client.Model())
	}
	if client.Provider() != "mock" {
		t.Errorf("expected mock, got %s", client.Provider())
	}
}

func TestRateLimitedClient_Generate(t *testing.T) {
	inner := &mockInnerClient{}
	limiter := NewLLMRateLimiter(map[string]LLMLimitConfig{}, nil)
	client := NewRateLimitedClient(inner, limiter)

	resp, err := client.Generate("test prompt", GenerateOptions{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp != "response" {
		t.Errorf("expected response, got %s", resp)
	}
}

func TestRateLimitedClient_Chat(t *testing.T) {
	inner := &mockInnerClient{}
	limiter := NewLLMRateLimiter(map[string]LLMLimitConfig{}, nil)
	client := NewRateLimitedClient(inner, limiter)

	resp, err := client.Chat([]Message{{Role: "user", Content: "test"}}, GenerateOptions{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp != "response" {
		t.Errorf("expected response, got %s", resp)
	}
}
