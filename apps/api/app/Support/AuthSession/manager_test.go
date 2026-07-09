package authsession

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestStartResetsSessionAndStoresUser(t *testing.T) {
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{HashSecret: "test-secret"})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		info, err := manager.Start(c, 42)
		if err != nil {
			return err
		}
		if info.ID == "" || info.Hash == "" {
			t.Fatalf("expected session info to include id and hash, got %#v", info)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodPost, "/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(resp.Cookies()) == 0 {
		t.Fatal("expected session cookie")
	}
}

func TestCurrentUserIDRefreshesAndRenewsSession(t *testing.T) {
	baseTime := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		RenewalInterval: time.Minute,
		HashSecret:      "test-secret",
	})
	manager.now = func() time.Time { return baseTime }

	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserID(c)
		if err != nil {
			return err
		}
		if !ok || userID != 42 {
			t.Fatalf("expected current user 42, got %d ok=%v", userID, ok)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	loginReq := httptest.NewRequest(fiber.MethodPost, "/login", nil)
	loginResp, err := app.Test(loginReq)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResp.Body.Close()
	oldCookie := loginResp.Cookies()[0]

	manager.now = func() time.Time { return baseTime.Add(2 * time.Minute) }
	sessionReq := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	sessionReq.AddCookie(oldCookie)
	sessionResp, err := app.Test(sessionReq)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	defer sessionResp.Body.Close()

	if sessionResp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", sessionResp.StatusCode)
	}
	if len(sessionResp.Cookies()) == 0 {
		t.Fatal("expected refreshed session cookie")
	}
	if sessionResp.Cookies()[0].Value == oldCookie.Value {
		t.Fatal("expected renewed session id after renewal interval")
	}
}

// TestTokenVersionInvalidationAfterIncrement 验证 M8：当用户令牌版本号递增后（如密码重置），
// 旧 session 的 CurrentUserID 返回 ok=false，会话立即失效。
func TestTokenVersionInvalidationAfterIncrement(t *testing.T) {
	currentVersion := int64(5)
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret",
		TokenVersion: func(_ context.Context, _ int64) (int64, error) {
			return currentVersion, nil
		},
	})

	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Get("/session", func(c fiber.Ctx) error {
		_, ok, err := manager.CurrentUserID(c)
		if err != nil {
			return err
		}
		if !ok {
			t.Fatal("expected current user to be valid before token version increment")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/session-after-reset", func(c fiber.Ctx) error {
		_, ok, err := manager.CurrentUserID(c)
		if err != nil {
			return err
		}
		if ok {
			t.Fatal("expected current user to be INVALID after token version increment (password reset)")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	// 1. 登录（写入版本号 5）。
	loginReq := httptest.NewRequest(fiber.MethodPost, "/login", nil)
	loginResp, err := app.Test(loginReq)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResp.Body.Close()
	cookie := loginResp.Cookies()[0]

	// 2. 登录后立即校验会话有效。
	sessionReq := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	sessionReq.AddCookie(cookie)
	sessionResp, err := app.Test(sessionReq)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	sessionResp.Body.Close()
	if sessionResp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected session valid 200, got %d", sessionResp.StatusCode)
	}

	// 3. 模拟密码重置：递增令牌版本号。
	currentVersion = 6

	// 4. 同一 session cookie 校验：版本不匹配，会话应失效。
	resetReq := httptest.NewRequest(fiber.MethodGet, "/session-after-reset", nil)
	resetReq.AddCookie(cookie)
	resetResp, err := app.Test(resetReq)
	if err != nil {
		t.Fatalf("session-after-reset request failed: %v", err)
	}
	resetResp.Body.Close()
	if resetResp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 (handler ok with invalid session), got %d", resetResp.StatusCode)
	}
}
