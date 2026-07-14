package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const hostCommandReceiptsMigration = "202607140017_host_command_receipts.sql"

func TestFilesIncludesHostCommandReceiptsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == hostCommandReceiptsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", hostCommandReceiptsMigration)
}

func TestHostCommandReceiptsPreserveExactReplayEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), hostCommandReceiptsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("Host Command receipts migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_host_command_receipts",
		"extension_version_id BIGINT NOT NULL",
		"package_digest TEXT NOT NULL",
		"authority_type TEXT NOT NULL",
		"REFERENCES extension_trust_grants(id) ON DELETE RESTRICT",
		"command_id TEXT NOT NULL",
		"command_version TEXT NOT NULL",
		"idempotency_key TEXT NOT NULL",
		"request_fingerprint TEXT NOT NULL",
		"result JSONB NOT NULL",
		"transaction_id TEXT NOT NULL UNIQUE",
		"audit_event_id BIGINT NOT NULL UNIQUE CHECK (audit_event_id > 0)",
		"UNIQUE (extension_id, command_id, command_version, idempotency_key)",
		"created_at TIMESTAMPTZ NOT NULL",
		"committed_at TIMESTAMPTZ NOT NULL",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("Host Command receipts migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES extensions(", "REFERENCES extension_versions(", "REFERENCES audit_events(",
		"ON DELETE CASCADE", "DELETE FROM", "TRUNCATE",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("Host Command receipts Up must not use %q", forbidden)
		}
	}
}

func TestHostCommandReceiptsDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), hostCommandReceiptsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_host_command_receipts)",
		"RAISE EXCEPTION 'cannot remove Host Command receipt evidence'",
		"DROP TABLE IF EXISTS extension_host_command_receipts",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("Host Command receipts Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("Host Command receipts Down contains %q", forbidden)
		}
	}
}
