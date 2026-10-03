package forum

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPublicContentQueriesDoNotSelectCanonicalSource(t *testing.T) {
	for name, query := range map[string]string{
		"topic detail": topicDetailSQL(),
		"comments":     commentSelectSQL(),
	} {
		for _, forbidden := range []string{"posts.raw_content", "posts.source_format", "posts.editor_type", "posts.editor_version", "posts.content_hash"} {
			if strings.Contains(query, forbidden) {
				t.Fatalf("%s query selects %q: %s", name, forbidden, query)
			}
		}
	}
}

func TestPublicContentJSONHasNoEditableSourceFields(t *testing.T) {
	payload, err := json.Marshal(struct {
		Topic    TopicDetail `json:"topic"`
		Comments CommentList `json:"comments"`
	}{
		Topic: TopicDetail{Content: PublicRenderedContent{
			ID: 1, HTMLContent: "<p>public</p>", PlainText: "public", Excerpt: "public", RenderVersion: RenderVersion,
		}},
		Comments: CommentList{Items: []Comment{{Content: PublicRenderedContent{
			ID: 2, HTMLContent: "<p>reply</p>", PlainText: "reply", Excerpt: "reply", RenderVersion: RenderVersion,
		}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"rawContent", "contentHash", "sourceFormat", "editorType", "editorVersion", "attachmentIds"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("public JSON contains %q: %s", forbidden, payload)
		}
	}
}
