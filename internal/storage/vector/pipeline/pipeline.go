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

// Package pipeline provides memory pipeline: decay, compression, vectorization, and recall
package pipeline

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/storage/vector"
)

// Embedder interface for generating embeddings
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	Dimension() int
	Model() string
}

// VectorMemoryItem represents a single memory item with vector representation
type VectorMemoryItem struct {
	ID        string
	Content   string
	Weight    float64 // importance weight (0.0-1.0)
	CreatedAt time.Time
	UpdatedAt time.Time
	Metadata  map[string]any
	VectorID  string // reference to vector store
	Vector    []float64
}

// MemoryPipelineConfig configuration for memory pipeline
type MemoryPipelineConfig struct {
	// Decay config
	DecayFactor   float64       // weight decay per decay interval (default 0.95)
	DecayInterval time.Duration // how often to apply decay (default 1 hour)
	MinWeight     float64       // minimum weight threshold for removal (default 0.1)

	// Compression config
	SimilarityThreshold float64 // threshold for merging similar memories (default 0.85)
	MaxGroupSize        int     // max items to merge in one group (default 5)

	// Embedding config
	BatchSize int // batch size for bulk embedding (default 32)

	// Vector store config
	VectorIndexName string // name of vector index (default "memory")

	// Maintenance config
	MaintenanceInterval time.Duration // how often to run maintenance (default 1 hour)
}

// DefaultMemoryPipelineConfig returns default configuration
func DefaultMemoryPipelineConfig() MemoryPipelineConfig {
	return MemoryPipelineConfig{
		DecayFactor:         0.95,
		DecayInterval:       time.Hour,
		MinWeight:           0.1,
		SimilarityThreshold: 0.85,
		MaxGroupSize:        5,
		BatchSize:           32,
		VectorIndexName:     "memory",
		MaintenanceInterval: time.Hour,
	}
}

// MemoryPipeline memory pipeline with decay, compression, vectorization, and recall
type MemoryPipeline struct {
	config    MemoryPipelineConfig
	embedder  Embedder
	store     vector.Store
	mu        sync.RWMutex
	items     map[string]*VectorMemoryItem // in-memory index for metadata
	lastDecay time.Time
}

// NewMemoryPipeline creates a new memory pipeline
func NewMemoryPipeline(config MemoryPipelineConfig, embedder Embedder, store vector.Store) (*MemoryPipeline, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	if store == nil {
		return nil, fmt.Errorf("vector store is required")
	}

	// Set defaults
	if config.DecayFactor <= 0 || config.DecayFactor > 1.0 {
		config.DecayFactor = 0.95
	}
	if config.DecayInterval <= 0 {
		config.DecayInterval = time.Hour
	}
	if config.MinWeight <= 0 {
		config.MinWeight = 0.1
	}
	if config.SimilarityThreshold <= 0 || config.SimilarityThreshold > 1.0 {
		config.SimilarityThreshold = 0.85
	}
	if config.MaxGroupSize <= 0 {
		config.MaxGroupSize = 5
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 32
	}
	if config.VectorIndexName == "" {
		config.VectorIndexName = "memory"
	}
	if config.MaintenanceInterval <= 0 {
		config.MaintenanceInterval = time.Hour
	}

	p := &MemoryPipeline{
		config:    config,
		embedder:  embedder,
		store:     store,
		items:     make(map[string]*VectorMemoryItem),
		lastDecay: time.Now(),
	}

	// Create vector index
	ctx := context.Background()
	index := &vector.Index{
		Name:      config.VectorIndexName,
		Dimension: embedder.Dimension(),
		Distance:  "cosine",
	}
	if err := store.Create(ctx, index); err != nil {
		// Index might already exist, that's OK
		// Log but don't fail
		log.Printf("warning: failed to create vector index (may already exist): %v", err)
	}

	return p, nil
}

