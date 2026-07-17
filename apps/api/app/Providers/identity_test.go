package providers

import (
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestProviderRegistersIdentityRoutes(t *testing.T) {
	provider := NewIdentityProvider(nil, session.NewStore())

	app := fiber.New()
	api := app.Group("/api/v1")
	provider.RegisterRoutes(api)

	for _, test := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "session", method: nethttp.MethodGet, path: "/api/v1/auth/session"},
		{name: "role suggestion list", method: nethttp.MethodGet, path: "/api/v1/roles/suggestions"},
		{name: "role suggestion decision", method: nethttp.MethodPost, path: "/api/v1/roles/suggestions/1/decision"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != nethttp.StatusUnauthorized {
				t.Fatalf("expected registered route to return 401, got %d", resp.StatusCode)
			}
		})
	}
}
