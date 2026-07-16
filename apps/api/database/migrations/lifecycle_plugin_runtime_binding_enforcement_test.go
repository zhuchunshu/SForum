package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecyclePluginRuntimeBindingEnforcementMigration = "202607160031_lifecycle_plugin_runtime_binding_enforcement.sql"

func TestLifecyclePluginRuntimeBindingEnforcementRequiresSameCAS(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecyclePluginRuntimeBindingEnforcementMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle plugin runtime binding enforcement migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE OR REPLACE FUNCTION enforce_lifecycle_plugin_runtime_binding() RETURNS trigger",
		"TG_OP = 'INSERT'",
		"NEW.commit_marker IS TRUE AND NEW.plugin_runtime_publication_revision IS NULL",
		"new committed lifecycle marker requires plugin runtime binding",
		"OLD.commit_marker IS TRUE AND NEW.commit_marker IS NOT TRUE",
		"committed lifecycle marker is immutable",
		"OLD.commit_marker IS FALSE AND NEW.commit_marker IS TRUE AND NEW.plugin_runtime_publication_revision IS NULL",
		"OLD.commit_marker IS TRUE AND OLD.plugin_runtime_publication_revision IS NULL AND NEW.plugin_runtime_publication_revision IS NOT NULL",
		"historical lifecycle marker binding cannot be backfilled",
		"OLD.plugin_runtime_publication_revision IS NOT NULL AND NEW.plugin_runtime_publication_revision IS DISTINCT FROM OLD.plugin_runtime_publication_revision",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("lifecycle runtime binding enforcement migration missing %q", clause)
		}
	}
}

func TestLifecyclePluginRuntimeBindingEnforcementDownProtectsEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecyclePluginRuntimeBindingEnforcementMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle plugin runtime binding enforcement migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"WHERE plugin_runtime_publication_revision IS NOT NULL",
		"RAISE EXCEPTION 'cannot weaken lifecycle plugin runtime binding enforcement'",
		"CREATE OR REPLACE FUNCTION enforce_lifecycle_plugin_runtime_binding() RETURNS trigger",
		"lifecycle plugin runtime binding requires committed marker",
		"lifecycle plugin runtime binding is immutable",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle runtime binding enforcement Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP TABLE", "DROP COLUMN"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle runtime binding enforcement Down destroys evidence with %q", forbidden)
		}
	}
}
