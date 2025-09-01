package postgres

import (
	"context"

	"github.com/baharkarakas/insider-backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditLogsRepo struct { pool *pgxpool.Pool }

func (r *auditLogsRepo) Create(l models.AuditLog) error {
	_, err := r.pool.Exec(context.Background(), `INSERT INTO audit_logs(entity_type, entity_id, action, details) VALUES($1,$2,$3,$4)`, l.EntityType, l.EntityID, l.Action, l.Details)
	return err
}