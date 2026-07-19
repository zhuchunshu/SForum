package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityExternalLinksMigration = "202607190037_identity_external_links.sql"

func TestIdentityExternalLinksMigrationDefinesHostOwnedLinkPersistence(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityExternalLinksMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity external links migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE identity_external_links",
		"id BIGSERIAL PRIMARY KEY",
		"user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"provider_id TEXT NOT NULL",
		"provider_contract_version TEXT NOT NULL",
		"owner_extension_id TEXT NOT NULL",
		"owner_extension_version_id BIGINT",
		"owner_extension_version_id IS NULL OR owner_extension_version_id > 0",
		"owner_extension_version TEXT NOT NULL",
		"owner_package_digest TEXT NOT NULL",
		"owner_package_digest ~ '^[0-9a-f]{64}$'",
		"declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0)",
		"owner_extension_id ~ '^core[.]' AND owner_extension_version_id IS NULL",
		"owner_extension_id !~ '^core[.]' AND owner_extension_version_id IS NOT NULL",
		"provider_subject_digest TEXT",
		"provider_subject_digest ~ '^[0-9a-f]{64}$'",
		"status TEXT NOT NULL CHECK (status IN ('active', 'unlinked', 'erased'))",
		"revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0)",
		"linked_at TIMESTAMPTZ NOT NULL",
		"unlinked_at TIMESTAMPTZ",
		"erased_at TIMESTAMPTZ",
		"actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0)",
		"created_at TIMESTAMPTZ NOT NULL",
		"updated_at TIMESTAMPTZ NOT NULL",
		"status = 'active' AND provider_subject_digest IS NOT NULL AND unlinked_at IS NULL AND erased_at IS NULL",
		"status = 'unlinked' AND provider_subject_digest IS NOT NULL AND unlinked_at IS NOT NULL AND erased_at IS NULL",
		"status = 'erased' AND provider_subject_digest IS NULL AND erased_at IS NOT NULL",
		"CREATE UNIQUE INDEX identity_external_links_active_provider_digest_uidx",
		"ON identity_external_links (provider_id, provider_subject_digest) WHERE status = 'active'",
		"CREATE INDEX identity_external_links_user_status_provider_idx",
		"ON identity_external_links (user_id, status, provider_id)",
		"CREATE INDEX identity_external_links_owner_status_provider_idx",
		"ON identity_external_links (owner_extension_id, status, provider_id)",
		"CREATE TABLE identity_external_link_events",
		"link_id BIGINT NOT NULL CHECK (link_id > 0)",
		"provider_id TEXT NOT NULL",
		"provider_contract_version TEXT NOT NULL",
		"owner_extension_id TEXT NOT NULL",
		"action TEXT NOT NULL CHECK (action IN ('link', 'unlink', 'erase'))",
		"idempotency_key TEXT NOT NULL UNIQUE",
		"idempotency_key !~ '[^!-~]'",
		"request_fingerprint TEXT NOT NULL",
		"request_fingerprint ~ '^[0-9a-f]{64}$'",
		"previous_revision BIGINT",
		"next_revision BIGINT NOT NULL CHECK (next_revision > 0)",
		"previous_status TEXT",
		"next_status TEXT NOT NULL",
		"action = 'link' AND previous_status IS NULL AND next_status = 'active'",
		"action = 'unlink' AND previous_status = 'active' AND next_status = 'unlinked'",
		"action = 'erase' AND previous_status IN ('active', 'unlinked') AND next_status = 'erased'",
		"CREATE INDEX identity_external_link_events_link_idx",
		"CREATE INDEX identity_external_link_events_provider_idx",
		"ON identity_external_link_events (provider_id, created_at DESC, id DESC)",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity external links migration missing %q", clause)
		}
	}

	// Events must not FK current rows: privacy user deletion may remove PII
	// while redacted evidence remains.
	if strings.Contains(up, "REFERENCES identity_external_links") {
		t.Fatal("identity_external_link_events must not FK identity_external_links")
	}
	if strings.Contains(up, "REFERENCES audit_events") {
		t.Fatal("identity external links migration must not reference audit_events")
	}
	eventParts := strings.SplitN(up, "CREATE TABLE identity_external_link_events", 2)
	if len(eventParts) != 2 || strings.Contains(eventParts[1], "provider_subject_digest") {
		t.Fatal("identity external link events must retain no provider subject digest")
	}
	for _, clause := range []string{
		"provider_id TEXT NOT NULL",
		"provider_contract_version TEXT NOT NULL",
		"owner_extension_id TEXT NOT NULL",
	} {
		if !strings.Contains(eventParts[1], clause) {
			t.Fatalf("identity external link events missing redacted provenance %q", clause)
		}
	}
	// Uniqueness is provider-subject active only; do not invent user+provider.
	if strings.Contains(up, "UNIQUE (user_id, provider_id)") || strings.Contains(up, "UNIQUE (provider_id, user_id)") {
		t.Fatal("identity external links must not invent (user_id, provider_id) uniqueness")
	}

	// Forbid raw identity/session secrets while allowing provider_subject_digest.
	sanitized := strings.ReplaceAll(up, "provider_subject_digest", "")
	lower := strings.ToLower(sanitized)
	for _, forbidden := range []string{
		"provider_subject",
		"access_token",
		"refresh_token",
		"password",
		"session_id",
		"cookie",
		"secret",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("identity external links migration must not store raw %q", forbidden)
		}
	}
	if !strings.Contains(up, "provider_subject_digest") {
		t.Fatal("identity external links migration must allow provider_subject_digest")
	}
}

func TestIdentityExternalLinksMigrationDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityExternalLinksMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity external links migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE identity_external_links, identity_external_link_events IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM identity_external_link_events) OR EXISTS (SELECT 1 FROM identity_external_links)",
		"RAISE EXCEPTION 'cannot remove identity external link evidence'",
		"DROP TABLE IF EXISTS identity_external_link_events",
		"DROP TABLE IF EXISTS identity_external_links",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity external links Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity external links Down contains %q", forbidden)
		}
	}
	if strings.Contains(down, "REFERENCES audit_events") || strings.Contains(down, "audit_events") {
		// Down should not touch audit_events either.
		if strings.Contains(down, "audit_events") {
			t.Fatal("identity external links Down must not reference audit_events")
		}
	}

	// Full file: no audit_events FK anywhere.
	full := strings.Join(strings.Fields(string(body)), " ")
	if strings.Contains(full, "REFERENCES audit_events") {
		t.Fatal("identity external links migration must not FK audit_events")
	}
}
