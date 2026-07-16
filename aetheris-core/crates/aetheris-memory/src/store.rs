//! Memory store trait — common interface for all memory tiers.

use async_trait::async_trait;

use crate::error::MemoryError;
use crate::types::{MemoryEntry, MemoryQuery, MemoryQueryResult};

/// Trait for memory stores at any tier.
#[async_trait]
pub trait MemoryStore: Send + Sync {
    /// Store a memory entry.
    async fn store(&self, entry: &MemoryEntry) -> Result<(), MemoryError>;

    /// Retrieve a memory by key.
    async fn get(&self, agent_id: &str, key: &str) -> Result<Option<MemoryEntry>, MemoryError>;

    /// Search memories by query.
    async fn search(&self, query: &MemoryQuery) -> Result<MemoryQueryResult, MemoryError>;

    /// Delete a memory by key.
    async fn delete(&self, agent_id: &str, key: &str) -> Result<(), MemoryError>;

    /// List all memories for an agent.
    async fn list(&self, agent_id: &str) -> Result<Vec<MemoryEntry>, MemoryError>;

    /// Clear all memories for an agent (used for cleanup).
    async fn clear(&self, agent_id: &str) -> Result<(), MemoryError>;
}
