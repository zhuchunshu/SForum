package extensions

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPostgresPluginRuntimeNotificationWakesDurablePoll(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "notify")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	wakes := make(chan struct{}, 4)
	errorsSeen := make(chan error, 4)
	done := make(chan struct{})
	var readyOnce sync.Once
	notifications := NewPostgresPluginRuntimeNotifications(fixture.store, 10*time.Millisecond, func(err error) {
		errorsSeen <- err
	})
	notifications.ready = func() { readyOnce.Do(func() { close(ready) }) }
	go func() {
		defer close(done)
		notifications.WatchPluginRuntimePublications(ctx, func() {
			select {
			case wakes <- struct{}{}:
			default:
			}
		})
	}()
	waitPluginRuntimeListenerReady(t, ready, errorsSeen)
	// 首次 LISTEN 成功会要求 durable startup reload；之后的 wake 才能证明
	// commit-time NOTIFY 被观察到。
	waitPluginRuntimeWake(t, wakes, errorsSeen)

	publication, err := fixture.store.PublishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	waitPluginRuntimeWake(t, wakes, errorsSeen)
	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || latest.Revision != publication.Revision {
		t.Fatalf("durable poll latest=%#v err=%v", latest, err)
	}
	stopPluginRuntimeListener(t, cancel, done)
}

func TestPostgresPluginRuntimeNotificationReconnectReloadsMissedPublication(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "notify-reconnect")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wakes := make(chan struct{}, 8)
	errorsSeen := make(chan error, 8)
	connected := make(chan uint32, 4)
	done := make(chan struct{})
	notifications := NewPostgresPluginRuntimeNotifications(fixture.store, 250*time.Millisecond, func(err error) {
		errorsSeen <- err
	})
	notifications.connected = func(pid uint32) { connected <- pid }
	go func() {
		defer close(done)
		notifications.WatchPluginRuntimePublications(ctx, func() {
			select {
			case wakes <- struct{}{}:
			default:
			}
		})
	}()

	firstPID := waitPluginRuntimeConnection(t, connected, errorsSeen)
	waitPluginRuntimeWake(t, wakes, errorsSeen)
	var terminated bool
	if err := fixture.admin.QueryRow(fixture.ctx, `SELECT pg_terminate_backend($1)`, firstPID).Scan(&terminated); err != nil {
		t.Fatal(err)
	}
	if !terminated {
		t.Fatalf("listener backend %d was not terminated", firstPID)
	}
	waitPluginRuntimeDisconnect(t, errorsSeen)

	// reconnectDelay keeps the listener offline while this revision commits.
	publication, err := fixture.store.PublishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondPID := waitPluginRuntimeConnection(t, connected, errorsSeen)
	if secondPID == firstPID {
		t.Fatalf("listener did not establish a new backend: pid=%d", secondPID)
	}
	waitPluginRuntimeWake(t, wakes, errorsSeen)
	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || latest.Revision != publication.Revision {
		t.Fatalf("reconnect durable poll latest=%#v err=%v", latest, err)
	}
	stopPluginRuntimeListener(t, cancel, done)
}

func waitPluginRuntimeListenerReady(t *testing.T, ready <-chan struct{}, errorsSeen <-chan error) {
	t.Helper()
	select {
	case <-ready:
	case err := <-errorsSeen:
		t.Fatalf("listen setup error=%v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("plugin runtime listener was not ready")
	}
}

func waitPluginRuntimeConnection(t *testing.T, connected <-chan uint32, errorsSeen <-chan error) uint32 {
	t.Helper()
	select {
	case pid := <-connected:
		return pid
	case err := <-errorsSeen:
		t.Fatalf("listen connection error=%v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("plugin runtime listener did not connect")
	}
	return 0
}

func waitPluginRuntimeDisconnect(t *testing.T, errorsSeen <-chan error) {
	t.Helper()
	select {
	case err := <-errorsSeen:
		if err == nil {
			t.Fatal("listener reported a nil disconnect error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("plugin runtime listener did not observe the terminated backend")
	}
}

func waitPluginRuntimeWake(t *testing.T, wakes <-chan struct{}, errorsSeen <-chan error) {
	t.Helper()
	select {
	case <-wakes:
	case err := <-errorsSeen:
		t.Fatalf("listen error=%v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("plugin runtime listener did not wake durable reload")
	}
}

func stopPluginRuntimeListener(t *testing.T, cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("plugin runtime listener did not stop")
	}
}
