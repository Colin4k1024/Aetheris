//! Aetheris effects crate.
//!
//! Two-phase commit for side-effect tool execution.
//! Phase 1: Record intent (pending) → Phase 2: Confirm or rollback.

mod error;
mod ledger;
mod memory;
mod types;

pub use error::EffectError;
pub use ledger::EffectLedger;
pub use memory::MemoryEffectLedger;
pub use types::*;

use async_trait::async_trait;

/// Effect store trait — records side-effect intents.
#[async_trait]
pub trait EffectStore: Send + Sync {
    /// Record a pending effect (Phase 1 of 2PC).
    async fn record_pending(
        &self,
        job_id: &str,
        attempt_id: &str,
        effect: &EffectEntry,
    ) -> Result<String, EffectError>;

    /// Confirm a pending effect (Phase 2 of 2PC — commit).
    async fn confirm(&self, effect_id: &str, output: &[u8]) -> Result<(), EffectError>;

    /// Rollback a pending effect (Phase 2 of 2PC — abort).
    async fn rollback(&self, effect_id: &str, reason: &str) -> Result<(), EffectError>;

    /// Get all effects for a job.
    async fn list_by_job(&self, job_id: &str) -> Result<Vec<EffectRecord>, EffectError>;

    /// Get a specific effect by ID.
    async fn get(&self, effect_id: &str) -> Result<Option<EffectRecord>, EffectError>;

    /// Check if an effect with the given idempotency key already exists.
    async fn exists_by_idempotency_key(
        &self,
        job_id: &str,
        key: &str,
    ) -> Result<Option<EffectRecord>, EffectError>;
}

/// Effect log trait — higher-level operations on the effect ledger.
#[async_trait]
pub trait EffectLog: Send + Sync {
    /// Begin a new effect (records pending, returns effect_id).
    async fn begin_effect(
        &self,
        job_id: &str,
        attempt_id: &str,
        kind: &str,
        input: &[u8],
        idempotency_key: &str,
    ) -> Result<String, EffectError>;

    /// Commit an effect (confirms with output).
    async fn commit_effect(&self, effect_id: &str, output: &[u8]) -> Result<(), EffectError>;

    /// Abort an effect (rolls back with reason).
    async fn abort_effect(&self, effect_id: &str, reason: &str) -> Result<(), EffectError>;

    /// Get all pending (uncommitted) effects for a job.
    async fn pending_effects(&self, job_id: &str) -> Result<Vec<EffectRecord>, EffectError>;
}
