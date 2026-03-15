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
	"testing"
)

func TestNewEmbedder(t *testing.T) {
	emb := NewEmbedder("text-embedding-3-small", 1536)
	if emb.Model() != "text-embedding-3-small" {
		t.Errorf("expected text-embedding-3-small, got %s", emb.Model())
	}
	if emb.Dimension() != 1536 {
		t.Errorf("expected 1536, got %d", emb.Dimension())
	}
}

func TestNewEmbedder_DefaultDimension(t *testing.T) {
	emb := NewEmbedder("model", 0)
	if emb.Dimension() != 1536 {
		t.Errorf("expected 1536 default, got %d", emb.Dimension())
	}
}

func TestEmbedder_Model_Nil(t *testing.T) {
	var emb *Embedder
	if emb.Model() != "" {
		t.Errorf("expected empty string for nil, got %s", emb.Model())
	}
}

func TestEmbedder_Dimension_Nil(t *testing.T) {
	var emb *Embedder
	if emb.Dimension() != 0 {
		t.Errorf("expected 0 for nil, got %d", emb.Dimension())
	}
}

func TestEmbedder_Embed(t *testing.T) {
	emb := NewEmbedder("model", 128)
	result, err := emb.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
	if len(result[0]) != 128 {
		t.Errorf("expected 128 dimensions, got %d", len(result[0]))
	}
}

func TestEmbedder_Embed_Nil(t *testing.T) {
	var emb *Embedder
	result, err := emb.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for nil embedder")
	}
}

func TestEmbedder_Embed_EmptyTexts(t *testing.T) {
	emb := NewEmbedder("model", 128)
	result, err := emb.Embed(context.Background(), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for empty texts")
	}
}
