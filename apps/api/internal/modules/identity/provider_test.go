package identity

import (
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestProviderRegistersIdentityRoutes(t *testing.T) {
	_, store := newTestService(t)
	provider := NewProvider(store, session.NewStore())

	app := fiber.New()
	api := app.Group("/api/v1")
	provider.RegisterRoutes(api)

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/session", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected registered route to return 401, got %d", resp.StatusCode)
	}
}
