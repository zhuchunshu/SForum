package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const hostCommandActorDelegationsMigration = "202607150024_host_command_actor_delegations.sql"

func TestHostCommandActorDelegationsPreserveExactOneUseEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), hostCommandActorDelegationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("Host Command actor delegation migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_host_command_actor_delegation_consumptions",
		"delegation_id_digest TEXT NOT NULL UNIQUE",
		"extension_version_id BIGINT NOT NULL",
		"package_digest TEXT NOT NULL",
		"runtime_epoch BIGINT NOT NULL",
		"runtime_instance_id TEXT NOT NULL",
		"actor_user_id BIGINT NOT NULL",
		"audience = 'sforum.host-command.v2'",
		"idempotency_key TEXT NOT NULL",
		"request_fingerprint TEXT NOT NULL",
		"issued_at TIMESTAMPTZ NOT NULL",
		"not_before TIMESTAMPTZ NOT NULL",
		"expires_at TIMESTAMPTZ NOT NULL",
		"consumed_at TIMESTAMPTZ NOT NULL",
		"UNIQUE (extension_id, command_id, command_version, idempotency_key)",
		"CREATE INDEX extension_host_command_actor_delegations_actor_idx",
		"CREATE INDEX extension_host_command_actor_delegations_artifact_idx",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("Host Command actor delegation migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES users(", "REFERENCES extensions(", "REFERENCES extension_versions(",
		"delegation_token", "jwt_token", "bearer_token", "ON DELETE CASCADE", "DELETE FROM", "TRUNCATE",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("Host Command actor delegation Up must not use %q", forbidden)
		}
	}
}

func TestHostCommandActorDelegationsDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), hostCommandActorDelegationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_host_command_actor_delegation_consumptions)",
		"RAISE EXCEPTION 'cannot remove Host Command actor delegation evidence'",
		"DROP TABLE IF EXISTS extension_host_command_actor_delegation_consumptions",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("Host Command actor delegation Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("Host Command actor delegation Down contains %q", forbidden)
		}
	}
}
