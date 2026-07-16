//! Shared types for the job store.

use serde::{Deserialize, Serialize};

/// A job event in the event stream.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct JobEvent {
    pub job_id: String,
    pub version: i32,
    pub event_type: String,
    pub payload: Vec<u8>,
    pub prev_hash: String,
    pub hash: String,
    pub timestamp_ms: i64,
}

/// Result of a successful claim operation.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ClaimResult {
    pub job_id: String,
    pub version: i32,
    pub attempt_id: String,
}

/// A snapshot entry stored in the database.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SnapshotEntry {
    pub job_id: String,
    pub version: i32,
    pub data: Vec<u8>,
    pub created_at_ms: i64,
}

impl JobEvent {
    /// Compute the SHA-256 hash for the proof chain.
    pub fn compute_hash(&self) -> String {
        use sha2::{Digest, Sha256};
        let mut hasher = Sha256::new();
        hasher.update(self.prev_hash.as_bytes());
        hasher.update(self.job_id.as_bytes());
        hasher.update(self.version.to_be_bytes());
        hasher.update(self.event_type.as_bytes());
        hasher.update(&self.payload);
        hex::encode(hasher.finalize())
    }
}
