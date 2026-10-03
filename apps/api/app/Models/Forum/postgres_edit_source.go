package forum

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) GetTopicEditSource(ctx context.Context, topicID int64) (EditableContentSource, error) {
	var source EditableContentSource
	err := s.pool.QueryRow(ctx, `
		SELECT posts.raw_content, posts.source_format, posts.editor_type,
		  posts.editor_version, posts.content_hash,
		  COALESCE((
		    SELECT array_agg(ar.attachment_id ORDER BY ar.attachment_id)
		    FROM attachment_references ar
		    WHERE ar.resource_type = 'topic' AND ar.resource_id = topics.id
		      AND ar.context = $2
		  ), ARRAY[]::bigint[]),
		  `+effectivePostCurrentRevisionSQL("posts")+`
		FROM topics
		JOIN posts ON posts.id = topics.content_id
		WHERE topics.id = $1
		  AND topics.status NOT IN ('hidden', 'deleted')
	`, topicID, forumAttachmentContext).Scan(
		&source.RawContent,
		&source.SourceFormat,
		&source.EditorType,
		&source.EditorVersion,
		&source.ContentHash,
		&source.AttachmentIDs,
		&source.CurrentRevision,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EditableContentSource{}, ErrTopicNotFound
	}
	if err != nil {
		return EditableContentSource{}, fmt.Errorf("get topic edit source: %w", err)
	}
	return source, nil
}

func (s *PostgresStore) GetCommentEditSource(ctx context.Context, commentID int64) (EditableContentSource, error) {
	var source EditableContentSource
	err := s.pool.QueryRow(ctx, `
		SELECT posts.raw_content, posts.source_format, posts.editor_type,
		  posts.editor_version, posts.content_hash,
		  COALESCE((
		    SELECT array_agg(ar.attachment_id ORDER BY ar.attachment_id)
		    FROM attachment_references ar
		    WHERE ar.resource_type = 'comment' AND ar.resource_id = comments.id
		      AND ar.context = $2
		  ), ARRAY[]::bigint[]),
		  `+effectivePostCurrentRevisionSQL("posts")+`
		FROM comments
		JOIN topics ON topics.id = comments.topic_id
		JOIN posts ON posts.id = comments.content_id
		WHERE comments.id = $1
		  AND comments.status NOT IN ('hidden', 'deleted')
		  AND topics.status NOT IN ('hidden', 'deleted')
	`, commentID, forumAttachmentContext).Scan(
		&source.RawContent,
		&source.SourceFormat,
		&source.EditorType,
		&source.EditorVersion,
		&source.ContentHash,
		&source.AttachmentIDs,
		&source.CurrentRevision,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EditableContentSource{}, ErrCommentNotFound
	}
	if err != nil {
		return EditableContentSource{}, fmt.Errorf("get comment edit source: %w", err)
	}
	return source, nil
}
