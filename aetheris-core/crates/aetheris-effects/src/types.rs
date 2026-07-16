//! Effect types.

use serde::{Deserialize, Serialize};

/// A single effect entry to be recorded.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct EffectEntry {
    pub kind: String,
    pub input: Vec<u8>,
    pub idempotency_key: String,
}

/// Status of an effect in the two-phase commit.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub enum EffectStatus {
    /// Phase 1 complete — waiting for confirm/rollback.
    Pending,
    /// Phase 2 complete — committed.
    Committed,
    /// Phase 2 complete — rolled back.
    RolledBack,
}

/// A recorded effect with its current state.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct EffectRecord {
    pub id: String,
    pub job_id: String,
    pub attempt_id: String,
    pub kind: String,
    pub input: Vec<u8>,
    pub output: Vec<u8>,
    pub status: EffectStatus,
    pub idempotency_key: String,
    pub error_message: String,
    pub created_at_ms: i64,
    pub committed_at_ms: Option<i64>,
}
