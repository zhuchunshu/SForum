package apitokens

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Create(ctx context.Context, userID int64, publicID, tokenHash, name string, scopes []string, expiresAt *time.Time) (Record, error) {
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return Record{}, err
	}
	var rec Record
	var scopesRaw []byte
	err = s.pool.QueryRow(ctx, `
INSERT INTO api_tokens (user_id, public_id, token_hash, name, scopes, expires_at)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id, user_id, public_id, token_hash, name, scopes, last_used_at, expires_at, revoked_at, created_at`,
		userID, publicID, tokenHash, name, scopesJSON, expiresAt,
	).Scan(&rec.ID, &rec.UserID, &rec.PublicID, &rec.TokenHash, &rec.Name, &scopesRaw,
		&rec.LastUsedAt, &rec.ExpiresAt, &rec.RevokedAt, &rec.CreatedAt)
	if err != nil {
		return Record{}, err
	}
	_ = json.Unmarshal(scopesRaw, &rec.Scopes)
	if rec.Scopes == nil {
		rec.Scopes = []string{}
	}
	return rec, nil
}

func (s *PostgresStore) ListByUser(ctx context.Context, userID int64, includeRevoked bool) ([]Record, error) {
	query := `
SELECT id, user_id, public_id, token_hash, name, scopes, last_used_at, expires_at, revoked_at, created_at
FROM api_tokens WHERE user_id=$1`
	if !includeRevoked {
		query += ` AND revoked_at IS NULL`
	}
	query += ` ORDER BY created_at DESC, id DESC`
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (s *PostgresStore) GetByPublicID(ctx context.Context, publicID string) (Record, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, public_id, token_hash, name, scopes, last_used_at, expires_at, revoked_at, created_at
FROM api_tokens WHERE public_id=$1`, publicID)
	return scanRecord(row)
}

func (s *PostgresStore) GetByIDForUser(ctx context.Context, userID, id int64) (Record, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, public_id, token_hash, name, scopes, last_used_at, expires_at, revoked_at, created_at
FROM api_tokens WHERE id=$1 AND user_id=$2`, id, userID)
	return scanRecord(row)
}

func (s *PostgresStore) Revoke(ctx context.Context, userID, id int64) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE api_tokens SET revoked_at=NOW()
WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	return nil
}

func (s *PostgresStore) TouchLastUsed(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE api_tokens SET last_used_at=NOW() WHERE id=$1`, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(row rowScanner) (Record, error) {
	var rec Record
	var scopesRaw []byte
	err := row.Scan(&rec.ID, &rec.UserID, &rec.PublicID, &rec.TokenHash, &rec.Name, &scopesRaw,
		&rec.LastUsedAt, &rec.ExpiresAt, &rec.RevokedAt, &rec.CreatedAt)
	if err == pgx.ErrNoRows {
		return Record{}, ErrTokenNotFound
	}
	if err != nil {
		return Record{}, err
	}
	_ = json.Unmarshal(scopesRaw, &rec.Scopes)
	if rec.Scopes == nil {
		rec.Scopes = []string{}
	}
	return rec, nil
}

func scanRecords(rows pgx.Rows) ([]Record, error) {
	items := []Record{}
	for rows.Next() {
		var rec Record
		var scopesRaw []byte
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.PublicID, &rec.TokenHash, &rec.Name, &scopesRaw,
			&rec.LastUsedAt, &rec.ExpiresAt, &rec.RevokedAt, &rec.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(scopesRaw, &rec.Scopes)
		if rec.Scopes == nil {
			rec.Scopes = []string{}
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}
