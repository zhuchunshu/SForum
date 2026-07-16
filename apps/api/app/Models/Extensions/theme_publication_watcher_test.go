package extensions

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type themeRuntimeWatcherTestRepository struct{}

func (themeRuntimeWatcherTestRepository) EnsureInitialThemeRuntimePublication(
	context.Context,
) (ThemeRuntimePublication, error) {
	return themeRuntimeWatcherTestPublication(), nil
}

func (themeRuntimeWatcherTestRepository) RegisterThemeRuntimeNode(
	_ context.Context,
	identity ThemeRuntimeNodeIdentity,
	_ time.Duration,
) (ThemeRuntimeNode, error) {
	return ThemeRuntimeNode{ThemeRuntimeNodeIdentity: identity, LastAppliedRevision: 1}, nil
}

func (themeRuntimeWatcherTestRepository) HeartbeatThemeRuntimeNode(
	_ context.Context,
	identity ThemeRuntimeNodeIdentity,
	_ time.Duration,
) (ThemeRuntimeNode, error) {
	return ThemeRuntimeNode{
		ThemeRuntimeNodeIdentity: identity,
		LastAppliedRevision:      1,
	}, nil
}

func (themeRuntimeWatcherTestRepository) GetThemeRuntimeNode(
	_ context.Context,
	identity ThemeRuntimeNodeIdentity,
) (ThemeRuntimeNode, error) {
	return ThemeRuntimeNode{
		ThemeRuntimeNodeIdentity: identity,
		LastAppliedRevision:      1,
	}, nil
}

func (themeRuntimeWatcherTestRepository) BeginThemeRuntimePublicationApply(
	context.Context,
	ThemeRuntimeNodeIdentity,
	int64,
) (ThemeRuntimePublicationAck, error) {
	return ThemeRuntimePublicationAck{}, nil
}

func (themeRuntimeWatcherTestRepository) CompleteThemeRuntimePublicationApply(
	context.Context,
	ThemeRuntimeNodeIdentity,
	ThemeRuntimePublication,
	int64,
) (ThemeRuntimePublicationAck, error) {
	return ThemeRuntimePublicationAck{}, nil
}

func (themeRuntimeWatcherTestRepository) FailThemeRuntimePublicationApply(
	context.Context,
	ThemeRuntimeNodeIdentity,
	int64,
	int64,
	string,
) (ThemeRuntimePublicationAck, error) {
	return ThemeRuntimePublicationAck{}, nil
}

func (themeRuntimeWatcherTestRepository) LatestThemeRuntimePublication(context.Context) (ThemeRuntimePublication, error) {
	return themeRuntimeWatcherTestPublication(), nil
}

func themeRuntimeWatcherTestPublication() ThemeRuntimePublication {
	return ThemeRuntimePublication{
		Revision: 1, DesiredState: ThemeRuntimePublicationActive,
		ThemeID: DefaultThemeID, ThemeVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
		Reason: ThemeRuntimePublicationStartupRepair,
	}
}

func (themeRuntimeWatcherTestRepository) ThemeRuntimePublicationByRevision(
	context.Context,
	int64,
) (ThemeRuntimePublication, error) {
	return ThemeRuntimePublication{}, ErrThemePublicationNotFound
}

type themeRuntimeWatcherTestApplier struct{}

func (themeRuntimeWatcherTestApplier) ApplyThemeRuntimePublication(
	context.Context,
	ThemeRuntimePublication,
) error {
	return nil
}

type themeRuntimeWatcherTestNotifications func(context.Context, func())

func (notifications themeRuntimeWatcherTestNotifications) WatchThemeRuntimePublications(
	ctx context.Context,
	wake func(),
) {
	notifications(ctx, wake)
}

func TestThemeRuntimeFailureReasonPreservesUTF8WithinDatabaseBound(t *testing.T) {
	reason := themeRuntimeFailureReason(errors.New(strings.Repeat("主题失败", 1000)))
	if reason == "" || len([]byte(reason)) > 2048 || !strings.HasPrefix(reason, "主题失败") {
		t.Fatalf("bounded reason bytes=%d value=%q", len([]byte(reason)), reason)
	}
}

func TestThemeRuntimeWatcherBoundsStubbornNotificationShutdown(t *testing.T) {
	notificationStarted := make(chan struct{})
	releaseNotification := make(chan struct{})
	reported := make(chan error, 1)
	ready := make(chan struct{})
	var readyOnce sync.Once
	watcher, err := NewThemeRuntimeWatcher(
		themeRuntimeWatcherTestRepository{},
		themeRuntimeWatcherTestApplier{},
		themeRuntimeWatcherTestNotifications(func(context.Context, func()) {
			close(notificationStarted)
			<-releaseNotification
		}),
		ThemeRuntimeWatcherConfig{
			Identity:          ThemeRuntimeNodeIdentity{NodeID: "api-test", BootID: "boot-test"},
			LeaseDuration:     time.Second,
			HeartbeatInterval: 100 * time.Millisecond,
			PollInterval:      time.Hour,
			StopTimeout:       10 * time.Millisecond,
			OnReady:           func() { readyOnce.Do(func() { close(ready) }) },
			OnError:           func(err error) { reported <- err },
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- watcher.Run(ctx) }()
	waitThemeRuntimeWatcherTestSignal(t, ready)
	waitThemeRuntimeWatcherTestSignal(t, notificationStarted)
	cancel()
	select {
	case runErr := <-result:
		if runErr != nil {
			t.Fatalf("bounded watcher stop error=%v", runErr)
		}
	case <-time.After(time.Second):
		t.Fatal("stubborn notification source blocked watcher shutdown")
	}
	select {
	case report := <-reported:
		if !errors.Is(report, context.DeadlineExceeded) {
			t.Fatalf("notification stop report=%v", report)
		}
	case <-time.After(time.Second):
		t.Fatal("notification stop timeout was not reported")
	}
	close(releaseNotification)
}

func waitThemeRuntimeWatcherTestSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for theme runtime watcher signal")
	}
}
