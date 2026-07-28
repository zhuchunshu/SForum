package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrChannelDeliveryNotFound = errors.New("notifications: channel delivery not found")
	ErrSubscriptionInvalid     = errors.New("notifications: subscription invalid")
	ErrSubscriptionNotFound    = errors.New("notifications: subscription not found")
)

type ChannelDelivery struct {
	ID                     int64           `json:"id"`
	NotificationID         *int64          `json:"notificationId,omitempty"`
	RecipientUserID        int64           `json:"-"`
	Type                   string          `json:"type"`
	Channel                string          `json:"channel"`
	ProviderExtensionID    string          `json:"providerExtensionId,omitempty"`
	ProviderArtifactDigest string          `json:"providerArtifactDigest,omitempty"`
	PayloadVersion         int             `json:"payloadVersion"`
	Payload                json.RawMessage `json:"-"`
	TargetMeta             json.RawMessage `json:"-"`
	IdempotencyKey         string          `json:"-"`
	Status                 string          `json:"status"`
	AttemptCount           int             `json:"attemptCount"`
	Reason                 string          `json:"reason,omitempty"`
	ErrorSummary           string          `json:"errorSummary,omitempty"`
	CreatedAt              time.Time       `json:"createdAt"`
	UpdatedAt              time.Time       `json:"updatedAt"`
	CompletedAt            *time.Time      `json:"completedAt,omitempty"`
}

type CreateChannelDeliveryInput struct {
	NotificationID  *int64
	RecipientUserID int64
	Type            string
	Channel         string
	PayloadVersion  int
	Payload         json.RawMessage
	TargetMeta      json.RawMessage
	IdempotencyKey  string
}

type ChannelDeliveryUpdate struct {
	ID                     int64
	Status                 string
	ProviderExtensionID    string
	ProviderArtifactDigest string
	AttemptCount           int
	Reason                 string
	ErrorSummary           string
}

func (s *PostgresStore) CreateChannelDeliveryTx(ctx context.Context, tx queryRunner, input CreateChannelDeliveryInput) (ChannelDelivery, error) {
	if input.Channel != "web_push" || input.RecipientUserID <= 0 || input.Type == "" || input.PayloadVersion <= 0 || input.IdempotencyKey == "" {
		return ChannelDelivery{}, ErrChannelDeliveryNotFound
	}
	payload, target := input.Payload, input.TargetMeta
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if len(target) == 0 {
		target = json.RawMessage(`{}`)
	}
	var item ChannelDelivery
	err := tx.QueryRow(ctx, `
		INSERT INTO notification_channel_deliveries
		  (notification_id,recipient_user_id,type,channel,payload_version,payload,target_meta,idempotency_key)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
		RETURNING id,notification_id,recipient_user_id,type,channel,provider_extension_id,
		  provider_artifact_digest,payload_version,payload,target_meta,idempotency_key,status,
		  attempt_count,reason,error_summary,created_at,updated_at,completed_at`,
		input.NotificationID, input.RecipientUserID, input.Type, input.Channel, input.PayloadVersion,
		payload, target, input.IdempotencyKey,
	).Scan(&item.ID, &item.NotificationID, &item.RecipientUserID, &item.Type, &item.Channel,
		&item.ProviderExtensionID, &item.ProviderArtifactDigest, &item.PayloadVersion, &item.Payload,
		&item.TargetMeta, &item.IdempotencyKey, &item.Status, &item.AttemptCount, &item.Reason,
		&item.ErrorSummary, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt)
	return item, err
}

func (s *PostgresStore) GetChannelDelivery(ctx context.Context, id int64) (ChannelDelivery, error) {
	var item ChannelDelivery
	err := s.runner.QueryRow(ctx, `SELECT id,notification_id,recipient_user_id,type,channel,
		provider_extension_id,provider_artifact_digest,payload_version,payload,target_meta,idempotency_key,
		status,attempt_count,reason,error_summary,created_at,updated_at,completed_at
		FROM notification_channel_deliveries WHERE id=$1`, id).Scan(
		&item.ID, &item.NotificationID, &item.RecipientUserID, &item.Type, &item.Channel,
		&item.ProviderExtensionID, &item.ProviderArtifactDigest, &item.PayloadVersion, &item.Payload,
		&item.TargetMeta, &item.IdempotencyKey, &item.Status, &item.AttemptCount, &item.Reason,
		&item.ErrorSummary, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelDelivery{}, ErrChannelDeliveryNotFound
	}
	return item, err
}

