package providers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

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
