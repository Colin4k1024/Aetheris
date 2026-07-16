//! Memory types.

use serde::{Deserialize, Serialize};

/// Memory namespace categorizes entries by scope.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum MemoryNamespace {
    /// Current session scope — ephemeral.
    ShortTerm,
    /// Current job scope — cleared after job completion.
    Working,
    /// Session/job summaries — append-only, persistent.
    Episodic,
    /// Durable across sessions — searchable, persistent.
    LongTerm,
}

/// A single memory entry.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MemoryEntry {
    pub id: String,
    pub agent_id: String,
    pub namespace: MemoryNamespace,
    pub key: String,
    pub value: Vec<u8>,
    pub relevance_score: f32,
    pub created_at_ms: i64,
    pub last_accessed_ms: i64,
    pub access_count: u32,
    pub metadata: Vec<u8>,
}

/// Query for searching memories.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MemoryQuery {
    pub agent_id: String,
    pub namespace: Option<MemoryNamespace>,
    pub query_text: String,
    pub max_results: usize,
    pub min_relevance: f32,
}

/// Result of a memory search.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MemoryQueryResult {
    pub entries: Vec<MemoryEntry>,
    pub total_count: usize,
}