func (s *PostgresStore) UpdateChannelDelivery(ctx context.Context, update ChannelDeliveryUpdate) error {
	completed := update.Status == DeliverySent || update.Status == DeliveryFailed || update.Status == DeliverySkipped
	_, err := s.runner.Exec(ctx, `UPDATE notification_channel_deliveries SET status=$2,
		provider_extension_id=$3,provider_artifact_digest=$4,attempt_count=$5,reason=$6,
		error_summary=$7,updated_at=now(),completed_at=CASE WHEN $8 THEN now() ELSE NULL END
		WHERE id=$1`, update.ID, update.Status, update.ProviderExtensionID, update.ProviderArtifactDigest,
		update.AttemptCount, update.Reason, update.ErrorSummary, completed)
	return err
}

func (s *PostgresStore) ListChannelDeliveries(ctx context.Context, limit int) ([]ChannelDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.runner.Query(ctx, `SELECT id,notification_id,recipient_user_id,type,channel,
		provider_extension_id,provider_artifact_digest,payload_version,'{}'::jsonb,'{}'::jsonb,''::text,
		status,attempt_count,reason,error_summary,created_at,updated_at,completed_at
		FROM notification_channel_deliveries ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ChannelDelivery, 0, limit)
	for rows.Next() {
		var item ChannelDelivery
		if err := rows.Scan(&item.ID, &item.NotificationID, &item.RecipientUserID, &item.Type, &item.Channel,
			&item.ProviderExtensionID, &item.ProviderArtifactDigest, &item.PayloadVersion, &item.Payload,
			&item.TargetMeta, &item.IdempotencyKey, &item.Status, &item.AttemptCount, &item.Reason,
			&item.ErrorSummary, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type WebPushSubscription struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"-"`
	EndpointOrigin    string     `json:"endpointOrigin"`
	Endpoint          string     `json:"-"`
	P256DHKey         []byte     `json:"-"`
	AuthKey           []byte     `json:"-"`
	ContentEncoding   string     `json:"contentEncoding"`
	Status            string     `json:"status"`
	LastFailureReason string     `json:"lastFailureReason,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	RevokedAt         *time.Time `json:"revokedAt,omitempty"`
}

type WebPushDeliveryAttempt struct {
	DeliveryID, SubscriptionID int64
	Status                     string
	AttemptCount               int
	Reason                     string
}

type CreateWebPushSubscriptionInput struct {
	UserID             int64
	Endpoint           string
	P256DHKey, AuthKey []byte
	ContentEncoding    string
}

func validateWebPushSubscription(input CreateWebPushSubscriptionInput) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(input.Endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || len(input.Endpoint) > 4096 ||
		len(input.P256DHKey) < 32 || len(input.P256DHKey) > 256 || len(input.AuthKey) < 8 || len(input.AuthKey) > 128 ||
		(input.ContentEncoding != "" && input.ContentEncoding != "aes128gcm") {
		return nil, ErrSubscriptionInvalid
	}
	return parsed, nil
}

func (s *PostgresStore) CreateWebPushSubscription(ctx context.Context, input CreateWebPushSubscriptionInput) (WebPushSubscription, error) {
	parsed, err := validateWebPushSubscription(input)
	if err != nil {
		return WebPushSubscription{}, err
	}
	digest := sha256.Sum256([]byte(input.Endpoint))
	encoding := input.ContentEncoding
	if encoding == "" {
		encoding = "aes128gcm"
	}
	var item WebPushSubscription
	err = s.runner.QueryRow(ctx, `
		INSERT INTO web_push_subscriptions (user_id,endpoint,endpoint_hash,p256dh_key,auth_key,content_encoding)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (endpoint_hash) DO UPDATE SET p256dh_key=EXCLUDED.p256dh_key,
		  auth_key=EXCLUDED.auth_key,content_encoding=EXCLUDED.content_encoding,status='active',
		  last_failure_reason='',updated_at=now(),revoked_at=NULL
		WHERE web_push_subscriptions.user_id=EXCLUDED.user_id
		RETURNING id,user_id,endpoint,p256dh_key,auth_key,content_encoding,status,last_failure_reason,
		  created_at,updated_at,revoked_at`, input.UserID, input.Endpoint, hex.EncodeToString(digest[:]),
		input.P256DHKey, input.AuthKey, encoding).Scan(&item.ID, &item.UserID, &item.Endpoint,
		&item.P256DHKey, &item.AuthKey, &item.ContentEncoding, &item.Status, &item.LastFailureReason,
		&item.CreatedAt, &item.UpdatedAt, &item.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebPushSubscription{}, ErrSubscriptionInvalid
	}
	item.EndpointOrigin = parsed.Scheme + "://" + parsed.Host
	return item, err
}

func (s *PostgresStore) ListWebPushSubscriptions(ctx context.Context, userID int64, activeOnly bool) ([]WebPushSubscription, error) {
	rows, err := s.runner.Query(ctx, `SELECT id,user_id,endpoint,p256dh_key,auth_key,content_encoding,
		status,last_failure_reason,created_at,updated_at,revoked_at FROM web_push_subscriptions
		WHERE user_id=$1 AND ($2=FALSE OR status='active') ORDER BY id DESC`, userID, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []WebPushSubscription{}
	for rows.Next() {
		var item WebPushSubscription
		if err := rows.Scan(&item.ID, &item.UserID, &item.Endpoint, &item.P256DHKey, &item.AuthKey,
			&item.ContentEncoding, &item.Status, &item.LastFailureReason, &item.CreatedAt, &item.UpdatedAt,
			&item.RevokedAt); err != nil {
			return nil, err
		}
		if parsed, parseErr := url.Parse(item.Endpoint); parseErr == nil {
			item.EndpointOrigin = parsed.Scheme + "://" + parsed.Host
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) RevokeWebPushSubscription(ctx context.Context, userID, id int64) error {
	tag, err := s.runner.Exec(ctx, `UPDATE web_push_subscriptions SET status='revoked',
		updated_at=now(),revoked_at=now() WHERE id=$1 AND user_id=$2 AND status<>'revoked'`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (s *PostgresStore) MarkWebPushSubscriptionFailed(ctx context.Context, id int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if len(reason) > 160 {
		reason = reason[:160]
	}
	_, err := s.runner.Exec(ctx, `UPDATE web_push_subscriptions SET status='failed',
		last_failure_reason=$2,updated_at=now() WHERE id=$1`, id, reason)
	return err
}

func (s *PostgresStore) GetWebPushDeliveryAttempt(ctx context.Context, deliveryID, subscriptionID int64) (WebPushDeliveryAttempt, error) {
	var item WebPushDeliveryAttempt
	err := s.runner.QueryRow(ctx, `INSERT INTO web_push_delivery_attempts (delivery_id,subscription_id)
		VALUES ($1,$2) ON CONFLICT (delivery_id,subscription_id) DO UPDATE SET delivery_id=EXCLUDED.delivery_id
		RETURNING delivery_id,subscription_id,status,attempt_count,reason`, deliveryID, subscriptionID).Scan(
		&item.DeliveryID, &item.SubscriptionID, &item.Status, &item.AttemptCount, &item.Reason)
	return item, err
}

func (s *PostgresStore) UpdateWebPushDeliveryAttempt(ctx context.Context, item WebPushDeliveryAttempt) error {
	_, err := s.runner.Exec(ctx, `UPDATE web_push_delivery_attempts SET status=$3,attempt_count=$4,
		reason=$5,updated_at=now() WHERE delivery_id=$1 AND subscription_id=$2`, item.DeliveryID,
		item.SubscriptionID, item.Status, item.AttemptCount, item.Reason)
	return err
}

func webPushTargetMeta(input CreateInput) json.RawMessage {
	body, _ := json.Marshal(map[string]any{"targetType": input.TargetType, "targetId": input.TargetID})
	return body
}

func channelProjectionKey(dedupe, channel string) string {
	return fmt.Sprintf("%s:%s", dedupe, channel)
}
