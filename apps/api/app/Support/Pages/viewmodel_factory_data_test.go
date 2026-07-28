package pages

import (
	"slices"
	"testing"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

func TestBuildCorePageViewModelCopiesTypedProductData(t *testing.T) {
	homeData := &themecompiler.HomePageViewModel{Topics: []themecompiler.TopicSummaryView{{ID: 42, Title: "Production topic"}}}
	model, err := BuildCorePageViewModel(CorePageViewModelRequest{
		PageID: "forum.home", Locale: "en-US", Path: "/", Data: CorePageViewModelData{Home: homeData},
	})
	if err != nil {
		t.Fatal(err)
	}
	home, ok := model.(themecompiler.HomePageViewModel)
	if !ok || len(home.Topics) != 1 || home.Topics[0].ID != 42 {
		t.Fatalf("typed product data lost: %#v", model)
	}
	if home.Base.PageID != "forum.home" || home.Base.SchemaVersion != "sforum.page.home@1" {
		t.Fatalf("Host base was not authoritative: %#v", home.Base)
	}
	if homeData.Base.PageID != "" {
		t.Fatal("factory mutated the source-owned payload")
	}
}

func TestBuildCorePageViewModelOverridesHostFormBoundary(t *testing.T) {
	input := &themecompiler.TopicCreatePageViewModel{Form: themecompiler.HostFormBoundary{
		ComponentID: "plugin.fake.form", ActionRouteIDs: []string{"plugin.fake.route"},
	}}
	model, err := BuildCorePageViewModel(CorePageViewModelRequest{
		PageID: "forum.topic.create", Locale: "en-US", Path: "/topics/new",
		Data: CorePageViewModelData{TopicCreate: input},
	})
	if err != nil {
		t.Fatal(err)
	}
	created := model.(themecompiler.TopicCreatePageViewModel)
	if created.Form.ComponentID != "forum.component.topic_composer" || len(created.Form.ActionRouteIDs) != 1 || created.Form.ActionRouteIDs[0] != "core.route.forum.create_topic" {
		t.Fatalf("untrusted form boundary survived: %#v", created.Form)
	}
}

func TestBuildNotificationSettingsViewModelFreezesHostBoundary(t *testing.T) {
	input := &themecompiler.NotificationSettingsPageViewModel{Form: themecompiler.HostFormBoundary{
		ComponentID: "plugin.capture.settings", ActionRouteIDs: []string{"plugin.capture.secrets"},
	}}
	model, err := BuildCorePageViewModel(CorePageViewModelRequest{
		PageID: "forum.settings.notifications", Locale: "en-US", Path: "/settings/notifications",
		Data: CorePageViewModelData{NotificationSettings: input},
	})
	if err != nil {
		t.Fatal(err)
	}
	settings := model.(themecompiler.NotificationSettingsPageViewModel)
	wantRoutes := []string{
		"core.route.notifications.get_preferences", "core.route.notifications.update_preferences",
		"core.route.notifications.restore_preferences", "core.route.notifications.web_push_config",
		"core.route.notifications.list_web_push_subscriptions", "core.route.notifications.create_web_push_subscription",
		"core.route.notifications.revoke_web_push_subscription",
	}
	if settings.Form.ComponentID != "notifications.component.settings" || !slices.Equal(settings.Form.ActionRouteIDs, wantRoutes) {
		t.Fatalf("Host notification boundary drifted: %#v", settings.Form)
	}
	if input.Form.ComponentID != "plugin.capture.settings" {
		t.Fatal("factory mutated source-owned notification payload")
	}
}
