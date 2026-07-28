package forum

import (
	"context"
	"errors"
	"fmt"
)

// ResolveNotificationTarget 复用公开论坛读路径的可见性规则。通知收件人身份
// 不能提升公开目标权限；后台审核入口不是公开通知链接。
func (s *PostgresStore) ResolveNotificationTarget(ctx context.Context, _ int64, targetType string, targetID int64) (bool, string, error) {
	if s == nil || targetID <= 0 {
		return false, "", nil
	}
	switch targetType {
	case "topic":
		if _, err := s.GetTopic(ctx, targetID); err != nil {
			if errors.Is(err, ErrTopicNotFound) {
				return false, "", nil
			}
			return false, "", err
		}
		return true, fmt.Sprintf("/t/%d", targetID), nil
	case "comment":
		comment, err := s.GetCommentSummary(ctx, targetID)
		if err != nil {
			if errors.Is(err, ErrCommentNotFound) {
				return false, "", nil
			}
			return false, "", err
		}
		if comment.Status != CommentStatusActive {
			return false, "", nil
		}
		if _, err := s.GetTopic(ctx, comment.TopicID); err != nil {
			if errors.Is(err, ErrTopicNotFound) {
				return false, "", nil
			}
			return false, "", err
		}
		return true, fmt.Sprintf("/t/%d#comment-%d", comment.TopicID, targetID), nil
	default:
		return false, "", nil
	}
}

// ResolveNotificationTargetPreview 只返回当前公开读路径可见的纯文本摘要。
// 评论上下文优先取直接父评论；根评论则取主题正文，便于收件人理解回复关系。
func (s *PostgresStore) ResolveNotificationTargetPreview(ctx context.Context, targetType string, targetID int64) (NotificationTargetPreview, bool, error) {
	if s == nil || targetID <= 0 {
		return NotificationTargetPreview{}, false, nil
	}
	switch targetType {
	case "topic":
		topic, err := s.GetTopic(ctx, targetID)
		if errors.Is(err, ErrTopicNotFound) {
			return NotificationTargetPreview{}, false, nil
		}
		if err != nil {
			return NotificationTargetPreview{}, false, err
		}
		return NotificationTargetPreview{
			TopicID: topic.ID, TopicTitle: topic.Title,
			Content: NotificationTargetPreviewContent{Type: "topic", ID: topic.ID, Excerpt: topic.Content.Excerpt, Author: topic.Author},
		}, true, nil
	case "comment":
		comment, err := getCommentByID(ctx, s.pool, targetID, s.avatarBuilder)
		if errors.Is(err, ErrCommentNotFound) {
			return NotificationTargetPreview{}, false, nil
		}
		if err != nil {
			return NotificationTargetPreview{}, false, err
		}
		if comment.Status != CommentStatusActive {
			return NotificationTargetPreview{}, false, nil
		}
		topic, err := s.GetTopic(ctx, comment.TopicID)
		if errors.Is(err, ErrTopicNotFound) {
			return NotificationTargetPreview{}, false, nil
		}
		if err != nil {
			return NotificationTargetPreview{}, false, err
		}
		preview := NotificationTargetPreview{
			TopicID: topic.ID, TopicTitle: topic.Title,
			Content: NotificationTargetPreviewContent{Type: "comment", ID: comment.ID, Excerpt: comment.Content.Excerpt, Author: comment.Author},
		}
		if comment.ReplyTo != nil {
			preview.Context = &NotificationTargetPreviewContent{Type: "comment", ID: comment.ReplyTo.ID, Excerpt: comment.ReplyTo.Excerpt, Author: comment.ReplyTo.Author}
		} else {
			preview.Context = &NotificationTargetPreviewContent{Type: "topic", ID: topic.ID, Excerpt: topic.Content.Excerpt, Author: topic.Author}
		}
		return preview, true, nil
	default:
		return NotificationTargetPreview{}, false, nil
	}
}
