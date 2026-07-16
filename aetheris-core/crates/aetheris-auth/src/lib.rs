//! Aetheris auth crate.
//!
//! Role-Based Access Control (RBAC) for multi-tenant agent execution.
//!
//! Model: Users belong to Tenants, have Roles, and Roles grant Permissions.
//! Policy checks determine whether a principal can perform an action on a resource.

mod error;
mod rbac;
mod types;

pub use error::AuthError;
pub use rbac::RbacEngine;
pub use types::*;

use async_trait::async_trait;

/// Trait for loading RBAC data (user roles, permissions).
#[async_trait]
pub trait RbacStore: Send + Sync {
    /// Get roles for a user in a tenant.
    async fn get_user_roles(
        &self,
        user_id: &str,
        tenant_id: &str,
    ) -> Result<Vec<Role>, AuthError>;

    /// Get permissions for a role.
    async fn get_role_permissions(
        &self,
        role: &Role,
    ) -> Result<Vec<Permission>, AuthError>;

    /// Check if a user has a specific role in a tenant.
    async fn user_has_role(
        &self,
        user_id: &str,
        tenant_id: &str,
        role: &Role,
    ) -> Result<bool, AuthError> {
        let roles = self.get_user_roles(user_id, tenant_id).await?;
        Ok(roles.contains(role))
    }
}
