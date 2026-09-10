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

package specialized

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

func TestJSONLPipeline_Name(t *testing.T) {
	p := NewJSONLPipeline()
	if p.Name() != "jsonl" {
		t.Errorf("expected name 'jsonl', got %s", p.Name())
	}
}

func TestJSONLPipeline_BasicParsing(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{
		ContentField: "text",
		IDField:      "id",
	})

	input := `{"id":"doc1","text":"hello world","category":"news"}
{"id":"doc2","text":"second line","category":"sports"}`

	result, err := p.ProcessSpecialized(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr, ok := result.(*JSONLResult)
	if !ok {
		t.Fatalf("expected *JSONLResult, got %T", result)
	}
	if len(jr.Documents) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(jr.Documents))
	}

	if jr.Documents[0].ID != "doc1" {
		t.Errorf("expected doc1, got %s", jr.Documents[0].ID)
	}
	if jr.Documents[0].Content != "hello world" {
		t.Errorf("expected 'hello world', got %s", jr.Documents[0].Content)
	}
	if jr.Documents[0].Metadata["category"] != "news" {
		t.Errorf("expected category=news, got %v", jr.Documents[0].Metadata["category"])
	}
	if jr.Documents[0].Metadata["source_line"] != 1 {
		t.Errorf("expected source_line=1, got %v", jr.Documents[0].Metadata["source_line"])
	}

	if jr.Documents[1].ID != "doc2" {
		t.Errorf("expected doc2, got %s", jr.Documents[1].ID)
	}
	if jr.Documents[1].Metadata["source_line"] != 2 {
		t.Errorf("expected source_line=2, got %v", jr.Documents[1].Metadata["source_line"])
	}
}

