package forum

import (
	"strings"
	"testing"
	"time"
)

func TestCommentListCursor_RoundTrip(t *testing.T) {
	t.Parallel()
	token, err := commentCursorFromItem(Comment{ID: 7, PathKey: "0001.0002"})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeCommentListCursor(token)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ID != 7 || decoded.Path != "0001.0002" {
		t.Fatalf("decoded %#v", decoded)
	}
}

func TestTopicKeysetPredicate_PinnedFirstDimension(t *testing.T) {
	t.Parallel()
	pred, err := topicKeysetPredicate("active", 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, frag := range []string{"topics.is_pinned", "topics.last_activity_at", "topics.id"} {
		if !strings.Contains(pred, frag) {
			t.Fatalf("predicate missing %q: %s", frag, pred)
		}
	}
	args, err := topicCursorSQLArgs(topicListCursor{
		Sort: "active", Pin: 1, Key: time.Now().UTC().Format(time.RFC3339Nano), ID: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pin, ok := args[0].(bool); !ok || !pin {
		t.Fatalf("pin arg want true bool, got %#v", args[0])
	}
}
