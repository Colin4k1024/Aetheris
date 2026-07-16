//! Checkpoint manager — creates and restores execution checkpoints.

use aetheris_planner::ExecutionPlan;

use crate::error::ExecutorError;

/// Manages checkpoints for crash recovery.
pub struct CheckpointManager {
    job_id: String,
}

impl CheckpointManager {
    pub fn new(job_id: impl Into<String>) -> Self {
        Self {
            job_id: job_id.into(),
        }
    }

    /// Create a checkpoint from the current execution plan state.
    pub fn create_checkpoint(&self, plan: &ExecutionPlan) -> Result<Vec<u8>, ExecutorError> {
        let checkpoint = CheckpointData {
            job_id: self.job_id.clone(),
            plan: plan.clone(),
            timestamp_ms: now_ms(),
        };

        serde_json::to_vec(&checkpoint)
            .map_err(|e| ExecutorError::CheckpointError(e.to_string()))
    }

    /// Restore an execution plan from checkpoint data.
    pub fn restore_plan(data: &[u8]) -> Result<ExecutionPlan, ExecutorError> {
        let checkpoint: CheckpointData = serde_json::from_slice(data)
            .map_err(|e| ExecutorError::CheckpointError(e.to_string()))?;
        Ok(checkpoint.plan)
    }
}

#[derive(serde::Serialize, serde::Deserialize)]
struct CheckpointData {
    job_id: String,
    plan: ExecutionPlan,
    timestamp_ms: i64,
}

fn now_ms() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

#[cfg(test)]
mod tests {
    use super::*;
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
                    timeout_ms: 30000,
                    status: PlanNodeStatus::Completed,
                    attempt: 1,
                    output: Some(b"done".to_vec()),
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
                    timeout_ms: 30000,
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

    #[test]
    fn test_checkpoint_roundtrip() {
        let plan = make_plan();
        let mgr = CheckpointManager::new("job-1");

        let data = mgr.create_checkpoint(&plan).unwrap();
        let restored = CheckpointManager::restore_plan(&data).unwrap();

        assert_eq!(restored.graph_id, plan.graph_id);
        assert_eq!(restored.nodes.len(), plan.nodes.len());
        assert_eq!(restored.nodes[0].status, PlanNodeStatus::Completed);
        assert_eq!(restored.nodes[1].status, PlanNodeStatus::Pending);
    }
}
