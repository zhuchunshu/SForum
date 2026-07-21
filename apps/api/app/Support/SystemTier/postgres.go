package systemtier

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists system tier membership for cross-process CLI recovery.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore builds a durable system tier store.
func NewPostgresStore(pool *pgxpool.Pool) (*PostgresStore, error) {
	if pool == nil {
		return nil, ErrInvalid
	}
	return &PostgresStore{pool: pool}, nil
}

// Upsert implements Store.
func (s *PostgresStore) Upsert(ctx context.Context, member Member) error {
	if s == nil || s.pool == nil || ctx == nil {
		return ErrInvalid
	}
	if member.UpdatedAt.IsZero() {
		member.UpdatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO system_tier_members (extension_id, role, priority, enabled, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (extension_id) DO UPDATE SET
			role = EXCLUDED.role,
			priority = EXCLUDED.priority,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`, member.ExtensionID, member.Role, member.Priority, member.Enabled, member.UpdatedAt, member.UpdatedBy)
	return err
}

// Disable implements Store.
func (s *PostgresStore) Disable(ctx context.Context, extensionID, actor string) error {
	if s == nil || s.pool == nil || ctx == nil {
		return ErrInvalid
	}
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	tag, err := s.pool.Exec(ctx, `
		UPDATE system_tier_members
		SET enabled = FALSE, updated_at = statement_timestamp(), updated_by = $2
		WHERE extension_id = $1
	`, extensionID, strings.TrimSpace(actor))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Get implements Store.
func (s *PostgresStore) Get(ctx context.Context, extensionID string) (Member, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return Member{}, ErrInvalid
	}
	var member Member
	err := s.pool.QueryRow(ctx, `
		SELECT extension_id, role, priority, enabled, updated_at, updated_by
		FROM system_tier_members WHERE extension_id = $1
	`, strings.ToLower(strings.TrimSpace(extensionID))).Scan(
		&member.ExtensionID, &member.Role, &member.Priority, &member.Enabled,
		&member.UpdatedAt, &member.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrNotFound
	}
	return member, err
}

// List implements Store.
func (s *PostgresStore) List(ctx context.Context) ([]Member, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return nil, ErrInvalid
	}
	rows, err := s.pool.Query(ctx, `
		SELECT extension_id, role, priority, enabled, updated_at, updated_by
		FROM system_tier_members ORDER BY extension_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Member, 0)
	for rows.Next() {
		var member Member
		if err := rows.Scan(
			&member.ExtensionID, &member.Role, &member.Priority, &member.Enabled,
			&member.UpdatedAt, &member.UpdatedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, member)
	}
	return out, rows.Err()
}

var _ Store = (*PostgresStore)(nil)
