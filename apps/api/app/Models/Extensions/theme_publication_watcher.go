package extensions

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	DefaultThemeRuntimeNodeLease         = 45 * time.Second
	DefaultThemeRuntimeHeartbeatInterval = 10 * time.Second
	DefaultThemeRuntimePollInterval      = 5 * time.Second
	DefaultThemeRuntimeReconnectDelay    = time.Second
)

type ThemeRuntimePublicationApplier interface {
	ApplyThemeRuntimePublication(context.Context, ThemeRuntimePublication) error
}

type ThemeRuntimePublicationNotificationSource interface {
	WatchThemeRuntimePublications(context.Context, func())
}

type themeRuntimeWatcherRepository interface {
	ThemeRuntimeNodeRepository
	ThemeRuntimePublicationRepository
}

type ThemeRuntimeWatcherConfig struct {
	Identity          ThemeRuntimeNodeIdentity
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	OnError           func(error)
}

type ThemeRuntimeWatcher struct {
	repository    themeRuntimeWatcherRepository
	applier       ThemeRuntimePublicationApplier
	notifications ThemeRuntimePublicationNotificationSource
	config        ThemeRuntimeWatcherConfig
}

func NewThemeRuntimeWatcher(
	repository themeRuntimeWatcherRepository,
	applier ThemeRuntimePublicationApplier,
	notifications ThemeRuntimePublicationNotificationSource,
	config ThemeRuntimeWatcherConfig,
) (*ThemeRuntimeWatcher, error) {
	if config.LeaseDuration == 0 {
		config.LeaseDuration = DefaultThemeRuntimeNodeLease
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = DefaultThemeRuntimeHeartbeatInterval
	}
	if config.PollInterval == 0 {
		config.PollInterval = DefaultThemeRuntimePollInterval
	}
	if repository == nil || applier == nil || !validThemeRuntimeNodeIdentity(config.Identity) ||
		!validThemeRuntimeNodeLease(config.LeaseDuration) || config.HeartbeatInterval <= 0 ||
		config.HeartbeatInterval >= config.LeaseDuration || config.PollInterval <= 0 {
		return nil, ErrThemeRuntimeNodeInvalid
	}
	return &ThemeRuntimeWatcher{
		repository: repository, applier: applier, notifications: notifications, config: config,
	}, nil
}

func (w *ThemeRuntimeWatcher) Run(ctx context.Context) error {
	if w == nil || ctx == nil {
		return ErrThemeRuntimeNodeInvalid
	}
	runCtx, cancel := context.WithCancel(ctx)
	if err := w.Initialize(runCtx); err != nil {
		cancel()
		return err
	}
	wake := make(chan struct{}, 1)
	var notificationWait sync.WaitGroup
	if w.notifications != nil {
		notificationWait.Add(1)
		go func() {
			defer notificationWait.Done()
			w.notifications.WatchThemeRuntimePublications(runCtx, func() {
				select {
				case wake <- struct{}{}:
				default:
				}
			})
		}()
	}
	defer func() {
		cancel()
		notificationWait.Wait()
	}()
	heartbeat := time.NewTicker(w.config.HeartbeatInterval)
	poll := time.NewTicker(w.config.PollInterval)
	defer heartbeat.Stop()
	defer poll.Stop()
	for {
		select {
		case <-runCtx.Done():
			return nil
		case <-heartbeat.C:
			if _, err := w.repository.HeartbeatThemeRuntimeNode(runCtx, w.config.Identity, w.config.LeaseDuration); err != nil {
				return err
			}
		case <-poll.C:
			if err := w.ReconcileOnce(runCtx); err != nil {
				w.report(err)
			}
		case <-wake:
			if err := w.ReconcileOnce(runCtx); err != nil {
				w.report(err)
			}
		}
	}
}

func (w *ThemeRuntimeWatcher) Initialize(ctx context.Context) error {
	if w == nil || ctx == nil {
		return ErrThemeRuntimeNodeInvalid
	}
	if _, err := w.repository.RegisterThemeRuntimeNode(ctx, w.config.Identity, w.config.LeaseDuration); err != nil {
		return err
	}
	return w.ReconcileOnce(ctx)
}

func (w *ThemeRuntimeWatcher) ReconcileOnce(ctx context.Context) error {
	if w == nil || ctx == nil {
		return ErrThemeRuntimeNodeInvalid
	}
	publication, err := w.repository.LatestThemeRuntimePublication(ctx)
	if errors.Is(err, ErrThemePublicationNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	node, err := w.repository.GetThemeRuntimeNode(ctx, w.config.Identity)
	if err != nil {
		return err
	}
	if node.LastAppliedRevision >= publication.Revision {
		return nil
	}
	ack, err := w.repository.BeginThemeRuntimePublicationApply(ctx, w.config.Identity, publication.Revision)
	if err != nil {
		return err
	}
	if err := w.applier.ApplyThemeRuntimePublication(ctx, publication); err != nil {
		failed, failErr := w.repository.FailThemeRuntimePublicationApply(
			ctx, w.config.Identity, publication.Revision, ack.Revision, themeRuntimeFailureReason(err),
		)
		if failErr != nil {
			return fmt.Errorf("apply theme runtime revision %d: %w; record failure: %v", publication.Revision, err, failErr)
		}
		return fmt.Errorf("apply theme runtime revision %d attempt %d: %w", publication.Revision, failed.AttemptCount, err)
	}
	if _, err := w.repository.CompleteThemeRuntimePublicationApply(
		ctx, w.config.Identity, publication, ack.Revision,
	); err != nil {
		return err
	}
	return nil
}

func themeRuntimeFailureReason(err error) string {
	if err == nil {
		return "theme runtime apply failed"
	}
	reason := err.Error()
	for len([]byte(reason)) > 2048 {
		_, size := utf8.DecodeLastRuneInString(reason)
		if size <= 0 {
			return "theme runtime apply failed"
		}
		reason = reason[:len(reason)-size]
	}
	if reason == "" {
		return "theme runtime apply failed"
	}
	return reason
}

func (w *ThemeRuntimeWatcher) report(err error) {
	if err != nil && w.config.OnError != nil {
		w.config.OnError(err)
	}
}