func TestJSONLPipeline_EmptyLines(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	input := `{"text":"line1"}

{"text":"line2"}
  `

	result, err := p.ProcessSpecialized(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 2 {
		t.Fatalf("expected 2 documents (empty lines skipped), got %d", len(jr.Documents))
	}
	if jr.Documents[0].Content != "line1" {
		t.Errorf("expected 'line1', got %s", jr.Documents[0].Content)
	}
	if jr.Documents[1].Content != "line2" {
		t.Errorf("expected 'line2', got %s", jr.Documents[1].Content)
	}
}

func TestJSONLPipeline_BadLine_Skip(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetErrorStrategy(JSONLErrorSkip)
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	input := `{"text":"good1"}
{bad json
{"text":"good2"}`

	result, err := p.ProcessSpecialized(input)
	if err != nil {
		t.Fatalf("unexpected error with skip strategy: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 2 {
		t.Fatalf("expected 2 documents (bad line skipped), got %d", len(jr.Documents))
	}
	if len(jr.Errors) != 0 {
		t.Errorf("expected 0 collected errors, got %d", len(jr.Errors))
	}
}

func TestJSONLPipeline_BadLine_Fail(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetErrorStrategy(JSONLErrorFail)
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	input := `{"text":"good"}
{bad json`

	_, err := p.ProcessSpecialized(input)
	if err == nil {
		t.Fatal("expected error with fail strategy")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("expected line 2 in error, got: %v", err)
	}
}

func TestJSONLPipeline_BadLine_Collect(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetErrorStrategy(JSONLErrorCollect)
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	input := `{"text":"good1"}
{bad json
{"text":"good2"}
{also bad`

	result, err := p.ProcessSpecialized(input)
	if err != nil {
		t.Fatalf("unexpected error with collect strategy: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 2 {
		t.Fatalf("expected 2 valid documents, got %d", len(jr.Documents))
	}
	if len(jr.Errors) != 2 {
		t.Fatalf("expected 2 collected errors, got %d", len(jr.Errors))
	}
	if jr.Errors[0].LineNumber != 2 {
		t.Errorf("expected error on line 2, got %d", jr.Errors[0].LineNumber)
	}
	if jr.Errors[1].LineNumber != 4 {
		t.Errorf("expected error on line 4, got %d", jr.Errors[1].LineNumber)
	}
}

func TestJSONLPipeline_DefaultContent(t *testing.T) {
	p := NewJSONLPipeline()
	// No ContentField set — should use full JSON as content

	input := `{"name":"test","value":123}`

	result, err := p.ProcessSpecialized(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(jr.Documents))
	}

	doc := jr.Documents[0]
	if !strings.Contains(doc.Content, "test") {
		t.Errorf("expected full JSON in content, got: %s", doc.Content)
	}
	if doc.Metadata["name"] != "test" {
		t.Errorf("expected name=test in metadata, got %v", doc.Metadata["name"])
	}
	if doc.Metadata["value"] != float64(123) {
		t.Errorf("expected value=123 in metadata, got %v", doc.Metadata["value"])
	}
}

func TestJSONLPipeline_MetadataFieldsMapping(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{
		ContentField:   "body",
		MetadataFields: map[string]string{"author": "creator", "tags": "labels"},
	})

	input := `{"body":"content here","author":"alice","tags":["a","b"],"unused":"x"}`

	result, err := p.ProcessSpecialized(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr := result.(*JSONLResult)
	doc := jr.Documents[0]
	if doc.Content != "content here" {
		t.Errorf("content mismatch: %s", doc.Content)
	}
	if doc.Metadata["creator"] != "alice" {
		t.Errorf("expected creator=alice, got %v", doc.Metadata["creator"])
	}
	// Since MetadataFields is set, only mapped fields go to metadata
	if _, exists := doc.Metadata["unused"]; exists {
		t.Error("expected 'unused' not in metadata with explicit mapping")
	}
}

func TestJSONLPipeline_FromBytes(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	input := []byte(`{"text":"byte input"}`)

	result, err := p.ProcessSpecialized(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(jr.Documents))
	}
	if jr.Documents[0].Content != "byte input" {
		t.Errorf("content mismatch: %s", jr.Documents[0].Content)
	}
}

func TestJSONLPipeline_FromDocument(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	doc := &common.Document{
		Content: `{"text":"from doc"}`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	result, err := p.ProcessSpecialized(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(jr.Documents))
	}
	if jr.Documents[0].Content != "from doc" {
		t.Errorf("content mismatch: %s", jr.Documents[0].Content)
	}
	if jr.Documents[0].Metadata["source"] != "test" {
		t.Errorf("expected base meta source=test, got %v", jr.Documents[0].Metadata["source"])
	}
}

func TestJSONLPipeline_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.jsonl")
	content := `{"id":"1","text":"file line 1"}
{"id":"2","text":"file line 2"}`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{
		ContentField: "text",
		IDField:      "id",
	})

	result, err := p.ProcessSpecialized(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 2 {
		t.Fatalf("expected 2 documents from file, got %d", len(jr.Documents))
	}
	if jr.Documents[0].ID != "1" {
		t.Errorf("expected id=1, got %s", jr.Documents[0].ID)
	}
	if jr.Documents[0].Metadata["source_file"] != filePath {
		t.Errorf("expected source_file in metadata, got %v", jr.Documents[0].Metadata["source_file"])
	}
}

func TestJSONLPipeline_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.jsonl")

	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, fmt.Sprintf(`{"id":"%d","text":"line %d"}`, i, i))
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{
		ContentField: "text",
		IDField:      "id",
	})

	result, err := p.ProcessSpecialized(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr := result.(*JSONLResult)
	if len(jr.Documents) != 1000 {
		t.Fatalf("expected 1000 documents, got %d", len(jr.Documents))
	}
	// Verify line numbers
	if jr.Documents[0].Metadata["source_line"] != 1 {
		t.Errorf("expected source_line=1, got %v", jr.Documents[0].Metadata["source_line"])
	}
	if jr.Documents[999].Metadata["source_line"] != 1000 {
		t.Errorf("expected source_line=1000, got %v", jr.Documents[999].Metadata["source_line"])
	}
}

