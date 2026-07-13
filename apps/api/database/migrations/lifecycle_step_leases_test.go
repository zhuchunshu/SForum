package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleStepLeasesMigration = "202607140002_extension_lifecycle_step_leases.sql"

func TestFilesIncludesLifecycleStepLeasesMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == lifecycleStepLeasesMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", lifecycleStepLeasesMigration)
}

func TestLifecycleStepLeasesMigrationDefinesCASContract(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleStepLeasesMigration)
	if err != nil {
		t.Fatalf("read lifecycle step leases migration: %v", err)
	}
	sql := strings.Join(strings.Fields(string(body)), " ")
	for _, clause := range []string{
		"lease_owner_token TEXT NOT NULL DEFAULT ''",
		"lease_expires_at TIMESTAMPTZ",
		"lease_revision BIGINT NOT NULL DEFAULT 0",
		"lease_heartbeat_at TIMESTAMPTZ",
		"CHECK (lease_revision >= 0)",
		"lease_owner_token = '' AND lease_expires_at IS NULL AND lease_heartbeat_at IS NULL",
		"octet_length(lease_owner_token) BETWEEN 1 AND 512",
		"lease_revision > 0",
		"lease_owner_token = '' OR status IN ('planned', 'running', 'waiting')",
		"lease_expires_at IS NULL OR lease_expires_at > lease_heartbeat_at",
		"CREATE INDEX extension_lifecycle_steps_claimable_idx",
		"lease_expires_at NULLS FIRST, created_at, id",
		"WHERE status IN ('planned', 'running', 'waiting')",
		"CREATE INDEX extension_lifecycle_steps_lease_owner_idx",
		"WHERE lease_owner_token <> ''",
	} {
		if !strings.Contains(sql, clause) {
			t.Fatalf("lifecycle step leases migration missing %q", clause)
		}
	}
	if strings.Contains(sql, "REFERENCES audit_events") || strings.Contains(sql, "REFERENCES extensions") {
		t.Fatal("step leases must not add retention-blocking foreign keys")
	}
	claimable := sql[strings.Index(sql, "CREATE INDEX extension_lifecycle_steps_claimable_idx"):]
	claimable = strings.SplitN(claimable, "CREATE INDEX extension_lifecycle_steps_lease_owner_idx", 2)[0]
	if strings.Contains(claimable, "now()") || strings.Contains(claimable, "transaction_timestamp()") || strings.Contains(claimable, "clock_timestamp()") {
		t.Fatal("claimable index predicate must not depend on wall-clock evaluation")
	}
}

func TestLifecycleStepLeasesDownOnlyRemovesLeaseColumnsAndIndexes(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleStepLeasesMigration)
	if err != nil {
		t.Fatalf("read lifecycle step leases migration: %v", err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle step leases migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"DROP INDEX IF EXISTS extension_lifecycle_steps_lease_owner_idx",
		"DROP INDEX IF EXISTS extension_lifecycle_steps_claimable_idx",
		"DROP COLUMN IF EXISTS lease_heartbeat_at",
		"DROP COLUMN IF EXISTS lease_revision",
		"DROP COLUMN IF EXISTS lease_expires_at",
		"DROP COLUMN IF EXISTS lease_owner_token",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle step leases Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"DROP TABLE", "DELETE FROM", "TRUNCATE", "DROP COLUMN operation_id",
		"DROP COLUMN audit_event_id", "DROP COLUMN actor_user_id",
	} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle step leases Down crosses retention boundary with %q", forbidden)
		}
	}
}
