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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

// JSONLErrorRowStrategy defines how to handle lines that fail JSON parsing.
type JSONLErrorRowStrategy string

const (
	// JSONLErrorSkip skips the bad line and continues processing.
	JSONLErrorSkip JSONLErrorRowStrategy = "skip"
	// JSONLErrorFail returns an error immediately on the first bad line.
	JSONLErrorFail JSONLErrorRowStrategy = "fail"
	// JSONLErrorCollect continues processing and returns errors in the result metadata.
	JSONLErrorCollect JSONLErrorRowStrategy = "collect"
)

// JSONLFieldMapping maps JSONL object fields to document content and metadata.
type JSONLFieldMapping struct {
	// ContentField is the JSON key whose value becomes the document Content.
	// If empty, the entire JSON line is marshaled as content.
	ContentField string
	// IDField is the JSON key whose value becomes the document ID.
	// If empty, a UUID is generated.
	IDField string
	// MetadataFields maps JSON keys to metadata keys in the output Document.
	// If empty, all JSON fields not used for Content/ID are copied to metadata.
	MetadataFields map[string]string
}

// JSONLResult is the output of ProcessSpecialized.
type JSONLResult struct {
	Documents []*common.Document
	Errors    []JSONLLineError
}

// JSONLLineError records a parsing failure on a specific line.
type JSONLLineError struct {
	LineNumber int
	Line       string
	Err        error
}

// JSONLPipeline processes JSONL (JSON Lines) format input.
// It streams line-by-line, maps fields to documents, and supports
// configurable error handling strategies.
type JSONLPipeline struct {
	name     string
	mu       sync.Mutex
	stages   []common.PipelineStage
	mapping  JSONLFieldMapping
	strategy JSONLErrorRowStrategy
}

// NewJSONLPipeline creates a JSONL Pipeline with default settings.
func NewJSONLPipeline() *JSONLPipeline {
	return &JSONLPipeline{
		name:     "jsonl",
		strategy: JSONLErrorSkip,
		mapping: JSONLFieldMapping{
			ContentField: "",
		},
	}
}

// SetFieldMapping configures how JSONL fields map to document content and metadata.
func (p *JSONLPipeline) SetFieldMapping(m JSONLFieldMapping) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mapping = m
}

// SetErrorStrategy configures how bad lines are handled.
func (p *JSONLPipeline) SetErrorStrategy(s JSONLErrorRowStrategy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.strategy = s
}

// Name implements Pipeline
func (p *JSONLPipeline) Name() string {
	return p.name
}

// Stages implements Pipeline, returning the current registered stages.
func (p *JSONLPipeline) Stages() []common.PipelineStage {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]common.PipelineStage, len(p.stages))
	copy(result, p.stages)
	return result
}

// Execute implements Pipeline
func (p *JSONLPipeline) Execute(ctx *common.PipelineContext, input interface{}) (interface{}, error) {
	return p.ProcessSpecialized(input)
}

