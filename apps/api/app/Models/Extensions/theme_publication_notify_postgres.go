package extensions

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const themeRuntimePublicationChannel = "sforum_theme_runtime_publication"

type PostgresThemeRuntimeNotifications struct {
	store          *PostgresStore
	reconnectDelay time.Duration
	onError        func(error)
	ready          func()
	connected      func(uint32)
}

func NewPostgresThemeRuntimeNotifications(store *PostgresStore, reconnectDelay time.Duration, onError func(error)) *PostgresThemeRuntimeNotifications {
	if reconnectDelay <= 0 {
		reconnectDelay = DefaultThemeRuntimeReconnectDelay
	}
	return &PostgresThemeRuntimeNotifications{store: store, reconnectDelay: reconnectDelay, onError: onError}
}

func (n *PostgresThemeRuntimeNotifications) WatchThemeRuntimePublications(ctx context.Context, wake func()) {
	if n == nil || n.store == nil || n.store.pool == nil || ctx == nil || wake == nil {
		return
	}
	for ctx.Err() == nil {
		config := n.store.pool.Config().ConnConfig.Copy()
		connection, err := pgx.ConnectConfig(ctx, config)
		if err == nil {
			_, err = connection.Exec(ctx, `LISTEN `+themeRuntimePublicationChannel)
		}
		if err != nil {
			if connection != nil {
				_ = connection.Close(context.Background())
			}
			n.report(err)
			if !waitThemeRuntimeReconnect(ctx, n.reconnectDelay) {
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
		if !waitThemeRuntimeReconnect(ctx, n.reconnectDelay) {
			return
		}
	}
}

func (n *PostgresThemeRuntimeNotifications) report(err error) {
	if err != nil && n.onError != nil {
		n.onError(err)
	}
}

func waitThemeRuntimeReconnect(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

var _ ThemeRuntimePublicationNotificationSource = (*PostgresThemeRuntimeNotifications)(nil)
