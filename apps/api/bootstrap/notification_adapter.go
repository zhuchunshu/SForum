package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
)

type forumNotificationAdapter struct{ outbox *notifications.Outbox }

type notificationTargetPreviewAdapter struct{ store *forum.PostgresStore }

func (a notificationTargetPreviewAdapter) ResolveNotificationTargetPreview(ctx context.Context, _ int64, targetType string, targetID int64) (notifications.TargetPreview, bool, error) {
	preview, available, err := a.store.ResolveNotificationTargetPreview(ctx, targetType, targetID)
	if err != nil || !available {
		return notifications.TargetPreview{}, available, err
	}
	convertAuthor := func(user *forum.UserSummary) *notifications.TargetPreviewAuthor {
		if user == nil {
			return nil
		}
		return &notifications.TargetPreviewAuthor{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Avatar: user.Avatar}
	}
	convertContent := func(content forum.NotificationTargetPreviewContent) notifications.TargetPreviewContent {
		return notifications.TargetPreviewContent{Type: content.Type, ID: content.ID, Excerpt: content.Excerpt, Author: convertAuthor(content.Author)}
	}
	result := notifications.TargetPreview{TopicID: preview.TopicID, TopicTitle: preview.TopicTitle, Content: convertContent(preview.Content)}
	if preview.Context != nil {
		context := convertContent(*preview.Context)
		result.Context = &context
	}
	return result, true, nil
}

func (a forumNotificationAdapter) NotifyCommentTx(ctx context.Context, tx pgx.Tx, input forum.CommentNotificationInput) error {
	return a.outbox.NotifyCommentTx(ctx, tx, notifications.CommentEvent{CommentID: input.CommentID, TopicID: input.TopicID, ActorUserID: input.ActorUserID, TopicAuthorUserID: input.TopicAuthorUserID, ParentAuthorUserID: input.ParentAuthorUserID, MentionedUsernames: input.MentionedUsernames})
}

func (a forumNotificationAdapter) NotifyTopicTx(ctx context.Context, tx pgx.Tx, input forum.TopicNotificationInput) error {
	return a.outbox.NotifyTopicTx(ctx, tx, notifications.TopicEvent{TopicID: input.TopicID, ActorUserID: input.ActorUserID, MentionedUsernames: input.MentionedUsernames})
}

type moderationNotificationAdapter struct{ outbox *notifications.Outbox }

func (a moderationNotificationAdapter) NotifyModerationTx(ctx context.Context, tx pgx.Tx, input moderation.DecisionNotificationInput) error {
	return a.outbox.NotifyModerationTx(ctx, tx, notifications.ModerationEvent{DecisionID: input.DecisionID, TargetType: input.TargetType, TargetID: input.TargetID, ReviewerUserID: input.ReviewerUserID, Approved: input.Approved, ReviewNote: input.ReviewNote})
}
