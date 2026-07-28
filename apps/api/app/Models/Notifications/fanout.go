package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
)

func channelsForType(policy options.NotificationPolicy, kind string) (bool, bool) {
	channel := policy.Moderation
	if kind == TypeReply {
		channel = policy.Reply
	} else if kind == TypeMention {
		channel = policy.Mention
	}
	return channel.InAppEnabled, channel.EmailEnabled
}

type CommentEvent struct {
	CommentID, TopicID, ActorUserID       int64
	TopicAuthorUserID, ParentAuthorUserID int64
	MentionedUsernames                    []string
}

type TopicEvent struct {
	TopicID, ActorUserID int64
	MentionedUsernames   []string
}

type ModerationEvent struct {
	DecisionID               int64
	TargetType               string
	TargetID, ReviewerUserID int64
	Approved                 bool
	ReviewNote               string
}

func (o *Outbox) NotifyModerationTx(ctx context.Context, tx pgx.Tx, event ModerationEvent) error {
	policy, err := o.notificationPolicy(ctx)
	if err != nil {
		return err
	}
	var authorID sql.NullInt64
	topicID := event.TargetID
	switch event.TargetType {
	case "topic":
		err = tx.QueryRow(ctx, `SELECT author_user_id FROM topics WHERE id=$1`, event.TargetID).Scan(&authorID)
	case "comment":
		err = tx.QueryRow(ctx, `SELECT author_user_id, topic_id FROM comments WHERE id=$1`, event.TargetID).Scan(&authorID, &topicID)
	default:
		return fmt.Errorf("unsupported moderation target type %q", event.TargetType)
	}
	if err != nil {
		return fmt.Errorf("load moderation notification target: %w", err)
	}
	kind := TypeModerationRejected
	if event.Approved {
		kind = TypeModerationApproved
	}
	if authorID.Valid && authorID.Int64 != event.ReviewerUserID {
		var email string
		err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1 AND status='active'`, authorID.Int64).Scan(&email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("load moderation notification recipient: %w", err)
		}
		if err == nil {
			payloadData := map[string]any{"targetType": event.TargetType, "targetId": event.TargetID, "topicId": topicID, "reviewNote": event.ReviewNote}
			if event.TargetType == "comment" {
				payloadData["commentId"] = event.TargetID
			}
			payload, _ := json.Marshal(payloadData)
			key := fmt.Sprintf("moderation:%d:%s:%d", event.DecisionID, kind, authorID.Int64)
			inApp, emailEnabled := channelsForType(policy, kind)
			if err := o.createProjectionsTx(ctx, tx, CreateBundleInput{
				Notification: CreateInput{RecipientUserID: authorID.Int64, Type: kind, ActorUserID: &event.ReviewerUserID, TargetType: event.TargetType, TargetID: event.TargetID, Payload: payload, DedupeKey: key},
				Delivery:     CreateDeliveryInput{Recipient: email, TemplateKey: "forum." + kind, TemplateData: payload, IdempotencyKey: key},
			}, inApp, emailEnabled); err != nil {
				return err
			}
		}
	}
	if event.Approved {
		return o.notifyApprovedContentTx(ctx, tx, event)
	}
	return nil
}

// notifyApprovedContentTx deliberately reloads the persisted source inside the
// moderation transaction. Request input is no longer authoritative after
// filters and review have run.
func (o *Outbox) notifyApprovedContentTx(ctx context.Context, tx pgx.Tx, event ModerationEvent) error {
	if event.TargetType == "topic" {
		var authorID sql.NullInt64
		var rawContent string
		if err := tx.QueryRow(ctx, `SELECT topics.author_user_id, posts.raw_content FROM topics JOIN posts ON posts.id=topics.content_id WHERE topics.id=$1`, event.TargetID).Scan(&authorID, &rawContent); err != nil {
			return err
		}
		if !authorID.Valid {
			return nil
		}
		return o.NotifyTopicTx(ctx, tx, TopicEvent{TopicID: event.TargetID, ActorUserID: authorID.Int64, MentionedUsernames: forum.MentionedUsernames(rawContent)})
	}
	if event.TargetType != "comment" {
		return nil
	}
	var actorID sql.NullInt64
	var topicID, topicAuthorID, parentAuthorID int64
	var rawContent string
	if err := tx.QueryRow(ctx, `
		SELECT comments.author_user_id, comments.topic_id, COALESCE(topics.author_user_id, 0),
			COALESCE(parent.author_user_id, 0), posts.raw_content
		FROM comments
		JOIN topics ON topics.id=comments.topic_id
		JOIN posts ON posts.id=comments.content_id
		LEFT JOIN comments parent ON parent.id=comments.parent_comment_id
		WHERE comments.id=$1`, event.TargetID).Scan(&actorID, &topicID, &topicAuthorID, &parentAuthorID, &rawContent); err != nil {
		return err
	}
	if !actorID.Valid {
		return nil
	}
	return o.NotifyCommentTx(ctx, tx, CommentEvent{CommentID: event.TargetID, TopicID: topicID, ActorUserID: actorID.Int64, TopicAuthorUserID: topicAuthorID, ParentAuthorUserID: parentAuthorID, MentionedUsernames: forum.MentionedUsernames(rawContent)})
}

func (o *Outbox) NotifyCommentTx(ctx context.Context, tx pgx.Tx, event CommentEvent) error {
	policy, err := o.notificationPolicy(ctx)
	if err != nil {
		return err
	}
	type recipient struct {
		id          int64
		email, kind string
	}
	recipients := map[string]recipient{}
	replyRecipientID := event.ParentAuthorUserID
	if replyRecipientID == 0 {
		replyRecipientID = event.TopicAuthorUserID
	}
	if replyRecipientID > 0 && replyRecipientID != event.ActorUserID {
		var email string
		err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1 AND status='active'`, replyRecipientID).Scan(&email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("load reply notification recipient: %w", err)
		}
		if err == nil {
			recipients[fmt.Sprintf("reply:%d", replyRecipientID)] = recipient{replyRecipientID, email, TypeReply}
		}
	}
	for _, username := range event.MentionedUsernames {
		var id int64
		var email string
		err := tx.QueryRow(ctx, `SELECT id,email FROM users WHERE username_lower=LOWER($1) AND status='active'`, username).Scan(&id, &email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("load mention notification recipient: %w", err)
		}
		if err != nil || id == event.ActorUserID {
			continue
		}
		recipients[fmt.Sprintf("mention:%d", id)] = recipient{id, email, TypeMention}
	}
	for key, item := range recipients {
		payload, _ := json.Marshal(map[string]any{"commentId": event.CommentID, "topicId": event.TopicID})
		dedupe := fmt.Sprintf("comment:%d:%s", event.CommentID, key)
		inApp, emailEnabled := channelsForType(policy, item.kind)
		if err := o.createProjectionsTx(ctx, tx, CreateBundleInput{
			Notification: CreateInput{RecipientUserID: item.id, Type: item.kind, ActorUserID: &event.ActorUserID, TargetType: "comment", TargetID: event.CommentID, Payload: payload, DedupeKey: dedupe},
			Delivery:     CreateDeliveryInput{Recipient: item.email, TemplateKey: "forum." + item.kind, TemplateData: payload, IdempotencyKey: dedupe},
		}, inApp, emailEnabled); err != nil {
			return err
		}
	}
	return nil
}

