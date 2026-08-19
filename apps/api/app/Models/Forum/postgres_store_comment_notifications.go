package forum

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) notifyCreatedCommentTx(ctx context.Context, tx pgx.Tx, input CreateCommentRecord, commentID int64) error {
	if s.notifications == nil {
		return nil
	}
	if input.Status == CommentStatusPending {
		if err := s.notifications.NotifyPendingReviewTx(ctx, tx, PendingReviewNotificationInput{TargetType: "comment", TargetID: commentID, TopicID: input.TopicID, AuthorUserID: input.AuthorUserID, Revision: 1}); err != nil {
			return fmt.Errorf("create pending comment notifications: %w", err)
		}
		return nil
	}
	if input.Status != CommentStatusActive {
		return nil
	}
	parentAuthorID := int64(0)
	if input.Parent != nil {
		parentAuthorID = input.Parent.AuthorUserID
	}
	if err := s.notifications.NotifyCommentTx(ctx, tx, CommentNotificationInput{CommentID: commentID, TopicID: input.TopicID, ActorUserID: input.AuthorUserID, TopicAuthorUserID: input.TopicAuthorUserID, ParentAuthorUserID: parentAuthorID, MentionedUsernames: input.MentionedUsernames}); err != nil {
		return fmt.Errorf("create comment notifications: %w", err)
	}
	return nil
}
