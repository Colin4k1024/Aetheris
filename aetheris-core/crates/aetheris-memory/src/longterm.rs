//! Long-term store — durable across sessions (persistent, searchable).

use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::RwLock;

use crate::error::MemoryError;
use crate::store::MemoryStore;
use crate::types::{MemoryEntry, MemoryNamespace, MemoryQuery, MemoryQueryResult};

/// In-memory long-term store for testing. Production would use PostgreSQL.
pub struct InMemoryLongTermStore {
    entries: Arc<RwLock<HashMap<String, MemoryEntry>>>,
}

impl InMemoryLongTermStore {
    pub fn new() -> Self {
        Self {
            entries: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}

impl Default for InMemoryLongTermStore {
    fn default() -> Self {
        Self::new()
    }
}

fn entry_key(agent_id: &str, key: &str) -> String {
    format!("{agent_id}:{key}")
}

fn now_ms() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

#[async_trait]
impl MemoryStore for InMemoryLongTermStore {
    async fn store(&self, entry: &MemoryEntry) -> Result<(), MemoryError> {
        let mut entries = self.entries.write().await;
        let mut e = entry.clone();
        e.namespace = MemoryNamespace::LongTerm;
        e.created_at_ms = now_ms();
        e.last_accessed_ms = e.created_at_ms;
        entries.insert(entry_key(&e.agent_id, &e.key), e);
        Ok(())
    }

    async fn get(&self, agent_id: &str, key: &str) -> Result<Option<MemoryEntry>, MemoryError> {
        let mut entries = self.entries.write().await;
        let k = entry_key(agent_id, key);
        if let Some(entry) = entries.get_mut(&k) {
            entry.last_accessed_ms = now_ms();
            entry.access_count += 1;
            Ok(Some(entry.clone()))
        } else {
            Ok(None)
        }
    }

    async fn search(&self, query: &MemoryQuery) -> Result<MemoryQueryResult, MemoryError> {
        let entries = self.entries.read().await;
        let results: Vec<MemoryEntry> = entries
            .values()
            .filter(|e| e.agent_id == query.agent_id)
            .filter(|e| {
                query.query_text.is_empty()
                    || e.key.contains(&query.query_text)
                    || String::from_utf8_lossy(&e.value).contains(&query.query_text)
            })
            .filter(|e| e.relevance_score >= query.min_relevance)
            .cloned()
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
        entries.remove(&entry_key(agent_id, key));
        Ok(())
    }

    async fn list(&self, agent_id: &str) -> Result<Vec<MemoryEntry>, MemoryError> {
        let entries = self.entries.read().await;
        Ok(entries
            .values()
            .filter(|e| e.agent_id == agent_id)
            .cloned()
            .collect())
    }

    async fn clear(&self, agent_id: &str) -> Result<(), MemoryError> {
        let mut entries = self.entries.write().await;
        entries.retain(|_, e| e.agent_id != agent_id);
        Ok(())
    }
}