// Store stores a new memory item with vectorization
func (p *MemoryPipeline) Store(ctx context.Context, item VectorMemoryItem) error {
	if item.ID == "" {
		item.ID = fmt.Sprintf("mem_%d", time.Now().UnixNano())
	}
	if item.Content == "" {
		return fmt.Errorf("content is required")
	}
	if item.Weight <= 0 {
		item.Weight = 1.0
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()

	// Generate embedding
	embeddings, err := p.embedder.Embed(ctx, []string{item.Content})
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	if len(embeddings) == 0 {
		return fmt.Errorf("no embedding generated")
	}

	item.Vector = embeddings[0]
	item.VectorID = fmt.Sprintf("vec_%s", item.ID)

	// Store vector in vector store
	p.mu.Lock()
	defer p.mu.Unlock()

	vec := &vector.Vector{
		ID:       item.VectorID,
		Values:   item.Vector,
		Metadata: p.itemToMetadata(item),
	}

	if err := p.store.Add(ctx, p.config.VectorIndexName, []*vector.Vector{vec}); err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

	// Store in memory index
	p.items[item.ID] = &item

	return nil
}

// Recall retrieves relevant memories based on query
func (p *MemoryPipeline) Recall(ctx context.Context, query string, topK int) ([]VectorMemoryItem, error) {
	if topK <= 0 {
		topK = 10
	}

	// Generate query embedding
	embeddings, err := p.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no query embedding generated")
	}

	queryVec := embeddings[0]

	// Search vector store
	results, err := p.store.Search(ctx, p.config.VectorIndexName, queryVec, &vector.SearchOptions{
		TopK:           topK,
		Threshold:      p.config.MinWeight,
		IncludeVectors: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// Convert results to memory items
	p.mu.RLock()
	defer p.mu.RUnlock()

	var items []VectorMemoryItem
	for _, result := range results {
		// Find the memory item by metadata
		if memID, exists := result.Metadata["memory_id"]; exists {
			if item, exists := p.items[memID]; exists {
				// Filter by weight
				if item.Weight >= p.config.MinWeight {
					items = append(items, *item)
				}
			}
		}
	}

	return items, nil
}

// Decay applies weight decay to old memories
func (p *MemoryPipeline) Decay(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	if now.Sub(p.lastDecay) < p.config.DecayInterval {
		return nil // Too soon to decay
	}

	p.lastDecay = now

	var toDelete []string
	var toUpdate []*VectorMemoryItem

	for id, item := range p.items {
		// Apply decay
		newWeight := item.Weight * p.config.DecayFactor
		item.Weight = newWeight
		item.UpdatedAt = now

		// Check if below threshold
		if newWeight < p.config.MinWeight {
			toDelete = append(toDelete, id)
		} else {
			toUpdate = append(toUpdate, item)
		}
	}

	// Delete items below threshold
	for _, id := range toDelete {
		if err := p.store.Delete(ctx, p.config.VectorIndexName, p.items[id].VectorID); err != nil {
			// Log but continue
			log.Printf("warning: failed to delete vector for item %s during decay: %v", id, err)
		}
		delete(p.items, id)
	}

	// Update metadata for remaining items
	// Note: In production, we'd batch this
	for _, item := range toUpdate {
		vec := &vector.Vector{
			ID:       item.VectorID,
			Values:   item.Vector,
			Metadata: p.itemToMetadata(*item),
		}
		// Delete and re-add with updated metadata
		_ = p.store.Delete(ctx, p.config.VectorIndexName, item.VectorID)
		_ = p.store.Add(ctx, p.config.VectorIndexName, []*vector.Vector{vec})
	}

	return nil
}

// Compress merges similar memories
func (p *MemoryPipeline) Compress(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.items) < 2 {
		return nil
	}

	// Get all items as slice
	var items []*VectorMemoryItem
	for _, item := range p.items {
		items = append(items, item)
	}

	// Group similar items
	var merged []string
	used := make(map[string]bool)

	for i, item := range items {
		if used[item.ID] {
			continue
		}

		// Find similar items
		var group []*VectorMemoryItem
		group = append(group, item)

		for j := i + 1; j < len(items); j++ {
			if used[items[j].ID] {
				continue
			}

			sim := cosineSimilarity(item.Vector, items[j].Vector)
			if sim >= p.config.SimilarityThreshold {
				group = append(group, items[j])
				if len(group) >= p.config.MaxGroupSize {
					break
				}
			}
		}

		// Merge group if more than 1 item
		if len(group) > 1 {
			mergedItem := p.mergeGroup(group)
			p.items[mergedItem.ID] = mergedItem

			// Mark items as used
			for _, g := range group {
				used[g.ID] = true
				merged = append(merged, g.ID)
			}
		}
	}

	// Delete merged items from vector store
	for _, id := range merged {
		if item, exists := p.items[id]; exists {
			_ = p.store.Delete(ctx, p.config.VectorIndexName, item.VectorID)
		}
		delete(p.items, id)
	}

	return nil
}

// Maintenance runs periodic maintenance: decay and compression
func (p *MemoryPipeline) Maintenance(ctx context.Context) error {
	if err := p.Decay(ctx); err != nil {
		return fmt.Errorf("decay failed: %w", err)
	}
	if err := p.Compress(ctx); err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}
	return nil
}

