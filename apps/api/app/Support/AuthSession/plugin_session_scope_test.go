package authsession

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// scopeHarness 用真实 Fiber 请求驱动会话生命周期，并在 handler 内取范围。
type scopeHarness struct {
	manager *Manager
	app     *fiber.App
	scopes  []RequestScope
	found   []bool
}

func newScopeHarness(manager *Manager) *scopeHarness {
	harness := &scopeHarness{manager: manager, app: fiber.New()}
	harness.app.Get("/scope", func(c fiber.Ctx) error {
		// 先走一次 CurrentUserID：续期（含 cookie 会话 id 轮换）只在认证路径上发生。
		if _, _, err := manager.CurrentUserID(c); err != nil {
			return err
		}
		scope, ok := manager.PluginSessionScope(c)
		harness.scopes = append(harness.scopes, scope)
		harness.found = append(harness.found, ok)
		return c.SendStatus(fiber.StatusOK)
	})
	harness.app.Post("/login", func(c fiber.Ctx) error {
		if _, err := manager.Start(c, 42); err != nil {
			return err
		}
		return c.SendStatus(fiber.StatusOK)
	})
	return harness
}

func (h *scopeHarness) last() (RequestScope, bool) {
	if len(h.scopes) == 0 {
		return RequestScope{}, false
	}
	return h.scopes[len(h.scopes)-1], h.found[len(h.found)-1]
}

// do 发一次请求，可选带上上一次响应的 cookie。
func (h *scopeHarness) do(t *testing.T, method, path string, cookies []*http.Cookie) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response, err := h.app.Test(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// 已认证会话的范围按 sid 派生，且必须跨 24h 轮换稳定（轮换换的是 cookie 会话 id，
// 不是 sid）。这条断言就是 S5「Rotation」行。
func TestPluginSessionScopeFollowsSIDAcrossRotation(t *testing.T) {
	baseTime := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	manager := NewManager(session.NewStore(session.Config{
		IdleTimeout: time.Hour, AbsoluteTimeout: 24 * time.Hour,
	}), Config{RenewalInterval: time.Minute, HashSecret: "test-secret"})
	manager.now = func() time.Time { return baseTime }
	harness := newScopeHarness(manager)

	login := harness.do(t, fiber.MethodPost, "/login", nil)
	loginCookies := login.Cookies()
	if len(loginCookies) == 0 {
		t.Fatal("login did not issue a session cookie")
	}
	first := harness.do(t, fiber.MethodGet, "/scope", loginCookies)
	before, ok := harness.last()
	if !ok {
		t.Fatal("authenticated session produced no scope")
	}
	if len(before.SessionFingerprint) != 64 {
		t.Fatalf("fingerprint = %q, want 64 hex", before.SessionFingerprint)
	}
	if before.ActorUserID != 42 {
		t.Fatalf("actor = %d, want 42", before.ActorUserID)
	}

	// 越过续期间隔：下一次请求会 Regenerate() 出新的 cookie 会话 id。
	baseTime = baseTime.Add(2 * time.Minute)
	rotated := harness.do(t, fiber.MethodGet, "/scope", first.Cookies())
	after, ok := harness.last()
	if !ok {
		t.Fatal("rotated session produced no scope")
	}
	if after.SessionFingerprint != before.SessionFingerprint {
		t.Fatalf("rotation changed the session fingerprint: %q -> %q",
			before.SessionFingerprint, after.SessionFingerprint)
	}
	// 确认这次请求真的轮换了 cookie 会话 id，否则上面的断言什么都没证明。
	if !sessionCookieChanged(first, rotated) {
		t.Fatal("the cookie session id did not rotate, so rotation stability was not exercised")
	}
}

// 绝对寿命必须以 created_at 为锚点，而不是每次读取都按「现在 + 配置值」重算——
// 后者会让一个已经存在一小时的会话把上界算晚一小时，插件状态于是可能活过 Core 会话。
func TestPluginSessionScopeAnchorsAbsoluteBoundaryAtCreation(t *testing.T) {
	baseTime := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	manager := NewManager(session.NewStore(session.Config{
		IdleTimeout: time.Hour, AbsoluteTimeout: 24 * time.Hour,
	}), Config{HashSecret: "test-secret"})
	manager.now = func() time.Time { return baseTime }
	harness := newScopeHarness(manager)

	login := harness.do(t, fiber.MethodPost, "/login", nil)
	cookies := login.Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not issue a session cookie")
	}
	harness.do(t, fiber.MethodGet, "/scope", cookies)
	issued, ok := harness.last()
	if !ok {
		t.Fatal("no scope after login")
	}
	if !issued.AbsoluteExpiresAt.Equal(baseTime.Add(24 * time.Hour)) {
		t.Fatalf("absolute boundary = %s, want %s", issued.AbsoluteExpiresAt, baseTime.Add(24*time.Hour))
	}

	baseTime = baseTime.Add(time.Hour)
	harness.do(t, fiber.MethodGet, "/scope", cookies)
	observed, ok := harness.last()
	if !ok {
		t.Fatal("no scope one hour later")
	}
	if !observed.AbsoluteExpiresAt.Equal(issued.AbsoluteExpiresAt) {
		t.Fatalf("absolute boundary moved with the clock: %s -> %s", issued.AbsoluteExpiresAt, observed.AbsoluteExpiresAt)
	}
	if !observed.IdleExpiresAt.After(issued.IdleExpiresAt) {
		t.Fatalf("idle boundary did not follow the request: %s -> %s", issued.IdleExpiresAt, observed.IdleExpiresAt)
	}
}

