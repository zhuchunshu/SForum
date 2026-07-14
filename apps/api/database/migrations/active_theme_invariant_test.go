package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestExactlyOneActiveThemeMigrationConvergesBeforeConstraint(t *testing.T) {
	body, err := fs.ReadFile(Files(), "202607150019_exactly_one_active_theme.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("active theme invariant migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"ROW_NUMBER() OVER ( ORDER BY (source = 'uploaded') DESC, updated_at DESC, id ASC )",
		"WHERE type = 'theme' AND status = 'enabled'",
		"SET status = 'disabled'",
		"CREATE UNIQUE INDEX extensions_one_active_theme_idx ON extensions ((type))",
		"WHERE type = 'theme' AND status = 'enabled'",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("active theme invariant migration missing %q", clause)
		}
	}
	if strings.Index(up, "SET status = 'disabled'") > strings.Index(up, "CREATE UNIQUE INDEX") {
		t.Fatal("historical active themes must converge before the unique index is created")
	}
	for _, forbidden := range []string{"DELETE FROM extensions", "DROP TABLE extensions"} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("active theme invariant migration must not use %q", forbidden)
		}
	}
}
