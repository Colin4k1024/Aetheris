//! Episodic store — session/job summaries (append-only, persistent).

use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::RwLock;

use crate::error::MemoryError;
use crate::store::MemoryStore;
use crate::types::{MemoryEntry, MemoryNamespace, MemoryQuery, MemoryQueryResult};

/// In-memory episodic store for testing. Production would use PostgreSQL.
pub struct InMemoryEpisodicStore {
    entries: Arc<RwLock<HashMap<String, Vec<MemoryEntry>>>>,
}

impl InMemoryEpisodicStore {
    pub fn new() -> Self {
        Self {
            entries: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}

impl Default for InMemoryEpisodicStore {
    fn default() -> Self {
        Self::new()
    }
}

fn now_ms() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

#[async_trait]
impl MemoryStore for InMemoryEpisodicStore {
    async fn store(&self, entry: &MemoryEntry) -> Result<(), MemoryError> {
        let mut entries = self.entries.write().await;
        let mut e = entry.clone();
        e.namespace = MemoryNamespace::Episodic;
        e.created_at_ms = now_ms();
        e.last_accessed_ms = e.created_at_ms;
        entries
            .entry(e.agent_id.clone())
            .or_default()
            .push(e);
        Ok(())
    }

    async fn get(&self, agent_id: &str, key: &str) -> Result<Option<MemoryEntry>, MemoryError> {
        let entries = self.entries.read().await;
        Ok(entries
            .get(agent_id)
            .and_then(|v| v.iter().find(|e| e.key == key).cloned()))
    }

    async fn search(&self, query: &MemoryQuery) -> Result<MemoryQueryResult, MemoryError> {
        let entries = self.entries.read().await;
        let agent_entries = entries.get(&query.agent_id).cloned().unwrap_or_default();

        let results: Vec<MemoryEntry> = agent_entries
            .into_iter()
            .filter(|e| {
                query.query_text.is_empty()
                    || e.key.contains(&query.query_text)
                    || String::from_utf8_lossy(&e.value).contains(&query.query_text)
            })
            .filter(|e| e.relevance_score >= query.min_relevance)
            .collect();

        let total = results.len();
        let limited = results.into_iter().take(query.max_results).collect();

        Ok(MemoryQueryResult {
            entries: limited,
            total_count: total,
        })
    }

    async fn delete(&self, agent_id: &str, key: &str) -> Result<(), MemoryError> {
        let mut entries = self.entries.write().await;
        if let Some(v) = entries.get_mut(agent_id) {
            v.retain(|e| e.key != key);
        }
        Ok(())
    }

    async fn list(&self, agent_id: &str) -> Result<Vec<MemoryEntry>, MemoryError> {
        let entries = self.entries.read().await;
        Ok(entries.get(agent_id).cloned().unwrap_or_default())
    }

    async fn clear(&self, agent_id: &str) -> Result<(), MemoryError> {
        let mut entries = self.entries.write().await;
        entries.remove(agent_id);
        Ok(())
    }
}
