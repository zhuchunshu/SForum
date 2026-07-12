package providers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestRecommendedForumSettingsIncludePaginationDefaults(t *testing.T) {
	settings := recommendedForumSettings()
	if settings.TopicsPerPage != 20 || settings.CommentsPerPage != 20 {
		t.Fatalf("pagination defaults = %d/%d, want 20/20", settings.TopicsPerPage, settings.CommentsPerPage)
	}
	for _, tc := range []struct {
		value string
		want  int
		ok    bool
	}{{"1", 1, true}, {"100", 100, true}, {"0", 0, false}, {"101", 0, false}} {
		got, ok := normalizeForumPageSize(tc.value)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("normalizeForumPageSize(%q) = %d/%v, want %d/%v", tc.value, got, ok, tc.want, tc.ok)
		}
	}
}

func TestExtensionTopicActionProviderBuildsSafeDescriptors(t *testing.T) {
	payload, err := json.Marshal(extensions.TopicActionContributionPayload{
		Type:    "extensionRoute",
		Method:  "POST",
		Path:    "/topic-actions/bookmark",
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	source := fakeContributionSource{items: []extensions.EffectiveContribution{
		{
			ExtensionID:   "demo.plugin",
			ExtensionName: "Demo Plugin",
			Point:         "forum.topic.actions",
			ID:            "demo.bookmark",
			Order:         100,
			Label:         map[string]string{"zh-CN": "收藏", "en-US": "Bookmark"},
			Icon:          "i-lucide-bookmark",
			Payload:       payload,
		},
		{
			ExtensionID: "demo.plugin",
			Point:       "forum.topic.actions",
			ID:          "demo.bad",
			Payload:     json.RawMessage(`{"type":"extensionRoute","method":"GET","path":"/topic-actions/bad"}`),
		},
		{
			ExtensionID: "other.plugin",
			Point:       "admin.pages",
			ID:          "other.page",
			Payload:     payload,
		},
	}}

	actions, err := NewExtensionTopicActionProvider(source).TopicExtensionActions(context.Background())
	if err != nil {
		t.Fatalf("TopicExtensionActions returned error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected one safe action, got %#v", actions)
	}
	action := actions[0]
	if action.ExtensionID != "demo.plugin" || action.ID != "demo.bookmark" || action.URL != "/extensions/demo.plugin/topic-actions/bookmark" {
		t.Fatalf("unexpected action descriptor: %#v", action)
	}
	if action.Method != "POST" || !action.Confirm || action.Label["zh-CN"] != "收藏" || action.Icon != "i-lucide-bookmark" {
		t.Fatalf("unexpected action payload fields: %#v", action)
	}
}

func TestExtensionTopicActionProviderPropagatesSourceErrors(t *testing.T) {
	expected := errors.New("list failed")
	_, err := NewExtensionTopicActionProvider(fakeContributionSource{err: expected}).TopicExtensionActions(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected source error, got %v", err)
	}
}

func TestExtensionComposerToolbarProvider(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"type": "extensionRoute", "method": "POST", "path": "/composer/insert",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := fakeContributionSource{items: []extensions.EffectiveContribution{{
		ExtensionID: "demo.plugin",
		Point:       "forum.composer.toolbar",
		ID:          "insert",
		Order:       50,
		Label:       map[string]string{"en-US": "Insert"},
		Icon:        "i-lucide-wand",
		Payload:     payload,
	}}}
	actions, err := NewExtensionComposerToolbarProvider(source).ComposerToolbarActions(context.Background())
	if err != nil || len(actions) != 1 {
		t.Fatalf("actions=%#v err=%v", actions, err)
	}
	if actions[0].URL != "/extensions/demo.plugin/composer/insert" || actions[0].Method != "POST" {
		t.Fatalf("unexpected %#v", actions[0])
	}
}

func TestExtensionProfileTabAndDashboardProviders(t *testing.T) {
	tabPayload, _ := json.Marshal(map[string]any{"type": "hostLink", "href": "/tags"})
	widgetPayload, _ := json.Marshal(map[string]any{"type": "adminLink", "route": "/jobs", "severity": "warning"})
	source := fakeContributionSource{items: []extensions.EffectiveContribution{
		{ExtensionID: "demo.plugin", Point: "forum.profile.tabs", ID: "tags", Order: 1, Label: map[string]string{"en-US": "Tags"}, Payload: tabPayload},
		{ExtensionID: "demo.plugin", Point: "admin.dashboard.widgets", ID: "jobs", Order: 2, Label: map[string]string{"en-US": "Jobs"}, Icon: "i-lucide-gauge", Payload: widgetPayload},
		{ExtensionID: "demo.plugin", Point: "admin.dashboard.widgets", ID: "evil", Payload: json.RawMessage(`{"type":"adminLink","route":"https://evil/","severity":"info"}`)},
	}}
	tabs, err := NewExtensionProfileTabProvider(source).ProfileTabs(context.Background())
	if err != nil || len(tabs) != 1 || tabs[0].URL != "/tags" || tabs[0].Kind != "hostLink" {
		t.Fatalf("tabs=%#v err=%v", tabs, err)
	}
	widgets, err := NewExtensionDashboardWidgetProvider(source).DashboardWidgets(context.Background())
	if err != nil || len(widgets) != 1 || widgets[0].Route != "/jobs" || widgets[0].Severity != "warning" {
		t.Fatalf("widgets=%#v err=%v", widgets, err)
	}
}

type fakeContributionSource struct {
	items []extensions.EffectiveContribution
	err   error
}

func (s fakeContributionSource) EffectiveContributions(context.Context) ([]extensions.EffectiveContribution, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}
