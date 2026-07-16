//! Execution plan — the compiled form of a TaskGraph.

use serde::{Deserialize, Serialize};

/// Status of a node in the execution plan.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub enum PlanNodeStatus {
    /// Waiting for dependencies to complete.
    Pending,
    /// Ready to execute (all dependencies satisfied).
    Ready,
    /// Currently executing.
    Running,
    /// Completed successfully.
    Completed,
    /// Failed (may be retried).
    Failed,
    /// Permanently failed (no more retries).
    PermanentFailure,
    /// Skipped (conditional branch not taken).
    Skipped,
}

/// A single node in the execution plan.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct PlanNode {
    /// Unique node identifier (from TaskGraph).
    pub id: String,
    /// Human-readable name.
    pub name: String,
    /// Node type (tool_call, llm_call, condition, etc.).
    pub node_type: String,
    /// Node configuration (JSON-encoded).
    pub config: Vec<u8>,
    /// IDs of nodes that must complete before this one.
    pub depends_on: Vec<String>,
    /// IDs of nodes that this one gates.
    pub gates: Vec<String>,
    /// Maximum retry attempts.
    pub max_retries: u32,
    /// Timeout in milliseconds.
    pub timeout_ms: u64,
    /// Current status.
    pub status: PlanNodeStatus,
    /// Current attempt number.
    pub attempt: u32,
    /// Optional output from execution.
    pub output: Option<Vec<u8>>,
    /// Optional error message.
    pub error: Option<String>,
    /// Topological order (lower = earlier).
    pub topo_order: usize,
    /// Parallel group index (nodes in the same group can run concurrently).
    pub parallel_group: usize,
}

impl PlanNode {
    /// Check if all dependencies are satisfied.
    pub fn dependencies_satisfied(&self, completed_nodes: &[&str]) -> bool {
        self.depends_on
            .iter()
            .all(|dep| completed_nodes.contains(&dep.as_str()))
    }

    /// Mark as running.
    pub fn start(&mut self) {
        self.status = PlanNodeStatus::Running;
        self.attempt += 1;
    }

    /// Mark as completed with output.
    pub fn complete(&mut self, output: Vec<u8>) {
        self.status = PlanNodeStatus::Completed;
        self.output = Some(output);
    }

    /// Mark as failed. Returns true if retryable.
    pub fn fail(&mut self, error: String) -> bool {
        if self.attempt < self.max_retries {
            self.status = PlanNodeStatus::Failed;
            self.error = Some(error);
            true
        } else {
            self.status = PlanNodeStatus::PermanentFailure;
            self.error = Some(error);
            false
        }
    }

    /// Reset to pending for retry.
    pub fn reset_for_retry(&mut self) {
        self.status = PlanNodeStatus::Pending;
    }

    /// Mark as skipped.
    pub fn skip(&mut self) {
        self.status = PlanNodeStatus::Skipped;
    }
}

/// The compiled execution plan — a DAG with topological ordering.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ExecutionPlan {
    /// Plan/graph identifier.
    pub graph_id: String,
    /// All nodes in topological order.
    pub nodes: Vec<PlanNode>,
    /// Entry node ID.
    pub entry_node: String,
}

impl ExecutionPlan {
    /// Get a node by ID.
    pub fn get_node(&self, id: &str) -> Option<&PlanNode> {
        self.nodes.iter().find(|n| n.id == id)
    }

    /// Get a mutable node by ID.
    pub fn get_node_mut(&mut self, id: &str) -> Option<&mut PlanNode> {
        self.nodes.iter_mut().find(|n| n.id == id)
    }

    /// Get all nodes that are ready to execute.
    pub fn ready_nodes(&self) -> Vec<&PlanNode> {
        let completed: Vec<&str> = self
            .nodes
            .iter()
            .filter(|n| n.status == PlanNodeStatus::Completed)
            .map(|n| n.id.as_str())
            .collect();

        self.nodes
            .iter()
            .filter(|n| {
                n.status == PlanNodeStatus::Pending && n.dependencies_satisfied(&completed)
            })
            .collect()
    }

    /// Check if all nodes are in a terminal state.
    pub fn is_complete(&self) -> bool {
        self.nodes.iter().all(|n| {
            matches!(
                n.status,
                PlanNodeStatus::Completed
                    | PlanNodeStatus::PermanentFailure
                    | PlanNodeStatus::Skipped
            )
        })
    }

    /// Check if any node has permanently failed.
    pub fn has_failure(&self) -> bool {
        self.nodes
            .iter()
            .any(|n| n.status == PlanNodeStatus::PermanentFailure)
    }

    /// Get the next node to execute (first ready node in topo order).
    pub fn next_to_execute(&self) -> Option<&PlanNode> {
        self.ready_nodes().into_iter().next()
    }

    /// Serialize to JSON bytes (for checkpoint storage).
    pub fn to_json_bytes(&self) -> Vec<u8> {
        serde_json::to_vec(self).unwrap_or_default()
    }

    /// Deserialize from JSON bytes.
    pub fn from_json_bytes(data: &[u8]) -> Option<Self> {
        serde_json::from_slice(data).ok()
    }
}
