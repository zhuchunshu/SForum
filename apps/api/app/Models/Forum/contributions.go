package forum

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// 侧栏 avatar group 最多展示人数；超出以 +N 表示。
const topicContributorDisplayLimit = 5

// ListTopicContributionTimeline 返回公开主题的贡献时间线（newest-first）。
// 与授权修订列表共用 ledger 行，但剥离 reason / restorableFields 等敏感或管理字段。
func (s *PostgresStore) ListTopicContributionTimeline(ctx context.Context, topicID int64, input RevisionListInput) (TopicContributionTimeline, error) {
	postID, err := s.publicTopicPostID(ctx, topicID)
	if err != nil {
		return TopicContributionTimeline{}, err
	}
	list, err := s.listPostRevisions(ctx, postID, "topic", input)
	if err != nil {
		return TopicContributionTimeline{}, err
	}
	items := make([]TopicContributionEvent, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, publicContributionEvent(item))
	}
	return TopicContributionTimeline{
		Items:      items,
		PerPage:    list.PerPage,
		HasMore:    list.HasMore,
		NextCursor: list.NextCursor,
	}, nil
}

func publicContributionEvent(item ForumRevisionSummary) TopicContributionEvent {
	return TopicContributionEvent{
		RevisionNo:             item.RevisionNo,
		Current:                item.Current,
		Actor:                  item.Actor,
		Operation:              item.Operation,
		Origin:                 item.Origin,
		ChangedFields:          item.ChangedFields,
		CommittedAt:            item.CommittedAt,
		RestoredFromRevisionNo: item.RestoredFromRevisionNo,
		Redacted:               item.Redacted,
	}
}

// publicTopicPostID 仅解析公开可读主题的 content_id（与 GetTopic 可见性一致）。
func (s *PostgresStore) publicTopicPostID(ctx context.Context, topicID int64) (int64, error) {
	if topicID <= 0 {
		return 0, ErrTopicNotFound
	}
	var postID int64
	err := s.pool.QueryRow(ctx, `
		SELECT topics.content_id
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		WHERE topics.id = $1
		  AND topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
	`, topicID).Scan(&postID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrTopicNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("public topic post id: %w", err)
	}
	return postID, nil
}

// attachTopicContributors 填充详情侧栏用的贡献者 stack。
// 无修订 ledger 时退回作者一人，避免侧栏空白。
func (s *PostgresStore) attachTopicContributors(ctx context.Context, topic *TopicDetail) error {
	if topic == nil {
		return nil
	}
	postID := topic.Content.ID
	if postID <= 0 {
		s.fallbackTopicContributors(topic)
		return nil
	}

	var count int
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT actor_user_id)::int
		FROM post_revisions
		WHERE post_id = $1
		  AND actor_user_id IS NOT NULL
		  AND revision_no IS NOT NULL
	`, postID).Scan(&count); err != nil {
		return fmt.Errorf("count topic contributors: %w", err)
	}
	if count == 0 {
		s.fallbackTopicContributors(topic)
		return nil
	}

	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
		  SELECT actor_user_id AS user_id,
		    MAX(COALESCE(committed_at, created_at)) AS last_at
		  FROM post_revisions
		  WHERE post_id = $1
		    AND actor_user_id IS NOT NULL
		    AND revision_no IS NOT NULL
		  GROUP BY actor_user_id
		)
		SELECT ranked.user_id, users.username, users.display_name, users.email,
		  author_profiles.avatar_attachment_id,
		  author_attachments.id, author_attachments.public_id, author_attachments.owner_user_id,
		  author_attachments.content_type, author_attachments.status
		FROM ranked
		JOIN users ON users.id = ranked.user_id
		LEFT JOIN user_profiles author_profiles ON author_profiles.user_id = users.id
		LEFT JOIN attachments author_attachments ON author_attachments.id = author_profiles.avatar_attachment_id
		ORDER BY CASE WHEN ranked.user_id = $2 THEN 0 ELSE 1 END, ranked.last_at DESC
		LIMIT $3
	`, postID, topic.AuthorUserID, topicContributorDisplayLimit)
	if err != nil {
		return fmt.Errorf("list topic contributors: %w", err)
	}
	defer rows.Close()

	items := make([]UserSummary, 0, topicContributorDisplayLimit)
	for rows.Next() {
		var userID sql.NullInt64
		var username, displayName, email sql.NullString
		var avatarAttachmentID, attachmentID, attachmentOwnerID sql.NullInt64
		var attachmentPublicID, attachmentContentType, attachmentStatus sql.NullString
		if err := rows.Scan(
			&userID,
			&username,
			&displayName,
			&email,
			&avatarAttachmentID,
			&attachmentID,
			&attachmentPublicID,
			&attachmentOwnerID,
			&attachmentContentType,
			&attachmentStatus,
		); err != nil {
			return fmt.Errorf("scan topic contributor: %w", err)
		}
		summary := userSummaryWithAvatar(
			s.avatarBuilder,
			userID, username, displayName, email,
			avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID,
			attachmentContentType, attachmentStatus,
		)
		if summary != nil {
			items = append(items, *summary)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate topic contributors: %w", err)
	}

	items, count = ensureAuthorInContributors(topic, items, count)
	topic.Contributors = items
	topic.ContributorCount = count
	return nil
}

func (s *PostgresStore) fallbackTopicContributors(topic *TopicDetail) {
	if topic.Author != nil {
		topic.Contributors = []UserSummary{*topic.Author}
		topic.ContributorCount = 1
		return
	}
	topic.Contributors = nil
	topic.ContributorCount = 0
}

// ensureAuthorInContributors 保证作者在 stack 首位；ledger 缺 create actor 时补上。
func ensureAuthorInContributors(topic *TopicDetail, items []UserSummary, count int) ([]UserSummary, int) {
	if topic.Author == nil {
		return items, count
	}
	for _, item := range items {
		if item.ID == topic.Author.ID {
			return items, count
		}
	}
	out := make([]UserSummary, 0, len(items)+1)
	out = append(out, *topic.Author)
	out = append(out, items...)
	if len(out) > topicContributorDisplayLimit {
		out = out[:topicContributorDisplayLimit]
	}
	return out, count + 1
}
