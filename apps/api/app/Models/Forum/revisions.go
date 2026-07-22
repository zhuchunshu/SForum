package forum

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	RevisionOperationCreate    = "create"
	RevisionOperationMigration = "migration"

	RevisionOriginSelf      = "self"
	RevisionOriginMigration = "migration"
)

type TopicRevisionSnapshotInput struct {
	TopicID      int64
	Title        string
	CategorySlug string
	TagSlugs     []string
}

type AcceptedRevisionSnapshotInput struct {
	PostID                 int64
	RevisionNo             int64
	ActorUserID            int64
	Operation              string
	Origin                 string
	ChangedFields          []string
	AttachmentIDs          []int64
	CommittedAt            *time.Time
	RestoredFromRevisionID *int64
	SnapshotComplete       bool
	Reason                 string
	Content                RenderedContent
	Topic                  *TopicRevisionSnapshotInput
}

type RevisionBackfillOptions struct {
	BatchSize int
}

type RevisionBackfillResult struct {
	Claimed   int64
	Completed int64
	Pending   int64
}

type legacyPostRevisionRow struct {
	ID                 int64
	SupersededByUserID sql.NullInt64
	CreatedAt          time.Time
}

type backfillPostRow struct {
	ID              int64
	CreatedByUserID sql.NullInt64
	UpdatedByUserID sql.NullInt64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Content         RenderedContent
}

type currentTopicRevisionSnapshot struct {
	TopicID      int64
	Title        string
	CategorySlug string
	TagSlugs     []string
}

