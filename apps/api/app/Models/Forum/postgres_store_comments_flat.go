package forum

// flat 评论流（时间流）读路径：列表分页、keyset 续页、楼层位置反查计数。
// 从 postgres_store_ops.go 拆出，避免该文件继续增长；排序契约（created_at ASC, id ASC）
// 与 CountCommentsBefore 的计数谓词必须保持严格对齐，放在同一文件里便于对照维护。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// listCommentsFlat 对 active 评论按 created_at ASC, id ASC（时间流）分页（M5：支持 after keyset）。
// 时间流顺序保证：新评论永远追加在列表末尾（最底楼），既有评论的楼层号不因他人回复而漂移；
// 回复关系由每行的 replyTo 引用块表达，不再把子孙插到父评论后面。
// 依赖 comments_topic_created_idx(topic_id, created_at, id)。
// 公开路径 Total 用 topics.comment_count；含软删扩展查询时仍精确计数。
func (s *PostgresStore) listCommentsFlat(ctx context.Context, input CommentListInput) (CommentList, error) {
	total, err := s.commentListTotalFlat(ctx, input)
	if err != nil {
		return CommentList{}, err
	}

	var cursor *commentListCursor
	if after := strings.TrimSpace(input.After); after != "" {
		decoded, decErr := decodeCommentListCursor(after)
		if decErr != nil {
			return CommentList{}, ErrInvalidCursor
		}
		cursor = &decoded
		input.Page = 1
	}

	fetchLimit := input.PerPage + 1
	var rows pgx.Rows
	if cursor != nil {
		key, keyErr := commentCursorKeyTime(*cursor)
		if keyErr != nil {
			return CommentList{}, keyErr
		}
		// created_at ASC, id ASC：之后 = 时间更大的行。
		// `created_at >= $4` 让 PG 把时间键推进 Index Cond，避免 OR 前缀退化成全量 Filter。
		rows, err = s.pool.Query(ctx, commentSelectSQL()+`
			WHERE comments.topic_id = $1
			  AND (comments.status = 'active' OR ($2::boolean AND comments.status = 'deleted'
			    AND ($3::bigint = 0 OR comments.author_user_id = $3)))
			  AND comments.created_at >= $4::timestamptz
			  AND (comments.created_at > $4::timestamptz OR comments.id > $5)
			ORDER BY comments.created_at ASC, comments.id ASC
			LIMIT $6
		`, input.TopicID, input.IncludeDeleted, input.DeletedAuthorUserID, key, cursor.ID, fetchLimit)
	} else {
		offset := (input.Page - 1) * input.PerPage
		rows, err = s.pool.Query(ctx, commentSelectSQL()+`
			WHERE comments.topic_id = $1
			  AND (comments.status = 'active' OR ($2::boolean AND comments.status = 'deleted'
			    AND ($3::bigint = 0 OR comments.author_user_id = $3)))
			ORDER BY comments.created_at ASC, comments.id ASC
			LIMIT $4 OFFSET $5
		`, input.TopicID, input.IncludeDeleted, input.DeletedAuthorUserID, fetchLimit, offset)
	}
	if err != nil {
		return CommentList{}, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	items, err := scanCommentsWithAvatar(rows, s.avatarBuilder)
	if err != nil {
		return CommentList{}, err
	}
	hasMore := len(items) > input.PerPage
	if hasMore {
		items = items[:input.PerPage]
	}
	var nextCursor string
	if hasMore && len(items) > 0 {
		nextCursor, err = commentCursorFromItem(items[len(items)-1])
		if err != nil {
			return CommentList{}, err
		}
	}
	return CommentList{
		Items: items, Total: total, Page: input.Page, PerPage: input.PerPage,
		View: "flat", HasMore: hasMore, NextCursor: nextCursor,
	}, nil
}

// CountCommentsBefore 返回同主题内排在 (createdAt, id) 之前、对 viewer 可见的评论数。
// flat 视图排序为 ORDER BY created_at ASC, id ASC，行值比较 (created_at, id) < ($2, $3)
// 与排序严格对齐；可见范围谓词与 listCommentsFlat 的 WHERE 完全一致
// （active + 按软删可见范围计入的 deleted 墓碑），保证反查页码与实际分页结果一致。
func (s *PostgresStore) CountCommentsBefore(ctx context.Context, topicID int64, createdAt time.Time, id int64, includeDeleted bool, deletedAuthorUserID int64) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM comments
		WHERE topic_id = $1
		  AND (comments.status = 'active' OR ($4::boolean AND comments.status = 'deleted'
		    AND ($5::bigint = 0 OR comments.author_user_id = $5)))
		  AND ROW(created_at, id) < ROW($2::timestamptz, $3)
	`, topicID, createdAt, id, includeDeleted, deletedAuthorUserID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count comments before: %w", err)
	}
	return count, nil
}

// commentListTotalFlat 公开 active-only 用 denormalized comment_count（无 COUNT）。
// IncludeDeleted 为作者/管理范围，QPS 低，允许精确 COUNT（M6 审计保留）。
func (s *PostgresStore) commentListTotalFlat(ctx context.Context, input CommentListInput) (int64, error) {
	if !input.IncludeDeleted {
		var total int64
		err := s.pool.QueryRow(ctx, `
			SELECT comment_count FROM topics WHERE id = $1
		`, input.TopicID).Scan(&total)
		if err != nil {
			return 0, fmt.Errorf("topic comment_count: %w", err)
		}
		return total, nil
	}
	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM comments
		WHERE topic_id = $1
		  AND (status = 'active' OR (status = 'deleted'
		    AND ($2::bigint = 0 OR author_user_id = $2)))
	`, input.TopicID, input.DeletedAuthorUserID).Scan(&total); err != nil {
		return 0, fmt.Errorf("count comments: %w", err)
	}
	return total, nil
}
