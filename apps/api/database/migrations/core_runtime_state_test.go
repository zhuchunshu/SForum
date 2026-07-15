package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const coreRuntimeStateMigration = "202607150026_core_runtime_state.sql"

func TestCoreRuntimeStateMigrationFailsClosedUntilPublication(t *testing.T) {
	body, err := fs.ReadFile(Files(), coreRuntimeStateMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("Core runtime state migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE sforum_core_runtime_state",
		"status IN ('migrating', 'ready')",
		"status = 'ready' AND current_version <> ''",
		"target_version = current_version",
		"INSERT INTO sforum_core_runtime_state (singleton) VALUES (TRUE)",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("Core runtime state migration missing %q", clause)
		}
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"status = 'ready' OR current_version <> '' OR revision > 1",
		"RAISE EXCEPTION 'cannot remove published Core runtime version state'",
		"DROP TABLE sforum_core_runtime_state",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("Core runtime state Down missing %q", clause)
		}
	}
}
