package notifications

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type queryRunner interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresStore struct {
	runner queryRunner
	pool   *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{runner: pool, pool: pool}
}
func newPostgresStore(runner queryRunner) *PostgresStore { return &PostgresStore{runner: runner} }

func (s *PostgresStore) CreateBundleTx(ctx context.Context, tx queryRunner, input CreateBundleInput) (Bundle, error) {
	notification, err := s.CreateNotificationTx(ctx, tx, input.Notification)
	if err != nil {
		return Bundle{}, err
	}
	result := Bundle{Notification: notification}
	templateData := input.Delivery.TemplateData
	if len(templateData) == 0 {
		templateData = json.RawMessage(`{}`)
	}
	err = tx.QueryRow(ctx, `
INSERT INTO mail_deliveries (recipient, template_key, template_data, idempotency_key, correlation_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
RETURNING id, recipient, template_key, template_data, idempotency_key, correlation_id, status, extension_id,
attempt_count, reason, error_summary, created_at, updated_at, completed_at`, input.Delivery.Recipient,
		input.Delivery.TemplateKey, templateData, input.Delivery.IdempotencyKey, input.Delivery.CorrelationID,
	).Scan(&result.Delivery.ID, &result.Delivery.Recipient, &result.Delivery.TemplateKey, &result.Delivery.TemplateData,
		&result.Delivery.IdempotencyKey, &result.Delivery.CorrelationID, &result.Delivery.Status,
		&result.Delivery.ExtensionID, &result.Delivery.AttemptCount, &result.Delivery.Reason,
		&result.Delivery.ErrorSummary, &result.Delivery.CreatedAt, &result.Delivery.UpdatedAt, &result.Delivery.CompletedAt)
	return result, err
}

func (s *PostgresStore) CreateNotificationTx(ctx context.Context, tx queryRunner, input CreateInput) (Notification, error) {
	payload := input.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	var result Notification
	err := tx.QueryRow(ctx, `
INSERT INTO notifications (recipient_user_id, type, actor_user_id, target_type, target_id, payload, dedupe_key)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (dedupe_key) DO UPDATE SET dedupe_key = EXCLUDED.dedupe_key
RETURNING id, recipient_user_id, type, actor_user_id, target_type, target_id, payload, dedupe_key, read_at, created_at`,
		input.RecipientUserID, input.Type, input.ActorUserID, input.TargetType, input.TargetID, payload, input.DedupeKey,
	).Scan(&result.ID, &result.RecipientUserID, &result.Type, &result.ActorUserID, &result.TargetType, &result.TargetID,
		&result.Payload, &result.DedupeKey, &result.ReadAt, &result.CreatedAt)
	return result, err
}

func (s *PostgresStore) CreateDeliveryTx(ctx context.Context, tx queryRunner, input CreateDeliveryInput) (MailDelivery, error) {
	templateData := input.TemplateData
	if len(templateData) == 0 {
		templateData = json.RawMessage(`{}`)
	}
	var item MailDelivery
	err := tx.QueryRow(ctx, `INSERT INTO mail_deliveries (recipient, template_key, template_data, idempotency_key, correlation_id)
VALUES ($1,$2,$3,$4,$5) ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
RETURNING id, recipient, template_key, template_data, idempotency_key, correlation_id, status, extension_id,
attempt_count, reason, error_summary, created_at, updated_at, completed_at`, input.Recipient, input.TemplateKey, templateData,
		input.IdempotencyKey, input.CorrelationID).Scan(&item.ID, &item.Recipient, &item.TemplateKey, &item.TemplateData,
		&item.IdempotencyKey, &item.CorrelationID, &item.Status, &item.ExtensionID, &item.AttemptCount, &item.Reason,
		&item.ErrorSummary, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt)
	return item, err
}

func (s *PostgresStore) List(ctx context.Context, input ListInput) (Page, error) {
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.runner.Query(ctx, `SELECT id, recipient_user_id, type, actor_user_id, target_type, target_id, payload,
dedupe_key, read_at, created_at FROM notifications WHERE recipient_user_id=$1 AND ($2::bigint=0 OR id<$2)
ORDER BY id DESC LIMIT $3`, input.RecipientUserID, input.BeforeID, limit+1)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	page := Page{Items: make([]Notification, 0, limit)}
	for rows.Next() {
		var item Notification
		if err := rows.Scan(&item.ID, &item.RecipientUserID, &item.Type, &item.ActorUserID, &item.TargetType, &item.TargetID, &item.Payload, &item.DedupeKey, &item.ReadAt, &item.CreatedAt); err != nil {
			return Page{}, err
		}
		if len(page.Items) == limit {
			page.HasMore = true
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, rows.Err()
}

func (s *PostgresStore) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := s.runner.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE recipient_user_id=$1 AND read_at IS NULL`, userID).Scan(&count)
	return count, err
}

func (s *PostgresStore) MarkRead(ctx context.Context, userID, id int64) error {
	tag, err := s.runner.Exec(ctx, `UPDATE notifications SET read_at=COALESCE(read_at, NOW()) WHERE id=$1 AND recipient_user_id=$2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return err
}

func (s *PostgresStore) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	tag, err := s.runner.Exec(ctx, `UPDATE notifications SET read_at=NOW() WHERE recipient_user_id=$1 AND read_at IS NULL`, userID)
	return tag.RowsAffected(), err
}

func (s *PostgresStore) GetDelivery(ctx context.Context, id int64) (MailDelivery, error) {
	var item MailDelivery
	err := s.runner.QueryRow(ctx, `SELECT id, recipient, template_key, template_data, idempotency_key, correlation_id,
status, extension_id, attempt_count, reason, error_summary, created_at, updated_at, completed_at FROM mail_deliveries WHERE id=$1`, id).Scan(
		&item.ID, &item.Recipient, &item.TemplateKey, &item.TemplateData, &item.IdempotencyKey, &item.CorrelationID,
		&item.Status, &item.ExtensionID, &item.AttemptCount, &item.Reason, &item.ErrorSummary, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt)
	return item, err
}

func (s *PostgresStore) UpdateDelivery(ctx context.Context, input DeliveryUpdate) error {
	completed := any(nil)
	if input.Status == DeliverySent || input.Status == DeliveryFailed || input.Status == DeliverySkipped {
		completed = time.Now().UTC()
	}
	_, err := s.runner.Exec(ctx, `UPDATE mail_deliveries SET status=$2, extension_id=$3, attempt_count=$4, reason=$5,
error_summary=$6, updated_at=NOW(), completed_at=COALESCE($7, completed_at) WHERE id=$1`, input.ID, input.Status,
		input.ExtensionID, input.AttemptCount, input.Reason, input.ErrorSummary, completed)
	return err
}

func (s *PostgresStore) ListDeliveries(ctx context.Context, limit int) ([]MailDelivery, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.runner.Query(ctx, `SELECT id, recipient, template_key, '{}'::jsonb, idempotency_key, correlation_id,
status, extension_id, attempt_count, reason, error_summary, created_at, updated_at, completed_at FROM mail_deliveries ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MailDelivery{}
	for rows.Next() {
		var item MailDelivery
		if err := rows.Scan(&item.ID, &item.Recipient, &item.TemplateKey, &item.TemplateData, &item.IdempotencyKey, &item.CorrelationID, &item.Status, &item.ExtensionID, &item.AttemptCount, &item.Reason, &item.ErrorSummary, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt); err != nil {
			return nil, err
		}
		item.IdempotencyKey = ""
		items = append(items, item)
	}
	return items, rows.Err()
}
