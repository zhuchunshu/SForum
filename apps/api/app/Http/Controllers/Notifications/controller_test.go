package notificationscontroller

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestNotificationListRejectsInvalidPATBeforeResolvingUserID(t *testing.T) {
	store := &notificationTestStore{}
	controller := NewController(store, nil, nil, nil)
	app := apphttp.NewApp(notificationTestConfig(), slog.Default(), apphttp.Dependencies{
		BearerTokens:   notificationBearer{err: apitokens.ErrTokenInvalid},
		RouteProviders: []apphttp.RouteProvider{controller},
	})

	resp := notificationRequest(t, app, "sft_disabled-user")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if store.listCalls != 0 {
		t.Fatalf("invalid PAT reached notification store %d times", store.listCalls)
	}
}

func TestNotificationListAcceptsAuthenticatedActivePAT(t *testing.T) {
	store := &notificationTestStore{}
	controller := NewController(store, nil, nil, nil)
	app := apphttp.NewApp(notificationTestConfig(), slog.Default(), apphttp.Dependencies{
		BearerTokens: notificationBearer{auth: apitokens.Authenticated{
			UserID: 42, TokenID: 7, PublicID: "active", Scopes: []string{"post.create"},
		}},
		RouteProviders: []apphttp.RouteProvider{controller},
	})

	resp := notificationRequest(t, app, "sft_active-user")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if store.lastUserID != 42 {
		t.Fatalf("notification query user=%d", store.lastUserID)
	}
}

type notificationBearer struct {
	auth apitokens.Authenticated
	err  error
}

func (b notificationBearer) AuthenticatePlaintext(fiber.Ctx, string) (apitokens.Authenticated, error) {
	return b.auth, b.err
}

type notificationTestStore struct {
	listCalls  int
	lastUserID int64
}

func (s *notificationTestStore) List(_ context.Context, input notifications.ListInput) (notifications.Page, error) {
	s.listCalls++
	s.lastUserID = input.RecipientUserID
	return notifications.Page{Items: []notifications.Notification{}}, nil
}

func (*notificationTestStore) UnreadCount(context.Context, int64) (int64, error) { return 0, nil }
func (*notificationTestStore) MarkRead(context.Context, int64, int64) error      { return nil }
func (*notificationTestStore) MarkAllRead(context.Context, int64) (int64, error) {
	return 0, nil
}
func (*notificationTestStore) GetDelivery(context.Context, int64) (notifications.MailDelivery, error) {
	return notifications.MailDelivery{}, nil
}
func (*notificationTestStore) UpdateDelivery(context.Context, notifications.DeliveryUpdate) error {
	return nil
}
func (*notificationTestStore) ListDeliveries(context.Context, int) ([]notifications.MailDelivery, error) {
	return nil, nil
}

func notificationRequest(t *testing.T, app *fiber.App, token string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func notificationTestConfig() config.Config {
	return config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}
}
