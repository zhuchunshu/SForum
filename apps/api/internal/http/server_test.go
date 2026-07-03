package http

import (
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/inkedus/sforum/apps/api/internal/config"
)

func TestHealthEndpoint(t *testing.T) {
	cfg := config.Config{
		AppName:          "SForum",
		AppEnv:           "test",
		AppLocale:        "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	}

	app := NewApp(cfg, slog.Default())
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/health", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected ok status, got %q", body.Status)
	}
	if body.Locale != "zh-CN" {
		t.Fatalf("expected zh-CN locale, got %q", body.Locale)
	}
	if len(body.SupportedLocales) != 2 {
		t.Fatalf("expected two supported locales, got %v", body.SupportedLocales)
	}
}
