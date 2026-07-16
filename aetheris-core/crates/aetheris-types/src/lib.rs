//! Aetheris types crate.
//!
//! Contains protobuf-generated domain types and shared type definitions.

/// Protobuf-generated domain types.
pub mod proto {
    include!(concat!(env!("OUT_DIR"), "/aetheris.domain.rs"));
}

// Re-export commonly used types
pub use proto::{
    JobEvent, JobStatus, StepResult, StepResultType,
    AgentState, AgentStatus, AgentInstance, AgentConfig,
    ToolInvocation, ToolInvocationStatus, ToolResult, ToolCapability,
    Checkpoint, TaskGraph, TaskNode, NodeType, Edge, CursorNode, CursorStatus,
    MemoryEntry, MemoryNamespace, MemoryQuery, MemoryQueryResult,
    AppendRequest, AppendResponse, ClaimRequest, ClaimResponse,
    ClaimJobRequest, HeartbeatRequest, ListEventsRequest, ListEventsResponse,
    CreateSnapshotRequest, SnapshotResponse,
    RunStepRequest, RunStepResponse, ScheduleRequest, ScheduleResponse,
};

/// FFI-compatible error codes.
#[repr(i32)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AetherisError {
    Success = 0,
    InvalidInput = -1,
    VersionConflict = -2,
    DatabaseError = -3,
    Timeout = -4,
    InternalError = -5,
}

impl AetherisError {
    /// Convert from i32 to AetherisError.
    pub fn from_i32(code: i32) -> Self {
        match code {
            0 => Self::Success,
            -1 => Self::InvalidInput,
            -2 => Self::VersionConflict,
            -3 => Self::DatabaseError,
            -4 => Self::Timeout,
            _ => Self::InternalError,
        }
    }
}

/// Proof chain verification — validates SHA-256 hash chains.
pub mod proof {
    use sha2::{Digest, Sha256};

    /// Verify a proof chain of events.
    ///
    /// Each event's `prev_hash` must match the previous event's `hash`.
    /// The first event's `prev_hash` must be empty.
    pub fn verify_chain(events: &[ProofEvent]) -> Result<(), ProofError> {
        for (i, event) in events.iter().enumerate() {
            // Compute expected hash
            let expected_hash = compute_hash(
                &event.prev_hash,
                &event.job_id,
                event.version,
                &event.event_type,
                &event.payload,
            );

            if event.hash != expected_hash {
                return Err(ProofError::HashMismatch {
                    index: i,
                    expected: expected_hash,
                    actual: event.hash.clone(),
                });
            }

            // Check chain linkage
            if i == 0 {
                if !event.prev_hash.is_empty() {
                    return Err(ProofError::ChainBreak {
                        index: i,
                        reason: "first event must have empty prev_hash".to_string(),
                    });
                }
            } else {
                let prev = &events[i - 1];
                if event.prev_hash != prev.hash {
                    return Err(ProofError::ChainBreak {
                        index: i,
                        reason: format!(
                            "prev_hash mismatch: expected {}, got {}",
                            prev.hash, event.prev_hash
                        ),
                    });
                }
            }
        }
        Ok(())
    }

    /// Compute SHA-256 hash for a proof chain event.
    pub fn compute_hash(
        prev_hash: &str,
        job_id: &str,
        version: i32,
        event_type: &str,
        payload: &[u8],
    ) -> String {
        let mut hasher = Sha256::new();
        hasher.update(prev_hash.as_bytes());
        hasher.update(job_id.as_bytes());
        hasher.update(version.to_be_bytes());
        hasher.update(event_type.as_bytes());
        hasher.update(payload);
        hex::encode(hasher.finalize())
    }

    /// A simplified event for proof chain verification.
    #[derive(Clone, Debug)]
    pub struct ProofEvent {
        pub job_id: String,
        pub version: i32,
        pub event_type: String,
        pub payload: Vec<u8>,
        pub prev_hash: String,
        pub hash: String,
    }

    /// Proof verification errors.
    #[derive(Debug, thiserror::Error)]
    pub enum ProofError {
        #[error("hash mismatch at index {index}: expected {expected}, got {actual}")]
        HashMismatch {
            index: usize,
            expected: String,
            actual: String,
        },
        #[error("chain break at index {index}: {reason}")]
        ChainBreak { index: usize, reason: String },
    }

    #[cfg(test)]
    mod tests {
        use super::*;

        #[test]
        fn test_valid_chain() {
            let hash1 = compute_hash("", "job-1", 1, "created", b"{}");
            let hash2 = compute_hash(&hash1, "job-1", 2, "started", b"{}");

            let events = vec![
                ProofEvent {
                    job_id: "job-1".to_string(),
                    version: 1,
                    event_type: "created".to_string(),
                    payload: b"{}".to_vec(),
                    prev_hash: String::new(),
                    hash: hash1,
                },
                ProofEvent {
                    job_id: "job-1".to_string(),
                    version: 2,
                    event_type: "started".to_string(),
                    payload: b"{}".to_vec(),
                    prev_hash: compute_hash("", "job-1", 1, "created", b"{}"),
                    hash: hash2,
                },
            ];

            verify_chain(&events).unwrap();
        }

        #[test]
        fn test_invalid_chain() {
            let events = vec![ProofEvent {
                job_id: "job-1".to_string(),
                version: 1,
                event_type: "created".to_string(),
                payload: b"{}".to_vec(),
                prev_hash: String::new(),
                hash: "wrong-hash".to_string(),
            }];

            assert!(verify_chain(&events).is_err());
        }
    }
}
