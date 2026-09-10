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

package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOpenAIEmbedder_MissingAPIKey(t *testing.T) {
	_, err := NewOpenAIEmbedder("", "text-embedding-3-small", "", 1536)
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestNewOpenAIEmbedder_Defaults(t *testing.T) {
	emb, err := NewOpenAIEmbedder("sk-test", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb.Model() != "text-embedding-3-small" {
		t.Errorf("expected default model, got %s", emb.Model())
	}
	if emb.Dimension() != 1536 {
		t.Errorf("expected default dimension 1536, got %d", emb.Dimension())
	}
	if emb.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default baseURL, got %s", emb.baseURL)
	}
}

func TestOpenAIEmbedder_EmptyTexts(t *testing.T) {
	emb, _ := NewOpenAIEmbedder("sk-test", "model", "http://test", 4)
	result, err := emb.Embed(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for empty texts")
	}
}

func TestOpenAIEmbedder_RealAPI(t *testing.T) {
	var capturedReq openaiEmbeddingRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("expected /embeddings, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("expected Bearer sk-test, got %s", r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)

		// Build raw JSON response matching OpenAI format
		var dataItems []string
		for range capturedReq.Input {
			dataItems = append(dataItems, `{"embedding":[0.1,0.2,0.3,0.4]}`)
		}
		raw := fmt.Sprintf(`{"data":[%s],"usage":{"prompt_tokens":2,"total_tokens":2}}`,
			joinJSON(dataItems))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()

	emb, err := NewOpenAIEmbedder("sk-test", "text-embedding-3-small", srv.URL, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	texts := []string{"hello world", "foo bar"}
	result, err := emb.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(result))
	}
	if len(result[0]) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(result[0]))
	}
	if result[0][0] != 0.1 {
		t.Errorf("expected 0.1, got %f", result[0][0])
	}

	if capturedReq.Model != "text-embedding-3-small" {
		t.Errorf("expected model in request, got %s", capturedReq.Model)
	}
	if len(capturedReq.Input) != 2 {
		t.Errorf("expected 2 inputs, got %d", len(capturedReq.Input))
	}
	if capturedReq.Input[0] != "hello world" {
		t.Errorf("expected 'hello world', got %s", capturedReq.Input[0])
	}
}

func TestOpenAIEmbedder_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	emb, err := NewOpenAIEmbedder("sk-bad", "model", srv.URL, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = emb.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestOpenAIEmbedder_EmptyVectors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := `{"data":[{"embedding":[]}],"usage":{"prompt_tokens":1,"total_tokens":1}}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()

	emb, err := NewOpenAIEmbedder("sk-test", "model", srv.URL, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = emb.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for empty vector")
	}
}

func TestOpenAIEmbedder_MismatchedCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := `{"data":[{"embedding":[0.1]}],"usage":{"prompt_tokens":1,"total_tokens":1}}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()

	emb, err := NewOpenAIEmbedder("sk-test", "model", srv.URL, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = emb.Embed(context.Background(), []string{"hello", "world"})
	if err == nil {
		t.Fatal("expected error for mismatched count")
	}
}

func TestNewOpenAIAdapter(t *testing.T) {
	adapter, err := NewOpenAIAdapter("sk-test", "text-embedding-3-small", 1536)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Model() != "text-embedding-3-small" {
		t.Errorf("expected text-embedding-3-small, got %s", adapter.Model())
	}
	if adapter.Dimension() != 1536 {
		t.Errorf("expected 1536, got %d", adapter.Dimension())
	}
}

func TestNewOpenAIAdapter_MissingAPIKey(t *testing.T) {
	_, err := NewOpenAIAdapter("", "model", 128)
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
}

// joinJSON joins string slice with comma
func joinJSON(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}
