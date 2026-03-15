// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"testing"
)

// 内存 RoleStore 实现（用于测试）
type memRoleStore struct {
	roles map[string]Role // key: tenantID:userID
}

func newMemRoleStore() *memRoleStore {
	return &memRoleStore{
		roles: make(map[string]Role),
	}
}

func (m *memRoleStore) GetUserRole(ctx context.Context, tenantID string, userID string) (Role, error) {
	key := tenantID + ":" + userID
	role, ok := m.roles[key]
	if !ok {
		return RoleUser, nil // 默认 user 角色
	}
	return role, nil
}

func (m *memRoleStore) SetUserRole(ctx context.Context, tenantID string, userID string, role Role) error {
	key := tenantID + ":" + userID
	m.roles[key] = role
	return nil
}

// TestRBAC_AdminHasAllPermissions Admin 角色拥有所有权限
func TestRBAC_AdminHasAllPermissions(t *testing.T) {
	store := newMemRoleStore()
	store.SetUserRole(context.Background(), "tenant1", "user1", RoleAdmin)

	rbac := NewSimpleRBACChecker(store)

	permissions := []Permission{
		PermissionJobView,
		PermissionJobCreate,
		PermissionJobExport,
		PermissionAuditView,
	}

	for _, perm := range permissions {
		allowed, err := rbac.CheckPermission(context.Background(), "tenant1", "user1", perm, "")
		if err != nil {
			t.Errorf("permission check failed: %v", err)
		}
		if !allowed {
			t.Errorf("admin should have permission %s", perm)
		}
	}
}

// TestRBAC_UserCannotExport User 角色不能导出
func TestRBAC_UserCannotExport(t *testing.T) {
	store := newMemRoleStore()
	store.SetUserRole(context.Background(), "tenant1", "user2", RoleUser)

	rbac := NewSimpleRBACChecker(store)

	allowed, err := rbac.CheckPermission(context.Background(), "tenant1", "user2", PermissionJobExport, "")
	if err != nil {
		t.Errorf("permission check failed: %v", err)
	}
	if allowed {
		t.Error("user should not have export permission")
	}
}

// TestRBAC_TenantIsolation 不同 tenant 之间隔离
func TestRBAC_TenantIsolation(t *testing.T) {
	store := newMemRoleStore()
	store.SetUserRole(context.Background(), "tenant1", "user1", RoleAdmin)
	store.SetUserRole(context.Background(), "tenant2", "user1", RoleUser)

	rbac := NewSimpleRBACChecker(store)

	// user1 在 tenant1 是 admin
	role1, _ := rbac.GetUserRole(context.Background(), "tenant1", "user1")
	if role1 != RoleAdmin {
		t.Errorf("expected admin role in tenant1, got %s", role1)
	}

	// user1 在 tenant2 是 user
	role2, _ := rbac.GetUserRole(context.Background(), "tenant2", "user1")
	if role2 != RoleUser {
		t.Errorf("expected user role in tenant2, got %s", role2)
	}
}

// TestRBAC_AuditorCanViewAndExport Auditor 可以查看和导出，但不能创建
func TestRBAC_AuditorCanViewAndExport(t *testing.T) {
	store := newMemRoleStore()
	store.SetUserRole(context.Background(), "tenant1", "auditor1", RoleAuditor)

	rbac := NewSimpleRBACChecker(store)

	// 可以查看
	if allowed, _ := rbac.CheckPermission(context.Background(), "tenant1", "auditor1", PermissionJobView, ""); !allowed {
		t.Error("auditor should have view permission")
	}

	// 可以导出
	if allowed, _ := rbac.CheckPermission(context.Background(), "tenant1", "auditor1", PermissionJobExport, ""); !allowed {
		t.Error("auditor should have export permission")
	}

	// 不能创建
	if allowed, _ := rbac.CheckPermission(context.Background(), "tenant1", "auditor1", PermissionJobCreate, ""); allowed {
		t.Error("auditor should not have create permission")
	}

	// 不能停止
	if allowed, _ := rbac.CheckPermission(context.Background(), "tenant1", "auditor1", PermissionJobStop, ""); allowed {
		t.Error("auditor should not have stop permission")
	}
}

func TestContext_WithTenantID(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenantID(ctx, "tenant-123")

	tenantID := GetTenantID(ctx)
	if tenantID != "tenant-123" {
		t.Errorf("expected tenant-123, got %s", tenantID)
	}
}

func TestContext_GetTenantID_Empty(t *testing.T) {
	ctx := context.Background()
	tenantID := GetTenantID(ctx)
	if tenantID != "" {
		t.Errorf("expected empty, got %s", tenantID)
	}
}

func TestContext_WithUserID(t *testing.T) {
	ctx := context.Background()
	ctx = WithUserID(ctx, "user-456")

	userID := GetUserID(ctx)
	if userID != "user-456" {
		t.Errorf("expected user-456, got %s", userID)
	}
}

func TestContext_GetUserID_Empty(t *testing.T) {
	ctx := context.Background()
	userID := GetUserID(ctx)
	if userID != "" {
		t.Errorf("expected empty, got %s", userID)
	}
}

func TestContext_WithRole(t *testing.T) {
	ctx := context.Background()
	ctx = WithRole(ctx, RoleAdmin)

	role := GetRole(ctx)
	if role != RoleAdmin {
		t.Errorf("expected RoleAdmin, got %s", role)
	}
}

func TestContext_GetRole_Default(t *testing.T) {
	ctx := context.Background()
	role := GetRole(ctx)
	if role != RoleUser {
		t.Errorf("expected RoleUser default, got %s", role)
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		role       Role
		permission Permission
		expected   bool
	}{
		{RoleAdmin, PermissionJobCreate, true},
		{RoleAdmin, PermissionJobView, true},
		{RoleUser, PermissionJobView, true},
		{RoleUser, PermissionJobExport, false},
		{RoleAuditor, PermissionJobExport, true},
		{RoleAuditor, PermissionJobCreate, false},
		{RoleOperator, PermissionJobStop, true},
	}

	for _, tt := range tests {
		result := HasPermission(tt.role, tt.permission)
		if result != tt.expected {
			t.Errorf("HasPermission(%s, %s) = %v, want %v", tt.role, tt.permission, result, tt.expected)
		}
	}
}
