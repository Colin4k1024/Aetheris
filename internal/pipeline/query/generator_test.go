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

package query

import (
	"context"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/model/llm"
	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

// mockLLMClient is a mock implementation of llm.Client
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) Generate(prompt string, opts llm.GenerateOptions) (string, error) {
	return m.response, m.err
}

func (m *mockLLMClient) GenerateWithContext(ctx context.Context, prompt string, opts llm.GenerateOptions) (string, error) {
	return m.response, m.err
}

func (m *mockLLMClient) Chat(messages []llm.Message, opts llm.GenerateOptions) (string, error) {
	return m.response, m.err
}

func (m *mockLLMClient) ChatWithContext(ctx context.Context, messages []llm.Message, opts llm.GenerateOptions) (string, error) {
	return m.response, m.err
}

func (m *mockLLMClient) Model() string {
	return "mock-model"
}

func (m *mockLLMClient) Provider() string {
	return "mock"
}

func (m *mockLLMClient) SetModel(model string) {}

func (m *mockLLMClient) SetAPIKey(apiKey string) {}

func TestNewGenerator(t *testing.T) {
	client := &mockLLMClient{response: "test response"}
	g := NewGenerator(client, 4096, 0.5)
	if g == nil {
		t.Fatal("expected non-nil generator")
	}
	if g.Name() != "generator" {
		t.Errorf("expected name generator, got %s", g.Name())
	}
}

func TestNewGenerator_Defaults(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 0, 0)
	if g.maxContextSize != 4096 {
		t.Errorf("expected default maxContextSize 4096, got %d", g.maxContextSize)
	}
	if g.temperature != 0.1 {
		t.Errorf("expected default temperature 0.1, got %f", g.temperature)
	}
}

func TestGenerator_Validate(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	err := g.Validate(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}

	err = g.Validate("invalid")
	if err == nil {
		t.Error("expected error for invalid type")
	}

	err = g.Validate(map[string]any{
		"query":            &common.Query{},
		"retrieval_result": &common.RetrievalResult{},
	})
	if err != nil {
		t.Errorf("expected no error for valid input, got %v", err)
	}
}

func TestGenerator_Validate_MissingQuery(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	err := g.Validate(map[string]any{
		"retrieval_result": &common.RetrievalResult{},
	})
	if err == nil {
		t.Error("expected error for missing query")
	}
}

func TestGenerator_Validate_MissingResult(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	err := g.Validate(map[string]any{
		"query": &common.Query{},
	})
	if err == nil {
		t.Error("expected error for missing retrieval_result")
	}
}

func TestGenerator_Validate_NoClient(t *testing.T) {
	g := &Generator{llmClient: nil}

	err := g.Validate(map[string]any{
		"query":            &common.Query{},
		"retrieval_result": &common.RetrievalResult{},
	})
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestGenerator_ProcessQuery(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	query := &common.Query{Text: "test query"}
	result, err := g.ProcessQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGenerator_GenerateWithRetrieval(t *testing.T) {
	client := &mockLLMClient{response: "test response"}
	g := NewGenerator(client, 4096, 0.5)

	query := &common.Query{Text: "test query"}
	result := &common.RetrievalResult{
		Chunks: []common.Chunk{
			{ID: "chunk-1", Content: "content 1", DocumentID: "doc-1", Index: 0},
			{ID: "chunk-2", Content: "content 2", DocumentID: "doc-1", Index: 1},
		},
	}

	genResult, err := g.GenerateWithRetrieval(query, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if genResult == nil {
		t.Fatal("expected non-nil result")
	}
	if genResult.Answer != "test response" {
		t.Errorf("expected response 'test response', got %s", genResult.Answer)
	}
	if len(genResult.References) != 2 {
		t.Errorf("expected 2 references, got %d", len(genResult.References))
	}
}

func TestGenerator_SetLLMClient(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	newClient := &mockLLMClient{}
	g.SetLLMClient(newClient)
	if g.llmClient != newClient {
		t.Error("LLM client not set correctly")
	}
}

func TestGenerator_GetLLMClient(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	got := g.GetLLMClient()
	if got != client {
		t.Error("GetLLMClient returned wrong client")
	}
}

func TestGenerator_SetMaxContextSize(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	g.SetMaxContextSize(8192)
	if g.maxContextSize != 8192 {
		t.Errorf("expected maxContextSize 8192, got %d", g.maxContextSize)
	}

	g.SetMaxContextSize(0)
	if g.maxContextSize != 8192 {
		t.Errorf("expected maxContextSize to remain 8192, got %d", g.maxContextSize)
	}
}

func TestGenerator_SetTemperature(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	g.SetTemperature(0.8)
	if g.temperature != 0.8 {
		t.Errorf("expected temperature 0.8, got %f", g.temperature)
	}

	g.SetTemperature(0)
	if g.temperature != 0.8 {
		t.Errorf("expected temperature to remain 0.8, got %f", g.temperature)
	}
}

func TestGenerator_BuildPrompt(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	query := &common.Query{Text: "What is Go?"}
	result := &common.RetrievalResult{
		Chunks: []common.Chunk{
			{ID: "chunk-1", Content: "Go is a programming language.", DocumentID: "doc-1", Index: 0},
		},
	}

	prompt := g.buildPrompt(query, result)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestGenerator_ExtractReferences(t *testing.T) {
	client := &mockLLMClient{}
	g := NewGenerator(client, 4096, 0.5)

	result := &common.RetrievalResult{
		Chunks: []common.Chunk{
			{ID: "chunk-1", Content: "content 1", DocumentID: "doc-1", Index: 0},
			{ID: "chunk-2", Content: "content 2", DocumentID: "doc-2", Index: 1},
		},
	}

	refs := g.extractReferences(result)
	if len(refs) != 2 {
		t.Errorf("expected 2 references, got %d", len(refs))
	}
}
