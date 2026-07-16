//! Runner — the core DAG execution loop.

use std::collections::HashMap;
use std::sync::Arc;

use aetheris_jobstore::{JobEvent, JobStore};
use aetheris_planner::{ExecutionPlan, PlanNodeStatus};
use tokio::sync::mpsc;

use crate::checkpoint::CheckpointManager;
use crate::compensation::CompensationRegistry;
use crate::error::ExecutorError;
use crate::step::StepExecutor;
use crate::types::{StepContext, StepResult, StepResultType};

/// Configuration for the runner.
#[derive(Clone, Debug)]
pub struct RunnerConfig {
    /// Maximum concurrent steps (parallel nodes).
    pub max_concurrency: usize,
    /// Default step timeout in milliseconds.
    pub default_timeout_ms: u64,
    /// Whether to checkpoint after each step.
    pub checkpoint_after_each_step: bool,
}

impl Default for RunnerConfig {
    fn default() -> Self {
        Self {
            max_concurrency: 4,
            default_timeout_ms: 60_000,
            checkpoint_after_each_step: true,
        }
    }
}

/// Events emitted by the runner during execution.
#[derive(Clone, Debug)]
pub enum RunnerEvent {
    StepStarted {
        node_id: String,
        attempt: u32,
    },
    StepCompleted {
        node_id: String,
        result_type: StepResultType,
    },
    StepFailed {
        node_id: String,
        error: String,
        will_retry: bool,
    },
    PlanCompleted,
    PlanFailed {
        reason: String,
    },
    CheckpointCreated {
        version: i32,
    },
    CompensationStarted {
        node_id: String,
    },
}

/// The core execution runner.
///
/// Drives a DAG execution plan by:
/// 1. Finding ready nodes (dependencies satisfied)
/// 2. Executing them via StepExecutor
/// 3. Recording results as events
/// 4. Creating checkpoints
/// 5. Handling retries and compensation
pub struct Runner {
    config: RunnerConfig,
    step_executor: Arc<dyn StepExecutor>,
    compensation: Arc<CompensationRegistry>,
}

impl Runner {
    pub fn new(
        config: RunnerConfig,
        step_executor: Arc<dyn StepExecutor>,
        compensation: Arc<CompensationRegistry>,
    ) -> Self {
        Self {
            config,
            step_executor,
            compensation,
        }
    }

    /// Execute a full DAG plan against a job store.
    pub async fn run<S: JobStore>(
        &self,
        store: &S,
        job_id: &str,
        attempt_id: &str,
        plan: &mut ExecutionPlan,
        event_tx: Option<mpsc::Sender<RunnerEvent>>,
    ) -> Result<(), ExecutorError> {
        let checkpoint_mgr = CheckpointManager::new(job_id);
        let mut outputs: HashMap<String, Vec<u8>> = HashMap::new();

        loop {
            // Find next ready node
            let next_node = plan.next_to_execute().map(|n| n.id.clone());

            let node_id = match next_node {
                Some(id) => id,
                None => {
                    if plan.is_complete() {
                        Self::emit(&event_tx, RunnerEvent::PlanCompleted).await;
                        return Ok(());
                    } else if plan.has_failure() {
                        let reason = plan
                            .nodes
                            .iter()
                            .filter(|n| n.status == PlanNodeStatus::PermanentFailure)
                            .map(|n| format!("{}: {}", n.id, n.error.as_deref().unwrap_or("unknown")))
                            .collect::<Vec<_>>()
                            .join("; ");
                        Self::emit(&event_tx, RunnerEvent::PlanFailed {
                            reason: reason.clone(),
                        })
                        .await;
                        return Err(ExecutorError::StepFailed(reason));
                    } else {
                        // All remaining nodes are running or waiting
                        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
                        continue;
                    }
                }
            };

            let node = plan
                .get_node_mut(&node_id)
                .ok_or_else(|| ExecutorError::StepNotFound(node_id.clone()))?;

            node.start();
            let attempt = node.attempt;
            let node_type = node.node_type.clone();
            let timeout_ms = if node.timeout_ms > 0 {
                node.timeout_ms
            } else {
                self.config.default_timeout_ms
            };

            Self::emit(
                &event_tx,
                RunnerEvent::StepStarted {
                    node_id: node_id.clone(),
                    attempt,
                },
            )
            .await;

            // Record step_started event
            let start_event = JobEvent {
                job_id: job_id.to_string(),
                version: 0, // will be set by store
                event_type: "step_started".to_string(),
                payload: serde_json::to_vec(&serde_json::json!({
                    "node_id": node_id,
                    "attempt": attempt,
                    "node_type": node_type,
                }))
                .unwrap_or_default(),
                prev_hash: String::new(),
                hash: String::new(),
                timestamp_ms: 0,
            };
            let _ = store.append(job_id, 0, &start_event).await;

            // Build step context
            let previous_outputs: Vec<(String, Vec<u8>)> = plan
                .nodes
                .iter()
                .filter(|n| n.status == PlanNodeStatus::Completed && n.output.is_some())
                .map(|n| (n.id.clone(), n.output.clone().unwrap()))
                .collect();

            let node = plan.get_node(&node_id).unwrap();
            let ctx = StepContext {
                job_id: job_id.to_string(),
                attempt_id: attempt_id.to_string(),
                node_id: node_id.clone(),
                step_name: node.name.clone(),
                input: node.config.clone(),
                previous_outputs,
            };

            // Execute with timeout
            let result = tokio::time::timeout(
                std::time::Duration::from_millis(timeout_ms),
                self.step_executor.execute(&ctx),
            )
            .await;

            let step_result = match result {
                Ok(Ok(r)) => r,
                Ok(Err(e)) => StepResult::retryable(&e.to_string(), 1000),
                Err(_) => StepResult::retryable(&format!("timeout after {timeout_ms}ms"), 2000),
            };

            // Record step result
            let node = plan.get_node_mut(&node_id).unwrap();

            if step_result.is_success() {
                let output = step_result.output.clone().unwrap_or_default();
                node.complete(output.clone());
                outputs.insert(node_id.clone(), output.clone());

                Self::emit(
                    &event_tx,
                    RunnerEvent::StepCompleted {
                        node_id: node_id.clone(),
                        result_type: step_result.result_type.clone(),
                    },
                )
                .await;

                // Record step_completed event
                let complete_event = JobEvent {
                    job_id: job_id.to_string(),
                    version: 0,
                    event_type: "step_completed".to_string(),
                    payload: serde_json::to_vec(&serde_json::json!({
                        "node_id": node_id,
                        "result_type": format!("{:?}", step_result.result_type),
                    }))
                    .unwrap_or_default(),
                    prev_hash: String::new(),
                    hash: String::new(),
                    timestamp_ms: 0,
                };
                let _ = store.append(job_id, 0, &complete_event).await;
            } else if step_result.is_retryable() {
                let will_retry = node.fail(step_result.error_message.clone());

                Self::emit(
                    &event_tx,
                    RunnerEvent::StepFailed {
                        node_id: node_id.clone(),
                        error: step_result.error_message.clone(),
                        will_retry,
                    },
                )
                .await;

                if will_retry {
                    // Wait before retry
                    tokio::time::sleep(std::time::Duration::from_millis(
                        step_result.retry_after_ms as u64,
                    ))
                    .await;
                    node.reset_for_retry();
                }
            } else if step_result.needs_compensation() {
                // Compensatable failure — run compensations then fail
                Self::emit(
                    &event_tx,
                    RunnerEvent::CompensationStarted {
                        node_id: node_id.clone(),
                    },
                )
                .await;

                node.fail(step_result.error_message.clone());
                if let Err(e) = self.compensation.compensate_all(job_id).await {
                    return Err(ExecutorError::CompensationFailed(e));
                }
            } else {
                // Permanent failure
                node.fail(step_result.error_message.clone());

                Self::emit(
                    &event_tx,
                    RunnerEvent::StepFailed {
                        node_id: node_id.clone(),
                        error: step_result.error_message,
                        will_retry: false,
                    },
                )
                .await;
            }

            // Checkpoint if configured
            if self.config.checkpoint_after_each_step {
                let checkpoint_data = checkpoint_mgr.create_checkpoint(plan)?;
                let checkpoint_event = JobEvent {
                    job_id: job_id.to_string(),
                    version: 0,
                    event_type: "checkpoint".to_string(),
                    payload: checkpoint_data,
                    prev_hash: String::new(),
                    hash: String::new(),
                    timestamp_ms: 0,
                };
                if let Ok(v) = store.append(job_id, 0, &checkpoint_event).await {
                    Self::emit(&event_tx, RunnerEvent::CheckpointCreated { version: v }).await;
                }
            }
        }
    }

