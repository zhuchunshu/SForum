package attachments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const compressionTaskLease = 15 * time.Minute

func (s *PostgresStore) CreateCompressionTask(ctx context.Context, attachment Attachment, settings CompressionSettings) (int64, bool, error) {
	settings = settings.normalized()
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO attachment_compression_tasks (
		  attachment_id, variant_name, source_sha256, policy_digest, compression_strength
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (attachment_id, variant_name, policy_digest) DO NOTHING
		RETURNING id
	`, attachment.ID, CompressionVariantDisplay, attachment.SHA256, settings.PolicyDigest, settings.Strength).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.pool.QueryRow(ctx, `
			SELECT id FROM attachment_compression_tasks
			WHERE attachment_id=$1 AND variant_name=$2 AND policy_digest=$3
		`, attachment.ID, CompressionVariantDisplay, settings.PolicyDigest).Scan(&id)
		return id, false, err
	}
	if err != nil {
		return 0, false, fmt.Errorf("create attachment compression task: %w", err)
	}
	return id, true, nil
}

func (s *PostgresStore) ClaimCompressionTask(ctx context.Context, id int64) (CompressionTask, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CompressionTask{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		SELECT t.id, t.variant_name, t.source_sha256, t.policy_digest,
		  t.compression_strength, t.attempts,
		  a.id, a.public_id, a.owner_user_id, a.provider, a.object_key, a.original_name,
		  a.content_type, a.extension, a.size_bytes, a.sha256, a.image_width, a.image_height,
		  a.visibility, a.status, a.reference_count, a.created_at, a.updated_at, a.deleted_at
		FROM attachment_compression_tasks t
		JOIN attachments a ON a.id=t.attachment_id
		WHERE t.id=$1 AND (
		  (t.status IN ('pending', 'failed') AND t.available_at <= now())
		  OR (t.status='running' AND t.started_at <= now() - $2::interval)
		)
		  AND t.attempts < 3 AND a.status='active'
		FOR UPDATE OF t SKIP LOCKED
	`, id, compressionTaskLease.String())
	var task CompressionTask
	var ownerID, width, height sql.NullInt64
	var deletedAt sql.NullTime
	if err := row.Scan(
		&task.ID, &task.VariantName, &task.SourceSHA256, &task.PolicyDigest,
		&task.CompressionStrength, &task.Attempts,
		&task.Attachment.ID, &task.Attachment.PublicID, &ownerID,
		&task.Attachment.Provider, &task.Attachment.ObjectKey, &task.Attachment.OriginalName,
		&task.Attachment.ContentType, &task.Attachment.Extension, &task.Attachment.SizeBytes,
		&task.Attachment.SHA256, &width, &height,
		&task.Attachment.Visibility, &task.Attachment.Status, &task.Attachment.ReferenceCount,
		&task.Attachment.CreatedAt, &task.Attachment.UpdatedAt, &deletedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CompressionTask{}, ErrAttachmentNotFound
		}
		return CompressionTask{}, fmt.Errorf("claim attachment compression task: %w", err)
	}
	if ownerID.Valid {
		task.Attachment.Owner = &OwnerSummary{ID: ownerID.Int64}
	}
	if width.Valid {
		value := int(width.Int64)
		task.Attachment.ImageWidth = &value
	}
	if height.Valid {
		value := int(height.Int64)
		task.Attachment.ImageHeight = &value
	}
	if deletedAt.Valid {
		value := deletedAt.Time
		task.Attachment.DeletedAt = &value
	}
	if _, err := tx.Exec(ctx, `
		UPDATE attachment_compression_tasks
		SET status='running', attempts=attempts+1, started_at=now(), updated_at=now(), error_code=''
		WHERE id=$1
	`, id); err != nil {
		return CompressionTask{}, err
	}
	task.Attempts++
	if err := tx.Commit(ctx); err != nil {
		return CompressionTask{}, err
	}
	return task, nil
}

