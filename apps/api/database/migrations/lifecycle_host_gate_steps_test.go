package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleHostGateStepsMigration = "202607140004_extension_lifecycle_host_gate_steps.sql"

func TestFilesIncludesLifecycleHostGateStepsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == lifecycleHostGateStepsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", lifecycleHostGateStepsMigration)
}

func TestLifecycleHostGateStepsMigrationWidensOnlyActionIdentity(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleHostGateStepsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle Host gate migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"DROP CONSTRAINT extension_lifecycle_steps_lifecycle_action_check",
		"ADD CONSTRAINT extension_lifecycle_steps_lifecycle_action_check",
		"'uninstall.after', 'host.gate'",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("Host gate migration Up missing %q", clause)
		}
	}
	if strings.Contains(down, "'host.gate'") || !strings.Contains(down, ")) NOT VALID") {
		t.Fatalf("Host gate migration Down must preserve rows while blocking new host.gate writes: %s", down)
	}
	for _, forbidden := range []string{"DELETE FROM", "UPDATE extension_lifecycle_steps", "DROP TABLE", "TRUNCATE"} {
		if strings.Contains(up, forbidden) || strings.Contains(down, forbidden) {
			t.Fatalf("Host gate migration mutates retained history with %q", forbidden)
		}
	}
}
