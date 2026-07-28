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
