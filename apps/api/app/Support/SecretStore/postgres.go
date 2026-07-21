package secretstore

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists versioned ciphertext secrets in secret_store.
// Append uses a transaction-scoped advisory lock so concurrent Put/Rotate/Clear
// cannot race nextVersion against insert.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore builds a production Secret Store backend.
func NewPostgresStore(pool *pgxpool.Pool) (*PostgresStore, error) {
	if pool == nil {
		return nil, ErrInvalid
	}
	return &PostgresStore{pool: pool}, nil
}

// Append implements Store with atomic version assignment.
func (s *PostgresStore) Append(ctx context.Context, row Row) (Row, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return Row{}, ErrInvalid
	}
	if row.Namespace == "" || row.SecretID == "" {
		return Row{}, ErrInvalid
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
	}

	// 并发 Put/Rotate 必须在同一事务内锁定 (namespace, secret_id) 再取 next version。
	const maxAttempts = 8
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		assigned, err := s.appendOnce(ctx, row)
		if err == nil {
			return assigned, nil
		}
		if isUniqueViolation(err) || isRetryableTx(err) {
			lastErr = err
			continue
		}
		return Row{}, mapStoreError(err)
	}
	return Row{}, fmt.Errorf("%w: concurrent version assignment failed: %v", ErrAlreadyExists, lastErr)
}

func (s *PostgresStore) appendOnce(ctx context.Context, row Row) (Row, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Row{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	// Advisory lock keyed by namespace+secret_id so concurrent writers serialize.
	lockKey := advisoryKey(row.Namespace, row.SecretID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lockKey); err != nil {
		return Row{}, err
	}

	var maxVersion *int64
	if err := tx.QueryRow(ctx, `
		SELECT MAX(version) FROM secret_store
		WHERE namespace = $1 AND secret_id = $2
	`, row.Namespace, row.SecretID).Scan(&maxVersion); err != nil {
		return Row{}, err
	}
	next := int64(1)
	if maxVersion != nil {
		next = *maxVersion + 1
	}
	row.Version = next
	purposes := row.Purposes
	if purposes == nil {
		purposes = []string{}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO secret_store (
			namespace, secret_id, version, value, media_type, purposes, revoked, created_at, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, row.Namespace, row.SecretID, row.Version, row.Cipher, row.MediaType, purposes, row.Revoked, row.UpdatedAt, row.UpdatedBy); err != nil {
		return Row{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Row{}, err
	}
	return cloneRow(row), nil
}

// Latest implements Store.
func (s *PostgresStore) Latest(ctx context.Context, namespace, secretID string, includeRevoked bool) (Row, bool, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return Row{}, false, ErrInvalid
	}
	query := `
		SELECT namespace, secret_id, version, value, media_type, purposes, revoked, created_at, created_by
		FROM secret_store
		WHERE namespace = $1 AND secret_id = $2
	`
	if !includeRevoked {
		query += ` AND revoked = FALSE`
	}
	query += ` ORDER BY version DESC LIMIT 1`

	row, err := scanRow(s.pool.QueryRow(ctx, query, namespace, secretID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, false, nil
	}
	if err != nil {
		return Row{}, false, mapStoreError(err)
	}
	return row, true, nil
}

// GetVersion implements Store.
func (s *PostgresStore) GetVersion(ctx context.Context, namespace, secretID string, version int64) (Row, bool, error) {
	if s == nil || s.pool == nil || ctx == nil || version <= 0 {
		return Row{}, false, nil
	}
	row, err := scanRow(s.pool.QueryRow(ctx, `
		SELECT namespace, secret_id, version, value, media_type, purposes, revoked, created_at, created_by
		FROM secret_store
		WHERE namespace = $1 AND secret_id = $2 AND version = $3
	`, namespace, secretID, version))
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, false, nil
	}
	if err != nil {
		return Row{}, false, mapStoreError(err)
	}
	return row, true, nil
}

// ListNamespace implements Store.
func (s *PostgresStore) ListNamespace(ctx context.Context, namespace string) ([]Row, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return nil, ErrInvalid
	}
	// Distinct on secret_id ordered by version DESC → latest non-revoked per id.
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (secret_id)
			namespace, secret_id, version, value, media_type, purposes, revoked, created_at, created_by
		FROM secret_store
		WHERE namespace = $1 AND revoked = FALSE
		ORDER BY secret_id ASC, version DESC
	`, namespace)
	if err != nil {
		return nil, mapStoreError(err)
	}
	defer rows.Close()
	out := make([]Row, 0)
	for rows.Next() {
		row, scanErr := scanRows(rows)
		if scanErr != nil {
			return nil, mapStoreError(scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, mapStoreError(err)
	}
	return out, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanRow(s scannable) (Row, error) {
	var row Row
	var purposes []string
	if err := s.Scan(
		&row.Namespace, &row.SecretID, &row.Version, &row.Cipher, &row.MediaType,
		&purposes, &row.Revoked, &row.UpdatedAt, &row.UpdatedBy,
	); err != nil {
		return Row{}, err
	}
	if purposes == nil {
		purposes = []string{}
	}
	row.Purposes = purposes
	return row, nil
}

func scanRows(rows pgx.Rows) (Row, error) {
	return scanRow(rows)
}

func advisoryKey(namespace, secretID string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(namespace))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(secretID))
	// 正数 int64，避免 advisory lock 符号位歧义。
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isRetryableTx(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if isUniqueViolation(err) {
		return ErrAlreadyExists
	}
	return err
}

// PostgresAuditStore persists Secret Store audit events without values.
// Uses secret_store_audit when present; construction is optional for production.
type PostgresAuditStore struct {
	pool *pgxpool.Pool
}

// NewPostgresAuditStore builds a durable audit backend (requires migration).
func NewPostgresAuditStore(pool *pgxpool.Pool) (*PostgresAuditStore, error) {
	if pool == nil {
		return nil, ErrInvalid
	}
	return &PostgresAuditStore{pool: pool}, nil
}

// AppendAudit implements AuditStore.
func (s *PostgresAuditStore) AppendAudit(ctx context.Context, event AuditEvent) error {
	if s == nil || s.pool == nil || ctx == nil {
		return ErrInvalid
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO secret_store_audit (
			audit_id, action, namespace, secret_id, version, actor, purpose, ok, at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, event.AuditID, event.Action, event.Namespace, event.SecretID, event.Version,
		event.Actor, event.Purpose, event.OK, event.At)
	return mapStoreError(err)
}

// ListRecentAudit implements AuditStore.
func (s *PostgresAuditStore) ListRecentAudit(ctx context.Context, limit int) ([]AuditEvent, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return nil, ErrInvalid
	}
	if limit <= 0 || limit > MaxAuditRing {
		limit = MaxAuditRing
	}
	rows, err := s.pool.Query(ctx, `
		SELECT audit_id, action, namespace, secret_id, version, actor, purpose, ok, at
		FROM secret_store_audit
		ORDER BY at DESC, audit_id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, mapStoreError(err)
	}
	defer rows.Close()
	out := make([]AuditEvent, 0, limit)
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(
			&event.AuditID, &event.Action, &event.Namespace, &event.SecretID,
			&event.Version, &event.Actor, &event.Purpose, &event.OK, &event.At,
		); err != nil {
			return nil, mapStoreError(err)
		}
		out = append(out, event)
	}
	return out, rows.Err()
}
