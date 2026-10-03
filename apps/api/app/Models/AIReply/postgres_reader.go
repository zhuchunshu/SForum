package aireply

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// PostgresReader 从数据库读取生成所需的线程快照。
//
// 它只读公开可见的内容：status 为 active/locked 且分类可见性为 public 的主题，
// 以及 status='active' 的评论。机器人不该因为「是系统调用的」而看到隐藏内容、
// 待审内容或非公开分类——可见性判断必须与访客看到的一致，否则机器人会把不该
// 公开的东西复述出来。
type PostgresReader struct {
	pool *pgxpool.Pool
}

func NewPostgresReader(pool *pgxpool.Pool) *PostgresReader {
	return &PostgresReader{pool: pool}
}

func (r *PostgresReader) ReplyContext(ctx context.Context, topicID int64, parentCommentID int64, triggerCommentID int64) (ReplyContext, error) {
	if r == nil || r.pool == nil {
		return ReplyContext{}, ErrGeneratorUnavailable
	}
	var thread ReplyContext
	err := r.pool.QueryRow(ctx, `
		SELECT topics.title, COALESCE(posts.plain_text, '')
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		WHERE topics.id = $1
		  AND topics.status IN ('active', 'locked')
		  AND topics.deleted_at IS NULL
		  AND categories.visibility = 'public'
	`, topicID).Scan(&thread.TopicTitle, &thread.TopicBody)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReplyContext{}, fmt.Errorf("reply context: topic %d is not publicly visible", topicID)
	}
	if err != nil {
		return ReplyContext{}, fmt.Errorf("load topic title: %w", err)
	}

	recent, err := r.recentComments(ctx, topicID)
	if err != nil {
		return ReplyContext{}, err
	}
	thread.RecentComments = recent

	if parentCommentID > 0 {
		name, content, isBot, err := r.comment(ctx, parentCommentID)
		if err != nil {
			return ReplyContext{}, err
		}
		thread.ParentAuthorName = name
		thread.ParentContent = content
		thread.ParentIsBot = isBot
	}
	if triggerCommentID > 0 && triggerCommentID != parentCommentID {
		name, content, _, err := r.comment(ctx, triggerCommentID)
		if err != nil {
			return ReplyContext{}, err
		}
		thread.TriggerAuthor = name
		thread.TriggerContent = content
	}
	return thread, nil
}

// recentComments 取最近若干条公开评论并按时间正序返回：提示词需要的是对话
// 顺序，而不是倒序的查询结果。
func (r *PostgresReader) recentComments(ctx context.Context, topicID int64) ([]ThreadComment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(users.display_name, ''), users.username, ''), COALESCE(posts.plain_text, ''), users.kind
		FROM comments
		JOIN posts ON posts.id = comments.content_id
		LEFT JOIN users ON users.id = comments.author_user_id
		WHERE comments.topic_id = $1 AND comments.status = 'active'
		ORDER BY comments.created_at DESC
		LIMIT $2
	`, topicID, RecentCommentWindow)
	if err != nil {
		return nil, fmt.Errorf("load recent comments: %w", err)
	}
	defer rows.Close()
	collected := make([]ThreadComment, 0, RecentCommentWindow)
	for rows.Next() {
		var name, content string
		var kind *string
		if err := rows.Scan(&name, &content, &kind); err != nil {
			return nil, err
		}
		collected = append(collected, ThreadComment{
			AuthorName: name,
			Content:    content,
			IsBot:      kind != nil && identity.UserKind(*kind).IsBot(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 查询是倒序的，翻回时间正序。
	for left, right := 0, len(collected)-1; left < right; left, right = left+1, right-1 {
		collected[left], collected[right] = collected[right], collected[left]
	}
	return collected, nil
}

func (r *PostgresReader) comment(ctx context.Context, commentID int64) (string, string, bool, error) {
	var name, content string
	var kind *string
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(users.display_name, ''), users.username, ''), COALESCE(posts.plain_text, ''), users.kind
		FROM comments
		JOIN posts ON posts.id = comments.content_id
		LEFT JOIN users ON users.id = comments.author_user_id
		WHERE comments.id = $1 AND comments.status = 'active'
	`, commentID).Scan(&name, &content, &kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, fmt.Errorf("reply context: comment %d is not publicly visible", commentID)
	}
	if err != nil {
		return "", "", false, fmt.Errorf("load comment: %w", err)
	}
	return name, content, kind != nil && identity.UserKind(*kind).IsBot(), nil
}
