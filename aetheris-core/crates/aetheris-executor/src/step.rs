//! Step executor trait — the pluggable execution interface.

use async_trait::async_trait;

use crate::error::ExecutorError;
use crate::types::{StepContext, StepResult};

/// The outcome of step execution — used by the runner to decide next action.
pub type StepOutcome = StepResult;

/// Trait for executing individual DAG nodes.
///
/// Implementations handle specific node types (tool calls, LLM calls, conditions, etc.)
/// The runner calls `execute` for each node in the DAG.
#[async_trait]
pub trait StepExecutor: Send + Sync {
    /// Execute a single step and return the outcome.
    async fn execute(&self, ctx: &StepContext) -> Result<StepOutcome, ExecutorError>;

    /// Check if this executor can handle the given node type.
    fn can_handle(&self, node_type: &str) -> bool;
}

/// A no-op executor that always returns success (for testing).
pub struct NoOpExecutor;

#[async_trait]
impl StepExecutor for NoOpExecutor {
    async fn execute(&self, _ctx: &StepContext) -> Result<StepOutcome, ExecutorError> {
        Ok(StepResult::success(b"no-op".to_vec()))
    }

    fn can_handle(&self, _node_type: &str) -> bool {
        true
    }
}
