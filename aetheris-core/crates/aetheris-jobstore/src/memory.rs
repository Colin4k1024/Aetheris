//! In-memory job store implementation for testing.

use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::RwLock;
use uuid::Uuid;

use crate::error::JobStoreError;
use crate::types::{ClaimResult, JobEvent, SnapshotEntry};
use crate::JobStore;

struct JobLog {
    events: Vec<JobEvent>,
    claim: Option<ClaimEntry>,
}

struct ClaimEntry {
    worker_id: String,
    attempt_id: String,
    expires_at_ms: i64,
}

/// In-memory job store for unit tests.
pub struct MemoryJobStore {
    jobs: Arc<RwLock<HashMap<String, JobLog>>>,
    snapshots: Arc<RwLock<HashMap<String, Vec<SnapshotEntry>>>>,
    lease_duration_ms: i64,
}

impl MemoryJobStore {
    pub fn new() -> Self {
        Self {
            jobs: Arc::new(RwLock::new(HashMap::new())),
            snapshots: Arc::new(RwLock::new(HashMap::new())),
            lease_duration_ms: 30_000,
        }
    }

    pub fn with_lease_duration(mut self, ms: i64) -> Self {
        self.lease_duration_ms = ms;
        self
    }
}

impl Default for MemoryJobStore {
    fn default() -> Self {
        Self::new()
    }
}

fn now_ms() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

#[async_trait]
impl JobStore for MemoryJobStore {
    async fn append(
        &self,
        job_id: &str,
        expected_version: i32,
        event: &JobEvent,
    ) -> Result<i32, JobStoreError> {
        let mut jobs = self.jobs.write().await;
        let log = jobs.entry(job_id.to_string()).or_insert_with(|| JobLog {
            events: Vec::new(),
            claim: None,
        });

        let current_version = log.events.last().map(|e| e.version).unwrap_or(0);
        if current_version != expected_version {
            return Err(JobStoreError::VersionConflict {
                expected: expected_version,
                actual: current_version,
            });
        }

        let new_version = current_version + 1;
        let prev_hash = log
            .events
            .last()
            .map(|e| e.hash.clone())
            .unwrap_or_default();

        let mut new_event = event.clone();
        new_event.job_id = job_id.to_string();
        new_event.version = new_version;
        new_event.prev_hash = prev_hash;
        new_event.hash = new_event.compute_hash();
        new_event.timestamp_ms = now_ms();

        log.events.push(new_event);
        Ok(new_version)
    }

    async fn list_events(&self, job_id: &str) -> Result<(Vec<JobEvent>, i32), JobStoreError> {
        let jobs = self.jobs.read().await;
        match jobs.get(job_id) {
            Some(log) => {
                let version = log.events.last().map(|e| e.version).unwrap_or(0);
                Ok((log.events.clone(), version))
            }
            None => Ok((Vec::new(), 0)),
        }
    }

    async fn claim(&self, worker_id: &str) -> Result<Option<ClaimResult>, JobStoreError> {
        let mut jobs = self.jobs.write().await;
        let now = now_ms();

        for (job_id, log) in jobs.iter_mut() {
            // Skip jobs that are already claimed with a valid lease
            if let Some(ref claim) = log.claim {
                if claim.expires_at_ms > now {
                    continue;
                }
            }

            // Check if the job has a "created" event (is ready to be claimed)
            let has_created = log
                .events
                .iter()
                .any(|e| e.event_type == "job_created" || e.event_type == "JobCreated");
            if !has_created {
                continue;
            }

            // Check if the job is already completed or failed
            let is_terminal = log.events.iter().any(|e| {
                matches!(
                    e.event_type.as_str(),
                    "job_completed" | "job_failed" | "JobCompleted" | "JobFailed"
                )
            });
            if is_terminal {
                continue;
            }

            let attempt_id = Uuid::new_v4().to_string();
            let version = log.events.last().map(|e| e.version).unwrap_or(0);

            log.claim = Some(ClaimEntry {
                worker_id: worker_id.to_string(),
                attempt_id: attempt_id.clone(),
                expires_at_ms: now + self.lease_duration_ms,
            });

            return Ok(Some(ClaimResult {
                job_id: job_id.clone(),
                version,
                attempt_id,
            }));
        }

        Ok(None)
    }

