package extensions

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	pluginRuntimePublicationChannel    = "sforum_plugin_runtime_publication"
	DefaultPluginRuntimeReconnectDelay = time.Second
)

type PostgresPluginRuntimeNotifications struct {
	store          *PostgresStore
	reconnectDelay time.Duration
	onError        func(error)
	ready          func()
	connected      func(uint32)
}

func NewPostgresPluginRuntimeNotifications(
	store *PostgresStore,
	reconnectDelay time.Duration,
	onError func(error),
) *PostgresPluginRuntimeNotifications {
	if reconnectDelay <= 0 {
		reconnectDelay = DefaultPluginRuntimeReconnectDelay
	}
	return &PostgresPluginRuntimeNotifications{
		store: store, reconnectDelay: reconnectDelay, onError: onError,
	}
}

// WatchPluginRuntimePublications emits wake hints only. Consumers must poll the
// PostgreSQL repository after every wake and on their own interval because a
// LISTEN connection may miss notifications while disconnected.
func (n *PostgresPluginRuntimeNotifications) WatchPluginRuntimePublications(ctx context.Context, wake func()) {
	if n == nil || n.store == nil || n.store.pool == nil || ctx == nil || wake == nil {
		return
	}
	for ctx.Err() == nil {
		config := n.store.pool.Config().ConnConfig.Copy()
		connection, err := pgx.ConnectConfig(ctx, config)
		if err == nil {
			_, err = connection.Exec(ctx, `LISTEN `+pluginRuntimePublicationChannel)
		}
		if err != nil {
			if connection != nil {
				_ = connection.Close(context.Background())
			}
			n.report(err)
			if !waitPluginRuntimeReconnect(ctx, n.reconnectDelay) {
				return
			}
			continue
		}
		if n.ready != nil {
			n.ready()
		}
		if n.connected != nil {
			n.connected(connection.PgConn().PID())
		}
		// LISTEN 只负责唤醒。每次首次连接或重连成功都必须先从 PostgreSQL
		// 重读 durable latest，补回断线窗口内遗漏的 revision。
		wake()
		for ctx.Err() == nil {
			if _, err = connection.WaitForNotification(ctx); err != nil {
				break
			}
			wake()
		}
		_ = connection.Close(context.Background())
		if ctx.Err() != nil {
			return
		}
		n.report(err)
		if !waitPluginRuntimeReconnect(ctx, n.reconnectDelay) {
			return
		}
	}
}

func (n *PostgresPluginRuntimeNotifications) report(err error) {
	if err != nil && n.onError != nil {
		n.onError(err)
	}
}

func waitPluginRuntimeReconnect(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

var _ PluginRuntimePublicationNotificationSource = (*PostgresPluginRuntimeNotifications)(nil)
