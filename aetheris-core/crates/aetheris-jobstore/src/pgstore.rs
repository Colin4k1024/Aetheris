//! PostgreSQL-backed job store using oris-runtime kernel-postgres.
//!
//! Uses oris-runtime's PostgresEventStore for the event log,
//! and direct sqlx queries for Aetheris-specific tables (claims, snapshots).

use std::sync::Arc;

use async_trait::async_trait;
use sqlx::postgres::PgPoolOptions;
use sqlx::PgPool;
use uuid::Uuid;

use crate::error::JobStoreError;
use crate::types::{ClaimResult, JobEvent, SnapshotEntry};
use crate::JobStore;

/// PostgreSQL-backed job store.
///
/// Event log uses oris-runtime's `PostgresEventStore` (kernel_events table).
/// Claim/heartbeat uses Aetheris's `job_claims` table.
/// Snapshots use Aetheris's `job_snapshots` table.
pub struct PgJobStore {
    pool: PgPool,
    lease_duration_secs: i64,
}

impl PgJobStore {
    /// Create a new PgJobStore connected to the given DSN.
    pub async fn new(dsn: &str) -> Result<Self, JobStoreError> {
        let pool = PgPoolOptions::new()
            .max_connections(10)
            .connect(dsn)
            .await
            .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(Self {
            pool,
            lease_duration_secs: 30,
        })
    }

    pub fn with_lease_duration(mut self, secs: i64) -> Self {
        self.lease_duration_secs = secs;
        self
    }

    /// Create from an existing pool.
    pub fn from_pool(pool: PgPool) -> Self {
        Self {
            pool,
            lease_duration_secs: 30,
        }
    }
}

