//! Memory error types.

#[derive(Debug, thiserror::Error)]
pub enum MemoryError {
    #[error("entry not found: {0}")]
    NotFound(String),

    #[error("storage error: {0}")]
    Storage(String),

    #[error("invalid input: {0}")]
    InvalidInput(String),

    #[error("capacity exceeded: {0}")]
    CapacityExceeded(String),
}
