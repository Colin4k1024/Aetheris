//! Compensation registry — tracks and executes side-effect rollbacks.

use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::RwLock;

/// A compensation action that can undo a side effect.
#[derive(Clone, Debug)]
pub struct CompensationAction {
    /// The effect ID this compensation is for.
    pub effect_id: String,
    /// The node that produced the effect.
    pub node_id: String,
    /// Human-readable description.
    pub description: String,
    /// Serialized compensation input.
    pub compensation_input: Vec<u8>,
}

/// Trait for executing compensations.
#[async_trait]
pub trait Compensator: Send + Sync {
    /// Execute the compensation action.
    async fn compensate(&self, action: &CompensationAction) -> Result<(), String>;
}

/// Registry that tracks compensation actions for a job execution.
pub struct CompensationRegistry {
    /// job_id -> list of compensation actions (in reverse order of execution)
    actions: Arc<RwLock<HashMap<String, Vec<CompensationAction>>>>,
    compensator: Option<Arc<dyn Compensator>>,
}

impl CompensationRegistry {
    pub fn new() -> Self {
        Self {
            actions: Arc::new(RwLock::new(HashMap::new())),
            compensator: None,
        }
    }

    pub fn with_compensator(mut self, compensator: Arc<dyn Compensator>) -> Self {
        self.compensator = Some(compensator);
        self
    }

    /// Register a compensation action for a job.
    pub async fn register(&self, job_id: &str, action: CompensationAction) {
        let mut actions = self.actions.write().await;
        actions
            .entry(job_id.to_string())
            .or_default()
            .push(action);
    }

    /// Execute all compensations for a job (in reverse order).
    pub async fn compensate_all(&self, job_id: &str) -> Result<(), String> {
        let compensator = match &self.compensator {
            Some(c) => c,
            None => return Err("no compensator registered".to_string()),
        };

        let mut actions = self.actions.write().await;
        let job_actions = match actions.get_mut(job_id) {
            Some(a) => a,
            None => return Ok(()),
        };

        // Execute in reverse order (LIFO)
        let mut errors = Vec::new();
        while let Some(action) = job_actions.pop() {
            if let Err(e) = compensator.compensate(&action).await {
                errors.push(format!(
                    "compensation failed for {}: {}",
                    action.effect_id, e
                ));
            }
        }

        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors.join("; "))
        }
    }

    /// Get pending compensation actions for a job.
    pub async fn pending(&self, job_id: &str) -> Vec<CompensationAction> {
        let actions = self.actions.read().await;
        actions.get(job_id).cloned().unwrap_or_default()
    }

    /// Clear all compensations for a job (after successful completion).
    pub async fn clear(&self, job_id: &str) {
        let mut actions = self.actions.write().await;
        actions.remove(job_id);
    }
}

impl Default for CompensationRegistry {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicUsize, Ordering};

    struct CountingCompensator {
        count: Arc<AtomicUsize>,
    }

    #[async_trait]
    impl Compensator for CountingCompensator {
        async fn compensate(&self, _action: &CompensationAction) -> Result<(), String> {
            self.count.fetch_add(1, Ordering::SeqCst);
            Ok(())
        }
    }

    #[tokio::test]
    async fn test_register_and_compensate() {
        let counter = Arc::new(AtomicUsize::new(0));
        let registry = CompensationRegistry::new().with_compensator(Arc::new(
            CountingCompensator {
                count: counter.clone(),
            },
        ));

        registry
            .register(
                "job-1",
                CompensationAction {
                    effect_id: "eff-1".to_string(),
                    node_id: "n1".to_string(),
                    description: "undo http call".to_string(),
                    compensation_input: vec![],
                },
            )
            .await;

        registry
            .register(
                "job-1",
                CompensationAction {
                    effect_id: "eff-2".to_string(),
                    node_id: "n2".to_string(),
                    description: "undo db write".to_string(),
                    compensation_input: vec![],
                },
            )
            .await;

        assert_eq!(registry.pending("job-1").await.len(), 2);

        registry.compensate_all("job-1").await.unwrap();
        assert_eq!(counter.load(Ordering::SeqCst), 2);
        assert_eq!(registry.pending("job-1").await.len(), 0);
    }
}
