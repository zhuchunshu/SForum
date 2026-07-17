package routes

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var ErrRouteStreamBudgetExceeded = errors.New("routes: stream route budget exceeded")

const routeStreamDefaultBudget = 24 * time.Hour

// RouteStreamCallerDetacher drops only the request-context link. Host budget
// and exact-runtime cancellation remain authoritative after a WebSocket upgrade.
type RouteStreamCallerDetacher interface {
	DetachCaller() error
}

// RouteStreamLifetimeSource lets the outer transport release its timer and
// caller callback after the exact runtime lease has ended without polling.
type RouteStreamLifetimeSource interface {
	Done() <-chan struct{}
	Cause() error
}

type routeStreamOpenLifetime struct {
	caller       context.Context
	ctx          context.Context
	cancelOpen   context.CancelCauseFunc
	cancelBudget context.CancelFunc

	mu             sync.Mutex
	callerLinked   bool
	callerWon      bool
	callerDetached bool
	closed         bool
	stopCaller     func() bool
	closeOnce      sync.Once
}

func newRouteStreamOpenLifetime(caller context.Context, budget time.Duration) *routeStreamOpenLifetime {
	if budget <= 0 {
		budget = routeStreamDefaultBudget
	}
	// WithoutCancel keeps Host budget independent of caller cancel so detach can
	// drop only the request link after a successful WebSocket upgrade.
	budgetCtx, cancelBudget := context.WithTimeoutCause(
		context.WithoutCancel(caller), budget, ErrRouteStreamBudgetExceeded,
	)
	ctx, cancelOpen := context.WithCancelCause(budgetCtx)
	lifetime := &routeStreamOpenLifetime{
		caller: caller, ctx: ctx, cancelOpen: cancelOpen, cancelBudget: cancelBudget,
		callerLinked: true,
	}
	lifetime.stopCaller = context.AfterFunc(caller, lifetime.cancelFromCaller)
	if caller.Err() != nil {
		lifetime.cancelFromCaller()
	}
	return lifetime
}

func (l *routeStreamOpenLifetime) Context() context.Context {
	if l == nil || l.ctx == nil {
		return context.Background()
	}
	return l.ctx
}

func (l *routeStreamOpenLifetime) cancelFromCaller() {
	if l == nil {
		return
	}
	l.mu.Lock()
	if l.closed || !l.callerLinked {
		l.mu.Unlock()
		return
	}
	l.callerLinked = false
	l.callerWon = true
	cause := context.Cause(l.caller)
	if cause == nil {
		cause = l.caller.Err()
	}
	if cause == nil {
		cause = context.Canceled
	}
	// Cancel while holding mu so Open cannot observe "caller claimed" with a
	// still-live open context between unlock and cancelOpen.
	l.cancelOpen(cause)
	l.mu.Unlock()
}

func (l *routeStreamOpenLifetime) detachCaller() error {
	if l == nil {
		return context.Canceled
	}
	l.mu.Lock()
	if cause := context.Cause(l.ctx); cause != nil {
		l.mu.Unlock()
		return cause
	}
	if l.closed || l.callerWon {
		cause := context.Cause(l.caller)
		l.mu.Unlock()
		if cause == nil {
			cause = context.Canceled
		}
		return cause
	}
	if l.callerDetached {
		l.mu.Unlock()
		return nil
	}
	if !l.callerLinked {
		l.mu.Unlock()
		return context.Canceled
	}
	if cause := context.Cause(l.caller); cause != nil {
		l.callerLinked = false
		l.callerWon = true
		l.cancelOpen(cause)
		l.mu.Unlock()
		return cause
	}
	stop := l.stopCaller
	// stop()==false means the caller callback already started or finished; the
	// caller owns cancellation and upgrade must not pretend detach succeeded.
	if stop != nil && !stop() {
		// Wait for the in-flight cancelFromCaller to publish open cancellation.
		l.mu.Unlock()
		<-l.ctx.Done()
		cause := context.Cause(l.ctx)
		if cause == nil {
			cause = context.Canceled
		}
		return cause
	}
	if cause := context.Cause(l.caller); cause != nil {
		l.callerLinked = false
		l.callerWon = true
		l.stopCaller = nil
		l.cancelOpen(cause)
		l.mu.Unlock()
		return cause
	}
	l.callerLinked = false
	l.callerDetached = true
	l.stopCaller = nil
	l.mu.Unlock()
	return nil
}

func (l *routeStreamOpenLifetime) close(cause error) {
	if l == nil {
		return
	}
	l.closeOnce.Do(func() {
		l.mu.Lock()
		l.closed = true
		l.callerLinked = false
		stop := l.stopCaller
		l.stopCaller = nil
		l.mu.Unlock()
		if stop != nil {
			stop()
		}
		if cause == nil {
			cause = context.Canceled
		}
		l.cancelOpen(cause)
		l.cancelBudget()
	})
}

// routeStreamLifetimeSession is the outer Host owner for budget timer, caller
// callback, and WebSocket detach. Wire cancel and lease release stay on inner.
type routeStreamLifetimeSession struct {
	inner    RouteStreamSession
	lifetime *routeStreamOpenLifetime
	done     chan struct{}
	finish   sync.Once
	finished atomic.Bool
	causeMu  sync.Mutex
	cause    error
}

