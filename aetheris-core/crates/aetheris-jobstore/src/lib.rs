//! Aetheris jobstore crate.
//!
//! Event-sourced job storage with optimistic concurrency, distributed lease,
//! proof chain, and snapshot support. Built on oris-runtime kernel-postgres.

mod error;
mod memory;
mod pgstore;
mod types;

pub use error::JobStoreError;
pub use memory::MemoryJobStore;
pub use pgstore::PgJobStore;
pub use types::*;

use async_trait::async_trait;

/// Core job store trait — the event sourcing contract.
///
/// All mutations go through `Append` (event sourcing).
/// `Claim` and `Heartbeat` use dedicated tables for distributed scheduling.
#[async_trait]
pub trait JobStore: Send + Sync {
    /// Append an event to a job's event stream with optimistic concurrency.
    ///
    /// Returns the new version on success, or `VersionConflict` if
    /// `expected_version` doesn't match the current head.
    async fn append(
        &self,
        job_id: &str,
        expected_version: i32,
        event: &JobEvent,
    ) -> Result<i32, JobStoreError>;

    /// List all events for a job, ordered by version ascending.
    async fn list_events(&self, job_id: &str) -> Result<(Vec<JobEvent>, i32), JobStoreError>;

    /// Claim the next available job for a worker using distributed lease.
    ///
    /// Returns (job_id, version, attempt_id) if a job was claimed, or None.
    async fn claim(&self, worker_id: &str) -> Result<Option<ClaimResult>, JobStoreError>;

    /// Claim a specific job by ID.
    async fn claim_job(
        &self,
        worker_id: &str,
        job_id: &str,
    ) -> Result<Option<ClaimResult>, JobStoreError>;

    /// Renew the lease heartbeat for a claimed job.
    async fn heartbeat(&self, worker_id: &str, job_id: &str) -> Result<(), JobStoreError>;

    /// Watch for new events on a job (returns events after the given version).
    async fn watch(
        &self,
        job_id: &str,
        after_version: i32,
    ) -> Result<Vec<JobEvent>, JobStoreError>;

    /// List job IDs with expired claims (for reclamation).
    async fn list_expired_claims(&self) -> Result<Vec<String>, JobStoreError>;

    /// Get the current attempt ID for a job.
    async fn get_attempt_id(&self, job_id: &str) -> Result<Option<String>, JobStoreError>;

    /// Create a snapshot of job state up to a given version.
    async fn create_snapshot(
        &self,
        job_id: &str,
        up_to_version: i32,
        snapshot: &[u8],
    ) -> Result<(), JobStoreError>;

    /// Get the latest snapshot for a job.
    async fn get_latest_snapshot(
        &self,
        job_id: &str,
    ) -> Result<Option<SnapshotEntry>, JobStoreError>;

    /// Delete snapshots before a given version.
    async fn delete_snapshots_before(
        &self,
        job_id: &str,
        before_version: i32,
    ) -> Result<(), JobStoreError>;
}
