package forum

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const forumAttachmentContext = "content"

// replaceForumAttachmentReferences 在事务内把一条 topic/comment 的正文附件引用
// 全量替换为 attachmentIDs。
//
// 一致性边界：
//   - 基表 UNIQUE (attachment_id, resource_type, resource_id, context) 保证同一个
//     附件不会在同一条内容里被重复绑定；
//   - (resource_type, resource_id, context) 的唯一索引只覆盖槽位型资源
//     (site/seo/user/extension)，topic/comment 属于多值型，可以绑定多个附件。
//     见 knowledge/decisions/2026-10-24-attachment-reference-uniqueness-scope.md。
//
// 写步骤刻意保持「先校验、再全量替换、最后按引用行重算计数」：任何一步失败都让
// 调用方回滚整个内容事务，不会留下半套引用。
func replaceForumAttachmentReferences(ctx context.Context, tx pgx.Tx, resourceType string, resourceID, actorUserID int64, attachmentIDs []int64) error {
	if resourceType != "topic" && resourceType != "comment" {
		return ErrInvalidContent
	}
	if len(attachmentIDs) > 0 {
		unavailable, err := lockUsableForumAttachments(ctx, tx, resourceType, resourceID, actorUserID, attachmentIDs)
		if err != nil {
			return err
		}
		if len(unavailable) > 0 {
			// 附件失效是可自愈的用户侧问题，但必须能在服务端日志里回答「哪几张图、
			// 哪条内容」，否则线上只能看到一个 422。
			slog.WarnContext(ctx, "forum: unavailable attachments in submitted content",
				"resourceType", resourceType,
				"resourceId", resourceID,
				"actorUserId", actorUserID,
				"attachmentIds", unavailable,
			)
			return &AttachmentUnavailableError{
				ResourceType:  resourceType,
				ResourceID:    resourceID,
				AttachmentIDs: unavailable,
			}
		}
	}

	removed, err := deleteForumAttachmentReferences(ctx, tx, resourceType, resourceID)
	if err != nil {
		return err
	}
	if len(attachmentIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO attachment_references
			  (attachment_id, resource_type, resource_id, context, created_by_user_id)
			SELECT unnest($1::bigint[]), $2, $3, $4, $5
			ON CONFLICT DO NOTHING
		`, attachmentIDs, resourceType, resourceID, forumAttachmentContext, actorUserID); err != nil {
			return mapForumAttachmentWriteError(err, resourceType, resourceID, attachmentIDs)
		}
	}

	// 计数按引用行重算而不是 +1/-1：即使历史数据已经漂移，也会在这次写入时自愈。
	touched := unionAttachmentIDs(removed, attachmentIDs)
	if len(touched) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE attachments a
		SET reference_count = counts.amount, updated_at = now()
		FROM (
			SELECT keep.id AS attachment_id, COUNT(ref.attachment_id)::integer AS amount
			FROM unnest($1::bigint[]) AS keep(id)
			LEFT JOIN attachment_references ref ON ref.attachment_id = keep.id
			GROUP BY keep.id
		) counts
		WHERE a.id = counts.attachment_id
	`, touched); err != nil {
		return fmt.Errorf("recount forum attachment references: %w", err)
	}
	return nil
}

// lockUsableForumAttachments 校验附件可用并锁行，返回不可用的附件 id。
//
// 可用口径与历史行为一致：active + public，且「自己上传的」或「已被同一条内容
// 引用过」（跨作者编辑/恢复可以保留他人图片）。ORDER BY id 固定加锁顺序，避免并发
// 发帖共享同一批附件时互相等待成环。
func lockUsableForumAttachments(ctx context.Context, tx pgx.Tx, resourceType string, resourceID, actorUserID int64, attachmentIDs []int64) ([]int64, error) {
	rows, err := tx.Query(ctx, `
		SELECT a.id
		FROM attachments a
		WHERE a.id = ANY($1::bigint[])
		  AND a.status = 'active'
		  AND a.visibility = 'public'
		  AND (
		    a.owner_user_id = $2
		    OR EXISTS (
		      SELECT 1
		      FROM attachment_references ar
		      WHERE ar.attachment_id = a.id
		        AND ar.resource_type = $3
		        AND ar.resource_id = $4
		        AND ar.context = $5
		    )
		  )
		ORDER BY a.id
		FOR UPDATE
	`, attachmentIDs, actorUserID, resourceType, resourceID, forumAttachmentContext)
	if err != nil {
		return nil, fmt.Errorf("validate forum attachments: %w", err)
	}
	defer rows.Close()

	usable := make(map[int64]struct{}, len(attachmentIDs))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan forum attachment: %w", err)
		}
		usable[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate forum attachments: %w", err)
	}

	var unavailable []int64
	for _, id := range attachmentIDs {
		if _, ok := usable[id]; !ok {
			unavailable = append(unavailable, id)
		}
	}
	return unavailable, nil
}

// deleteForumAttachmentReferences 清空一条内容的正文引用，返回被清掉的附件 id。
func deleteForumAttachmentReferences(ctx context.Context, tx pgx.Tx, resourceType string, resourceID int64) ([]int64, error) {
	rows, err := tx.Query(ctx, `
		DELETE FROM attachment_references
		WHERE resource_type = $1 AND resource_id = $2 AND context = $3
		RETURNING attachment_id
	`, resourceType, resourceID, forumAttachmentContext)
	if err != nil {
		return nil, fmt.Errorf("clear forum attachment references: %w", err)
	}
	defer rows.Close()

	var removed []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan cleared forum attachment reference: %w", err)
		}
		removed = append(removed, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cleared forum attachment references: %w", err)
	}
	return removed, nil
}

func unionAttachmentIDs(left, right []int64) []int64 {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(left)+len(right))
	merged := make([]int64, 0, len(left)+len(right))
	for _, group := range [][]int64{left, right} {
		for _, id := range group {
			if id <= 0 {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			merged = append(merged, id)
		}
	}
	return merged
}

// mapForumAttachmentWriteError 把引用写入的唯一约束冲突翻译成稳定错误。
//
// 正常情况下 topic/comment 不会再撞 attachment_references 的唯一性（多值型已被
// 排除在槽位索引之外，基表唯一性由 ON CONFLICT DO NOTHING 吸收）。真的撞上说明
// schema 与代码假设不一致，此时宁可给调用方一个具名 409，也不要冒泡成不可诊断的
// 500 internal_error。
func mapForumAttachmentWriteError(err error, resourceType string, resourceID int64, attachmentIDs []int64) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.HasPrefix(pgErr.ConstraintName, "attachment_references_") {
		return &AttachmentConflictError{
			Constraint:    pgErr.ConstraintName,
			ResourceType:  resourceType,
			ResourceID:    resourceID,
			AttachmentIDs: attachmentIDs,
		}
	}
	return fmt.Errorf("insert forum attachment references: %w", err)
}
