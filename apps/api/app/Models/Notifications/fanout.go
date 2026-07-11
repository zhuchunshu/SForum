package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type CommentEvent struct {
	CommentID, TopicID, ActorUserID int64
	ParentAuthorUserID              int64
	MentionedUsernames              []string
}

type ModerationEvent struct {
	DecisionID               int64
	TargetType               string
	TargetID, ReviewerUserID int64
	Approved                 bool
	ReviewNote               string
}

func (o *Outbox) NotifyModerationTx(ctx context.Context, tx pgx.Tx, event ModerationEvent) error {
	table := "topics"
	if event.TargetType == "comment" {
		table = "comments"
	}
	var userID int64
	var email string
	if err := tx.QueryRow(ctx, `SELECT users.id, users.email FROM `+table+` target JOIN users ON users.id=target.author_user_id WHERE target.id=$1 AND users.status='active'`, event.TargetID).Scan(&userID, &email); err != nil || userID == event.ReviewerUserID {
		return nil
	}
	kind := TypeModerationRejected
	if event.Approved {
		kind = TypeModerationApproved
	}
	payload, _ := json.Marshal(map[string]any{"targetType": event.TargetType, "targetId": event.TargetID, "reviewNote": event.ReviewNote})
	key := fmt.Sprintf("moderation:%d:%s:%d", event.DecisionID, kind, userID)
	_, err := o.CreateBundleTx(ctx, tx, CreateBundleInput{
		Notification: CreateInput{RecipientUserID: userID, Type: kind, ActorUserID: &event.ReviewerUserID, TargetType: event.TargetType, TargetID: event.TargetID, Payload: payload, DedupeKey: key},
		Delivery:     CreateDeliveryInput{Recipient: email, TemplateKey: "forum." + kind, TemplateData: payload, IdempotencyKey: key},
	})
	return err
}

func (o *Outbox) NotifyCommentTx(ctx context.Context, tx pgx.Tx, event CommentEvent) error {
	type recipient struct {
		id          int64
		email, kind string
	}
	recipients := map[string]recipient{}
	if event.ParentAuthorUserID > 0 && event.ParentAuthorUserID != event.ActorUserID {
		var email string
		if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1 AND status='active'`, event.ParentAuthorUserID).Scan(&email); err == nil {
			recipients[fmt.Sprintf("reply:%d", event.ParentAuthorUserID)] = recipient{event.ParentAuthorUserID, email, TypeReply}
		}
	}
	for _, username := range event.MentionedUsernames {
		var id int64
		var email string
		if err := tx.QueryRow(ctx, `SELECT id,email FROM users WHERE username_lower=LOWER($1) AND status='active'`, username).Scan(&id, &email); err != nil || id == event.ActorUserID {
			continue
		}
		recipients[fmt.Sprintf("mention:%d", id)] = recipient{id, email, TypeMention}
	}
	for key, item := range recipients {
		payload, _ := json.Marshal(map[string]any{"commentId": event.CommentID, "topicId": event.TopicID})
		dedupe := fmt.Sprintf("comment:%d:%s", event.CommentID, key)
		if _, err := o.CreateBundleTx(ctx, tx, CreateBundleInput{
			Notification: CreateInput{RecipientUserID: item.id, Type: item.kind, ActorUserID: &event.ActorUserID, TargetType: "comment", TargetID: event.CommentID, Payload: payload, DedupeKey: dedupe},
			Delivery:     CreateDeliveryInput{Recipient: item.email, TemplateKey: "forum." + item.kind, TemplateData: payload, IdempotencyKey: dedupe},
		}); err != nil {
			return err
		}
	}
	return nil
}