func bindRouteStreamLifetime(
	inner RouteStreamSession,
	lifetime *routeStreamOpenLifetime,
) RouteStreamSession {
	if inner == nil || lifetime == nil {
		if lifetime != nil {
			lifetime.close(context.Canceled)
		}
		return nil
	}
	session := &routeStreamLifetimeSession{
		inner: inner, lifetime: lifetime, done: make(chan struct{}),
	}
	go session.watch()
	return session
}

func (s *routeStreamLifetimeSession) watch() {
	var innerDone <-chan struct{}
	var source RouteStreamLifetimeSource
	if value, ok := s.inner.(RouteStreamLifetimeSource); ok {
		source = value
		innerDone = source.Done()
	}
	if innerDone == nil {
		<-s.lifetime.Context().Done()
		cause := context.Cause(s.lifetime.Context())
		if !s.finished.Load() {
			s.inner.Cancel()
		}
		s.finishLifetime(cause)
		return
	}
	select {
	case <-s.lifetime.Context().Done():
		// Host budget / caller cancel: ask inner to release, then wait so a typed
		// inner cause is not erased and a late cancel cannot wipe a terminal win.
		lifetimeCause := context.Cause(s.lifetime.Context())
		if !s.finished.Load() {
			s.inner.Cancel()
		}
		<-innerDone
		if source != nil {
			if source.Cause() == nil {
				// Terminal won the atomic race; budget/caller arrived too late.
				s.finishLifetime(nil)
				return
			}
			if lifetimeCause != nil {
				s.finishLifetime(lifetimeCause)
				return
			}
			s.finishLifetime(source.Cause())
			return
		}
		s.finishLifetime(lifetimeCause)
		return
	case <-innerDone:
		// Normal EOF, transport failure, or ForceCancel completed on the inner session.
		s.finishLifetime(source.Cause())
		return
	}
}

func (s *routeStreamLifetimeSession) Send(data []byte, final bool) error {
	if s == nil || s.inner == nil {
		return ErrDispatchTransport
	}
	return s.inner.Send(data, final)
}

func (s *routeStreamLifetimeSession) CloseRequest() error {
	if s == nil || s.inner == nil {
		return ErrDispatchTransport
	}
	return s.inner.CloseRequest()
}

func (s *routeStreamLifetimeSession) Recv() (RouteStreamChunk, error) {
	if s == nil || s.inner == nil {
		return RouteStreamChunk{}, ErrDispatchTransport
	}
	chunk, err := s.inner.Recv()
	switch {
	case errors.Is(err, io.EOF):
		s.finishLifetime(nil)
	case err != nil:
		// Preserve the exact cancel/transport cause for outer observers.
		s.finishLifetime(err)
	}
	return chunk, err
}

func (s *routeStreamLifetimeSession) Response() (DispatchResponse, bool) {
	if s == nil || s.inner == nil {
		return DispatchResponse{}, false
	}
	return s.inner.Response()
}

func (s *routeStreamLifetimeSession) Cancel() {
	if s == nil {
		return
	}
	if s.inner != nil {
		s.inner.Cancel()
	}
	if source, ok := s.inner.(RouteStreamLifetimeSource); ok {
		// Wait for wire/lease ownership to finish so Cause reflects the atomic winner.
		<-source.Done()
		if source.Cause() == nil {
			// Terminal already won; do not invent a generic cancel cause.
			s.finishLifetime(nil)
			return
		}
		if s.lifetime != nil {
			if lifetimeCause := context.Cause(s.lifetime.Context()); lifetimeCause != nil {
				s.finishLifetime(lifetimeCause)
				return
			}
		}
		s.finishLifetime(source.Cause())
		return
	}
	s.finishLifetime(context.Canceled)
}

func (s *routeStreamLifetimeSession) DetachCaller() error {
	if s == nil || s.lifetime == nil {
		return context.Canceled
	}
	return s.lifetime.detachCaller()
}

func (s *routeStreamLifetimeSession) Done() <-chan struct{} {
	if s == nil || s.done == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return s.done
}

func (s *routeStreamLifetimeSession) Cause() error {
	if s == nil {
		return context.Canceled
	}
	s.causeMu.Lock()
	defer s.causeMu.Unlock()
	return s.cause
}

func (s *routeStreamLifetimeSession) finishLifetime(cause error) {
	if s == nil {
		return
	}
	s.finish.Do(func() {
		s.finished.Store(true)
		s.causeMu.Lock()
		s.cause = cause
		s.causeMu.Unlock()
		if s.lifetime != nil {
			s.lifetime.close(cause)
		}
		// Done closes only after state/cause and outer timer release are published.
		close(s.done)
	})
}

var _ RouteStreamSession = (*routeStreamLifetimeSession)(nil)
var _ RouteStreamCallerDetacher = (*routeStreamLifetimeSession)(nil)
var _ RouteStreamLifetimeSource = (*routeStreamLifetimeSession)(nil)
