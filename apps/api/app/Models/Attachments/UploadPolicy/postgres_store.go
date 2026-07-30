package uploadpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) RoleLimitsForUser(ctx context.Context, userID int64) ([]RoleLimit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT roles.key, policies.max_file_size_bytes
		FROM user_roles
		JOIN roles ON roles.id = user_roles.role_id AND roles.is_enabled = TRUE
		JOIN role_permissions ON role_permissions.role_id = roles.id
		LEFT JOIN attachment_role_upload_policies policies ON policies.role_id = roles.id
		WHERE user_roles.user_id = $1
		  AND role_permissions.permission_key = 'attachment.upload'
		ORDER BY roles.key
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list attachment upload role limits: %w", err)
	}
	defer rows.Close()

	items := []RoleLimit{}
	for rows.Next() {
		var item RoleLimit
		var limit pgtype.Int8
		if err := rows.Scan(&item.RoleKey, &limit); err != nil {
			return nil, fmt.Errorf("scan attachment upload role limit: %w", err)
		}
		if limit.Valid {
			value := limit.Int64
			item.MaxFileSizeBytes = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachment upload role limits: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) UserLimit(ctx context.Context, userID int64) (*int64, error) {
	var value int64
	err := s.pool.QueryRow(ctx, `
		SELECT max_file_size_bytes
		FROM attachment_user_upload_policies
		WHERE user_id = $1
	`, userID).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get attachment upload user limit: %w", err)
	}
	return &value, nil
}

func (s *PostgresStore) ListRolePolicies(ctx context.Context) ([]StoredRolePolicy, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT roles.key, roles.alias, roles.is_enabled,
		       roles.key = 'super_admin' OR EXISTS (
		         SELECT 1 FROM role_permissions
		         WHERE role_permissions.role_id = roles.id
		           AND role_permissions.permission_key = 'attachment.upload'
		       ) AS grants_upload,
		       policies.max_file_size_bytes, policies.updated_at
		FROM roles
		LEFT JOIN attachment_role_upload_policies policies ON policies.role_id = roles.id
		ORDER BY roles.is_system DESC, roles.alias, roles.key
	`)
	if err != nil {
		return nil, fmt.Errorf("list attachment upload role policies: %w", err)
	}
	defer rows.Close()

	items := []StoredRolePolicy{}
	for rows.Next() {
		var item StoredRolePolicy
		var limit pgtype.Int8
		var updatedAt pgtype.Timestamptz
		if err := rows.Scan(&item.RoleKey, &item.Alias, &item.Enabled, &item.GrantsUpload, &limit, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan attachment upload role policy: %w", err)
		}
		if limit.Valid {
			value := limit.Int64
			item.MaxFileSizeBytes = &value
		}
		if updatedAt.Valid {
			value := updatedAt.Time
			item.UpdatedAt = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachment upload role policies: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) GetUserPolicy(ctx context.Context, userID int64) (StoredUserPolicy, error) {
	var item StoredUserPolicy
	var limit pgtype.Int8
	var updatedAt pgtype.Timestamptz
	err := s.pool.QueryRow(ctx, `
		SELECT users.id, users.username, users.display_name, users.status,
		       policies.max_file_size_bytes, policies.updated_at
		FROM users
		LEFT JOIN attachment_user_upload_policies policies ON policies.user_id = users.id
		WHERE users.id = $1
	`, userID).Scan(
		&item.UserID, &item.Username, &item.DisplayName, &item.Status, &limit, &updatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUserPolicy{}, ErrInvalidPolicy
	}
	if err != nil {
		return StoredUserPolicy{}, fmt.Errorf("get attachment upload user policy: %w", err)
	}
	if limit.Valid {
		value := limit.Int64
		item.MaxFileSizeBytes = &value
	}
	if updatedAt.Valid {
		value := updatedAt.Time
		item.UpdatedAt = &value
	}
	return item, nil
}

func (s *PostgresStore) SetRoleLimit(ctx context.Context, actorUserID int64, roleKey string, maxBytes int64) error {
	return s.changeRoleLimit(ctx, actorUserID, roleKey, &maxBytes)
}

func (s *PostgresStore) DeleteRoleLimit(ctx context.Context, actorUserID int64, roleKey string) error {
	return s.changeRoleLimit(ctx, actorUserID, roleKey, nil)
}

func (s *PostgresStore) changeRoleLimit(ctx context.Context, actorUserID int64, roleKey string, maxBytes *int64) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin attachment upload role policy: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var roleID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE key = $1 FOR UPDATE`, roleKey).Scan(&roleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidPolicy
		}
		return fmt.Errorf("lock attachment upload policy role: %w", err)
	}
	if maxBytes == nil {
		if _, err := tx.Exec(ctx, `DELETE FROM attachment_role_upload_policies WHERE role_id = $1`, roleID); err != nil {
			return fmt.Errorf("delete attachment upload role policy: %w", err)
		}
	} else if _, err := tx.Exec(ctx, `
		INSERT INTO attachment_role_upload_policies (
		  role_id, max_file_size_bytes, updated_by_user_id
		) VALUES ($1, $2, $3)
		ON CONFLICT (role_id) DO UPDATE SET
		  max_file_size_bytes = EXCLUDED.max_file_size_bytes,
		  updated_by_user_id = EXCLUDED.updated_by_user_id,
		  updated_at = now()
	`, roleID, *maxBytes, nullableInt8(actorUserID)); err != nil {
		return fmt.Errorf("upsert attachment upload role policy: %w", err)
	}
	if err := insertAudit(ctx, tx, actorUserID, 0, "attachment.upload_policy.role.update", map[string]any{
		"roleKey": roleKey, "maxFileSizeBytes": maxBytes,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit attachment upload role policy: %w", err)
	}
	return nil
}