    async fn emit(tx: &Option<mpsc::Sender<RunnerEvent>>, event: RunnerEvent) {
        if let Some(tx) = tx {
            let _ = tx.send(event).await;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::step::NoOpExecutor;
    use aetheris_jobstore::MemoryJobStore;
    use aetheris_planner::{PlanNode, PlanNodeStatus};

    fn make_plan() -> ExecutionPlan {
        ExecutionPlan {
            graph_id: "test".to_string(),
            nodes: vec![
                PlanNode {
                    id: "n1".to_string(),
                    name: "step1".to_string(),
                    node_type: "tool_call".to_string(),
                    config: vec![],
                    depends_on: vec![],
                    gates: vec!["n2".to_string()],
                    max_retries: 3,
                    timeout_ms: 5000,
                    status: PlanNodeStatus::Pending,
                    attempt: 0,
                    output: None,
                    error: None,
                    topo_order: 0,
                    parallel_group: 0,
                },
                PlanNode {
                    id: "n2".to_string(),
                    name: "step2".to_string(),
                    node_type: "llm_call".to_string(),
                    config: vec![],
                    depends_on: vec!["n1".to_string()],
                    gates: vec![],
                    max_retries: 3,
                    timeout_ms: 5000,
                    status: PlanNodeStatus::Pending,
                    attempt: 0,
                    output: None,
                    error: None,
                    topo_order: 1,
                    parallel_group: 1,
                },
            ],
            entry_node: "n1".to_string(),
        }
    }

    #[tokio::test]
    async fn test_runner_executes_plan() {
        let store = MemoryJobStore::new();
        let executor = Arc::new(NoOpExecutor);
        let compensation = Arc::new(CompensationRegistry::new());
        let config = RunnerConfig {
            checkpoint_after_each_step: false,
            ..Default::default()
        };
        let runner = Runner::new(config, executor, compensation);
        let mut plan = make_plan();

        // Create job first
        store
            .append(
                "job-test",
                0,
                &JobEvent {
                    job_id: "job-test".to_string(),
                    version: 0,
                    event_type: "job_created".to_string(),
                    payload: vec![],
                    prev_hash: String::new(),
                    hash: String::new(),
                    timestamp_ms: 0,
                },
            )
            .await
            .unwrap();

        runner
            .run(&store, "job-test", "attempt-1", &mut plan, None)
            .await
            .unwrap();

        assert!(plan.is_complete());
        assert!(!plan.has_failure());
    }
}