func (s *PostgresStore) BackfillContentRevisions(ctx context.Context, opts RevisionBackfillOptions) (RevisionBackfillResult, error) {
	if s == nil || s.pool == nil {
		return RevisionBackfillResult{}, fmt.Errorf("forum revision backfill store is unavailable")
	}
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return RevisionBackfillResult{}, fmt.Errorf("begin revision backfill: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rows, err := tx.Query(ctx, `
		SELECT id
		FROM posts
		WHERE current_revision = 0
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return RevisionBackfillResult{}, fmt.Errorf("claim revision backfill batch: %w", err)
	}
	postIDs := make([]int64, 0, batchSize)
	for rows.Next() {
		var postID int64
		if err := rows.Scan(&postID); err != nil {
			rows.Close()
			return RevisionBackfillResult{}, fmt.Errorf("scan revision backfill claim: %w", err)
		}
		postIDs = append(postIDs, postID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return RevisionBackfillResult{}, fmt.Errorf("iterate revision backfill claims: %w", err)
	}
	rows.Close()

	for _, postID := range postIDs {
		if err := backfillPostRevisionLedger(ctx, tx, postID); err != nil {
			return RevisionBackfillResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return RevisionBackfillResult{}, fmt.Errorf("commit revision backfill: %w", err)
	}

	pending, err := s.PendingContentRevisionBackfill(ctx)
	if err != nil {
		return RevisionBackfillResult{}, err
	}
	completed := int64(len(postIDs))
	return RevisionBackfillResult{Claimed: completed, Completed: completed, Pending: pending}, nil
}

func (s *PostgresStore) PendingContentRevisionBackfill(ctx context.Context) (int64, error) {
	if s == nil || s.pool == nil {
		return 0, fmt.Errorf("forum revision backfill store is unavailable")
	}
	var pending int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM posts WHERE current_revision = 0
	`).Scan(&pending); err != nil {
		return 0, fmt.Errorf("count pending revision backfill: %w", err)
	}
	return pending, nil
}

func backfillPostRevisionLedger(ctx context.Context, tx pgx.Tx, postID int64) error {
	post, err := loadBackfillPost(ctx, tx, postID)
	if err != nil {
		return err
	}
	legacyRows, err := loadLegacyRevisionRows(ctx, tx, postID)
	if err != nil {
		return err
	}

	for i, row := range legacyRows {
		revisionNo := int64(i + 1)
		actorUserID := backfillLegacyActor(post, legacyRows, i)
		committedAt := backfillLegacyCommittedAt(post, legacyRows, i)
		if _, err := tx.Exec(ctx, `
			UPDATE post_revisions
			SET revision_no = $2,
			    actor_user_id = $3,
			    operation = 'migration',
			    origin = 'migration',
			    changed_fields = ARRAY['content']::text[],
			    committed_at = $4,
			    snapshot_complete = false
			WHERE id = $1 AND revision_no IS NULL
		`, row.ID, revisionNo, nullSQLInt64(actorUserID), committedAt); err != nil {
			return fmt.Errorf("number legacy post revision: %w", err)
		}
	}

	currentRevision := int64(len(legacyRows) + 1)
	attachments, err := currentPostAttachmentIDs(ctx, tx, postID)
	if err != nil {
		return err
	}
	topicSnapshot, err := currentTopicSnapshotForPost(ctx, tx, postID)
	if err != nil {
		return err
	}
	committedAt := post.UpdatedAt
	revisionID, err := insertAcceptedPostRevision(ctx, tx, AcceptedRevisionSnapshotInput{
		PostID:           post.ID,
		RevisionNo:       currentRevision,
		ActorUserID:      sqlNullInt64Value(post.UpdatedByUserID),
		Operation:        RevisionOperationMigration,
		Origin:           RevisionOriginMigration,
		ChangedFields:    []string{"content"},
		AttachmentIDs:    attachments,
		CommittedAt:      &committedAt,
		SnapshotComplete: true,
		Content:          post.Content,
		Topic:            topicSnapshot,
	})
	if err != nil {
		return err
	}
	if revisionID <= 0 {
		return fmt.Errorf("insert current post revision: missing revision id")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE posts
		SET current_revision = $2
		WHERE id = $1 AND current_revision = 0
	`, post.ID, currentRevision); err != nil {
		return fmt.Errorf("mark post revision backfilled: %w", err)
	}
	return nil
}

func loadBackfillPost(ctx context.Context, tx pgx.Tx, postID int64) (backfillPostRow, error) {
	var post backfillPostRow
	if err := tx.QueryRow(ctx, `
		SELECT id, raw_content, html_content, plain_text, source_format, editor_type,
		  editor_version, render_version, content_hash,
		  created_by_user_id, updated_by_user_id, created_at, updated_at
		FROM posts
		WHERE id = $1
		FOR UPDATE
	`, postID).Scan(
		&post.ID,
		&post.Content.RawContent,
		&post.Content.HTMLContent,
		&post.Content.PlainText,
		&post.Content.SourceFormat,
		&post.Content.EditorType,
		&post.Content.EditorVersion,
		&post.Content.RenderVersion,
		&post.Content.ContentHash,
		&post.CreatedByUserID,
		&post.UpdatedByUserID,
		&post.CreatedAt,
		&post.UpdatedAt,
	); err != nil {
		return backfillPostRow{}, fmt.Errorf("load post for revision backfill: %w", err)
	}
	post.Content.ID = post.ID
	return post, nil
}

func loadLegacyRevisionRows(ctx context.Context, tx pgx.Tx, postID int64) ([]legacyPostRevisionRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, superseded_by_user_id, created_at
		FROM post_revisions
		WHERE post_id = $1 AND revision_no IS NULL
		ORDER BY created_at ASC, id ASC
		FOR UPDATE
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("load legacy post revisions: %w", err)
	}
	defer rows.Close()

	out := []legacyPostRevisionRow{}
	for rows.Next() {
		var row legacyPostRevisionRow
		if err := rows.Scan(&row.ID, &row.SupersededByUserID, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan legacy post revision: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy post revisions: %w", err)
	}
	return out, nil
}

func insertAcceptedPostRevision(ctx context.Context, tx pgx.Tx, input AcceptedRevisionSnapshotInput) (int64, error) {
	fields := normalizeRevisionTextArray(input.ChangedFields)
	attachments := normalizeRevisionInt64Array(input.AttachmentIDs)
	operation := strings.TrimSpace(input.Operation)
	origin := strings.TrimSpace(input.Origin)
	if operation == "" || origin == "" {
		return 0, fmt.Errorf("insert accepted post revision: operation and origin are required")
	}
	var revisionID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO post_revisions (
		  post_id, revision_no, actor_user_id, raw_content,
		  source_format, editor_type, editor_version, render_version, content_hash,
		  reason, operation, origin, changed_fields, attachment_ids, committed_at,
		  restored_from_revision_id, snapshot_complete
		)
		VALUES (
		  $1, $2, $3, $4,
		  $5, $6, $7, $8, $9,
		  $10, $11, $12, $13, $14, COALESCE($15, now()),
		  $16, $17
		)
		RETURNING id
	`, input.PostID, input.RevisionNo, nullSQLInt64(input.ActorUserID), input.Content.RawContent,
		input.Content.SourceFormat, input.Content.EditorType, input.Content.EditorVersion,
		input.Content.RenderVersion, input.Content.ContentHash, strings.TrimSpace(input.Reason),
		operation, origin, fields, attachments, input.CommittedAt,
		input.RestoredFromRevisionID, input.SnapshotComplete).Scan(&revisionID)
	if err != nil {
		return 0, fmt.Errorf("insert accepted post revision: %w", err)
	}
	if input.Topic != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO topic_revision_snapshots
			  (post_revision_id, topic_id, title, category_slug, tag_slugs)
			VALUES ($1, $2, $3, $4, $5)
		`, revisionID, input.Topic.TopicID, strings.TrimSpace(input.Topic.Title),
			strings.TrimSpace(input.Topic.CategorySlug), normalizeRevisionTextArray(input.Topic.TagSlugs)); err != nil {
			return 0, fmt.Errorf("insert topic revision snapshot: %w", err)
		}
	}
	return revisionID, nil
}

