//! In-memory effect store for testing.

use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::RwLock;
use uuid::Uuid;

use crate::error::EffectError;
use crate::types::{EffectEntry, EffectRecord, EffectStatus};
use crate::EffectStore;

fn now_ms() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

/// In-memory effect store for unit tests.
pub struct MemoryEffectLedger {
    effects: Arc<RwLock<HashMap<String, EffectRecord>>>,
}

impl MemoryEffectLedger {
    pub fn new() -> Self {
        Self {
            effects: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}

impl Default for MemoryEffectLedger {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl EffectStore for MemoryEffectLedger {
    async fn record_pending(
        &self,
        job_id: &str,
        attempt_id: &str,
        effect: &EffectEntry,
    ) -> Result<String, EffectError> {
        // Check idempotency
        {
            let effects = self.effects.read().await;
            for record in effects.values() {
                if record.job_id == job_id
                    && record.idempotency_key == effect.idempotency_key
                    && record.status == EffectStatus::Committed
                {
                    return Err(EffectError::IdempotencyConflict(format!(
                        "effect with key {} already committed",
                        effect.idempotency_key
                    )));
                }
            }
        }

        let id = Uuid::new_v4().to_string();
        let record = EffectRecord {
            id: id.clone(),
            job_id: job_id.to_string(),
            attempt_id: attempt_id.to_string(),
            kind: effect.kind.clone(),
            input: effect.input.clone(),
            output: Vec::new(),
            status: EffectStatus::Pending,
            idempotency_key: effect.idempotency_key.clone(),
            error_message: String::new(),
            created_at_ms: now_ms(),
            committed_at_ms: None,
        };

        let mut effects = self.effects.write().await;
        effects.insert(id.clone(), record);
        Ok(id)
    }

    async fn confirm(&self, effect_id: &str, output: &[u8]) -> Result<(), EffectError> {
        let mut effects = self.effects.write().await;
        let record = effects
            .get_mut(effect_id)
            .ok_or_else(|| EffectError::NotFound(effect_id.to_string()))?;

        match record.status {
            EffectStatus::Committed => {
                return Err(EffectError::AlreadyCommitted(effect_id.to_string()))
            }
            EffectStatus::RolledBack => {
                return Err(EffectError::AlreadyRolledBack(effect_id.to_string()))
            }
            EffectStatus::Pending => {}
        }

        record.status = EffectStatus::Committed;
        record.output = output.to_vec();
        record.committed_at_ms = Some(now_ms());
        Ok(())
    }

    async fn rollback(&self, effect_id: &str, reason: &str) -> Result<(), EffectError> {
        let mut effects = self.effects.write().await;
        let record = effects
            .get_mut(effect_id)
            .ok_or_else(|| EffectError::NotFound(effect_id.to_string()))?;

        match record.status {
            EffectStatus::Committed => {
                return Err(EffectError::AlreadyCommitted(effect_id.to_string()))
            }
            EffectStatus::RolledBack => {
                return Err(EffectError::AlreadyRolledBack(effect_id.to_string()))
            }
            EffectStatus::Pending => {}
        }

        record.status = EffectStatus::RolledBack;
        record.error_message = reason.to_string();
        Ok(())
    }

    async fn list_by_job(&self, job_id: &str) -> Result<Vec<EffectRecord>, EffectError> {
        let effects = self.effects.read().await;
        Ok(effects
            .values()
            .filter(|r| r.job_id == job_id)
            .cloned()
            .collect())
    }

    async fn get(&self, effect_id: &str) -> Result<Option<EffectRecord>, EffectError> {
        let effects = self.effects.read().await;
        Ok(effects.get(effect_id).cloned())
    }

    async fn exists_by_idempotency_key(
        &self,
        job_id: &str,
        key: &str,
    ) -> Result<Option<EffectRecord>, EffectError> {
        let effects = self.effects.read().await;
        Ok(effects
            .values()
            .find(|r| r.job_id == job_id && r.idempotency_key == key)
            .cloned())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_effect(kind: &str, key: &str) -> EffectEntry {
        EffectEntry {
            kind: kind.to_string(),
            input: b"test-input".to_vec(),
            idempotency_key: key.to_string(),
        }
    }

    #[tokio::test]
    async fn test_two_phase_commit() {
        let store = MemoryEffectLedger::new();

        // Phase 1: record pending
        let id = store
            .record_pending("job-1", "attempt-1", &make_effect("http_call", "key-1"))
            .await
            .unwrap();

        let record = store.get(&id).await.unwrap().unwrap();
        assert_eq!(record.status, EffectStatus::Pending);

        // Phase 2: confirm
        store.confirm(&id, b"output-data").await.unwrap();

        let record = store.get(&id).await.unwrap().unwrap();
        assert_eq!(record.status, EffectStatus::Committed);
        assert_eq!(record.output, b"output-data");
    }

    #[tokio::test]
    async fn test_rollback() {
        let store = MemoryEffectLedger::new();

        let id = store
            .record_pending("job-1", "attempt-1", &make_effect("http_call", "key-1"))
            .await
            .unwrap();

        store.rollback(&id, "timeout").await.unwrap();

        let record = store.get(&id).await.unwrap().unwrap();
        assert_eq!(record.status, EffectStatus::RolledBack);
        assert_eq!(record.error_message, "timeout");
    }

    #[tokio::test]
    async fn test_idempotency_conflict() {
        let store = MemoryEffectLedger::new();

        // First call — pending
        let id = store
            .record_pending("job-1", "attempt-1", &make_effect("http_call", "key-1"))
            .await
            .unwrap();

        // Commit it
        store.confirm(&id, b"done").await.unwrap();

        // Second call with same idempotency key — should conflict
        let result = store
            .record_pending("job-1", "attempt-1", &make_effect("http_call", "key-1"))
            .await;
        assert!(matches!(result, Err(EffectError::IdempotencyConflict(_))));
    }

    #[tokio::test]
    async fn test_double_confirm_fails() {
        let store = MemoryEffectLedger::new();

        let id = store
            .record_pending("job-1", "attempt-1", &make_effect("http_call", "key-1"))
            .await
            .unwrap();

        store.confirm(&id, b"done").await.unwrap();

        let result = store.confirm(&id, b"done-again").await;
        assert!(matches!(result, Err(EffectError::AlreadyCommitted(_))));
    }
}
