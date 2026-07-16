//! RBAC engine — evaluates authorization requests.

use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::RwLock;

use crate::error::AuthError;
use crate::types::*;
use crate::RbacStore;

/// In-memory RBAC store for testing.
pub struct InMemoryRbacStore {
    /// (user_id, tenant_id) -> roles
    user_roles: Arc<RwLock<HashMap<(String, String), Vec<Role>>>>,
    /// role -> permissions
    role_permissions: Arc<RwLock<HashMap<Role, Vec<Permission>>>>,
}

impl InMemoryRbacStore {
    pub fn new() -> Self {
        let mut role_permissions = HashMap::new();
        // Initialize with default permissions for built-in roles
        for role in &[Role::Admin, Role::AgentManager, Role::Operator, Role::Viewer] {
            role_permissions.insert(role.clone(), default_permissions(role));
        }

        Self {
            user_roles: Arc::new(RwLock::new(HashMap::new())),
            role_permissions: Arc::new(RwLock::new(role_permissions)),
        }
    }

    /// Assign a role to a user in a tenant.
    pub async fn assign_role(
        &self,
        user_id: &str,
        tenant_id: &str,
        role: Role,
    ) -> Result<(), AuthError> {
        let mut user_roles = self.user_roles.write().await;
        let key = (user_id.to_string(), tenant_id.to_string());
        let roles = user_roles.entry(key).or_default();
        if !roles.contains(&role) {
            roles.push(role);
        }
        Ok(())
    }

    /// Set custom permissions for a role.
    pub async fn set_permissions(
        &self,
        role: Role,
        permissions: Vec<Permission>,
    ) -> Result<(), AuthError> {
        let mut role_permissions = self.role_permissions.write().await;
        role_permissions.insert(role, permissions);
        Ok(())
    }
}

impl Default for InMemoryRbacStore {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl RbacStore for InMemoryRbacStore {
    async fn get_user_roles(
        &self,
        user_id: &str,
        tenant_id: &str,
    ) -> Result<Vec<Role>, AuthError> {
        let user_roles = self.user_roles.read().await;
        let key = (user_id.to_string(), tenant_id.to_string());
        Ok(user_roles.get(&key).cloned().unwrap_or_default())
    }

    async fn get_role_permissions(&self, role: &Role) -> Result<Vec<Permission>, AuthError> {
        let role_permissions = self.role_permissions.read().await;
        Ok(role_permissions.get(role).cloned().unwrap_or_default())
    }
}

/// RBAC engine that evaluates authorization requests.
pub struct RbacEngine {
    store: Arc<dyn RbacStore>,
}

impl RbacEngine {
    pub fn new(store: Arc<dyn RbacStore>) -> Self {
        Self { store }
    }

    /// Check if a user has a specific permission in a tenant.
    pub async fn check_permission(
        &self,
        user_id: &str,
        tenant_id: &str,
        required: &Permission,
    ) -> Result<AuthResult, AuthError> {
        let roles = self.store.get_user_roles(user_id, tenant_id).await?;

        if roles.is_empty() {
            return Ok(AuthResult {
                allowed: false,
                reason: format!("user {user_id} has no roles in tenant {tenant_id}"),
            });
        }

        for role in &roles {
            let permissions = self.store.get_role_permissions(role).await?;
            if permissions.contains(required) {
                return Ok(AuthResult {
                    allowed: true,
                    reason: format!(
                        "user {user_id} has role {:?} with permission {:?}",
                        role, required
                    ),
                });
            }
        }

        Ok(AuthResult {
            allowed: false,
            reason: format!(
                "user {user_id} lacks permission {:?} in tenant {tenant_id}",
                required
            ),
        })
    }

    /// Evaluate an auth request.
    pub async fn authorize(&self, request: &AuthRequest) -> Result<AuthResult, AuthError> {
        self.check_permission(&request.user_id, &request.tenant_id, &request.action)
            .await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_admin_has_all_permissions() {
        let store = Arc::new(InMemoryRbacStore::new());
        store
            .assign_role("alice", "tenant-1", Role::Admin)
            .await
            .unwrap();

        let engine = RbacEngine::new(store);

        let result = engine
            .check_permission("alice", "tenant-1", &Permission::CreateAgent)
            .await
            .unwrap();
        assert!(result.allowed);

        let result = engine
            .check_permission("alice", "tenant-1", &Permission::ManageConfig)
            .await
            .unwrap();
        assert!(result.allowed);
    }

    #[tokio::test]
    async fn test_viewer_cannot_manage() {
        let store = Arc::new(InMemoryRbacStore::new());
        store
            .assign_role("bob", "tenant-1", Role::Viewer)
            .await
            .unwrap();

        let engine = RbacEngine::new(store);

        let result = engine
            .check_permission("bob", "tenant-1", &Permission::ViewJobs)
            .await
            .unwrap();
        assert!(result.allowed);

        let result = engine
            .check_permission("bob", "tenant-1", &Permission::ManageJobs)
            .await
            .unwrap();
        assert!(!result.allowed);
    }

    #[tokio::test]
    async fn test_no_role_denied() {
        let store = Arc::new(InMemoryRbacStore::new());
        let engine = RbacEngine::new(store);

        let result = engine
            .check_permission("unknown", "tenant-1", &Permission::ViewJobs)
            .await
            .unwrap();
        assert!(!result.allowed);
    }

    #[tokio::test]
    async fn test_custom_role_with_permissions() {
        let store = Arc::new(InMemoryRbacStore::new());
        let custom_role = Role::Custom("deployer".to_string());
        store
            .set_permissions(
                custom_role.clone(),
                vec![Permission::ManageJobs, Permission::ViewMetrics],
            )
            .await
            .unwrap();
        store
            .assign_role("charlie", "tenant-1", custom_role)
            .await
            .unwrap();

        let engine = RbacEngine::new(store);

        let result = engine
            .check_permission("charlie", "tenant-1", &Permission::ManageJobs)
            .await
            .unwrap();
        assert!(result.allowed);

        let result = engine
            .check_permission("charlie", "tenant-1", &Permission::CreateAgent)
            .await
            .unwrap();
        assert!(!result.allowed);
    }
}
