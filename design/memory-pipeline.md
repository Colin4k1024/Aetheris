# Memory Pipeline Design - EPIC 4

## Overview
Long-running agent tasks require persistent memory with intelligent management: decay old info, compress redundant content, and enable semantic retrieval via vector search.

## Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Memory Pipeline                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   Ingest     │───▶│   Decay      │───▶│   Compression        │  │
│  │  (new items) │    │  (decay old) │    │  (merge redundant)   │  │
│  └──────────────┘    └──────────────┘    └──────────┬───────────┘  │
│                                                      │               │
│                                                      ▼               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   Recall     │◀───│   Vector     │◀───│   Embedding          │  │
│  │ (retrieve)   │    │   Search     │    │   (generate)        │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Memory Decay (衰减)
- **Purpose**: Reduce importance/weight of old memories over time
- **Implementation**: 
  - Each memory item has a `weight` field (0.0 - 1.0)
  - On each access, decay the weight: `weight = weight * decay_factor` (e.g., 0.95)
  - Periodically remove items below threshold
- **Config**:
  - `decay_factor`: 0.0-1.0 (default 0.95)
  - `decay_interval`: how often to decay (default 1 hour)
  - `min_weight`: threshold for removal (default 0.1)

### 2. Memory Compression (压缩)
- **Purpose**: Merge similar/redundant memories to save space
- **Implementation**:
  - Group memories by similarity (cosine similarity > threshold)
  - Merge by combining content and keeping most recent timestamp
  - Update aggregated weight
- **Config**:
  - `similarity_threshold`: 0.8-0.95 (default 0.85)
  - `max_group_size`: max items to merge (default 5)

### 3. Vector Embedding (向量化)
- **Purpose**: Convert memory content to vectors for semantic search
- **Implementation**:
  - Use embedding model (configurable: OpenAI, local, etc.)
  - Store vectors in vector store (MemoryStore, Milvus, Pinecone)
- **Config**:
  - `model`: embedding model name
  - `dimension`: vector dimension (default 1536)
  - `batch_size`: batch for bulk embedding (default 32)

### 4. Vector Store Adapters (向量库适配)
- **MemoryStore**: In-memory (for testing/dev)
- **Milvus**: Production-scale vector DB
- **Pinecone**: Cloud-managed vector DB

### 5. Recall Mechanism (召回)
- **Purpose**: Retrieve relevant memories based on query
- **Flow**:
  1. Embed query using same embedding model
  2. Search vector store with query vector
  3. Filter by weight threshold
  4. Return top-K results
  5. Re-inject into agent context

## Data Structures

### VectorMemoryItem
```go
type VectorMemoryItem struct {
    ID        string    // unique identifier
    Content   string    // memory text
    Weight    float64   // importance weight (0.0-1.0)
    CreatedAt time.Time
    UpdatedAt time.Time
    Metadata  map[string]any
    VectorID  string    // reference to vector store
}
```

### MemoryPipelineConfig
```go
type MemoryPipelineConfig struct {
    // Decay config
    DecayFactor    float64
    DecayInterval time.Duration
    MinWeight     float64
    
    // Compression config
    SimilarityThreshold float64
    MaxGroupSize        int
    
    // Embedding config
    EmbeddingModel  string
    EmbeddingDim    int
    BatchSize      int
    
    // Vector store config
    VectorStoreType string  // "memory", "milvus", "pinecone"
    VectorStoreURI string
}
```

## Usage Example

```go
// Create pipeline
config := MemoryPipelineConfig{
    DecayFactor:          0.95,
    DecayInterval:        time.Hour,
    MinWeight:            0.1,
    SimilarityThreshold:  0.85,
    EmbeddingModel:       "text-embedding-3-small",
    EmbeddingDim:         1536,
    VectorStoreType:      "memory",
}
pipeline := NewMemoryPipeline(config, embedder, vectorStore)

// Store new memory
item := VectorMemoryItem{
    Content: "User prefers dark mode",
    Weight:  1.0,
}
await pipeline.Store(ctx, item)

// Recall relevant memories
results, err := pipeline.Recall(ctx, "theme preferences")
// results contain top-K similar memories

// Periodic maintenance (decay + compress)
await pipeline.Maintenance(ctx)
```

## Milvus/Pinecone Adapter Notes

### Milvus
- Requires milvus service running
- Collection-based storage
- Support for filtered search
- Good for on-premise deployments

### Pinecone
- Cloud-managed, serverless option
- Simple API for upsert/search
- Good for production with minimal ops

## Testing Strategy
1. Unit tests for decay logic
2. Unit tests for compression logic  
3. Integration tests with MemoryStore
4. Mock tests for Milvus/Pinecone adapters
