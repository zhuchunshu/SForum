package http_test

import (
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestAbortUsesDefaultReason(t *testing.T) {
	apiErr := apphttp.Abort(fiber.StatusNotFound)

	if apiErr.Status != fiber.StatusNotFound {
		t.Fatalf("expected status 404, got %d", apiErr.Status)
	}
	if apiErr.Reason != "not_found" {
		t.Fatalf("expected not_found reason, got %q", apiErr.Reason)
	}
}

func TestAbortPreservesExplicitReason(t *testing.T) {
	apiErr := apphttp.Abort(fiber.StatusForbidden, "permission.denied")

	if apiErr.Status != fiber.StatusForbidden {
		t.Fatalf("expected status 403, got %d", apiErr.Status)
	}
	if apiErr.Reason != "permission.denied" {
		t.Fatalf("expected explicit reason, got %q", apiErr.Reason)
	}
}

func TestAbortConditionHelpers(t *testing.T) {
	if err := apphttp.AbortIf(false, fiber.StatusForbidden); err != nil {
		t.Fatalf("expected AbortIf(false) to return nil, got %v", err)
	}
	if err := apphttp.AbortUnless(true, fiber.StatusForbidden); err != nil {
		t.Fatalf("expected AbortUnless(true) to return nil, got %v", err)
	}

	if err := apphttp.AbortIf(true, fiber.StatusUnauthorized); err == nil || err.Status != fiber.StatusUnauthorized || err.Reason != "auth.required" {
		t.Fatalf("expected unauthorized APIError from AbortIf(true), got %#v", err)
	}
	if err := apphttp.AbortUnless(false, fiber.StatusTooManyRequests); err == nil || err.Status != fiber.StatusTooManyRequests || err.Reason != "rate_limit.exceeded" {
		t.Fatalf("expected rate limit APIError from AbortUnless(false), got %#v", err)
	}
}

func TestAbortHelperUsesErrorEnvelope(t *testing.T) {
	cfg := config.Config{
		AppName:          "SForum",
		AppEnv:           "test",
		AppLocale:        "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	}
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{routeProviderFunc(func(api fiber.Router) {
			api.Get("/abort-503", func(c fiber.Ctx) error {
				return apphttp.Abort(fiber.StatusServiceUnavailable)
			})
		})},
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/abort-503", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("abort request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected envelope code 503, got %d", body.Code)
	}
	if body.Message != "服务器暂时不可用，请稍后再试。" {
		t.Fatalf("expected internal error message, got %q", body.Message)
	}
	if body.Data.Reason != "internal_error" {
		t.Fatalf("expected internal_error reason, got %q", body.Data.Reason)
	}
}