// 匿名请求：本轮实测 Core 没有全局 session 中间件，新生成的匿名会话不会被持久化，
// 因此它没有可达的命名空间。范围必须 fail closed，而不是给插件一个下一请求就失效的键。
func TestPluginSessionScopeRefusesUnpersistedAnonymousSession(t *testing.T) {
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{HashSecret: "test-secret"})
	harness := newScopeHarness(manager)
	response := harness.do(t, fiber.MethodGet, "/scope", nil)
	if _, ok := harness.last(); ok {
		t.Fatal("an unpersisted anonymous session produced a scope")
	}
	if len(response.Cookies()) != 0 {
		t.Fatal("an anonymous request unexpectedly persisted a session cookie")
	}
}

func TestPluginSessionScopeContextCarrierIsFailClosed(t *testing.T) {
	if _, ok := PluginSessionScopeFromContext(context.Background()); ok {
		t.Fatal("an empty context produced a scope")
	}
	if _, ok := PluginSessionScopeFromContext(nil); ok {
		t.Fatal("a nil context produced a scope")
	}
	scope := RequestScope{SessionFingerprint: "abc", ActorUserID: 7}
	if _, ok := PluginSessionScopeFromContext(WithPluginSessionScope(context.Background(), RequestScope{})); ok {
		t.Fatal("an empty fingerprint produced a scope")
	}
	restored, ok := PluginSessionScopeFromContext(WithPluginSessionScope(context.Background(), scope))
	if !ok || restored.SessionFingerprint != scope.SessionFingerprint || restored.ActorUserID != scope.ActorUserID {
		t.Fatalf("carrier round trip lost data: %#v %t", restored, ok)
	}
	if WithPluginSessionScope(nil, scope) != nil {
		t.Fatal("nil context must stay nil")
	}
}

// sessionCookieChanged 报告第二次响应是否轮换了 cookie 会话 id。名字取默认 extractor
// 的键（session_id），而不是生产配置里的 sforum_session。
func sessionCookieChanged(before, after *http.Response) bool {
	extract := func(response *http.Response) string {
		for _, cookie := range response.Cookies() {
			if cookie.Name == "session_id" {
				return cookie.Value
			}
		}
		return ""
	}
	first, second := extract(before), extract(after)
	return first != "" && second != "" && first != second
}
