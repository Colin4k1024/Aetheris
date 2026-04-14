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

package ingest

import (
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
	"github.com/cloudwego/eino/schema"
)

func TestCommonDocumentToSchema(t *testing.T) {
	doc := &common.Document{
		ID:      "doc-1",
		Content: "test content",
		Metadata: map[string]interface{}{
			"author": "test",
		},
	}
	result := CommonDocumentToSchema(doc, "test-uri")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "doc-1" {
		t.Errorf("expected ID doc-1, got %s", result.ID)
	}
}

func TestCommonDocumentToSchema_Nil(t *testing.T) {
	result := CommonDocumentToSchema(nil, "test-uri")
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestSchemaDocumentToCommonDocument(t *testing.T) {
	schemaDoc := &schema.Document{
		ID:      "doc-1",
		Content: "test content",
		MetaData: map[string]any{
			"author": "test",
		},
	}
	result := SchemaDocumentToCommonDocument(schemaDoc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "doc-1" {
		t.Errorf("expected ID doc-1, got %s", result.ID)
	}
}

func TestSchemaDocumentToCommonDocument_Nil(t *testing.T) {
	result := SchemaDocumentToCommonDocument(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestCommonChunkToSchemaDocument(t *testing.T) {
	chunk := common.Chunk{
		ID:         "chunk-1",
		Content:    "test chunk",
		Index:      1,
		TokenCount: 100,
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	result := CommonChunkToSchemaDocument(chunk, "doc-1", []float64{0.1, 0.2})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "chunk-1" {
		t.Errorf("expected ID chunk-1, got %s", result.ID)
	}
}

func TestChunksToSchemaDocuments(t *testing.T) {
	doc := &common.Document{
		ID: "doc-1",
		Chunks: []common.Chunk{
			{ID: "chunk-1", Content: "chunk 1", Index: 0},
			{ID: "chunk-2", Content: "chunk 2", Index: 1},
		},
	}
	result := ChunksToSchemaDocuments(doc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(result))
	}
}

func TestChunksToSchemaDocuments_Nil(t *testing.T) {
	result := ChunksToSchemaDocuments(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}

	result = ChunksToSchemaDocuments(&common.Document{})
	if result != nil {
		t.Error("expected nil for empty chunks")
	}
}
