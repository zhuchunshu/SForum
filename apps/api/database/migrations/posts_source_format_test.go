package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const postsDropLegacyJSONSourceFormatMigration = "202607230054_posts_drop_legacy_json_source_format.sql"

func TestPostsDropLegacyJSONSourceFormatMigrationKeepsOnlyPublishedFormats(t *testing.T) {
	body, err := fs.ReadFile(Files(), postsDropLegacyJSONSourceFormatMigration)
	if err != nil {
		t.Fatal(err)
	}
	up := strings.SplitN(string(body), "-- +goose Down", 2)[0]
	normalized := strings.Join(strings.Fields(up), " ")

	for _, format := range []string{"'markdown'::text", "'html'::text", "'editor-document'::text"} {
		if !strings.Contains(normalized, format) {
			t.Fatalf("source format migration missing %s", format)
		}
	}
	if strings.Contains(normalized, "'json'::text") {
		t.Fatal("legacy json source format must not remain in the Up constraint")
	}
	if !strings.Contains(normalized, "IF EXISTS (SELECT 1 FROM posts WHERE source_format = 'json')") {
		t.Fatal("migration must fail explicitly instead of guessing how to convert legacy json rows")
	}
}
