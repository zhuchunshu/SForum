package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type TxEnqueuer interface {
	EnqueueTx(context.Context, pgx.Tx, river.JobArgs, supportjobs.EnqueueOptions) (*rivertype.JobInsertResult, error)
}

type Outbox struct {
	pool interface {
		Begin(context.Context) (pgx.Tx, error)
	}
	store *PostgresStore
	jobs  TxEnqueuer
}

func NewOutbox(pool interface {
	Begin(context.Context) (pgx.Tx, error)
}, store *PostgresStore, jobs TxEnqueuer) *Outbox {
	return &Outbox{pool: pool, store: store, jobs: jobs}
}

type QueueMailInput struct {
	Recipient, TemplateKey, IdempotencyKey, CorrelationID string
	TemplateData                                          json.RawMessage
}
type QueuePasswordResetInput struct {
	UserID        int64
	TokenHash     string
	ExpiresAt     time.Time
	RequestIPHash string
	Mail          QueueMailInput
}

type deliverMailArgs struct {
	DeliveryID int64 `json:"delivery_id" river:"unique"`
}

func (deliverMailArgs) Kind() string { return "mail.deliver" }
func (deliverMailArgs) enqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{Queue: supportjobs.QueueMail, MaxAttempts: 5, Unique: river.UniqueOpts{ByArgs: true}}
}

func (o *Outbox) QueueMail(ctx context.Context, input QueueMailInput) (MailDelivery, error) {
	return o.inTx(ctx, input, nil)
}

func (o *Outbox) QueuePasswordReset(ctx context.Context, input QueuePasswordResetInput) (MailDelivery, error) {
	return o.inTx(ctx, input.Mail, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, request_ip_hash) VALUES ($1,$2,$3,$4)`, input.UserID, input.TokenHash, input.ExpiresAt, input.RequestIPHash)
		return err
	})
}

func (o *Outbox) inTx(ctx context.Context, input QueueMailInput, before func(pgx.Tx) error) (MailDelivery, error) {
	if o == nil || o.pool == nil || o.store == nil || o.jobs == nil {
		return MailDelivery{}, fmt.Errorf("notifications: outbox is not configured")
	}
	tx, err := o.pool.Begin(ctx)
	if err != nil {
		return MailDelivery{}, err
	}
	defer tx.Rollback(ctx)
	if before != nil {
		if err := before(tx); err != nil {
			return MailDelivery{}, err
		}
	}
	delivery, err := o.store.CreateDeliveryTx(ctx, tx, CreateDeliveryInput{Recipient: input.Recipient, TemplateKey: input.TemplateKey, TemplateData: input.TemplateData, IdempotencyKey: input.IdempotencyKey, CorrelationID: input.CorrelationID})
	if err != nil {
		return MailDelivery{}, err
	}
	args := deliverMailArgs{DeliveryID: delivery.ID}
	if _, err = o.jobs.EnqueueTx(ctx, tx, args, args.enqueueOptions()); err != nil {
		return MailDelivery{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return MailDelivery{}, err
	}
	return delivery, nil
}

func (o *Outbox) CreateBundleTx(ctx context.Context, tx pgx.Tx, input CreateBundleInput) (Bundle, error) {
	bundle, err := o.store.CreateBundleTx(ctx, tx, input)
	if err != nil {
		return Bundle{}, err
	}
	args := deliverMailArgs{DeliveryID: bundle.Delivery.ID}
	if _, err = o.jobs.EnqueueTx(ctx, tx, args, args.enqueueOptions()); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}
