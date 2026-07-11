package bootstrap

import (
	"context"
	"encoding/json"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
)

type passwordResetOutbox struct{ outbox *notifications.Outbox }

func (a passwordResetOutbox) QueuePasswordReset(ctx context.Context, token identity.CreatePasswordResetTokenInput, message identity.PasswordResetMail) error {
	data, err := json.Marshal(map[string]string{"subject": message.Subject, "textBody": message.TextBody})
	if err != nil {
		return err
	}
	_, err = a.outbox.QueuePasswordReset(ctx, notifications.QueuePasswordResetInput{
		UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt, RequestIPHash: token.RequestIPHash,
		Mail: notifications.QueueMailInput{Recipient: message.Recipient, TemplateKey: "identity.password_reset", TemplateData: data, IdempotencyKey: message.IdempotencyKey},
	})
	return err
}
