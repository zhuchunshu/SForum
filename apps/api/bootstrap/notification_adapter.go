package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
)

type forumNotificationAdapter struct{ outbox *notifications.Outbox }

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
