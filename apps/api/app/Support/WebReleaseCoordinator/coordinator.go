package webreleasecoordinator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
)

const (
	CheckpointPending          = "pending"
	CheckpointRuntimePrepared  = "runtime_prepared"
	CheckpointEffectsCommitted = "effects_committed"
	CheckpointPointerWritten   = "pointer_written"
	CheckpointSupervisorActive = "supervisor_active"

	activationLockKey = "sforum.web_release.activate"
)

var ErrNoActivation = errors.New("web release coordinator: no activation pending")

type Store interface {
	NextActivation(context.Context) (extensions.WebReleaseDetail, error)
	Transition(context.Context, extensions.WebReleaseTransitionInput) (extensions.WebRelease, error)
	SetCheckpoint(context.Context, int64, string, string) error
	ApplyEffects(context.Context, extensions.WebReleaseDetail, bool) error
	FinalizeRevocations(context.Context, int64) error
}

type RuntimeManager interface {
	Prepare(context.Context, extensions.WebReleaseDetail) error
	Finalize(context.Context, extensions.WebReleaseDetail) error
	Compensate(context.Context, extensions.WebReleaseDetail) error
}

type PointerStore interface {
	WriteCurrent(context.Context, webreleaseruntime.CurrentRelease) error
	ReadActive(context.Context) (webreleaseruntime.ActiveRelease, error)
	ReadFailure(context.Context, int64) (webreleaseruntime.Failure, error)
	RestorePrevious(context.Context, extensions.WebReleaseDetail) error
}

type AdvisoryLocker interface {
	WithLock(context.Context, string, func(context.Context) error) error
}

type Coordinator struct {
	store    Store
	runtime  RuntimeManager
	pointers PointerStore
	lock     AdvisoryLocker
	interval time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func New(store Store, runtime RuntimeManager, pointers PointerStore, lock AdvisoryLocker) *Coordinator {
	return &Coordinator{store: store, runtime: runtime, pointers: pointers, lock: lock, interval: time.Second}
}

func (c *Coordinator) Reconcile(ctx context.Context) error {
	if c == nil || c.store == nil || c.runtime == nil || c.pointers == nil || c.lock == nil {
		return fmt.Errorf("web release coordinator dependencies are incomplete")
	}
	return c.lock.WithLock(ctx, activationLockKey, c.reconcileLocked)
}

func (c *Coordinator) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	done := make(chan struct{})
	c.done = done
	go func() {
		defer close(done)
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			_ = c.Reconcile(runCtx)
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return nil
}

func (c *Coordinator) Stop(ctx context.Context) error {
	c.mu.Lock()
	cancel, done := c.cancel, c.done
	c.cancel, c.done = nil, nil
	c.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
