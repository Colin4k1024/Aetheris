//! Auth error types.

#[derive(Debug, thiserror::Error)]
pub enum AuthError {
    #[error("permission denied: {0}")]
    PermissionDenied(String),

    #[error("role not found: {0}")]
    RoleNotFound(String),

    #[error("user not found: {0}")]
    UserNotFound(String),

    #[error("storage error: {0}")]
    Storage(String),
}
