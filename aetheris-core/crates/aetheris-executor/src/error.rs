//! Executor error types.

#[derive(Debug, thiserror::Error)]
pub enum ExecutorError {
    #[error("step not found: {0}")]
    StepNotFound(String),

    #[error("step failed: {0}")]
    StepFailed(String),

    #[error("step timeout after {0}ms")]
    StepTimeout(u64),

    #[error("compensation failed: {0}")]
    CompensationFailed(String),

    #[error("checkpoint error: {0}")]
    CheckpointError(String),

    #[error("replay error: {0}")]
    ReplayError(String),

    #[error("plan error: {0}")]
    PlanError(String),

    #[error("job store error: {0}")]
    JobStore(String),

    #[error("internal error: {0}")]
    Internal(String),
}
