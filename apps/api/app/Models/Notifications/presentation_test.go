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

type previewResolverFixture struct {
	preview   TargetPreview
	available bool
	err       error
	calls     *int
}

func (r previewResolverFixture) ResolveNotificationTargetPreview(context.Context, int64, string, int64) (TargetPreview, bool, error) {
	if r.calls != nil {
		*r.calls++
	}
	return r.preview, r.available, r.err
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
		if item.ActorUserID != nil || item.Actor != nil || item.TargetType != "unavailable" || item.TargetID != 0 || item.TargetAvailable || item.TargetPath != "" || string(item.Payload) != "{}" {
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

func TestResolveNotificationDetailAddsPreviewOnlyAfterVisibilityCheck(t *testing.T) {
	item := Notification{ID: 1, TargetType: "comment", TargetID: 42, Payload: json.RawMessage(`{"topicId":8}`)}
	preview := TargetPreview{TopicID: 8, TopicTitle: "Topic", Content: TargetPreviewContent{Type: "comment", ID: 42, Excerpt: "reply"}}
	detail, err := ResolveNotificationDetail(context.Background(), targetResolverFixture{available: true, path: "/t/8#comment-42"}, previewResolverFixture{preview: preview, available: true}, 7, item)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Preview == nil || detail.Preview.TopicTitle != "Topic" || detail.TargetPath != "/t/8#comment-42" {
		t.Fatalf("detail=%#v", detail)
	}

	calls := 0
	scrubbed, err := ResolveNotificationDetail(context.Background(), targetResolverFixture{available: false}, previewResolverFixture{preview: preview, available: true, calls: &calls}, 7, item)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || scrubbed.Preview != nil || scrubbed.TargetAvailable || scrubbed.TargetID != 0 || string(scrubbed.Payload) != "{}" {
		t.Fatalf("unsafe detail=%#v previewCalls=%d", scrubbed, calls)
	}
}
