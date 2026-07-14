package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleRegistryPublicationsMigration = "202607140011_extension_lifecycle_registry_publications.sql"

func TestLifecycleRegistryPublicationsMigrationPersistsAggregateFence(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleRegistryPublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle registry publication migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_lifecycle_registry_publications",
		"publication_id BIGINT NOT NULL UNIQUE",
		"REFERENCES extension_lifecycle_publications(id) ON DELETE CASCADE",
		"operation_id BIGINT NOT NULL",
		"source_runtime_instance_id TEXT",
		"source_plan_digest TEXT NOT NULL",
		"target_runtime_instance_id TEXT NOT NULL",
		"target_plan_digest TEXT NOT NULL",
		"transaction_state TEXT NOT NULL DEFAULT 'source'",
		"CHECK (transaction_state IN ('source', 'target'))",
		"transaction_state <> 'target' OR published_at IS NOT NULL",
		"transaction_state <> 'source' OR published_at IS NULL OR restored_at IS NOT NULL",
		"UNIQUE (operation_id, step_id, publication_mode)",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("lifecycle registry migration missing %q", clause)
		}
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"-- +goose StatementBegin",
		"IF EXISTS (SELECT 1 FROM extension_lifecycle_registry_publications)",
		"RAISE EXCEPTION 'cannot remove lifecycle registry publication history'",
		"-- +goose StatementEnd",
		"DROP TABLE IF EXISTS extension_lifecycle_registry_publications",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle registry migration Down missing %q", clause)
		}
	}
}
