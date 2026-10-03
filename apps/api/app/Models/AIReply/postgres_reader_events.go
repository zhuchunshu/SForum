package aireply

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// CommentContext 补齐 comment.created 事件载荷里没有的判定输入：正文、父评论
// 作者与当前状态。事件只带定位信息，是为了让发布路径保持轻量。
func (r *PostgresReader) CommentContext(ctx context.Context, commentID int64) (CommentContext, error) {
	if r == nil || r.pool == nil {
		return CommentContext{}, ErrGeneratorUnavailable
	}
	var snapshot CommentContext
	err := r.pool.QueryRow(ctx, `
		SELECT comments.id, comments.topic_id, COALESCE(comments.author_user_id, 0),
		       COALESCE(comments.parent_comment_id, 0), comments.status,
		       COALESCE(parent.author_user_id, 0),
		       COALESCE(posts.plain_text, '')
		FROM comments
		JOIN posts ON posts.id = comments.content_id
		LEFT JOIN comments AS parent ON parent.id = comments.parent_comment_id
		WHERE comments.id = $1
	`, commentID).Scan(
		&snapshot.CommentID,
		&snapshot.TopicID,
		&snapshot.AuthorUserID,
		&snapshot.ParentID,
		&snapshot.Status,
		&snapshot.ParentAuthorUserID,
		&snapshot.RawContent,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommentContext{}, fmt.Errorf("comment %d not found", commentID)
	}
	if err != nil {
		return CommentContext{}, fmt.Errorf("load comment context: %w", err)
	}
	return snapshot, nil
}

// UserKind 查询账号类型，供防循环判定使用。
func (r *PostgresReader) UserKind(ctx context.Context, userID int64) (identity.UserKind, error) {
	if r == nil || r.pool == nil {
		return "", ErrGeneratorUnavailable
	}
	var raw string
	err := r.pool.QueryRow(ctx, `SELECT kind FROM users WHERE id = $1`, userID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("user %d not found", userID)
	}
	if err != nil {
		return "", fmt.Errorf("load user kind: %w", err)
	}
	if !identity.ValidUserKind(identity.UserKind(raw)) {
		return identity.UserKindHuman, nil
	}
	return identity.UserKind(raw), nil
}