// GetStats returns pipeline statistics
func (p *MemoryPipeline) GetStats() (totalItems int, avgWeight float64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalItems = len(p.items)
	if totalItems == 0 {
		return 0, 0
	}

	var totalWeight float64
	for _, item := range p.items {
		totalWeight += item.Weight
	}

	return totalItems, totalWeight / float64(totalItems)
}

// Close closes the pipeline and underlying store
func (p *MemoryPipeline) Close() error {
	return p.store.Close()
}

// mergeGroup merges a group of similar memories
func (p *MemoryPipeline) mergeGroup(group []*VectorMemoryItem) *VectorMemoryItem {
	// Sort by timestamp (most recent first)
	sort.Slice(group, func(i, j int) bool {
		return group[i].CreatedAt.After(group[j].CreatedAt)
	})

	// Use most recent item as base
	merged := *group[0]

	// Ensure Metadata is initialized
	if merged.Metadata == nil {
		merged.Metadata = make(map[string]any)
	}

	// Combine content (keep unique sentences)
	contents := make(map[string]bool)
	for _, item := range group {
		contents[item.Content] = true
	}

	// Use first content but note that there are merged items
	merged.Metadata["merged_count"] = len(group)
	merged.Metadata["merged_from"] = func() []string {
		ids := make([]string, len(group))
		for i, item := range group {
			ids[i] = item.ID
		}
		return ids
	}()

	// Average the vectors
	if len(group) > 1 {
		merged.Vector = make([]float64, len(merged.Vector))
		for _, item := range group {
			for i, v := range item.Vector {
				merged.Vector[i] += v
			}
		}
		for i := range merged.Vector {
			merged.Vector[i] /= float64(len(group))
		}
	}

	// Weight is average
	var totalWeight float64
	for _, item := range group {
		totalWeight += item.Weight
	}
	merged.Weight = totalWeight / float64(len(group))

	return &merged
}

// itemToMetadata converts VectorMemoryItem to metadata map
func (p *MemoryPipeline) itemToMetadata(item VectorMemoryItem) map[string]string {
	metadata := map[string]string{
		"memory_id":  item.ID,
		"content":    item.Content,
		"weight":     fmt.Sprintf("%.4f", item.Weight),
		"created_at": item.CreatedAt.Format(time.RFC3339),
		"updated_at": item.UpdatedAt.Format(time.RFC3339),
	}
	for k, v := range item.Metadata {
		metadata[k] = fmt.Sprintf("%v", v)
	}
	return metadata
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
