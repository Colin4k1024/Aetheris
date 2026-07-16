//! Effect error types.

use aetheris_types::AetherisError;

#[derive(Debug, thiserror::Error)]
pub enum EffectError {
    #[error("effect not found: {0}")]
    NotFound(String),

    #[error("effect already committed: {0}")]
    AlreadyCommitted(String),

    #[error("effect already rolled back: {0}")]
    AlreadyRolledBack(String),

    #[error("idempotency conflict: {0}")]
    IdempotencyConflict(String),

    #[error("database error: {0}")]
    Database(String),

    #[error("invalid input: {0}")]
    InvalidInput(String),

    #[error("internal error: {0}")]
    Internal(String),
}

impl EffectError {
    pub fn to_ffi_code(&self) -> i32 {
        match self {
            Self::NotFound(_) => AetherisError::InvalidInput as i32,
            Self::AlreadyCommitted(_) | Self::AlreadyRolledBack(_) => {
                AetherisError::VersionConflict as i32
            }
            Self::IdempotencyConflict(_) => AetherisError::VersionConflict as i32,
            Self::Database(_) => AetherisError::DatabaseError as i32,
            Self::InvalidInput(_) => AetherisError::InvalidInput as i32,
            Self::Internal(_) => AetherisError::InternalError as i32,
        }
    }
}
