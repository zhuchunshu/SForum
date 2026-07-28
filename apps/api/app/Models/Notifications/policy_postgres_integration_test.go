package notifications

import "testing"

func TestNotificationPolicyUsesPostgresBooleanAggregates(t *testing.T) {
	ctx, tx := notificationChannelPostgresTx(t)
	store := newPostgresStore(tx)

	if _, err := tx.Exec(ctx, `
		UPDATE notification_type_policies
		SET enabled = CASE
			WHEN type = 'reply' AND channel = 'email' THEN FALSE
			ELSE TRUE
		END
		WHERE type IN ('reply', 'mention', 'moderation_approved', 'moderation_rejected')
		  AND channel IN ('in_app', 'email')`); err != nil {
		t.Fatal(err)
	}

	policy, err := store.NotificationPolicy(ctx)
	if err != nil {
		t.Fatalf("read compatibility notification policy: %v", err)
	}
	if !policy.Reply.InAppEnabled || policy.Reply.EmailEnabled ||
		!policy.Mention.InAppEnabled || !policy.Mention.EmailEnabled ||
		!policy.Moderation.InAppEnabled || !policy.Moderation.EmailEnabled {
		t.Fatalf("unexpected compatibility policy: %#v", policy)
	}
}