#[async_trait]
impl JobStore for PgJobStore {
    async fn append(
        &self,
        job_id: &str,
        expected_version: i32,
        event: &JobEvent,
    ) -> Result<i32, JobStoreError> {
        // Use advisory lock per job_id for serialized append (same pattern as oris-runtime)
        let mut tx = self
            .pool
            .begin()
            .await
            .map_err(|e| JobStoreError::Database(e.to_string()))?;

        // Advisory lock on job_id hash
        sqlx::query("SELECT pg_advisory_xact_lock(hashtext($1))")
            .bind(job_id)
            .execute(&mut *tx)
            .await
            .map_err(|e| JobStoreError::Database(e.to_string()))?;

        // Get current version
        let current: i32 = sqlx::query_scalar(
            "SELECT COALESCE(MAX(version), 0) FROM job_events WHERE job_id = $1",
        )
        .bind(job_id)
        .fetch_one(&mut *tx)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        if current != expected_version {
            return Err(JobStoreError::VersionConflict {
                expected: expected_version,
                actual: current,
            });
        }

        let new_version = expected_version + 1;

        // Get previous hash for proof chain
        let prev_hash: Option<String> = sqlx::query_scalar(
            "SELECT hash FROM job_events WHERE job_id = $1 ORDER BY version DESC LIMIT 1",
        )
        .bind(job_id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        let prev_hash = prev_hash.unwrap_or_default();

        // Compute hash
        let mut new_event = event.clone();
        new_event.job_id = job_id.to_string();
        new_event.version = new_version;
        new_event.prev_hash = prev_hash;
        let hash = new_event.compute_hash();
        let timestamp_ms = chrono::Utc::now().timestamp_millis();

        sqlx::query(
            "INSERT INTO job_events (job_id, version, type, payload, prev_hash, hash, created_at)
             VALUES ($1, $2, $3, $4, $5, $6, NOW())",
        )
        .bind(job_id)
        .bind(new_version)
        .bind(&event.event_type)
        .bind(serde_json::to_value(&event.payload).unwrap_or_default())
        .bind(&new_event.prev_hash)
        .bind(&hash)
        .execute(&mut *tx)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        tx.commit()
            .await
            .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(new_version)
    }

    async fn list_events(&self, job_id: &str) -> Result<(Vec<JobEvent>, i32), JobStoreError> {
        let rows: Vec<(i32, String, serde_json::Value, String, String)> = sqlx::query_as(
            "SELECT version, type, payload, prev_hash, hash
             FROM job_events WHERE job_id = $1 ORDER BY version ASC",
        )
        .bind(job_id)
        .fetch_all(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        let events: Vec<JobEvent> = rows
            .into_iter()
            .map(|(version, event_type, payload, prev_hash, hash)| JobEvent {
                job_id: job_id.to_string(),
                version,
                event_type,
                payload: serde_json::to_vec(&payload).unwrap_or_default(),
                prev_hash,
                hash,
                timestamp_ms: 0,
            })
            .collect();

        let latest = events.last().map(|e| e.version).unwrap_or(0);
        Ok((events, latest))
    }

    async fn claim(&self, worker_id: &str) -> Result<Option<ClaimResult>, JobStoreError> {
        // Find an unclaimed job with a "job_created" event
        let row: Option<(String, i32)> = sqlx::query_as(
            "SELECT j.job_id, j.version FROM (
                SELECT DISTINCT ON (je.job_id) je.job_id, je.version
                FROM job_events je
                WHERE je.type IN ('job_created', 'JobCreated')
                ORDER BY je.job_id, je.version DESC
             ) j
             LEFT JOIN job_claims c ON c.job_id = j.job_id AND c.expires_at > NOW()
             WHERE c.job_id IS NULL
             LIMIT 1",
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        let (job_id, version) = match row {
            Some(r) => r,
            None => return Ok(None),
        };

        let attempt_id = Uuid::new_v4().to_string();
        let lease_secs = self.lease_duration_secs;

        sqlx::query(
            "INSERT INTO job_claims (job_id, worker_id, attempt_id, expires_at)
             VALUES ($1, $2, $3, NOW() + ($4 || ' seconds')::interval)
             ON CONFLICT (job_id)
             DO UPDATE SET worker_id = $2, attempt_id = $3, expires_at = NOW() + ($4 || ' seconds')::interval
             WHERE job_claims.expires_at <= NOW()",
        )
        .bind(&job_id)
        .bind(worker_id)
        .bind(&attempt_id)
        .bind(lease_secs.to_string())
        .execute(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(Some(ClaimResult {
            job_id,
            version,
            attempt_id,
        }))
    }

    async fn claim_job(
        &self,
        worker_id: &str,
        job_id: &str,
    ) -> Result<Option<ClaimResult>, JobStoreError> {
        let attempt_id = Uuid::new_v4().to_string();
        let lease_secs = self.lease_duration_secs;

        let result = sqlx::query(
            "INSERT INTO job_claims (job_id, worker_id, attempt_id, expires_at)
             VALUES ($1, $2, $3, NOW() + ($4 || ' seconds')::interval)
             ON CONFLICT (job_id) DO NOTHING
             RETURNING job_id",
        )
        .bind(job_id)
        .bind(worker_id)
        .bind(&attempt_id)
        .bind(lease_secs.to_string())
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        if result.is_some() {
            let version: i32 = sqlx::query_scalar(
                "SELECT COALESCE(MAX(version), 0) FROM job_events WHERE job_id = $1",
            )
            .bind(job_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| JobStoreError::Database(e.to_string()))?;

            Ok(Some(ClaimResult {
                job_id: job_id.to_string(),
                version,
                attempt_id,
            }))
        } else {
            Ok(None)
        }
    }

    async fn heartbeat(&self, worker_id: &str, job_id: &str) -> Result<(), JobStoreError> {
        let lease_secs = self.lease_duration_secs;
        let result = sqlx::query(
            "UPDATE job_claims SET expires_at = NOW() + ($3 || ' seconds')::interval
             WHERE job_id = $1 AND worker_id = $2",
        )
        .bind(job_id)
        .bind(worker_id)
        .bind(lease_secs.to_string())
        .execute(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        if result.rows_affected() == 0 {
            return Err(JobStoreError::InvalidInput(format!(
                "no valid claim for worker {worker_id} on job {job_id}"
            )));
        }
        Ok(())
    }

    async fn watch(
        &self,
        job_id: &str,
        after_version: i32,
    ) -> Result<Vec<JobEvent>, JobStoreError> {
        let rows: Vec<(i32, String, serde_json::Value, String, String)> = sqlx::query_as(
            "SELECT version, type, payload, prev_hash, hash
             FROM job_events WHERE job_id = $1 AND version > $2 ORDER BY version ASC",
        )
        .bind(job_id)
        .bind(after_version)
        .fetch_all(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(rows
            .into_iter()
            .map(|(version, event_type, payload, prev_hash, hash)| JobEvent {
                job_id: job_id.to_string(),
                version,
                event_type,
                payload: serde_json::to_vec(&payload).unwrap_or_default(),
                prev_hash,
                hash,
                timestamp_ms: 0,
            })
            .collect())
    }

    async fn list_expired_claims(&self) -> Result<Vec<String>, JobStoreError> {
        let rows: Vec<String> = sqlx::query_scalar(
            "SELECT job_id FROM job_claims WHERE expires_at <= NOW()",
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(rows)
    }

    async fn get_attempt_id(&self, job_id: &str) -> Result<Option<String>, JobStoreError> {
        let result: Option<String> = sqlx::query_scalar(
            "SELECT attempt_id FROM job_claims WHERE job_id = $1",
        )
        .bind(job_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(result)
    }

    async fn create_snapshot(
        &self,
        job_id: &str,
        up_to_version: i32,
        snapshot: &[u8],
    ) -> Result<(), JobStoreError> {
        sqlx::query(
            "INSERT INTO job_snapshots (job_id, version, snapshot, created_at)
             VALUES ($1, $2, $3, NOW())",
        )
        .bind(job_id)
        .bind(up_to_version)
        .bind(snapshot)
        .execute(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(())
    }

    async fn get_latest_snapshot(
        &self,
        job_id: &str,
    ) -> Result<Option<SnapshotEntry>, JobStoreError> {
        let row: Option<(i32, Vec<u8>)> = sqlx::query_as(
            "SELECT version, snapshot FROM job_snapshots
             WHERE job_id = $1 ORDER BY version DESC LIMIT 1",
        )
        .bind(job_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(row.map(|(version, data)| SnapshotEntry {
            job_id: job_id.to_string(),
            version,
            data,
            created_at_ms: 0,
        }))
    }

    async fn delete_snapshots_before(
        &self,
        job_id: &str,
        before_version: i32,
    ) -> Result<(), JobStoreError> {
        sqlx::query("DELETE FROM job_snapshots WHERE job_id = $1 AND version < $2")
            .bind(job_id)
            .bind(before_version)
            .execute(&self.pool)
            .await
            .map_err(|e| JobStoreError::Database(e.to_string()))?;

        Ok(())
    }
}
