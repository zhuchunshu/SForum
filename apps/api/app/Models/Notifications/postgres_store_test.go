package notifications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresStoreCreateBundleUsesDeterministicKeys(t *testing.T) {
	runner := newMemoryRunner()
	store := newPostgresStore(runner)
	input := CreateBundleInput{
		Notification: CreateInput{RecipientUserID: 7, Type: TypeMention, TargetType: "comment", TargetID: 42, DedupeKey: "comment:42:mention:7"},
		Delivery:     CreateDeliveryInput{Recipient: "member@example.com", TemplateKey: "forum.mention", IdempotencyKey: "comment:42:mention:7"},
	}

	first, err := store.CreateBundleTx(context.Background(), runner, input)
	if err != nil {
		t.Fatalf("create first bundle: %v", err)
	}
	second, err := store.CreateBundleTx(context.Background(), runner, input)
	if err != nil {
		t.Fatalf("create duplicate bundle: %v", err)
	}
	if first.Notification.ID != second.Notification.ID || first.Delivery.ID != second.Delivery.ID {
		t.Fatalf("duplicate keys created new records: first=%#v second=%#v", first, second)
	}
	if first.Delivery.Status != DeliveryQueued {
		t.Fatalf("unexpected delivery status: %s", first.Delivery.Status)
	}
}

type memoryRunner struct {
	nextID        int64
	notifications map[string]Notification
	deliveries    map[string]MailDelivery
}

func newMemoryRunner() *memoryRunner {
	return &memoryRunner{nextID: 1, notifications: map[string]Notification{}, deliveries: map[string]MailDelivery{}}
}

func (m *memoryRunner) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	if strings.Contains(query, "INSERT INTO notifications") {
		key := args[6].(string)
		item, ok := m.notifications[key]
		if !ok {
			item = Notification{ID: m.nextID, RecipientUserID: args[0].(int64), Type: args[1].(string), TargetType: args[3].(string), TargetID: args[4].(int64), Payload: args[5].(json.RawMessage), DedupeKey: key, CreatedAt: time.Now()}
			m.nextID++
			m.notifications[key] = item
		}
		return memoryRow{values: []any{item.ID, item.RecipientUserID, item.Type, item.ActorUserID, item.TargetType, item.TargetID, item.Payload, item.DedupeKey, item.ReadAt, item.CreatedAt}}
	}
	if strings.Contains(query, "INSERT INTO mail_deliveries") {
		key := args[3].(string)
		item, ok := m.deliveries[key]
		if !ok {
			now := time.Now()
			item = MailDelivery{ID: m.nextID, Recipient: args[0].(string), TemplateKey: args[1].(string), TemplateData: args[2].(json.RawMessage), IdempotencyKey: key, CorrelationID: args[4].(string), Status: DeliveryQueued, CreatedAt: now, UpdatedAt: now}
			m.nextID++
			m.deliveries[key] = item
		}
		return memoryRow{values: []any{item.ID, item.Recipient, item.TemplateKey, item.TemplateData, item.IdempotencyKey, item.CorrelationID, item.Status, item.ExtensionID, item.AttemptCount, item.Reason, item.ErrorSummary, item.CreatedAt, item.UpdatedAt, item.CompletedAt}}
	}
	if strings.Contains(query, "COUNT(*)") {
		userID := args[0].(int64)
		var count int64
		for _, item := range m.notifications {
			if item.RecipientUserID == userID && item.ReadAt == nil {
				count++
			}
		}
		return memoryRow{values: []any{count}}
	}
	return memoryRow{err: pgx.ErrNoRows}
}

func (m *memoryRunner) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "UPDATE notifications SET read_at") {
		id, userID := args[0].(int64), args[1].(int64)
		for key, item := range m.notifications {
			if item.ID == id && item.RecipientUserID == userID {
				now := time.Now()
				item.ReadAt = &now
				m.notifications[key] = item
				return pgconn.NewCommandTag("UPDATE 1"), nil
			}
		}
	}
	return pgconn.NewCommandTag("UPDATE 0"), nil
}

func (*memoryRunner) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

type memoryRow struct {
	values []any
	err    error
}

func (r memoryRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch target := dest[i].(type) {
		case *int64:
			*target = r.values[i].(int64)
		case *int:
			*target = r.values[i].(int)
		case *string:
			*target = r.values[i].(string)
		case **int64:
			*target = r.values[i].(*int64)
		case **time.Time:
			*target = r.values[i].(*time.Time)
		case *time.Time:
			*target = r.values[i].(time.Time)
		case *json.RawMessage:
			*target = r.values[i].(json.RawMessage)
		}
	}
	return nil
}

func TestPostgresStoreMarksOnlyRecipientsNotificationRead(t *testing.T) {
	runner := newMemoryRunner()
	store := newPostgresStore(runner)
	bundle, err := store.CreateBundleTx(context.Background(), runner, CreateBundleInput{
		Notification: CreateInput{RecipientUserID: 7, Type: TypeReply, TargetType: "comment", TargetID: 9, DedupeKey: "comment:9:reply:7"},
		Delivery:     CreateDeliveryInput{Recipient: "member@example.com", TemplateKey: "forum.reply", IdempotencyKey: "comment:9:reply:7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkRead(context.Background(), 8, bundle.Notification.ID); err != ErrNotificationNotFound {
		t.Fatalf("other recipient changed notification: %v", err)
	}
	if err := store.MarkRead(context.Background(), 7, bundle.Notification.ID); err != nil {
		t.Fatalf("owner mark read: %v", err)
	}
	count, err := store.UnreadCount(context.Background(), 7)
	if err != nil || count != 0 {
		t.Fatalf("unexpected unread count: count=%d err=%v", count, err)
	}
}
