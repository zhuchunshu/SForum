package authsession

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/valyala/fasthttp"
)

// Pending 是 Begin 后尚未持久化的会话；调用方可设置设备展示信息后再 Save。
type Pending struct {
	manager    *Manager
	session    *session.Session
	state      *requestSessionState
	request    fiber.Ctx
	context    context.Context
	info       Info
	userID     int64
	generation uint64
	// deviceInfo 由调用方在 Save 前设置；非空时 Save 会写入会话目录。
	deviceInfo *SessionRecordInput
	consumed   atomic.Bool
}

func (p *Pending) Info() Info {
	if p == nil {
		return Info{}
	}
	return p.info
}

// SetDeviceInfo 设置设备展示信息（device_name/browser/os/ip_prefix 等，由调用方解析 UA/IP 得到）。
// 必须在 Save 前调用；Save 时会据此写入会话目录。
func (p *Pending) SetDeviceInfo(info SessionRecordInput) {
	if p == nil {
		return
	}
	// 用 pending 已有的 sid/hash 补齐，确保一致。
	info.UserID = p.userID
	info.SID = p.info.SID
	info.SessionHash = p.info.Hash
	p.deviceInfo = &info
}

func (p *Pending) Save() error {
	if p == nil || p.context == nil {
		return p.SaveContext(context.Background())
	}
	return p.SaveContext(p.context)
}

// SaveContext persists the accepted session effect and its directory record.
// A Pending value is one-shot so duplicate/concurrent saves cannot create
// multiple directory effects.
func (p *Pending) SaveContext(ctx context.Context) error {
	if p == nil || p.session == nil {
		return nil
	}
	if !p.consumed.CompareAndSwap(false, true) {
		return ErrPendingConsumed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if p.state == nil {
		return ErrSessionAuthorityChanged
	}
	p.state.lifecycleMu.Lock()
	defer p.state.lifecycleMu.Unlock()
	if !p.state.pendingActive || p.generation == 0 || p.generation != p.state.generation {
		return ErrSessionAuthorityChanged
	}
	cleanupStarted := false
	defer func() {
		if panicValue := recover(); panicValue != nil {
			if !cleanupStarted {
				p.state.invalidateLocked()
				func() {
					defer func() { _ = recover() }()
					p.manager.abortSessionCredential(
						ctx, p.request, p.state, p.session, p.info.ID,
						p.deviceUserID(), p.info.SID, "issue_failed",
					)
				}()
			}
			panic(panicValue)
		}
	}()
	if err := p.state.run(ctx, p.session.Save); err != nil {
		p.state.invalidateLocked()
		cleanupStarted = true
		return p.abortIssue(ctx, err)
	}
	// 目录写入失败不能静默成功：后续鉴权依赖目录判断远程下线状态。
	if p.manager.sessions != nil && p.deviceInfo != nil {
		if err := p.manager.sessions.CreateSession(ctx, *p.deviceInfo); err != nil {
			p.state.invalidateLocked()
			cleanupStarted = true
			return p.abortIssue(ctx, err)
		}
	}
	p.state.pendingActive = false
	p.state.pending.Store(false)
	return nil
}

func (p *Pending) abortIssue(ctx context.Context, cause error) error {
	p.manager.abortSessionCredential(
		ctx, p.request, p.state, p.session, p.info.ID,
		p.deviceUserID(), p.info.SID, "issue_failed",
	)
	return cause
}

func (p *Pending) deviceUserID() int64 {
	if p == nil {
		return 0
	}
	return p.userID
}

func (m *Manager) abortSessionCredential(
	ctx context.Context,
	c fiber.Ctx,
	state *requestSessionState,
	sess *session.Session,
	sessionID string,
	userID int64,
	sid string,
	reason string,
) {
	if m == nil || sess == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cleanupCtx, cancelCleanup := m.sessionCleanupRoot(ctx)
	defer cancelCleanup()
	var cleanup sync.WaitGroup
	cleanup.Add(1)
	go func() {
		defer cleanup.Done()
		defer func() { _ = recover() }()
		m.revokeSessionDirectoryWithin(cleanupCtx, userID, sid, reason)
	}()
	func() {
		defer func() { _ = recover() }()
		m.scrubSessionCredentialWithin(cleanupCtx, c, state, sess, sessionID)
	}()
	cleanup.Wait()
}

func (m *Manager) revokeSessionDirectory(ctx context.Context, userID int64, sid string, reason string) {
	defer func() { _ = recover() }()
	cleanupCtx, cancelCleanup := m.sessionCleanupRoot(ctx)
	defer cancelCleanup()
	m.revokeSessionDirectoryWithin(cleanupCtx, userID, sid, reason)
}

func (m *Manager) revokeSessionDirectoryWithin(ctx context.Context, userID int64, sid string, reason string) {
	if m == nil || m.sessions == nil || userID <= 0 || sid == "" {
		return
	}
	_ = m.sessions.RevokeSession(ctx, userID, sid, reason)
}

func (m *Manager) scrubSessionCredential(
	ctx context.Context,
	c fiber.Ctx,
	state *requestSessionState,
	sess *session.Session,
	sessionID string,
) {
	cleanupCtx, cancelCleanup := m.sessionCleanupRoot(ctx)
	defer cancelCleanup()
	m.scrubSessionCredentialWithin(cleanupCtx, c, state, sess, sessionID)
}

func (m *Manager) scrubSessionCredentialWithin(
	ctx context.Context,
	c fiber.Ctx,
	state *requestSessionState,
	sess *session.Session,
	sessionID string,
) {
	if m == nil || sess == nil {
		return
	}
	defer func() { _ = recover() }()
	defer m.expireSessionTransport(c)
	attemptTimeout := m.sessionCleanupTimeout() / 3
	if attemptTimeout <= 0 {
		attemptTimeout = time.Nanosecond
	}
	destroyFailed := true
	func() {
		destroyCtx, cancelDestroy := context.WithTimeout(ctx, attemptTimeout)
		defer cancelDestroy()
		defer func() {
			if recover() != nil {
				destroyFailed = true
			}
		}()
		var destroyErr error
		if state != nil {
			destroyErr = state.run(destroyCtx, sess.Destroy)
		} else {
			destroyErr = sess.Destroy()
		}
		destroyFailed = destroyErr != nil
	}()
	if destroyFailed && sessionID != "" {
		// Fiber clears the in-memory payload before Delete. Retry with a detached
		// budget, then best-effort overwrite a commit-unknown payload with the
		// already-cleared session data.
		func() {
			deleteCtx, cancelDelete := context.WithTimeout(ctx, attemptTimeout)
			defer cancelDelete()
			defer func() { _ = recover() }()
			_ = m.store.Storage.DeleteWithContext(deleteCtx, sessionID)
		}()
		func() {
			saveCtx, cancelSave := context.WithTimeout(ctx, attemptTimeout)
			defer cancelSave()
			defer func() { _ = recover() }()
			if state != nil {
				_ = state.run(saveCtx, sess.Save)
			} else {
				_ = sess.Save()
			}
		}()
	}
}

func (m *Manager) sessionCleanupRoot(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(ctx), m.sessionCleanupTimeout())
}

