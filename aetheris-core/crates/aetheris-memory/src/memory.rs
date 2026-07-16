//! Memory manager — coordinates all 4 memory tiers.

use crate::error::MemoryError;
use crate::store::MemoryStore;
use crate::types::{MemoryEntry, MemoryNamespace, MemoryQuery, MemoryQueryResult};

/// Memory manager that coordinates short-term, working, episodic, and long-term memory.
pub struct MemoryManager {
    short_term: Box<dyn MemoryStore>,
    working: Box<dyn MemoryStore>,
    episodic: Box<dyn MemoryStore>,
    long_term: Box<dyn MemoryStore>,
}

impl MemoryManager {
    pub fn new(
        short_term: Box<dyn MemoryStore>,
        working: Box<dyn MemoryStore>,
        episodic: Box<dyn MemoryStore>,
        long_term: Box<dyn MemoryStore>,
    ) -> Self {
        Self {
            short_term,
            working,
            episodic,
            long_term,
        }
    }

    /// Get the appropriate store for a namespace.
    fn store_for(&self, namespace: &MemoryNamespace) -> &dyn MemoryStore {
        match namespace {
            MemoryNamespace::ShortTerm => self.short_term.as_ref(),
            MemoryNamespace::Working => self.working.as_ref(),
            MemoryNamespace::Episodic => self.episodic.as_ref(),
            MemoryNamespace::LongTerm => self.long_term.as_ref(),
        }
    }

    /// Store a memory entry in the appropriate tier.
    pub async fn store(&self, entry: &MemoryEntry) -> Result<(), MemoryError> {
        self.store_for(&entry.namespace).store(entry).await
    }

    /// Get a memory from a specific namespace.
    pub async fn get(
        &self,
        namespace: &MemoryNamespace,
        agent_id: &str,
        key: &str,
    ) -> Result<Option<MemoryEntry>, MemoryError> {
        self.store_for(namespace).get(agent_id, key).await
    }

    /// Search across all tiers (or a specific one).
    pub async fn search(&self, query: &MemoryQuery) -> Result<MemoryQueryResult, MemoryError> {
        match &query.namespace {
            Some(ns) => self.store_for(ns).search(query).await,
            None => {
                // Search all tiers
                let mut all_entries = Vec::new();
                for ns in &[
                    MemoryNamespace::ShortTerm,
                    MemoryNamespace::Working,
                    MemoryNamespace::Episodic,
                    MemoryNamespace::LongTerm,
                ] {
                    let mut q = query.clone();
                    q.namespace = Some(ns.clone());
                    let result = self.store_for(ns).search(&q).await?;
                    all_entries.extend(result.entries);
                }

                // Sort by relevance, limit results
                all_entries.sort_by(|a, b| {
                    b.relevance_score
                        .partial_cmp(&a.relevance_score)
                        .unwrap_or(std::cmp::Ordering::Equal)
                });
                let total = all_entries.len();
                all_entries.truncate(query.max_results);

                Ok(MemoryQueryResult {
                    entries: all_entries,
                    total_count: total,
                })
            }
        }
    }

    /// Delete from a specific namespace.
    pub async fn delete(
        &self,
        namespace: &MemoryNamespace,
        agent_id: &str,
        key: &str,
    ) -> Result<(), MemoryError> {
        self.store_for(namespace).delete(agent_id, key).await
    }

    /// Clear working memory for an agent (called after job completion).
    pub async fn clear_working(&self, agent_id: &str) -> Result<(), MemoryError> {
        self.working.clear(agent_id).await
    }

    /// Clear short-term memory for an agent (called after session ends).
    pub async fn clear_short_term(&self, agent_id: &str) -> Result<(), MemoryError> {
        self.short_term.clear(agent_id).await
    }

