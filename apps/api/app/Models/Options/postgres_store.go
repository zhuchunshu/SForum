package options

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) List(ctx context.Context) ([]Option, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT name, value
		FROM web_options
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list web options: %w", err)
	}
	defer rows.Close()

	options := []Option{}
	for rows.Next() {
		var option Option
		if err := rows.Scan(&option.Name, &option.Value); err != nil {
			return nil, fmt.Errorf("scan web option: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate web options: %w", err)
	}
	return options, nil
}

func (s *PostgresStore) InsertMissing(ctx context.Context, input UpdateInput) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO web_options (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO NOTHING
	`, input.Name, input.Value)
	if err != nil {
		return fmt.Errorf("insert missing web option: %w", err)
	}
	return nil
}

func (s *PostgresStore) Upsert(ctx context.Context, input UpdateInput) (Option, error) {
	if referenceContext, ok := siteAttachmentReferenceContext(input.Name); ok {
		return s.upsertSiteAttachmentOption(ctx, input, referenceContext)
	}
	var option Option
	err := s.pool.QueryRow(ctx, `
		INSERT INTO web_options (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET value = EXCLUDED.value
		RETURNING name, value
	`, input.Name, input.Value).Scan(&option.Name, &option.Value)
	if err != nil {
		return Option{}, fmt.Errorf("upsert web option: %w", err)
	}
	return option, nil
}

func siteAttachmentReferenceContext(name string) (string, bool) {
	switch strings.TrimSpace(name) {
	case NameSiteLogoAttachmentID:
		return "logo", true
	case NameSiteFaviconAttachmentID:
		return "favicon", true
	case NameSiteAppleTouchIconAttachmentID:
		return "apple-touch-icon", true
	default:
		return "", false
	}
}

func (s *PostgresStore) upsertSiteAttachmentOption(ctx context.Context, input UpdateInput, referenceContext string) (Option, error) {
	var attachmentID int64
	if value := strings.TrimSpace(input.Value); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return Option{}, ErrInvalidOption
		}
		attachmentID = parsed
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Option{}, fmt.Errorf("begin site attachment option update: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldAttachmentID int64
	err = tx.QueryRow(ctx, `
		SELECT attachment_id
		FROM attachment_references
		WHERE resource_type = 'site' AND resource_id = 0 AND context = $1
		FOR UPDATE
	`, referenceContext).Scan(&oldAttachmentID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Option{}, fmt.Errorf("load site attachment reference: %w", err)
	}
	if oldAttachmentID != attachmentID {
		if oldAttachmentID > 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM attachment_references WHERE resource_type = 'site' AND resource_id = 0 AND context = $1`, referenceContext); err != nil {
				return Option{}, fmt.Errorf("delete site attachment reference: %w", err)
			}
			if _, err := tx.Exec(ctx, `UPDATE attachments SET reference_count = GREATEST(reference_count - 1, 0), updated_at = now() WHERE id = $1`, oldAttachmentID); err != nil {
				return Option{}, fmt.Errorf("decrement site attachment reference: %w", err)
			}
		}
		if attachmentID > 0 {
			result, err := tx.Exec(ctx, `
				INSERT INTO attachment_references (attachment_id, resource_type, resource_id, context)
				SELECT id, 'site', 0, $2 FROM attachments
				WHERE id = $1 AND status = 'active' AND visibility = 'public' AND content_type LIKE 'image/%'
			`, attachmentID, referenceContext)
			if err != nil {
				return Option{}, fmt.Errorf("insert site attachment reference: %w", err)
			}
			if result.RowsAffected() != 1 {
				return Option{}, ErrInvalidOption
			}
			if _, err := tx.Exec(ctx, `UPDATE attachments SET reference_count = reference_count + 1, updated_at = now() WHERE id = $1`, attachmentID); err != nil {
				return Option{}, fmt.Errorf("increment site attachment reference: %w", err)
			}
		}
	}

	var option Option
	if err := tx.QueryRow(ctx, `
		INSERT INTO web_options (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
		RETURNING name, value
	`, input.Name, strings.TrimSpace(input.Value)).Scan(&option.Name, &option.Value); err != nil {
		return Option{}, fmt.Errorf("upsert site attachment option: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Option{}, fmt.Errorf("commit site attachment option: %w", err)
	}
	return option, nil
}
