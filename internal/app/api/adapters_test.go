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

package api

import (
	"context"
	"testing"
)

type mockEmbedderForAPI struct{}

func (m *mockEmbedderForAPI) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	result := make([][]float64, len(texts))
	for i := range texts {
		result[i] = []float64{0.1, 0.2, 0.3}
	}
	return result, nil
}

func TestNewEinoEmbedderAdapter(t *testing.T) {
	embedder := &mockEmbedderForAPI{}
	adapter := NewEinoEmbedderAdapter(embedder)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestNewRetrieverAdapter_Defaults(t *testing.T) {
	embedder := &mockEmbedderForAPI{}
	// Passing nil for einoRetriever - will panic if called
	adapter := NewRetrieverAdapter(embedder, nil, 0)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestNewRetrieverAdapter_WithThreshold(t *testing.T) {
	embedder := &mockEmbedderForAPI{}
	adapter := NewRetrieverAdapter(embedder, nil, 0.8)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}
