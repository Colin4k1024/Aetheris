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
	"testing"
)

func TestChunk(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-1",
		Content:    "test content",
		DocumentID: "doc-1",
		Metadata:   map[string]interface{}{"key": "value"},
	}
	if chunk.ID != "chunk-1" {
		t.Errorf("expected ID chunk-1, got %s", chunk.ID)
	}
	if chunk.Content != "test content" {
		t.Errorf("expected Content test content, got %s", chunk.Content)
	}
	if chunk.DocumentID != "doc-1" {
		t.Errorf("expected DocumentID doc-1, got %s", chunk.DocumentID)
	}
}

func TestChunk_WithNilMetadata(t *testing.T) {
	chunk := Chunk{
		ID:      "chunk-1",
		Content: "test",
	}
	if chunk.Metadata != nil {
		t.Error("expected nil metadata")
	}
}

type mockRetriever struct {
	chunks []Chunk
	err    error
}

func (m *mockRetriever) Retrieve(ctx context.Context, query, collection string, topK int) ([]Chunk, error) {
	return m.chunks, m.err
}

func TestRetrieverInterface(t *testing.T) {
	mock := &mockRetriever{
		chunks: []Chunk{{ID: "1", Content: "content"}},
	}
	var _ Retriever = mock

	chunks, err := mock.Retrieve(context.Background(), "query", "col", 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}

type mockGenerator struct {
	output string
	err    error
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	return m.output, m.err
}

func TestGeneratorInterface(t *testing.T) {
	mock := &mockGenerator{output: "test output"}
	var _ Generator = mock

	result, err := mock.Generate(context.Background(), "prompt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "test output" {
		t.Errorf("expected test output, got %s", result)
	}
}

type mockDocumentLoader struct {
	result interface{}
	err    error
}

func (m *mockDocumentLoader) Load(ctx context.Context, input interface{}) (interface{}, error) {
	return m.result, m.err
}

func TestDocumentLoaderInterface(t *testing.T) {
	mock := &mockDocumentLoader{result: "loaded"}
	var _ DocumentLoader = mock

	result, err := mock.Load(context.Background(), "input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "loaded" {
		t.Errorf("expected loaded, got %v", result)
	}
}

type mockDocumentParser struct {
	result interface{}
	err    error
}

func (m *mockDocumentParser) Parse(ctx context.Context, doc interface{}) (interface{}, error) {
	return m.result, m.err
}

func TestDocumentParserInterface(t *testing.T) {
	mock := &mockDocumentParser{result: "parsed"}
	var _ DocumentParser = mock

	result, err := mock.Parse(context.Background(), "doc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "parsed" {
		t.Errorf("expected parsed, got %v", result)
	}
}

type mockDocumentSplitter struct {
	result interface{}
	err    error
}

func (m *mockDocumentSplitter) Split(ctx context.Context, doc interface{}) (interface{}, error) {
	return m.result, m.err
}

func TestDocumentSplitterInterface(t *testing.T) {
	mock := &mockDocumentSplitter{result: []string{"chunk1", "chunk2"}}
	var _ DocumentSplitter = mock

	result, err := mock.Split(context.Background(), "doc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

type mockDocumentEmbedding struct {
	result interface{}
	err    error
}

func (m *mockDocumentEmbedding) Embed(ctx context.Context, doc interface{}) (interface{}, error) {
	return m.result, m.err
}

func TestDocumentEmbeddingInterface(t *testing.T) {
	mock := &mockDocumentEmbedding{result: []float64{0.1, 0.2}}
	var _ DocumentEmbedding = mock

	result, err := mock.Embed(context.Background(), "doc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

type mockDocumentIndexer struct {
	result interface{}
	err    error
}

func (m *mockDocumentIndexer) Index(ctx context.Context, doc interface{}) (interface{}, error) {
	return m.result, m.err
}

func TestDocumentIndexerInterface(t *testing.T) {
	mock := &mockDocumentIndexer{result: "indexed"}
	var _ DocumentIndexer = mock

	result, err := mock.Index(context.Background(), "doc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "indexed" {
		t.Errorf("expected indexed, got %v", result)
	}
}
