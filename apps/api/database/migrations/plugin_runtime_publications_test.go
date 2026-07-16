package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const pluginRuntimePublicationsMigration = "202607160027_plugin_runtime_publications.sql"

func TestPluginRuntimePublicationsMigrationPersistsExactFullSetConvergence(t *testing.T) {
	body, err := fs.ReadFile(Files(), pluginRuntimePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("plugin runtime publication migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE plugin_runtime_publications",
		"revision BIGSERIAL PRIMARY KEY",
		"member_count INTEGER NOT NULL CHECK (member_count >= 0)",
		"members_digest TEXT NOT NULL CHECK (members_digest ~ '^[0-9a-f]{64}$')",
		"CREATE UNIQUE INDEX extension_versions_plugin_runtime_identity_idx",
		"CREATE TABLE plugin_runtime_publication_members",
		"PRIMARY KEY (publication_revision, extension_id)",
		"REFERENCES extension_versions(id, extension_id, version, package_digest) ON DELETE RESTRICT",
		"CREATE FUNCTION enforce_plugin_runtime_member_type() RETURNS trigger",
		"stored_type IS DISTINCT FROM 'plugin'",
		"FOR NO KEY UPDATE OF e",
		"CREATE TRIGGER plugin_runtime_publication_member_type",
		"CREATE FUNCTION reject_published_plugin_type_change() RETURNS trigger",
		"CREATE TRIGGER plugin_runtime_extension_type_immutable",
		"CREATE FUNCTION reject_plugin_runtime_desired_mutation() RETURNS trigger",
		"CREATE CONSTRAINT TRIGGER plugin_runtime_publication_full_set",
		"DEFERRABLE INITIALLY DEFERRED",
		"encode(sha256(convert_to(coalesce(string_agg(",
		"ORDER BY extension_id COLLATE \"C\"",
		"CREATE TABLE plugin_runtime_nodes",
		"CHECK (process_role IN ('api', 'worker'))",
		"PRIMARY KEY (node_id, process_role, boot_id)",
		"lease_expires_at > last_seen_at",
		"CREATE FUNCTION enforce_plugin_runtime_node_monotonicity() RETURNS trigger",
		"NEW.last_applied_revision <> 0",
		"NEW.last_applied_revision < OLD.last_applied_revision",
		"NEW.first_seen_at IS DISTINCT FROM OLD.first_seen_at",
		"NEW.last_seen_at < OLD.last_seen_at",
		"CREATE TRIGGER plugin_runtime_node_monotonic",
		"CREATE TABLE plugin_runtime_publication_acks",
		"CHECK (status IN ('applying', 'applied', 'failed'))",
		"PRIMARY KEY (publication_revision, node_id, process_role, boot_id)",
		"revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0)",
		"applied_at >= started_at",
		"CREATE FUNCTION enforce_plugin_runtime_ack_cas() RETURNS trigger",
		"node_lease_expires_at <= statement_timestamp()",
		"NEW.publication_revision <= node_applied_revision",
		"NEW.revision <> OLD.revision + 1",
		"OLD.status = 'failed'",
		"OLD.status = 'applying'",
		"CREATE TABLE plugin_runtime_applied_members",
		"runtime_instance_id TEXT NOT NULL",
		"CREATE FUNCTION enforce_plugin_runtime_applied_member_lease() RETURNS trigger",
		"CREATE TRIGGER plugin_runtime_applied_member_lease",
		"REFERENCES plugin_runtime_publication_acks( publication_revision, node_id, process_role, boot_id ) ON DELETE RESTRICT",
		"REFERENCES plugin_runtime_publication_members( publication_revision, extension_id, extension_version_id, extension_version, package_digest ) ON DELETE RESTRICT",
		"CREATE FUNCTION validate_plugin_runtime_applied_full_set() RETURNS trigger",
		"actual_digest IS DISTINCT FROM desired_digest",
		"CREATE FUNCTION validate_plugin_runtime_node_progress() RETURNS trigger",
		"CREATE CONSTRAINT TRIGGER plugin_runtime_node_progress",
		"node_applied_revision IS DISTINCT FROM target_publication",
		"CREATE CONSTRAINT TRIGGER plugin_runtime_ack_applied_full_set",
		"CREATE FUNCTION notify_plugin_runtime_publication() RETURNS trigger",
		"pg_notify('sforum_plugin_runtime_publication', NEW.revision::text)",
		"CREATE TRIGGER plugin_runtime_publication_notify",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("plugin runtime publication migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"references users(id)", "on delete cascade", "safe_mode",
		"canary", "marketplace", "system_extension",
	} {
		if strings.Contains(strings.ToLower(up), forbidden) {
			t.Fatalf("plugin runtime publication migration must not contain %q", forbidden)
		}
	}
}

func TestPluginRuntimePublicationsMigrationProtectsHistoryOnDown(t *testing.T) {
	body, err := fs.ReadFile(Files(), pluginRuntimePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("plugin runtime publication migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM plugin_runtime_publications)",
		"OR EXISTS (SELECT 1 FROM plugin_runtime_publication_acks)",
		"OR EXISTS (SELECT 1 FROM plugin_runtime_applied_members)",
		"RAISE EXCEPTION 'cannot remove plugin runtime publication history'",
		"DROP TABLE IF EXISTS plugin_runtime_applied_members",
		"DROP TABLE IF EXISTS plugin_runtime_publication_acks",
		"DROP TABLE IF EXISTS plugin_runtime_nodes",
		"DROP TABLE IF EXISTS plugin_runtime_publication_members",
		"DROP TRIGGER IF EXISTS plugin_runtime_extension_type_immutable ON extensions",
		"DROP FUNCTION IF EXISTS validate_plugin_runtime_node_progress()",
		"DROP FUNCTION IF EXISTS enforce_plugin_runtime_applied_member_lease()",
		"DROP FUNCTION IF EXISTS enforce_plugin_runtime_node_monotonicity()",
		"DROP FUNCTION IF EXISTS reject_published_plugin_type_change()",
		"DROP FUNCTION IF EXISTS enforce_plugin_runtime_member_type()",
		"DROP INDEX IF EXISTS extension_versions_plugin_runtime_identity_idx",
		"DROP TABLE IF EXISTS plugin_runtime_publications",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("plugin runtime publication Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS extension_versions",
	} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("plugin runtime publication Down contains %q", forbidden)
		}
	}
}
