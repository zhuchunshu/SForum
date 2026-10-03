package forum

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCommentListCursor_RoundTrip(t *testing.T) {
	t.Parallel()
	publishedAt := time.Date(2026, 10, 2, 18, 13, 16, 246234000, time.UTC)
	token, err := commentCursorFromItem(Comment{ID: 7, CreatedAt: publishedAt})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeCommentListCursor(token)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ID != 7 || decoded.Key != publishedAt.Format(time.RFC3339Nano) {
		t.Fatalf("decoded %#v", decoded)
	}
	key, err := commentCursorKeyTime(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !key.Equal(publishedAt) {
		t.Fatalf("key = %s, want %s", key, publishedAt)
	}
}

// 旧格式（path_key 载荷）必须被拒绝：客户端拿到 ErrInvalidCursor 后重新取第一页，
// 不能把字典序键当成时间键继续翻页。
func TestCommentListCursor_RejectsLegacyPathPayload(t *testing.T) {
	t.Parallel()
	legacy := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"pk":"0001.0002","i":7}`))
	if _, err := decodeCommentListCursor(legacy); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("decode legacy cursor err = %v, want ErrInvalidCursor", err)
	}
}

func TestCommentCursorFromItem_RequiresCreatedAt(t *testing.T) {
	t.Parallel()
	if _, err := commentCursorFromItem(Comment{ID: 7}); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("err = %v, want ErrInvalidCursor", err)
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
