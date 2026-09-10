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
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

// HiveQueryClient abstracts a Hive query execution client.
// Implementations may use HiveServer2 Thrift, JDBC, or a REST API.
type HiveQueryClient interface {
	// Query executes a HiveQL query and returns rows as a slice of maps.
	// Each map maps column name → value (as interface{}).
	Query(ctx context.Context, query string) ([]map[string]interface{}, error)
	// Close releases any resources held by the client.
	Close() error
}

// HiveSyncProgressStore persists sync cursor and state between runs.
type HiveSyncProgressStore interface {
	// GetCursor returns the last sync cursor (e.g., max primary key or timestamp) for the given sync key.
	GetCursor(syncKey string) (string, error)
	// SaveCursor persists the sync cursor for the given sync key.
	SaveCursor(syncKey string, cursor string) error
}

// HiveConfig configures the Hive pipeline's query and sync behavior.
type HiveConfig struct {
	// QueryTemplate is the HiveQL template with optional {cursor} placeholder for incremental sync.
	// Example: SELECT * FROM events WHERE id > {cursor} ORDER BY id LIMIT 1000
	QueryTemplate string
	// CursorField is the column name used for incremental sync (e.g., "id" or "created_at").
	// If empty, incremental sync is disabled (full load each time).
	CursorField string
	// PageSize limits the number of rows per query batch.
	// If 0, no pagination is applied.
	PageSize int
	// SyncKey identifies the sync job for progress persistence (e.g., "events_daily_sync").
	SyncKey string
	// ContentField maps a column to the document Content field.
	ContentField string
	// IDField maps a column to the document ID field.
	// If empty, a UUID is generated.
	IDField string
	// MetadataFields maps column names to metadata keys.
	// If empty, all columns are copied to metadata.
	MetadataFields map[string]string
}

// HiveResult is the output of ProcessSpecialized.
type HiveResult struct {
	Documents  []*common.Document
	LastCursor string
	TotalRows  int
}

// HIVEPipeline processes data from a Hive data source.
// It supports incremental sync via cursor, pagination, and field mapping.
type HIVEPipeline struct {
	name     string
	mu       sync.Mutex
	stages   []common.PipelineStage
	client   HiveQueryClient
	config   HiveConfig
	progress HiveSyncProgressStore
}

// NewHIVEPipeline creates a HIVE Pipeline.
func NewHIVEPipeline() *HIVEPipeline {
	return &HIVEPipeline{name: "hive"}
}

// SetClient injects the Hive query client.
func (p *HIVEPipeline) SetClient(client HiveQueryClient) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.client = client
}

// SetConfig configures the pipeline's query and sync parameters.
func (p *HIVEPipeline) SetConfig(config HiveConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
}

// SetProgressStore injects the sync progress persistence store.
func (p *HIVEPipeline) SetProgressStore(store HiveSyncProgressStore) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress = store
}

// Name implements Pipeline
func (p *HIVEPipeline) Name() string {
	return p.name
}

// Stages implements Pipeline
func (p *HIVEPipeline) Stages() []common.PipelineStage {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]common.PipelineStage, len(p.stages))
	copy(result, p.stages)
	return result
}

// Execute implements Pipeline
func (p *HIVEPipeline) Execute(ctx *common.PipelineContext, input interface{}) (interface{}, error) {
	return p.ProcessSpecialized(input)
}

// AddStage implements Pipeline
func (p *HIVEPipeline) AddStage(stage common.PipelineStage) error {
	if stage == nil {
		return fmt.Errorf("stage cannot be nil")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stages = append(p.stages, stage)
	return nil
}

// RemoveStage implements Pipeline
func (p *HIVEPipeline) RemoveStage(name string) error {
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
// Input should be a context.Context for cancellation control.
// Returns *HiveResult with the queried documents and sync cursor.
func (p *HIVEPipeline) ProcessSpecialized(input interface{}) (interface{}, error) {
	p.mu.Lock()
	client := p.client
	config := p.config
	progress := p.progress
	p.mu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("Hive query client not configured")
	}
	if config.QueryTemplate == "" {
		return nil, fmt.Errorf("Hive query template not configured")
	}

	ctx, ok := input.(context.Context)
	if !ok {
		ctx = context.Background()
	}

	// Load last cursor for incremental sync
	var lastCursor string
	if config.CursorField != "" && progress != nil {
		cursor, err := progress.GetCursor(config.SyncKey)
		if err != nil {
			return nil, fmt.Errorf("load sync cursor: %w", err)
		}
		lastCursor = cursor
	}

	result := &HiveResult{
		Documents: make([]*common.Document, 0),
	}

	// Execute query with cursor substitution
	query := config.QueryTemplate
	if config.CursorField != "" && lastCursor != "" {
		query = substituteCursor(config.QueryTemplate, lastCursor)
	}

	rows, err := client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("Hive query failed: %w", err)
	}

	// Apply pagination
	if config.PageSize > 0 && len(rows) > config.PageSize {
		rows = rows[:config.PageSize]
	}

	for i, row := range rows {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		doc := mapRowToDocument(row, config, i)

		// Track new cursor
		if config.CursorField != "" {
			if cursorVal, ok := row[config.CursorField]; ok {
				result.LastCursor = fmt.Sprintf("%v", cursorVal)
			}
		}

		result.Documents = append(result.Documents, doc)
	}
	result.TotalRows = len(result.Documents)

	// Persist new cursor
	if config.CursorField != "" && result.LastCursor != "" && progress != nil {
		if err := progress.SaveCursor(config.SyncKey, result.LastCursor); err != nil {
			return nil, fmt.Errorf("save sync cursor: %w", err)
		}
	}

	return result, nil
}

// substituteCursor replaces the {cursor} placeholder in the query template.
func substituteCursor(template, cursor string) string {
	// Simple substitution — production code would use parameterized queries
	result := ""
	for i := 0; i < len(template); i++ {
		if i+7 < len(template) && template[i:i+8] == "{cursor}" {
			result += "'" + cursor + "'"
			i += 7
		} else {
			result += string(template[i])
		}
	}
	return result
}

// mapRowToDocument converts a Hive row (map of column→value) to a Document.
func mapRowToDocument(row map[string]interface{}, config HiveConfig, rowIndex int) *common.Document {
	doc := &common.Document{
		ID:        uuid.New().String(),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if config.IDField != "" {
		if idVal, ok := row[config.IDField]; ok {
			doc.ID = fmt.Sprintf("%v", idVal)
		}
	}

	if config.ContentField != "" {
		if contentVal, ok := row[config.ContentField]; ok {
			doc.Content = fmt.Sprintf("%v", contentVal)
		}
	} else {
		// Use all columns as content (concatenated)
		content := ""
		for k, v := range row {
			if content != "" {
				content += " "
			}
			content += fmt.Sprintf("%s=%v", k, v)
		}
		doc.Content = content
	}

	if len(config.MetadataFields) > 0 {
		for colName, metaKey := range config.MetadataFields {
			if val, ok := row[colName]; ok {
				doc.Metadata[metaKey] = val
			}
		}
	} else {
		for k, v := range row {
			if k == config.ContentField || k == config.IDField {
				continue
			}
			doc.Metadata[k] = v
		}
	}

	doc.Metadata["row_index"] = rowIndex
	doc.Metadata["hive_pipeline"] = "hive"

	return doc
}