    async fn claim_job(
        &self,
        worker_id: &str,
        job_id: &str,
    ) -> Result<Option<ClaimResult>, JobStoreError> {
        let mut jobs = self.jobs.write().await;
        let now = now_ms();

        let log = match jobs.get_mut(job_id) {
            Some(l) => l,
            None => return Ok(None),
        };

        // Check if already claimed with valid lease
        if let Some(ref claim) = log.claim {
            if claim.expires_at_ms > now && claim.worker_id != worker_id {
                return Ok(None); // Another worker holds the lease
            }
        }

        let attempt_id = Uuid::new_v4().to_string();
        let version = log.events.last().map(|e| e.version).unwrap_or(0);

        log.claim = Some(ClaimEntry {
            worker_id: worker_id.to_string(),
            attempt_id: attempt_id.clone(),
            expires_at_ms: now + self.lease_duration_ms,
        });

        Ok(Some(ClaimResult {
            job_id: job_id.to_string(),
            version,
            attempt_id,
        }))
    }

    async fn heartbeat(&self, worker_id: &str, job_id: &str) -> Result<(), JobStoreError> {
        let mut jobs = self.jobs.write().await;
        let now = now_ms();

        if let Some(log) = jobs.get_mut(job_id) {
            if let Some(ref mut claim) = log.claim {
                if claim.worker_id == worker_id {
                    claim.expires_at_ms = now + self.lease_duration_ms;
                    return Ok(());
                }
            }
        }

        Err(JobStoreError::InvalidInput(format!(
            "no valid claim for worker {worker_id} on job {job_id}"
        )))
    }

    async fn watch(
        &self,
        job_id: &str,
        after_version: i32,
    ) -> Result<Vec<JobEvent>, JobStoreError> {
        let jobs = self.jobs.read().await;
        match jobs.get(job_id) {
            Some(log) => Ok(log
                .events
                .iter()
                .filter(|e| e.version > after_version)
                .cloned()
                .collect()),
            None => Ok(Vec::new()),
        }
    }

    async fn list_expired_claims(&self) -> Result<Vec<String>, JobStoreError> {
        let jobs = self.jobs.read().await;
        let now = now_ms();

        Ok(jobs
            .iter()
            .filter_map(|(job_id, log)| {
                log.claim.as_ref().and_then(|claim| {
                    if claim.expires_at_ms <= now {
                        Some(job_id.clone())
                    } else {
                        None
                    }
                })
            })
            .collect())
    }

    async fn get_attempt_id(&self, job_id: &str) -> Result<Option<String>, JobStoreError> {
        let jobs = self.jobs.read().await;
        Ok(jobs
            .get(job_id)
            .and_then(|log| log.claim.as_ref().map(|c| c.attempt_id.clone())))
    }

    async fn create_snapshot(
        &self,
        job_id: &str,
        up_to_version: i32,
        snapshot: &[u8],
    ) -> Result<(), JobStoreError> {
        let mut snapshots = self.snapshots.write().await;
        let entries = snapshots.entry(job_id.to_string()).or_default();
        entries.push(SnapshotEntry {
            job_id: job_id.to_string(),
            version: up_to_version,
            data: snapshot.to_vec(),
            created_at_ms: now_ms(),
        });
        Ok(())
    }

    async fn get_latest_snapshot(
        &self,
        job_id: &str,
    ) -> Result<Option<SnapshotEntry>, JobStoreError> {
        let snapshots = self.snapshots.read().await;
        Ok(snapshots
            .get(job_id)
            .and_then(|entries| entries.iter().max_by_key(|s| s.version).cloned()))
    }

