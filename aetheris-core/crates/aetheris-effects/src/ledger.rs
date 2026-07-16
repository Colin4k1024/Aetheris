//! Effect ledger — higher-level EffectLog implementation.

use async_trait::async_trait;

use crate::error::EffectError;
use crate::types::{EffectEntry, EffectRecord, EffectStatus};
use crate::{EffectLog, EffectStore};

/// Generic effect ledger that wraps any EffectStore.
pub struct EffectLedger<S: EffectStore> {
    store: S,
}

impl<S: EffectStore> EffectLedger<S> {
    pub fn new(store: S) -> Self {
        Self { store }
    }

    pub fn store(&self) -> &S {
        &self.store
    }
}

#[async_trait]
impl<S: EffectStore> EffectLog for EffectLedger<S> {
    async fn begin_effect(
        &self,
        job_id: &str,
        attempt_id: &str,
        kind: &str,
        input: &[u8],
        idempotency_key: &str,
    ) -> Result<String, EffectError> {
        let entry = EffectEntry {
            kind: kind.to_string(),
            input: input.to_vec(),
            idempotency_key: idempotency_key.to_string(),
        };
        self.store.record_pending(job_id, attempt_id, &entry).await
    }

    async fn commit_effect(&self, effect_id: &str, output: &[u8]) -> Result<(), EffectError> {
        self.store.confirm(effect_id, output).await
    }

    async fn abort_effect(&self, effect_id: &str, reason: &str) -> Result<(), EffectError> {
        self.store.rollback(effect_id, reason).await
    }

    async fn pending_effects(&self, job_id: &str) -> Result<Vec<EffectRecord>, EffectError> {
        let all = self.store.list_by_job(job_id).await?;
        Ok(all
            .into_iter()
            .filter(|r| r.status == EffectStatus::Pending)
            .collect())
    }
}
