package forum

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
)

type PostgresStore struct {
	pool          *pgxpool.Pool
	avatarBuilder *avatar.ViewBuilder
	notifications CommentNotificationWriter
	auditor       audit.TxWriter
}

type CommentNotificationInput struct {
	CommentID, TopicID, ActorUserID, ParentAuthorUserID int64
	MentionedUsernames                                  []string
}
type CommentNotificationWriter interface {
	NotifyCommentTx(context.Context, pgx.Tx, CommentNotificationInput) error
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return NewPostgresStoreWithAvatar(pool, nil)
}

func (s *PostgresStore) WithCommentNotifications(writer CommentNotificationWriter) *PostgresStore {
	s.notifications = writer
	return s
}

// WithAuditor wires the shared transaction-aware audit boundary. The no-op
// default keeps local tools/read-only stores usable; production injects it.
func (s *PostgresStore) WithAuditor(writer audit.TxWriter) *PostgresStore {
	if writer == nil {
		writer = audit.NoopWriter{}
	}
	s.auditor = writer
	return s
}

func NewPostgresStoreWithAvatar(pool *pgxpool.Pool, avatarOptions avatar.OptionResolver) *PostgresStore {
	return &PostgresStore{pool: pool, avatarBuilder: avatar.NewViewBuilder(avatarOptions)}
}