func (s *PostgresStore) CompleteCompressionTask(ctx context.Context, task CompressionTask, variant AttachmentVariant) (*AttachmentVariant, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var previous *AttachmentVariant
	row := tx.QueryRow(ctx, `
		SELECT id, attachment_id, name, provider, object_key, content_type, size_bytes, sha256,
		  image_width, image_height, source_sha256, policy_digest, compression_strength, created_at, updated_at
		FROM attachment_variants WHERE attachment_id=$1 AND name=$2 FOR UPDATE
	`, task.Attachment.ID, task.VariantName)
	if item, scanErr := scanAttachmentVariant(row); scanErr == nil {
		previous = &item
	} else if !errors.Is(scanErr, pgx.ErrNoRows) {
		return nil, scanErr
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO attachment_variants (
		  attachment_id, name, provider, object_key, content_type, size_bytes, sha256,
		  image_width, image_height, source_sha256, policy_digest, compression_strength
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (attachment_id, name) DO UPDATE SET
		  provider=EXCLUDED.provider, object_key=EXCLUDED.object_key,
		  content_type=EXCLUDED.content_type, size_bytes=EXCLUDED.size_bytes,
		  sha256=EXCLUDED.sha256, image_width=EXCLUDED.image_width,
		  image_height=EXCLUDED.image_height, source_sha256=EXCLUDED.source_sha256,
		  policy_digest=EXCLUDED.policy_digest,
		  compression_strength=EXCLUDED.compression_strength, updated_at=now()
	`, variant.AttachmentID, variant.Name, variant.Provider, variant.ObjectKey,
		variant.ContentType, variant.SizeBytes, variant.SHA256, variant.ImageWidth,
		variant.ImageHeight, variant.SourceSHA256, variant.PolicyDigest, variant.CompressionStrength)
	if err != nil {
		return nil, fmt.Errorf("store attachment variant: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE attachment_compression_tasks
		SET status='succeeded', finished_at=now(), updated_at=now(), error_code=''
		WHERE id=$1
	`, task.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return previous, nil
}

func (s *PostgresStore) FinishCompressionTask(ctx context.Context, id int64, status, errorCode string) error {
	if status != CompressionStatusSkipped && status != CompressionStatusFailed {
		return ErrInvalidAttachment
	}
	availableAt := time.Now().UTC()
	if status == CompressionStatusFailed {
		availableAt = availableAt.Add(time.Minute)
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE attachment_compression_tasks
		SET status=$2, error_code=$3, available_at=$4, finished_at=now(), updated_at=now()
		WHERE id=$1
	`, id, status, errorCode, availableAt)
	return err
}

func (s *PostgresStore) GetAttachmentVariant(ctx context.Context, attachmentID int64, name string) (AttachmentVariant, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, attachment_id, name, provider, object_key, content_type, size_bytes, sha256,
		  image_width, image_height, source_sha256, policy_digest, compression_strength, created_at, updated_at
		FROM attachment_variants WHERE attachment_id=$1 AND name=$2
	`, attachmentID, name)
	item, err := scanAttachmentVariant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttachmentVariant{}, ErrAttachmentNotFound
	}
	return item, err
}

func (s *PostgresStore) ListAttachmentVariants(ctx context.Context, attachmentID int64) ([]AttachmentVariant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, attachment_id, name, provider, object_key, content_type, size_bytes, sha256,
		  image_width, image_height, source_sha256, policy_digest, compression_strength, created_at, updated_at
		FROM attachment_variants WHERE attachment_id=$1 ORDER BY name
	`, attachmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AttachmentVariant{}
	for rows.Next() {
		item, scanErr := scanAttachmentVariant(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CompressionStats(ctx context.Context) (CompressionStats, error) {
	var stats CompressionStats
	err := s.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM attachment_compression_tasks WHERE status='pending'),
		  (SELECT count(*) FROM attachment_compression_tasks WHERE status='running'),
		  (SELECT count(*) FROM attachment_compression_tasks WHERE status='failed'),
		  count(*), COALESCE(sum(a.size_bytes),0), COALESCE(sum(v.size_bytes),0)
		FROM attachment_variants v JOIN attachments a ON a.id=v.attachment_id
		WHERE v.name=$1 AND a.status='active'
	`, CompressionVariantDisplay).Scan(&stats.Pending, &stats.Running, &stats.Failed,
		&stats.ReadyVariants, &stats.OriginalBytes, &stats.VariantBytes)
	stats.SavedBytes = stats.OriginalBytes - stats.VariantBytes
	if stats.SavedBytes < 0 {
		stats.SavedBytes = 0
	}
	return stats, err
}

func (s *PostgresStore) ListPendingCompressionTaskIDs(ctx context.Context, limit int) ([]int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE attachment_compression_tasks
		SET status='failed', error_code='worker_timeout', finished_at=now(), updated_at=now()
		WHERE status='running' AND attempts >= 3 AND started_at <= now() - $1::interval
	`, compressionTaskLease.String()); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM attachment_compression_tasks
		WHERE (
		  (status IN ('pending', 'failed') AND available_at <= now())
		  OR (status='running' AND started_at <= now() - $2::interval)
		) AND attempts < 3
		ORDER BY available_at, id LIMIT $1
	`, limit, compressionTaskLease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *PostgresStore) BackfillCompressionTasks(ctx context.Context, settings CompressionSettings, limit int) ([]int64, error) {
	if limit <= 0 || limit > 10000 {
		limit = 5000
	}
	settings = settings.normalized()
	rows, err := s.pool.Query(ctx, `
		WITH candidates AS (
		  SELECT a.id, a.sha256
		  FROM attachments a
		  WHERE a.status='active' AND a.content_type IN ('image/jpeg','image/png')
		    AND (a.size_bytes >= $1 OR a.image_width > $2 OR a.image_height > $2)
		    AND NOT EXISTS (
		      SELECT 1 FROM attachment_compression_tasks t
		      WHERE t.attachment_id=a.id AND t.variant_name=$3 AND t.policy_digest=$4
		    )
		  ORDER BY a.id LIMIT $6
		)
		INSERT INTO attachment_compression_tasks (
		  attachment_id, variant_name, source_sha256, policy_digest, compression_strength
		)
		SELECT id, $3, sha256, $4, $5 FROM candidates
		RETURNING id
	`, int64(settings.MinSizeKB)*1024, settings.MaxDimension, CompressionVariantDisplay,
		settings.PolicyDigest, settings.Strength, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanAttachmentVariant(row rowScanner) (AttachmentVariant, error) {
	var item AttachmentVariant
	err := row.Scan(&item.ID, &item.AttachmentID, &item.Name, &item.Provider,
		&item.ObjectKey, &item.ContentType, &item.SizeBytes, &item.SHA256,
		&item.ImageWidth, &item.ImageHeight, &item.SourceSHA256, &item.PolicyDigest,
		&item.CompressionStrength, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}