func setPostCurrentRevision(ctx context.Context, tx pgx.Tx, postID, revisionNo int64) error {
	if _, err := tx.Exec(ctx, `
		UPDATE posts
		SET current_revision = $2
		WHERE id = $1
	`, postID, revisionNo); err != nil {
		return fmt.Errorf("set post current revision: %w", err)
	}
	return nil
}

func currentPostAttachmentIDs(ctx context.Context, tx pgx.Tx, postID int64) ([]int64, error) {
	var resourceType string
	var resourceID int64
	err := tx.QueryRow(ctx, `
		SELECT 'topic', topics.id
		FROM topics
		WHERE topics.content_id = $1
		UNION ALL
		SELECT 'comment', comments.id
		FROM comments
		WHERE comments.content_id = $1
		LIMIT 1
	`, postID).Scan(&resourceType, &resourceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return []int64{}, nil
		}
		return nil, fmt.Errorf("resolve post attachment resource: %w", err)
	}
	return loadForumAttachmentIDs(ctx, tx, resourceType, resourceID)
}

func loadForumAttachmentIDs(ctx context.Context, tx pgx.Tx, resourceType string, resourceID int64) ([]int64, error) {
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT attachment_id
		FROM attachment_references
		WHERE resource_type = $1 AND resource_id = $2 AND context = $3
		ORDER BY attachment_id ASC
	`, resourceType, resourceID, forumAttachmentContext)
	if err != nil {
		return nil, fmt.Errorf("load forum attachment ids: %w", err)
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan forum attachment id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate forum attachment ids: %w", err)
	}
	return out, nil
}

func currentTopicSnapshotForPost(ctx context.Context, tx pgx.Tx, postID int64) (*TopicRevisionSnapshotInput, error) {
	var snapshot currentTopicRevisionSnapshot
	err := tx.QueryRow(ctx, `
		SELECT topics.id, topics.title, categories.slug
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		WHERE topics.content_id = $1
	`, postID).Scan(&snapshot.TopicID, &snapshot.Title, &snapshot.CategorySlug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load topic revision snapshot: %w", err)
	}
	tagSlugs, err := currentTopicTagSlugs(ctx, tx, snapshot.TopicID)
	if err != nil {
		return nil, err
	}
	return &TopicRevisionSnapshotInput{
		TopicID:      snapshot.TopicID,
		Title:        snapshot.Title,
		CategorySlug: snapshot.CategorySlug,
		TagSlugs:     tagSlugs,
	}, nil
}

func currentTopicTagSlugs(ctx context.Context, tx pgx.Tx, topicID int64) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT tags.slug
		FROM topic_tags
		JOIN tags ON tags.id = topic_tags.tag_id
		WHERE topic_tags.topic_id = $1
		ORDER BY tags.slug ASC
	`, topicID)
	if err != nil {
		return nil, fmt.Errorf("load topic revision tag slugs: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("scan topic revision tag slug: %w", err)
		}
		out = append(out, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topic revision tag slugs: %w", err)
	}
	return out, nil
}

func normalizeRevisionTextArray(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizeRevisionInt64Array(values []int64) []int64 {
	if len(values) == 0 {
		return []int64{}
	}
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func topicTagSlugs(tags []TopicTagSummary) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, tag.Slug)
	}
	return normalizeRevisionTextArray(out)
}

func backfillLegacyActor(post backfillPostRow, legacy []legacyPostRevisionRow, index int) int64 {
	if index == 0 {
		return sqlNullInt64Value(post.CreatedByUserID)
	}
	return sqlNullInt64Value(legacy[index-1].SupersededByUserID)
}

func backfillLegacyCommittedAt(post backfillPostRow, legacy []legacyPostRevisionRow, index int) time.Time {
	if index == 0 {
		return post.CreatedAt
	}
	return legacy[index-1].CreatedAt
}

func sqlNullInt64Value(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

func nullSQLInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}