func (s *PostgresStore) SetUserLimit(ctx context.Context, actorUserID int64, userID int64, maxBytes int64) error {
	return s.changeUserLimit(ctx, actorUserID, userID, &maxBytes)
}

func (s *PostgresStore) DeleteUserLimit(ctx context.Context, actorUserID int64, userID int64) error {
	return s.changeUserLimit(ctx, actorUserID, userID, nil)
}

func (s *PostgresStore) changeUserLimit(ctx context.Context, actorUserID int64, userID int64, maxBytes *int64) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin attachment upload user policy: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedUserID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&lockedUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidPolicy
		}
		return fmt.Errorf("lock attachment upload policy user: %w", err)
	}
	if maxBytes == nil {
		if _, err := tx.Exec(ctx, `DELETE FROM attachment_user_upload_policies WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("delete attachment upload user policy: %w", err)
		}
	} else if _, err := tx.Exec(ctx, `
		INSERT INTO attachment_user_upload_policies (
		  user_id, max_file_size_bytes, updated_by_user_id
		) VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
		  max_file_size_bytes = EXCLUDED.max_file_size_bytes,
		  updated_by_user_id = EXCLUDED.updated_by_user_id,
		  updated_at = now()
	`, userID, *maxBytes, nullableInt8(actorUserID)); err != nil {
		return fmt.Errorf("upsert attachment upload user policy: %w", err)
	}
	if err := insertAudit(ctx, tx, actorUserID, userID, "attachment.upload_policy.user.update", map[string]any{
		"maxFileSizeBytes": maxBytes,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit attachment upload user policy: %w", err)
	}
	return nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, actorUserID int64, targetUserID int64, action string, metadata map[string]any) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal attachment upload policy audit: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
		VALUES ($1, $2, $3, $4)
	`, nullableInt8(actorUserID), nullableInt8(targetUserID), action, data); err != nil {
		return fmt.Errorf("insert attachment upload policy audit: %w", err)
	}
	return nil
}

func nullableInt8(value int64) pgtype.Int8 {
	if value <= 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: value, Valid: true}
}
