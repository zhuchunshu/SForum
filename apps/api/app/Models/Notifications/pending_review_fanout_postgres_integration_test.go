package notifications

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type permissionRecipientFixture struct{ userIDs []int64 }

func (f permissionRecipientFixture) ListActiveUserIDsWithPermissionTx(context.Context, pgx.Tx, string) ([]int64, error) {
	return f.userIDs, nil
}

func TestPendingReviewFanoutIsPermissionResolvedAuthorExcludedAndIdempotentPostgres(t *testing.T) {
	ctx, tx := notificationChannelPostgresTx(t)
	authorID := insertNotificationChannelUser(t, ctx, tx, "pending_author")
	reviewerA := insertNotificationChannelUser(t, ctx, tx, "pending_reviewer_a")
	reviewerB := insertNotificationChannelUser(t, ctx, tx, "pending_reviewer_b")
	if _, err := tx.Exec(ctx, `
		UPDATE notification_type_policies
		SET enabled=(channel='in_app'), recommended_enabled=(channel='in_app')
		WHERE type='moderation_pending'
	`); err != nil {
		t.Fatal(err)
	}
	outbox := NewOutbox(nil, newPostgresStore(tx), &channelProjectionJobs{}).
		WithPermissionRecipients(permissionRecipientFixture{userIDs: []int64{authorID, reviewerA, reviewerB}})
	event := PendingReviewEvent{TargetType: "topic", TargetID: time.Now().UnixNano(), TopicID: 42, AuthorUserID: authorID, Revision: 2, Title: "Needs review"}
	if err := outbox.NotifyPendingReviewTx(ctx, tx, event); err != nil {
		t.Fatal(err)
	}
	if err := outbox.NotifyPendingReviewTx(ctx, tx, event); err != nil {
		t.Fatal(err)
	}
	pattern := fmt.Sprintf("moderation-pending:topic:%d:2:%%", event.TargetID)
	rows, err := tx.Query(ctx, `SELECT recipient_user_id,target_type,payload->>'title' FROM notifications WHERE dedupe_key LIKE $1 ORDER BY recipient_user_id`, pattern)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := []int64{}
	for rows.Next() {
		var userID int64
		var targetType, title string
		if err := rows.Scan(&userID, &targetType, &title); err != nil {
			t.Fatal(err)
		}
		if targetType != "moderation_topic" || title != "Needs review" {
			t.Fatalf("pending notification target=%q title=%q", targetType, title)
		}
		got = append(got, userID)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != reviewerA || got[1] != reviewerB {
		t.Fatalf("pending review recipients=%v want [%d %d]", got, reviewerA, reviewerB)
	}
}
