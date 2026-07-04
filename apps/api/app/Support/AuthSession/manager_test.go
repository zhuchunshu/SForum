package authsession

import (
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
