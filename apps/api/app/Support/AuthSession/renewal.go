package authsession

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func (m *Manager) refresh(
	c fiber.Ctx,
	ctx context.Context,
	state *requestSessionState,
	sess *session.Session,
	generation uint64,
	userID int64,
	tokenVersion int64,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var (
		now                 time.Time
		renewedAt           time.Time
		renew               bool
		shouldTouchLastSeen bool
		touchSID            string
	)
	if err := func() error {
		state.lifecycleMu.Lock()
		defer state.lifecycleMu.Unlock()
		if state.pending.Load() || state.pendingActive || state.generation != generation {
			return ErrRenewalRejected
		}
		now = m.currentTime()
		createdAt := sessionTime(sess.Get(sessionCreatedAtKey))
		if createdAt.IsZero() {
			createdAt = now
			sess.Set(sessionCreatedAtKey, createdAt)
		}

		renewedAt = sessionTime(sess.Get(sessionRenewedAtKey))
		if renewedAt.IsZero() {
			renewedAt = createdAt
		}

		// 定期换 session id，降低长期复用同一个 id 带来的泄露影响。
		// sid 不随 cookie session id 轮换而变（保留在 payload），设备列表稳定。
		renew = m.renewalInterval > 0 && !renewedAt.After(now) && now.Sub(renewedAt) >= m.renewalInterval

		// 节流刷新 last_seen：距上次刷新超过 lastSeenInterval 时才写库，避免每请求都写。
		// 与 session 续期解耦：续期是安全相关（轮换 id），last_seen 是展示相关（最后活跃）。
		if m.sessions != nil {
			lastSeenTouched := sessionTime(sess.Get(sessionLastSeenTouchedKey))
			if lastSeenTouched.IsZero() || now.Sub(lastSeenTouched) >= lastSeenInterval {
				sid, _ := sess.Get(sessionSIDKey).(string)
				if sid != "" {
					shouldTouchLastSeen = true
					touchSID = sid
					sess.Set(sessionLastSeenTouchedKey, now)
				}
			}
		}
		return nil
	}(); err != nil {
		return err
	}

	save := func(effectCtx context.Context) error {
		if renew {
			if m.tokenVersion != nil {
				currentVersion, err := m.tokenVersion(effectCtx, userID)
				if err != nil || currentVersion != tokenVersion {
					return ErrRenewalRejected
				}
			}
		}
		state.lifecycleMu.Lock()
		defer state.lifecycleMu.Unlock()
		if state.pending.Load() || state.pendingActive || state.generation != generation {
			return ErrRenewalRejected
		}
		if renew {
			defer func() {
				if panicValue := recover(); panicValue != nil {
					state.invalidateLocked()
					sid, _ := sess.Get(sessionSIDKey).(string)
					func() {
						defer func() { _ = recover() }()
						m.abortSessionCredential(
							effectCtx, c, state, sess, sess.ID(), userID, sid, "renew_failed",
						)
					}()
					panic(panicValue)
				}
			}()
		}
		err := state.run(effectCtx, func() error {
			if renew {
				if err := sess.Regenerate(); err != nil {
					return err
				}
				renewedAt = now
				sess.Set(sessionRenewedAtKey, renewedAt)
			}
			return sess.Save()
		})
		if err != nil {
			if renew {
				state.invalidateLocked()
				sid, _ := sess.Get(sessionSIDKey).(string)
				m.abortSessionCredential(effectCtx, c, state, sess, sess.ID(), userID, sid, "renew_failed")
			}
			return err
		}
		return nil
	}
	if renew && m.renewalEffectGate != nil {
		if err := m.runRenewalEffectGate(ctx, userID, tokenVersion, save); err != nil {
			return err
		}
	} else {
		if renew && m.renewalGate != nil {
			if err := m.renewalGate(ctx, userID); err != nil {
				return fmt.Errorf("%w: %v", ErrRenewalRejected, err)
			}
		}
		if err := save(ctx); err != nil {
			return err
		}
	}
	if shouldTouchLastSeen {
		_ = m.sessions.TouchSessionLastSeen(ctx, userID, touchSID)
	}
	return nil
}

func (m *Manager) runRenewalEffectGate(
	ctx context.Context,
	userID int64,
	tokenVersion int64,
	effect RenewalEffect,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var mu sync.Mutex
	open := true
	started := false
	var effectResult error
	var callbackPanic any
	done := make(chan struct{})

	wrapped := RenewalEffect(func(effectCtx context.Context) (result error) {
		mu.Lock()
		if !open || ctx.Err() != nil {
			mu.Unlock()
			return &renewalEffectError{cause: ErrRenewalRejected}
		}
		if started {
			mu.Unlock()
			return &renewalEffectError{cause: ErrRenewalRejected}
		}
		started = true
		if effectCtx == nil {
			effectCtx = ctx
		}
		mu.Unlock()
		defer func() {
			panicValue := recover()
			if panicValue != nil {
				result = &renewalEffectError{cause: ErrRenewalRejected}
			}
			mu.Lock()
			effectResult = result
			callbackPanic = panicValue
			close(done)
			mu.Unlock()
		}()
		if err := effect(effectCtx); err != nil {
			return &renewalEffectError{cause: err}
		}
		return nil
	})

	var gateErr error
	var gatePanic any
	func() {
		defer func() { gatePanic = recover() }()
		gateErr = m.renewalEffectGate(ctx, userID, tokenVersion, wrapped)
	}()
	mu.Lock()
	open = false
	wasStarted := started
	mu.Unlock()
	if wasStarted {
		<-done
	}
	mu.Lock()
	result := effectResult
	panicValue := callbackPanic
	mu.Unlock()
	if panicValue != nil {
		panic(panicValue)
	}
	if gatePanic != nil {
		panic(gatePanic)
	}
	if !wasStarted {
		return errors.Join(ErrRenewalRejected, gateErr)
	}
	var effectErr *renewalEffectError
	if errors.As(result, &effectErr) {
		return effectErr.cause
	}
	if result != nil {
		return result
	}
	// Once the Host mutation has committed, it is the terminal truth. A malformed
	// gate cannot turn the renewed credential back into a denial after the fact.
	return nil
}

type renewalEffectError struct {
	cause error
}

func (e *renewalEffectError) Error() string {
	if e == nil || e.cause == nil {
		return "authsession: renewal effect failed"
	}
	return e.cause.Error()
}

func (e *renewalEffectError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}
