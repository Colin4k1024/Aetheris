package middleware

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAuditStore persists API access audit logs.
type PostgresAuditStore struct {
	pool *pgxpool.Pool
}

func NewPostgresAuditStore(pool *pgxpool.Pool) *PostgresAuditStore {
	if pool == nil {
		return nil
	}
	return &PostgresAuditStore{pool: pool}
}

func (s *PostgresAuditStore) LogAccess(ctx context.Context, log AuditLog) error {
	if s == nil || s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO access_audit_log (tenant_id, user_id, action, resource_type, resource_id, success, duration_ms, client_ip, user_agent, request_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		log.TenantID, log.UserID, log.Action, log.ResourceType, log.ResourceID, log.Success, log.DurationMS, log.ClientIP, log.UserAgent, log.RequestID, log.CreatedAt,
	)
	return err
}
