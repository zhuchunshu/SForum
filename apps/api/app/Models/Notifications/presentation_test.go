package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type targetResolverFixture struct {
	available bool
	path      string
	err       error
}

func (r targetResolverFixture) ResolveNotificationTarget(context.Context, int64, string, int64) (bool, string, error) {
	return r.available, r.path, r.err
}

func TestResolveSafeTargetsScrubsUnavailableAndResolverFailures(t *testing.T) {
	actorID := int64(9)
	input := Page{Items: []Notification{{
		ID: 1, ActorUserID: &actorID, TargetType: "comment", TargetID: 42,
		Payload: json.RawMessage(`{"title":"private","reviewNote":"private","route":"/private"}`),
	}}}
	for _, resolver := range []TargetVisibilityResolver{
		targetResolverFixture{available: false},
		targetResolverFixture{err: errors.New("lookup failed")},
		nil,
	} {
		page, err := ResolveSafeTargets(context.Background(), resolver, 7, input)
		if err != nil {
			t.Fatal(err)
		}
		item := page.Items[0]
		if item.ActorUserID != nil || item.TargetType != "unavailable" || item.TargetID != 0 || item.TargetAvailable || item.TargetPath != "" || string(item.Payload) != "{}" {
			t.Fatalf("unsafe fallback: %#v", item)
		}
	}
}

func TestResolveSafeTargetsUsesOnlyHostResolvedPath(t *testing.T) {
	input := Page{Items: []Notification{{ID: 1, TargetType: "comment", TargetID: 42, Payload: json.RawMessage(`{"topicId":999}`)}}}
	page, err := ResolveSafeTargets(context.Background(), targetResolverFixture{available: true, path: "/t/8#comment-42"}, 7, input)
	if err != nil {
		t.Fatal(err)
	}
	item := page.Items[0]
	if !item.TargetAvailable || item.TargetPath != "/t/8#comment-42" {
		t.Fatalf("host target not applied: %#v", item)
	}
}
