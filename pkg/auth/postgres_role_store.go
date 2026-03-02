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
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRoleStore Postgres 角色存储实现
type PostgresRoleStore struct {
	pool *pgxpool.Pool
}

// NewPostgresRoleStore 创建 Postgres RoleStore
func NewPostgresRoleStore(pool *pgxpool.Pool) (*PostgresRoleStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("PostgresRoleStore: pool is nil")
	}
	return &PostgresRoleStore{pool: pool}, nil
}

// InitSchema 初始化表结构
func (s *PostgresRoleStore) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS rbac_roles (
		id SERIAL PRIMARY KEY,
		tenant_id VARCHAR(255) NOT NULL,
		user_id VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		UNIQUE(tenant_id, user_id)
	);
	CREATE INDEX IF NOT EXISTS idx_rbac_roles_tenant_user ON rbac_roles(tenant_id, user_id);
	`
	_, err := s.pool.Exec(ctx, schema)
	return err
}

// GetUserRole 获取用户在租户中的角色
func (s *PostgresRoleStore) GetUserRole(ctx context.Context, tenantID string, userID string) (Role, error) {
	var role string
	err := s.pool.QueryRow(ctx,
		"SELECT role FROM rbac_roles WHERE tenant_id = $1 AND user_id = $2",
		tenantID, userID,
	).Scan(&role)

	if err != nil {
		// 未找到时返回默认角色
		return RoleUser, nil
	}

	return Role(role), nil
}

// SetUserRole 设置用户在租户中的角色
func (s *PostgresRoleStore) SetUserRole(ctx context.Context, tenantID string, userID string, role Role) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO rbac_roles (tenant_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (tenant_id, user_id)
		DO UPDATE SET role = $3, updated_at = NOW()`,
		tenantID, userID, string(role),
	)
	return err
}

// DeleteUserRole 删除用户角色
func (s *PostgresRoleStore) DeleteUserRole(ctx context.Context, tenantID string, userID string) error {
	_, err := s.pool.Exec(ctx,
		"DELETE FROM rbac_roles WHERE tenant_id = $1 AND user_id = $2",
		tenantID, userID,
	)
	return err
}

// ListUserRoles 列出租户中所有用户角色
func (s *PostgresRoleStore) ListUserRoles(ctx context.Context, tenantID string) ([]UserRole, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT user_id, role, created_at, updated_at FROM rbac_roles WHERE tenant_id = $1 ORDER BY user_id",
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UserRole
	for rows.Next() {
		var ur UserRole
		if err := rows.Scan(&ur.UserID, &ur.Role, &ur.CreatedAt, &ur.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, ur)
	}
	return results, nil
}

// UserRole 用户角色信息
type UserRole struct {
	UserID    string
	Role      Role
	CreatedAt interface{}
	UpdatedAt interface{}
}