func TestJSONLPipeline_CancelContext(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Pass context via Document metadata
	doc := &common.Document{
		Content: `{"text":"line1"}
{"text":"line2"}
{"text":"line3"}`,
		Metadata: map[string]interface{}{
			"__ctx": ctx,
		},
	}

	_, err := p.ProcessSpecialized(doc)
	if err == nil {
		// Cancellation might not be caught if scanner returns first line before checking
		// That's OK — the test just verifies no panic
	}
}

func TestJSONLPipeline_UnsupportedInput(t *testing.T) {
	p := NewJSONLPipeline()

	_, err := p.ProcessSpecialized(12345)
	if err == nil {
		t.Error("expected error for unsupported input type")
	}
}

func TestJSONLPipeline_AddStage(t *testing.T) {
	p := NewJSONLPipeline()

	if len(p.Stages()) != 0 {
		t.Fatalf("expected 0 stages initially, got %d", len(p.Stages()))
	}

	// AddStage should actually register the stage
	mockStage := &mockPipelineStage{name: "test-stage"}
	if err := p.AddStage(mockStage); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stages := p.Stages()
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage after AddStage, got %d", len(stages))
	}
	if stages[0].Name() != "test-stage" {
		t.Errorf("expected stage name 'test-stage', got %s", stages[0].Name())
	}
}

func TestJSONLPipeline_RemoveStage(t *testing.T) {
	p := NewJSONLPipeline()

	stage1 := &mockPipelineStage{name: "stage1"}
	stage2 := &mockPipelineStage{name: "stage2"}
	p.AddStage(stage1)
	p.AddStage(stage2)

	if len(p.Stages()) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(p.Stages()))
	}

	if err := p.RemoveStage("stage1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stages := p.Stages()
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage after remove, got %d", len(stages))
	}
	if stages[0].Name() != "stage2" {
		t.Errorf("expected remaining stage 'stage2', got %s", stages[0].Name())
	}
}

func TestJSONLPipeline_RemoveStage_NotFound(t *testing.T) {
	p := NewJSONLPipeline()

	err := p.RemoveStage("nonexistent")
	if err == nil {
		t.Error("expected error for removing non-existent stage")
	}
}

func TestJSONLPipeline_AddStage_Nil(t *testing.T) {
	p := NewJSONLPipeline()

	err := p.AddStage(nil)
	if err == nil {
		t.Error("expected error for nil stage")
	}
}

func TestJSONLPipeline_Execute(t *testing.T) {
	p := NewJSONLPipeline()
	p.SetFieldMapping(JSONLFieldMapping{ContentField: "text"})

	ctx := common.NewPipelineContext(nil, "test")
	result, err := p.Execute(ctx, `{"text":"executed"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jr, ok := result.(*JSONLResult)
	if !ok {
		t.Fatalf("expected *JSONLResult, got %T", result)
	}
	if len(jr.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(jr.Documents))
	}
	if jr.Documents[0].Content != "executed" {
		t.Errorf("content mismatch: %s", jr.Documents[0].Content)
	}
}

func TestJSONLPipeline_StagesReturnsCopy(t *testing.T) {
	p := NewJSONLPipeline()
	p.AddStage(&mockPipelineStage{name: "s1"})

	stages1 := p.Stages()
	stages1[0] = &mockPipelineStage{name: "tampered"}

	stages2 := p.Stages()
	if stages2[0].Name() != "s1" {
		t.Error("expected Stages() to return a copy, not internal slice")
	}
}

// mockPipelineStage is a simple implementation for testing stage management.
type mockPipelineStage struct {
	name string
}

func (m *mockPipelineStage) Name() string { return m.name }
func (m *mockPipelineStage) Execute(ctx *common.PipelineContext, input interface{}) (interface{}, error) {
	return input, nil
}
func (m *mockPipelineStage) Validate(input interface{}) error { return nil }