    /// Promote a short-term memory to long-term.
    pub async fn promote_to_long_term(
        &self,
        agent_id: &str,
        key: &str,
    ) -> Result<(), MemoryError> {
        let entry = self
            .short_term
            .get(agent_id, key)
            .await?
            .ok_or_else(|| MemoryError::NotFound(key.to_string()))?;

        let mut promoted = entry.clone();
        promoted.namespace = MemoryNamespace::LongTerm;
        self.long_term.store(&promoted).await?;
        self.short_term.delete(agent_id, key).await?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::episodic::InMemoryEpisodicStore;
    use crate::longterm::InMemoryLongTermStore;
    use crate::shortterm::ShortTermMemory;
    use crate::working::WorkingMemory;

    fn make_entry(ns: MemoryNamespace, key: &str, value: &str) -> MemoryEntry {
        MemoryEntry {
            id: format!("id-{key}"),
            agent_id: "agent-1".to_string(),
            namespace: ns,
            key: key.to_string(),
            value: value.as_bytes().to_vec(),
            relevance_score: 1.0,
            created_at_ms: 0,
            last_accessed_ms: 0,
            access_count: 0,
            metadata: vec![],
        }
    }

    fn make_manager() -> MemoryManager {
        MemoryManager::new(
            Box::new(ShortTermMemory::new(100)),
            Box::new(WorkingMemory::new()),
            Box::new(InMemoryEpisodicStore::new()),
            Box::new(InMemoryLongTermStore::new()),
        )
    }

    #[tokio::test]
    async fn test_store_and_get_all_tiers() {
        let mgr = make_manager();

        mgr.store(&make_entry(MemoryNamespace::ShortTerm, "k1", "v1"))
            .await
            .unwrap();
        mgr.store(&make_entry(MemoryNamespace::Working, "k2", "v2"))
            .await
            .unwrap();
        mgr.store(&make_entry(MemoryNamespace::Episodic, "k3", "v3"))
            .await
            .unwrap();
        mgr.store(&make_entry(MemoryNamespace::LongTerm, "k4", "v4"))
            .await
            .unwrap();

        assert!(mgr
            .get(&MemoryNamespace::ShortTerm, "agent-1", "k1")
            .await
            .unwrap()
            .is_some());
        assert!(mgr
            .get(&MemoryNamespace::Working, "agent-1", "k2")
            .await
            .unwrap()
            .is_some());
        assert!(mgr
            .get(&MemoryNamespace::Episodic, "agent-1", "k3")
            .await
            .unwrap()
            .is_some());
        assert!(mgr
            .get(&MemoryNamespace::LongTerm, "agent-1", "k4")
            .await
            .unwrap()
            .is_some());
    }

    #[tokio::test]
    async fn test_search_across_tiers() {
        let mgr = make_manager();

        mgr.store(&make_entry(MemoryNamespace::ShortTerm, "user_pref", "dark_mode"))
            .await
            .unwrap();
        mgr.store(&make_entry(MemoryNamespace::LongTerm, "user_name", "alice"))
            .await
            .unwrap();

        let result = mgr
            .search(&MemoryQuery {
                agent_id: "agent-1".to_string(),
                namespace: None,
                query_text: "user".to_string(),
                max_results: 10,
                min_relevance: 0.0,
            })
            .await
            .unwrap();

        assert_eq!(result.total_count, 2);
    }

    #[tokio::test]
    async fn test_promote_to_long_term() {
        let mgr = make_manager();

        mgr.store(&make_entry(MemoryNamespace::ShortTerm, "important", "data"))
            .await
            .unwrap();

        mgr.promote_to_long_term("agent-1", "important")
            .await
            .unwrap();

        // Should be gone from short-term
        assert!(mgr
            .get(&MemoryNamespace::ShortTerm, "agent-1", "important")
            .await
            .unwrap()
            .is_none());

        // Should be in long-term
        assert!(mgr
            .get(&MemoryNamespace::LongTerm, "agent-1", "important")
            .await
            .unwrap()
            .is_some());
    }

    #[tokio::test]
    async fn test_clear_working() {
        let mgr = make_manager();

        mgr.store(&make_entry(MemoryNamespace::Working, "temp", "data"))
            .await
            .unwrap();

        mgr.clear_working("agent-1").await.unwrap();

        assert!(mgr
            .get(&MemoryNamespace::Working, "agent-1", "temp")
            .await
            .unwrap()
            .is_none());
    }
}
