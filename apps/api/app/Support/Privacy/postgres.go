package privacy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAuditor persists privacy export/erase audits without payload bodies.
type PostgresAuditor struct {
	pool *pgxpool.Pool
}

// NewPostgresAuditor builds a durable privacy audit sink (migration 202607220046).
func NewPostgresAuditor(pool *pgxpool.Pool) (*PostgresAuditor, error) {
	if pool == nil {
		return nil, ErrInvalid
	}
	return &PostgresAuditor{pool: pool}, nil
}

// Append implements Auditor.
func (a *PostgresAuditor) Append(ctx context.Context, event AuditEvent) error {
	if a == nil || a.pool == nil {
		return ErrInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	auditID := strings.TrimSpace(event.AuditID)
	if auditID == "" {
		auditID = fmt.Sprintf("privacy-%d", time.Now().UTC().UnixNano())
	}
	operation := strings.ToLower(strings.TrimSpace(event.Operation))
	status := strings.ToLower(strings.TrimSpace(event.Status))
	if operation == "" || status == "" {
		return ErrInvalid
	}
	detail := map[string]string{}
	if strings.TrimSpace(event.Detail) != "" {
		detail["message"] = event.Detail
	}
	blob, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	at := event.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err = a.pool.Exec(ctx, `
		INSERT INTO privacy_operation_audit (audit_id, operation, actor, user_id, status, detail, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		ON CONFLICT (audit_id) DO NOTHING
	`, auditID, operation, event.Actor, event.UserID, status, string(blob), at)
	if err != nil {
		return fmt.Errorf("privacy audit append: %w", err)
	}
	return nil
}

var _ Auditor = (*PostgresAuditor)(nil)
