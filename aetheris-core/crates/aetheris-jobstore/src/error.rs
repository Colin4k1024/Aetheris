//! Job store error types.

use aetheris_types::AetherisError;

/// Errors from job store operations.
#[derive(Debug, thiserror::Error)]
pub enum JobStoreError {
    #[error("version conflict: expected {expected}, got {actual}")]
    VersionConflict { expected: i32, actual: i32 },

    #[error("database error: {0}")]
    Database(String),

    #[error("timeout: {0}")]
    Timeout(String),

    #[error("invalid input: {0}")]
    InvalidInput(String),

    #[error("internal error: {0}")]
    Internal(String),
}

impl JobStoreError {
    /// Convert to FFI error code.
    pub fn to_ffi_code(&self) -> i32 {
        match self {
            Self::VersionConflict { .. } => AetherisError::VersionConflict as i32,
            Self::Database(_) => AetherisError::DatabaseError as i32,
            Self::Timeout(_) => AetherisError::Timeout as i32,
            Self::InvalidInput(_) => AetherisError::InvalidInput as i32,
            Self::Internal(_) => AetherisError::InternalError as i32,
        }
    }
}

impl From<JobStoreError> for i32 {
    fn from(e: JobStoreError) -> i32 {
        e.to_ffi_code()
    }
}
