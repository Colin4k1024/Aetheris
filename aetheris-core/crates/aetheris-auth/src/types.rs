//! RBAC types.

use serde::{Deserialize, Serialize};

/// Roles in the system.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum Role {
    /// Full system access.
    Admin,
    /// Can create and manage agents.
    AgentManager,
    /// Can execute jobs and view results.
    Operator,
    /// Read-only access to job history and traces.
    Viewer,
    /// Custom role with specific permissions.
    Custom(String),
}

/// Permissions that can be granted to roles.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum Permission {
    /// Create agents.
    CreateAgent,
    /// Delete agents.
    DeleteAgent,
    /// Start/stop jobs.
    ManageJobs,
    /// View job history and traces.
    ViewJobs,
    /// Approve human-in-the-loop gates.
    ApproveDecisions,
    /// View system metrics and health.
    ViewMetrics,
    /// Modify system configuration.
    ManageConfig,
    /// Access compliance and forensics data.
    AccessCompliance,
    /// Custom permission.
    Custom(String),
}

/// A request to check authorization.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct AuthRequest {
    pub user_id: String,
    pub tenant_id: String,
    pub action: Permission,
    pub resource: String,
}

/// Result of an authorization check.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct AuthResult {
    pub allowed: bool,
    pub reason: String,
}

/// Default permissions for each role.
pub fn default_permissions(role: &Role) -> Vec<Permission> {
    match role {
        Role::Admin => vec![
            Permission::CreateAgent,
            Permission::DeleteAgent,
            Permission::ManageJobs,
            Permission::ViewJobs,
            Permission::ApproveDecisions,
            Permission::ViewMetrics,
            Permission::ManageConfig,
            Permission::AccessCompliance,
        ],
        Role::AgentManager => vec![
            Permission::CreateAgent,
            Permission::DeleteAgent,
            Permission::ManageJobs,
            Permission::ViewJobs,
            Permission::ViewMetrics,
        ],
        Role::Operator => vec![
            Permission::ManageJobs,
            Permission::ViewJobs,
            Permission::ApproveDecisions,
            Permission::ViewMetrics,
        ],
        Role::Viewer => vec![
            Permission::ViewJobs,
            Permission::ViewMetrics,
        ],
        Role::Custom(_) => vec![], // Custom roles must have permissions assigned explicitly
    }
}