// AddStage implements Pipeline, registering a new stage.
func (p *JSONLPipeline) AddStage(stage common.PipelineStage) error {
	if stage == nil {
		return fmt.Errorf("stage cannot be nil")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stages = append(p.stages, stage)
	return nil
}

// RemoveStage implements Pipeline, removing a stage by name.
func (p *JSONLPipeline) RemoveStage(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, s := range p.stages {
		if s.Name() == name {
			p.stages = append(p.stages[:i], p.stages[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("stage %q not found", name)
}

// ProcessSpecialized implements SpecializedPipeline.
// Input can be: string (JSONL content or file path), []byte, *bufio.Reader, or io.Reader.
// Returns *JSONLResult containing parsed documents and any collected errors.
func (p *JSONLPipeline) ProcessSpecialized(input interface{}) (interface{}, error) {
	reader, baseMeta, err := p.normalizeInput(input)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			_ = closer.Close()
		}
	}()

	result := &JSONLResult{
		Documents: make([]*common.Document, 0),
		Errors:    make([]JSONLLineError, 0),
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // support lines up to 10MB
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		// Skip empty lines
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		// Check for context cancellation via baseMeta if available
		if ctx, ok := baseMeta["__ctx"]; ok {
			if ctxVal, ok2 := ctx.(context.Context); ok2 {
				select {
				case <-ctxVal.Done():
					return nil, ctxVal.Err()
				default:
				}
			}
		}

		doc, lineErr := p.parseLine(line, lineNum, baseMeta)
		if lineErr != nil {
			switch p.strategy {
			case JSONLErrorFail:
				return nil, fmt.Errorf("line %d: %w", lineNum, lineErr)
			case JSONLErrorCollect:
				result.Errors = append(result.Errors, JSONLLineError{
					LineNumber: lineNum,
					Line:       string(line),
					Err:        lineErr,
				})
				continue
			default: // JSONLErrorSkip
				continue
			}
		}
		if doc != nil {
			result.Documents = append(result.Documents, doc)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return result, nil
}

// parseLine parses a single JSONL line into a Document.
func (p *JSONLPipeline) parseLine(line []byte, lineNum int, baseMeta map[string]interface{}) (*common.Document, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(line, &obj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	p.mu.Lock()
	mapping := p.mapping
	p.mu.Unlock()

	doc := &common.Document{
		ID:        uuid.New().String(),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set document ID from mapped field
	if mapping.IDField != "" {
		if idVal, ok := obj[mapping.IDField]; ok {
			doc.ID = fmt.Sprintf("%v", idVal)
		}
	}

	// Set content from mapped field or full JSON
	if mapping.ContentField != "" {
		if contentVal, ok := obj[mapping.ContentField]; ok {
			doc.Content = fmt.Sprintf("%v", contentVal)
		}
	} else {
		// Use full JSON as content
		contentBytes, _ := json.Marshal(obj)
		doc.Content = string(contentBytes)
	}

	// Set metadata
	if len(mapping.MetadataFields) > 0 {
		for jsonKey, metaKey := range mapping.MetadataFields {
			if val, ok := obj[jsonKey]; ok {
				doc.Metadata[metaKey] = val
			}
		}
	} else {
		// Copy all fields not used for content/ID to metadata
		for k, v := range obj {
			if k == mapping.ContentField || k == mapping.IDField {
				continue
			}
			doc.Metadata[k] = v
		}
	}

	// Add source line number
	doc.Metadata["source_line"] = lineNum
	doc.Metadata["jsonl_pipeline"] = p.name

	// Merge base metadata
	for k, v := range baseMeta {
		if k == "__ctx" {
			continue
		}
		if _, exists := doc.Metadata[k]; !exists {
			doc.Metadata[k] = v
		}
	}

	return doc, nil
}

// normalizeInput converts the input to a reader and extracts base metadata.
func (p *JSONLPipeline) normalizeInput(input interface{}) (io.Reader, map[string]interface{}, error) {
	baseMeta := make(map[string]interface{})

	switch v := input.(type) {
	case string:
		// If the string looks like a file path, read from file
		if _, err := os.Stat(v); err == nil {
			f, err := os.Open(v)
			if err != nil {
				return nil, nil, fmt.Errorf("open file %s: %w", v, err)
			}
			baseMeta["source_file"] = v
			return f, baseMeta, nil
		}
		// Otherwise treat as JSONL content
		baseMeta["source_type"] = "inline"
		return strings.NewReader(v), baseMeta, nil

	case []byte:
		baseMeta["source_type"] = "bytes"
		return bytes.NewReader(v), baseMeta, nil

	case *common.Document:
		if v == nil {
			return nil, nil, fmt.Errorf("document is nil")
		}
		baseMeta["source_type"] = "document"
		if v.Metadata != nil {
			for k, val := range v.Metadata {
				baseMeta[k] = val
			}
		}
		return strings.NewReader(v.Content), baseMeta, nil

	case io.Reader:
		baseMeta["source_type"] = "reader"
		return v, baseMeta, nil

	case *bufio.Reader:
		baseMeta["source_type"] = "reader"
		return v, baseMeta, nil

	default:
		return nil, nil, fmt.Errorf("unsupported input type: %T", input)
	}
}