// NotifyTopicTx mirrors comment mention fanout for an active topic creation.
// Topic authors are not notified about their own new topic; only explicit active
// mentions become projections.
func (o *Outbox) NotifyTopicTx(ctx context.Context, tx pgx.Tx, event TopicEvent) error {
	policy, err := o.notificationPolicy(ctx)
	if err != nil {
		return err
	}
	recipients := make(map[int64]string, len(event.MentionedUsernames))
	for _, username := range event.MentionedUsernames {
		var recipientID int64
		var email string
		err := tx.QueryRow(ctx, `SELECT id, email FROM users WHERE username_lower=LOWER($1) AND status='active'`, username).Scan(&recipientID, &email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("load topic mention notification recipient: %w", err)
		}
		if err != nil || recipientID == event.ActorUserID {
			continue
		}
		recipients[recipientID] = email
	}
	for recipientID, email := range recipients {
		payload, _ := json.Marshal(map[string]any{"topicId": event.TopicID})
		key := fmt.Sprintf("topic:%d:mention:%d", event.TopicID, recipientID)
		inApp, emailEnabled := channelsForType(policy, TypeMention)
		if err := o.createProjectionsTx(ctx, tx, CreateBundleInput{
			Notification: CreateInput{RecipientUserID: recipientID, Type: TypeMention, ActorUserID: &event.ActorUserID, TargetType: "topic", TargetID: event.TopicID, Payload: payload, DedupeKey: key},
			Delivery:     CreateDeliveryInput{Recipient: email, TemplateKey: "forum." + TypeMention, TemplateData: payload, IdempotencyKey: key},
		}, inApp, emailEnabled); err != nil {
			return err
		}
	}
	return nil
}