func (m *Manager) sessionCleanupTimeout() time.Duration {
	timeout := sessionIssueCleanupTimeout
	if m != nil && m.cleanupTimeout > 0 {
		timeout = m.cleanupTimeout
	}
	return timeout
}

func (m *Manager) expireSessionTransport(c fiber.Ctx) {
	if m == nil || m.store == nil || c == nil {
		return
	}
	extractor := m.store.Extractor
	relevant := []extractors.Extractor{extractor}
	if len(extractor.Chain) > 0 {
		relevant = extractor.Chain
	}
	for _, candidate := range relevant {
		if candidate.Key == "" {
			continue
		}
		switch candidate.Source {
		case extractors.SourceHeader:
			c.Request().Header.Del(candidate.Key)
			c.Response().Header.Del(candidate.Key)
		case extractors.SourceCookie:
			c.Request().Header.DelCookie(candidate.Key)
			c.Response().Header.DelCookie(candidate.Key)
			cookie := fasthttp.AcquireCookie()
			cookie.SetKey(candidate.Key)
			cookie.SetPath(m.store.CookiePath)
			cookie.SetDomain(m.store.CookieDomain)
			cookie.SetMaxAge(-1)
			cookie.SetExpire(time.Now().Add(-time.Minute))
			switch {
			case strings.EqualFold(m.store.CookieSameSite, fiber.CookieSameSiteStrictMode):
				cookie.SetSameSite(fasthttp.CookieSameSiteStrictMode)
			case strings.EqualFold(m.store.CookieSameSite, fiber.CookieSameSiteNoneMode):
				cookie.SetSameSite(fasthttp.CookieSameSiteNoneMode)
			default:
				cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
			}
			cookie.SetSecure(m.store.CookieSecure || cookie.SameSite() == fasthttp.CookieSameSiteNoneMode)
			cookie.SetHTTPOnly(m.store.CookieHTTPOnly)
			c.Response().Header.SetCookie(cookie)
			fasthttp.ReleaseCookie(cookie)
		}
	}
}