    async fn delete_snapshots_before(
        &self,
        job_id: &str,
        before_version: i32,
    ) -> Result<(), JobStoreError> {
        let mut snapshots = self.snapshots.write().await;
        if let Some(entries) = snapshots.get_mut(job_id) {
            entries.retain(|s| s.version >= before_version);
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_event(event_type: &str, payload: &[u8]) -> JobEvent {
        JobEvent {
            job_id: String::new(),
            version: 0,
            event_type: event_type.to_string(),
            payload: payload.to_vec(),
            prev_hash: String::new(),
            hash: String::new(),
            timestamp_ms: 0,
        }
    }

    #[tokio::test]
    async fn test_append_and_list() {
        let store = MemoryJobStore::new();

        let v = store
            .append("job-1", 0, &make_event("job_created", b"{}"))
            .await
            .unwrap();
        assert_eq!(v, 1);

        let v = store
            .append("job-1", 1, &make_event("step_started", b"{}"))
            .await
            .unwrap();
        assert_eq!(v, 2);

        let (events, latest) = store.list_events("job-1").await.unwrap();
        assert_eq!(events.len(), 2);
        assert_eq!(latest, 2);
        assert_eq!(events[0].event_type, "job_created");
        assert_eq!(events[1].event_type, "step_started");
    }

    #[tokio::test]
    async fn test_version_conflict() {
        let store = MemoryJobStore::new();

        store
            .append("job-1", 0, &make_event("job_created", b"{}"))
            .await
            .unwrap();

        let result = store
            .append("job-1", 0, &make_event("step_started", b"{}"))
            .await;
        assert!(matches!(result, Err(JobStoreError::VersionConflict { .. })));
    }

    #[tokio::test]
    async fn test_proof_chain_hashing() {
        let store = MemoryJobStore::new();

        let v1 = store
            .append("job-1", 0, &make_event("job_created", b"{}"))
            .await
            .unwrap();

        let (events, _) = store.list_events("job-1").await.unwrap();
        assert_eq!(events[0].version, 1);
        assert!(!events[0].hash.is_empty());
        assert!(events[0].prev_hash.is_empty()); // first event

        let v2 = store
            .append("job-1", v1, &make_event("step_started", b"{}"))
            .await
            .unwrap();

        let (events, _) = store.list_events("job-1").await.unwrap();
        assert_eq!(events[1].prev_hash, events[0].hash); // chain linked
        assert_eq!(v2, 2);
    }

    #[tokio::test]
    async fn test_claim_and_heartbeat() {
        let store = MemoryJobStore::new();

        store
            .append("job-1", 0, &make_event("job_created", b"{}"))
            .await
            .unwrap();

        let claim = store.claim("worker-1").await.unwrap().unwrap();
        assert_eq!(claim.job_id, "job-1");
        assert_eq!(claim.version, 1);

        // Second claim should return None (already claimed)
        let result = store.claim("worker-2").await.unwrap();
        assert!(result.is_none());

        // Heartbeat should succeed
        store.heartbeat("worker-1", "job-1").await.unwrap();

        // Heartbeat with wrong worker should fail
        let result = store.heartbeat("worker-2", "job-1").await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_snapshot() {
        let store = MemoryJobStore::new();

        store
            .create_snapshot("job-1", 5, b"snapshot-data")
            .await
            .unwrap();

        let snap = store.get_latest_snapshot("job-1").await.unwrap().unwrap();
        assert_eq!(snap.version, 5);
        assert_eq!(snap.data, b"snapshot-data");

        store
            .create_snapshot("job-1", 10, b"newer-data")
            .await
            .unwrap();

        let snap = store.get_latest_snapshot("job-1").await.unwrap().unwrap();
        assert_eq!(snap.version, 10);
    }

    #[tokio::test]
    async fn test_watch() {
        let store = MemoryJobStore::new();

        store
            .append("job-1", 0, &make_event("job_created", b"{}"))
            .await
            .unwrap();
        store
            .append("job-1", 1, &make_event("step_started", b"{}"))
            .await
            .unwrap();
        store
            .append("job-1", 2, &make_event("step_completed", b"{}"))
            .await
            .unwrap();

        let new_events = store.watch("job-1", 1).await.unwrap();
        assert_eq!(new_events.len(), 2);
        assert_eq!(new_events[0].version, 2);
        assert_eq!(new_events[1].version, 3);
    }
}
