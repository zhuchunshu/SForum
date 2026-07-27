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
	// 仅供同包 PostgreSQL 并发回归测试在锁真正取得后建立确定性交错。
	registrationPolicyLockObserver func()
}

const registrationPolicyAdvisoryLock = "sforum.identity.registration_policy"

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
	if registrationPolicyOptionName(input.Name) {
		updated, err := s.UpsertMany(ctx, []UpdateInput{input})
		if err != nil {
			return Option{}, err
		}
		return updated[0], nil
	}
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

// UpsertMany keeps the registration-policy advisory lock and every affected
// write in one explicit PostgreSQL transaction. pg_advisory_xact_lock called
// through Pool.Exec would release before the following upsert, which is not a
// fence at all.
func (s *PostgresStore) UpsertMany(ctx context.Context, inputs []UpdateInput) ([]Option, error) {
	if len(inputs) == 0 {
		return []Option{}, nil
	}
	hasRegistrationPolicy := false
	for _, input := range inputs {
		if registrationPolicyOptionName(input.Name) {
			hasRegistrationPolicy = true
			break
		}
	}
	if !hasRegistrationPolicy {
		out := make([]Option, 0, len(inputs))
		for _, input := range inputs {
			updated, err := s.Upsert(ctx, input)
			if err != nil {
				return nil, err
			}
			out = append(out, updated)
		}
		return out, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin registration policy update: %w", err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, registrationPolicyAdvisoryLock); err != nil {
		return nil, fmt.Errorf("lock registration policy update: %w", err)
	}
	if s.registrationPolicyLockObserver != nil {
		s.registrationPolicyLockObserver()
	}

	out := make([]Option, 0, len(inputs))
	for _, input := range inputs {
		var option Option
		if err := tx.QueryRow(ctx, `
			INSERT INTO web_options (name, value)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE
			SET value = EXCLUDED.value
			RETURNING name, value
		`, input.Name, input.Value).Scan(&option.Name, &option.Value); err != nil {
			return nil, fmt.Errorf("upsert registration policy option: %w", err)
		}
		out = append(out, option)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit registration policy update: %w", err)
	}
	return out, nil
}

// RegistrationPolicyValuesTx returns the uncached authoritative registration
// options while holding the same PostgreSQL advisory transaction lock used by
// normal option updates. It is intentionally narrow so external registration
// can share its user/link/audit transaction without importing Options internals.
func (s *PostgresStore) RegistrationPolicyValuesTx(ctx context.Context, tx pgx.Tx) (map[string]string, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, registrationPolicyAdvisoryLock); err != nil {
		return nil, fmt.Errorf("lock registration policy read: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT name, value
		FROM web_options
		WHERE name = ANY($1)
	`, []string{NameIdentityRegistrationEnabled, NameIdentityRegistrationMode})
	if err != nil {
		return nil, fmt.Errorf("read registration policy options: %w", err)
	}
	defer rows.Close()
	values := make(map[string]string, 2)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan registration policy option: %w", err)
		}
		values[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate registration policy options: %w", err)
	}
	return values, nil
}

func registrationPolicyOptionName(name string) bool {
	switch strings.TrimSpace(name) {
	case NameIdentityRegistrationEnabled, NameIdentityRegistrationMode:
		return true
	default:
		return false
	}
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
