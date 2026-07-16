package extensions

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

const (
	DefaultThemeRuntimeNodeLease         = 45 * time.Second
	DefaultThemeRuntimeHeartbeatInterval = 10 * time.Second
	DefaultThemeRuntimePollInterval      = 5 * time.Second
	DefaultThemeRuntimeReconnectDelay    = time.Second
	DefaultThemeRuntimeStopTimeout       = 10 * time.Second
)

type ThemeRuntimePublicationApplier interface {
	// Implementations must stop promptly after ctx cancellation. Lease heartbeat
	// failure cancels an in-flight apply before this boot may publish more state.
	ApplyThemeRuntimePublication(context.Context, ThemeRuntimePublication) error
}

type ThemeRuntimePublicationNotificationSource interface {
	WatchThemeRuntimePublications(context.Context, func())
}

type themeRuntimeWatcherRepository interface {
	ThemeRuntimeNodeRepository
	ThemeRuntimePublicationRepository
	EnsureInitialThemeRuntimePublication(context.Context) (ThemeRuntimePublication, error)
}

type ThemeRuntimeWatcherConfig struct {
	Identity          ThemeRuntimeNodeIdentity
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	StopTimeout       time.Duration
	OnError           func(error)
	OnReady           func()
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
	if config.StopTimeout == 0 {
		config.StopTimeout = DefaultThemeRuntimeStopTimeout
	}
	if repository == nil || applier == nil || !validThemeRuntimeNodeIdentity(config.Identity) ||
		!validThemeRuntimeNodeLease(config.LeaseDuration) || config.HeartbeatInterval <= 0 ||
		config.HeartbeatInterval > config.LeaseDuration/3 || config.PollInterval <= 0 || config.StopTimeout <= 0 {
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
	if ctx.Err() != nil {
		return nil
	}
	if err := w.ensureInitialPublication(ctx); err != nil {
		return err
	}
	if err := w.registerNode(ctx); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	wake := make(chan struct{}, 1)
	heartbeatErrors := make(chan error, 1)
	workersDone := make(chan struct{}, 2)
	workerCount := 1
	go w.heartbeat(runCtx, cancel, heartbeatErrors, workersDone)
	if w.notifications != nil {
		workerCount++
		go func() {
			defer func() { workersDone <- struct{}{} }()
			w.notifications.WatchThemeRuntimePublications(runCtx, func() {
				select {
				case wake <- struct{}{}:
				default:
				}
			})
		}()
	}
	defer w.stopWorkers(cancel, workersDone, workerCount)
	poll := time.NewTicker(w.config.PollInterval)
	defer poll.Stop()
	if err := w.reconcileUntilCurrent(runCtx); err != nil {
		return themeRuntimeRunError(ctx, heartbeatErrors, err)
	}
	if heartbeatErr := receiveThemeRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
		return heartbeatErr
	}
	if ctx.Err() != nil {
		return nil
	}
	if w.config.OnReady != nil {
		w.config.OnReady()
	}
	for {
		select {
		case heartbeatErr := <-heartbeatErrors:
			return heartbeatErr
		case <-ctx.Done():
			if heartbeatErr := receiveThemeRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
				return heartbeatErr
			}
			return nil
		case <-poll.C:
			if err := w.reconcileUntilCurrent(runCtx); err != nil {
				return themeRuntimeRunError(ctx, heartbeatErrors, err)
			}
		case <-wake:
			if err := w.reconcileUntilCurrent(runCtx); err != nil {
				return themeRuntimeRunError(ctx, heartbeatErrors, err)
			}
		}
	}
}

func (w *ThemeRuntimeWatcher) Initialize(ctx context.Context) error {
	if w == nil || ctx == nil {
		return ErrThemeRuntimeNodeInvalid
	}
	if err := w.ensureInitialPublication(ctx); err != nil {
		return err
	}
	if err := w.registerNode(ctx); err != nil {
		return err
	}
	return w.reconcileUntilCurrent(ctx)
}

func (w *ThemeRuntimeWatcher) ReconcileOnce(ctx context.Context) error {
	if w == nil || ctx == nil {
		return ErrThemeRuntimeNodeInvalid
	}
	_, err := w.reconcileOnce(ctx)
	return err
}

func (w *ThemeRuntimeWatcher) reconcileUntilCurrent(ctx context.Context) error {
	for {
		applied, err := w.reconcileOnce(ctx)
		if err != nil || !applied {
			return err
		}
	}
}

