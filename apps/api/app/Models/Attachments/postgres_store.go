package attachments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

func (s *PostgresStore) Create(ctx context.Context, input CreateAttachmentInput) (Attachment, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO attachments (
		  public_id, owner_user_id, provider, object_key, original_name,
		  content_type, extension, size_bytes, sha256, image_width, image_height,
		  visibility, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'active')
		RETURNING id, public_id, owner_user_id, provider, object_key, original_name,
		  content_type, extension, size_bytes, sha256, image_width, image_height,
		  visibility, status, reference_count, created_at, updated_at, deleted_at
	`, input.PublicID, input.OwnerUserID, input.Provider, input.ObjectKey, input.OriginalName,
		input.ContentType, input.Extension, input.SizeBytes, input.SHA256, input.ImageWidth,
		input.ImageHeight, input.Visibility)
	attachment, err := scanAttachment(row)
	if err != nil {
		return Attachment{}, fmt.Errorf("create attachment: %w", err)
	}
	return s.hydrateOwner(ctx, attachment)
}

func (s *PostgresStore) GetByPublicID(ctx context.Context, publicID string) (Attachment, error) {
	row := s.pool.QueryRow(ctx, attachmentSelectSQL()+` WHERE attachments.public_id = $1`, publicID)
	return s.scanOne(ctx, row)
}

func (s *PostgresStore) GetByID(ctx context.Context, id int64) (Attachment, error) {
	row := s.pool.QueryRow(ctx, attachmentSelectSQL()+` WHERE attachments.id = $1`, id)
	return s.scanOne(ctx, row)
}

func (s *PostgresStore) List(ctx context.Context, input AttachmentListInput) (AttachmentList, error) {
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	input.Query = strings.TrimSpace(input.Query)
	input.Provider = strings.TrimSpace(input.Provider)
	input.Status = strings.TrimSpace(input.Status)
	input.ContentType = strings.TrimSpace(input.ContentType)
	input.ReferenceStatus = strings.TrimSpace(input.ReferenceStatus)

	where := `
		WHERE ($1 = '' OR attachments.public_id = $1 OR attachments.original_name ILIKE '%' || $1 || '%' ESCAPE '\' OR attachments.object_key ILIKE '%' || $1 || '%' ESCAPE '\')
		  AND ($2 = '' OR attachments.provider = $2)
		  AND ($3 = '' OR attachments.status = $3)
		  AND ($4 = '' OR attachments.content_type ILIKE $4 || '%' ESCAPE '\')
		  AND ($5 = 0 OR attachments.owner_user_id = $5)
		  AND ($6 = '' OR ($6 = 'referenced' AND attachments.reference_count > 0) OR ($6 = 'orphan' AND attachments.reference_count = 0))
		  AND ($7::timestamptz IS NULL OR attachments.created_at >= $7)
		  AND ($8::timestamptz IS NULL OR attachments.created_at <= $8)
	`
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM attachments `+where,
		input.Query, input.Provider, input.Status, input.ContentType, input.OwnerUserID,
		input.ReferenceStatus, nullableTime(input.CreatedFrom), nullableTime(input.CreatedTo),
	).Scan(&total); err != nil {
		return AttachmentList{}, fmt.Errorf("count attachments: %w", err)
	}

	rows, err := s.pool.Query(ctx, attachmentSelectSQL()+" "+where+`
		ORDER BY attachments.created_at DESC, attachments.id DESC
		LIMIT $9 OFFSET $10
	`, input.Query, input.Provider, input.Status, input.ContentType, input.OwnerUserID,
		input.ReferenceStatus, nullableTime(input.CreatedFrom), nullableTime(input.CreatedTo),
		input.PerPage, (input.Page-1)*input.PerPage,
	)
	if err != nil {
		return AttachmentList{}, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	items := []Attachment{}
	for rows.Next() {
		item, err := scanAttachment(rows)
		if err != nil {
			return AttachmentList{}, err
		}
		item, err = s.hydrateOwner(ctx, item)
		if err != nil {
			return AttachmentList{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AttachmentList{}, fmt.Errorf("iterate attachments: %w", err)
	}
	return AttachmentList{Items: items, Total: total, Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *PostgresStore) ListReferences(ctx context.Context, attachmentID int64) ([]AttachmentReference, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, attachment_id, resource_type, resource_id, context, created_at
		FROM attachment_references
		WHERE attachment_id = $1
		ORDER BY created_at DESC, id DESC
	`, attachmentID)
	if err != nil {
		return nil, fmt.Errorf("list attachment references: %w", err)
	}
	defer rows.Close()
	references := []AttachmentReference{}
	for rows.Next() {
		var reference AttachmentReference
		if err := rows.Scan(&reference.ID, &reference.AttachmentID, &reference.ResourceType, &reference.ResourceID, &reference.Context, &reference.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan attachment reference: %w", err)
		}
		references = append(references, reference)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachment references: %w", err)
	}
	return references, nil
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id int64, status string, deleted bool) (Attachment, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE attachments
		SET status = $2,
		    deleted_at = CASE WHEN $3 THEN COALESCE(deleted_at, now()) ELSE NULL END,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, public_id, owner_user_id, provider, object_key, original_name,
		  content_type, extension, size_bytes, sha256, image_width, image_height,
		  visibility, status, reference_count, created_at, updated_at, deleted_at
	`, id, status, deleted)
	attachment, err := scanAttachment(row)
	if err != nil {
		return Attachment{}, fmt.Errorf("update attachment status: %w", err)
	}
	return s.hydrateOwner(ctx, attachment)
}

func (s *PostgresStore) ListCleanupCandidates(ctx context.Context, cutoff time.Time, limit int) ([]Attachment, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, attachmentSelectSQL()+`
		WHERE attachments.status = 'deleted'
		  AND attachments.reference_count = 0
		  AND attachments.deleted_at IS NOT NULL
		  AND attachments.deleted_at <= $1
		ORDER BY attachments.deleted_at ASC
		LIMIT $2
	`, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list attachment cleanup candidates: %w", err)
	}
	defer rows.Close()
	items := []Attachment{}
	for rows.Next() {
		item, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachment cleanup candidates: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) DeleteMetadata(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM attachments
		WHERE id = $1 AND reference_count = 0
	`, id)
	if err != nil {
		return fmt.Errorf("delete attachment metadata: %w", err)
	}
	return nil
}

func (s *PostgresStore) scanOne(ctx context.Context, row pgx.Row) (Attachment, error) {
	attachment, err := scanAttachment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Attachment{}, ErrAttachmentNotFound
		}
		return Attachment{}, err
	}
	return s.hydrateOwner(ctx, attachment)
}

func (s *PostgresStore) hydrateOwner(ctx context.Context, attachment Attachment) (Attachment, error) {
	if attachment.Owner == nil {
		return attachment, nil
	}
	err := s.pool.QueryRow(ctx, `
		SELECT username, display_name
		FROM users
		WHERE id = $1
	`, attachment.Owner.ID).Scan(&attachment.Owner.Username, &attachment.Owner.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		attachment.Owner = nil
		return attachment, nil
	}
	if err != nil {
		return Attachment{}, fmt.Errorf("hydrate attachment owner: %w", err)
	}
	return attachment, nil
}

func attachmentSelectSQL() string {
	return `
		SELECT attachments.id, attachments.public_id, attachments.owner_user_id, attachments.provider,
		  attachments.object_key, attachments.original_name, attachments.content_type, attachments.extension,
		  attachments.size_bytes, attachments.sha256, attachments.image_width, attachments.image_height,
		  attachments.visibility, attachments.status, attachments.reference_count,
		  attachments.created_at, attachments.updated_at, attachments.deleted_at
		FROM attachments
	`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAttachment(row rowScanner) (Attachment, error) {
	var attachment Attachment
	var ownerID sql.NullInt64
	var width sql.NullInt64
	var height sql.NullInt64
	var deletedAt sql.NullTime
	if err := row.Scan(
		&attachment.ID,
		&attachment.PublicID,
		&ownerID,
		&attachment.Provider,
		&attachment.ObjectKey,
		&attachment.OriginalName,
		&attachment.ContentType,
		&attachment.Extension,
		&attachment.SizeBytes,
		&attachment.SHA256,
		&width,
		&height,
		&attachment.Visibility,
		&attachment.Status,
		&attachment.ReferenceCount,
		&attachment.CreatedAt,
		&attachment.UpdatedAt,
		&deletedAt,
	); err != nil {
		return Attachment{}, fmt.Errorf("scan attachment: %w", err)
	}
	if ownerID.Valid {
		attachment.Owner = &OwnerSummary{ID: ownerID.Int64}
	}
	if width.Valid {
		value := int(width.Int64)
		attachment.ImageWidth = &value
	}
	if height.Valid {
		value := int(height.Int64)
		attachment.ImageHeight = &value
	}
	if deletedAt.Valid {
		attachment.DeletedAt = &deletedAt.Time
	}
	return attachment, nil
}

// maxAdminListPage 限制后台附件列表的最大页数（M6），避免深 OFFSET DoS。
const maxAdminListPage = 200

func normalizePage(page int, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if page > maxAdminListPage {
		page = maxAdminListPage
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

// escapeLike 转义 SQL LIKE/ILIKE 元字符，配合 ESCAPE '\' 使用（M6/L4）。
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func nullableTime(value interface{ IsZero() bool }) any {
	if value.IsZero() {
		return nil
	}
	return value
}