func (w *ThemeRuntimeWatcher) reconcileOnce(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	publication, err := w.repository.LatestThemeRuntimePublication(ctx)
	if err != nil {
		return false, err
	}
	if publication.Revision <= 0 || !validThemeRuntimePublication(publication) {
		return false, ErrThemePublicationConflict
	}
	node, err := w.repository.GetThemeRuntimeNode(ctx, w.config.Identity)
	if err != nil {
		return false, err
	}
	if !validThemeRuntimeWatcherNode(node, w.config.Identity) {
		return false, ErrThemeRuntimeNodeInvalid
	}
	if node.LastAppliedRevision >= publication.Revision {
		return false, nil
	}
	ack, err := w.repository.BeginThemeRuntimePublicationApply(ctx, w.config.Identity, publication.Revision)
	if err != nil {
		return false, err
	}
	if !validThemeRuntimeApplyingAck(ack, w.config.Identity, publication.Revision) {
		return false, ErrThemeRuntimeAckConflict
	}
	if err := w.applier.ApplyThemeRuntimePublication(ctx, publication); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return false, errors.Join(err, contextErr)
		}
		failed, failErr := w.repository.FailThemeRuntimePublicationApply(
			ctx, w.config.Identity, publication.Revision, ack.Revision, themeRuntimeFailureReason(err),
		)
		if failErr != nil {
			return false, errors.Join(
				fmt.Errorf("apply theme runtime revision %d: %w", publication.Revision, err),
				fmt.Errorf("record theme runtime failure: %w", failErr),
			)
		}
		return false, fmt.Errorf("apply theme runtime revision %d attempt %d: %w", publication.Revision, failed.AttemptCount, err)
	}
	completed, err := w.repository.CompleteThemeRuntimePublicationApply(
		ctx, w.config.Identity, publication, ack.Revision,
	)
	if err != nil {
		advanced, verifyErr := w.repository.GetThemeRuntimeNode(ctx, w.config.Identity)
		if verifyErr == nil && validThemeRuntimeWatcherNode(advanced, w.config.Identity) &&
			advanced.LastAppliedRevision >= publication.Revision {
			return true, nil
		}
		if verifyErr != nil {
			return false, errors.Join(err, fmt.Errorf("verify theme runtime completion: %w", verifyErr))
		}
		if !validThemeRuntimeWatcherNode(advanced, w.config.Identity) {
			return false, errors.Join(err, ErrThemeRuntimeNodeInvalid)
		}
		return false, err
	}
	if !validThemeRuntimeCompletedAck(completed, ack, publication) {
		return false, ErrThemeRuntimeAckConflict
	}
	return true, nil
}

func (w *ThemeRuntimeWatcher) ensureInitialPublication(ctx context.Context) error {
	publication, err := w.repository.EnsureInitialThemeRuntimePublication(ctx)
	if err != nil {
		return err
	}
	if publication.Revision <= 0 || !validThemeRuntimePublication(publication) {
		return ErrThemePublicationConflict
	}
	return nil
}

func (w *ThemeRuntimeWatcher) registerNode(ctx context.Context) error {
	node, err := w.repository.RegisterThemeRuntimeNode(ctx, w.config.Identity, w.config.LeaseDuration)
	if err != nil {
		return err
	}
	if !validThemeRuntimeWatcherNode(node, w.config.Identity) {
		return ErrThemeRuntimeNodeInvalid
	}
	return nil
}

func (w *ThemeRuntimeWatcher) heartbeat(
	ctx context.Context,
	cancel context.CancelFunc,
	errorsCh chan<- error,
	done chan<- struct{},
) {
	defer func() { done <- struct{}{} }()
	ticker := time.NewTicker(w.config.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			heartbeatCtx, stopHeartbeat := context.WithTimeout(ctx, w.config.HeartbeatInterval)
			node, err := w.repository.HeartbeatThemeRuntimeNode(
				heartbeatCtx, w.config.Identity, w.config.LeaseDuration,
			)
			heartbeatContextErr := heartbeatCtx.Err()
			stopHeartbeat()
			if ctx.Err() != nil {
				return
			}
			if heartbeatContextErr != nil {
				err = heartbeatContextErr
			}
			if err == nil && !validThemeRuntimeWatcherNode(node, w.config.Identity) {
				err = ErrThemeRuntimeNodeInvalid
			}
			if err != nil {
				errorsCh <- err
				cancel()
				return
			}
		}
	}
}

func (w *ThemeRuntimeWatcher) stopWorkers(
	cancel context.CancelFunc,
	done <-chan struct{},
	count int,
) {
	cancel()
	timer := time.NewTimer(w.config.StopTimeout)
	defer timer.Stop()
	for range count {
		select {
		case <-done:
		case <-timer.C:
			w.report(fmt.Errorf("theme runtime worker stop: %w", context.DeadlineExceeded))
			return
		}
	}
}

func themeRuntimeRunError(
	parent context.Context,
	heartbeatErrors <-chan error,
	err error,
) error {
	if heartbeatErr := receiveThemeRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
		return heartbeatErr
	}
	if parent.Err() != nil {
		return nil
	}
	return err
}

func receiveThemeRuntimeHeartbeatError(errorsCh <-chan error) error {
	select {
	case err := <-errorsCh:
		return err
	default:
		return nil
	}
}

func validThemeRuntimeWatcherNode(node ThemeRuntimeNode, identity ThemeRuntimeNodeIdentity) bool {
	return node.ThemeRuntimeNodeIdentity == identity && node.LastAppliedRevision >= 0
}

func validThemeRuntimeApplyingAck(
	ack ThemeRuntimePublicationAck,
	identity ThemeRuntimeNodeIdentity,
	publicationRevision int64,
) bool {
	return ack.ThemeRuntimeNodeIdentity == identity && ack.PublicationRevision == publicationRevision &&
		ack.Status == ThemeRuntimeAckApplying && ack.AttemptCount > 0 && ack.Revision > 0
}

func validThemeRuntimeCompletedAck(
	completed ThemeRuntimePublicationAck,
	applying ThemeRuntimePublicationAck,
	publication ThemeRuntimePublication,
) bool {
	return completed.ThemeRuntimeNodeIdentity == applying.ThemeRuntimeNodeIdentity &&
		completed.PublicationRevision == publication.Revision && completed.Status == ThemeRuntimeAckApplied &&
		completed.AttemptCount == applying.AttemptCount && completed.Revision > applying.Revision &&
		completed.AppliedState == publication.DesiredState && completed.AppliedThemeID == publication.ThemeID &&
		completed.AppliedThemeVersion == publication.ThemeVersion &&
		completed.AppliedPackageDigest == publication.PackageDigest
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
