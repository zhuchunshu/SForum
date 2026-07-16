package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const hostInstallationIdentityMigration = "202607160032_host_installation_identity.sql"

func TestHostInstallationIdentityMigrationIsHostOwnedAndImmutable(t *testing.T) {
	body, err := fs.ReadFile(Files(), hostInstallationIdentityMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("host installation identity migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE host_installation_identity",
		"singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton)",
		"installation_id TEXT NOT NULL UNIQUE CHECK (installation_id ~ '^[0-9a-f]{64}$')",
		"CREATE FUNCTION reject_host_installation_identity_mutation() RETURNS trigger",
		"CREATE TRIGGER host_installation_identity_immutable BEFORE UPDATE OR DELETE",
		"CREATE TRIGGER host_installation_identity_no_truncate BEFORE TRUNCATE",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("host installation identity migration missing %q", clause)
		}
	}
	lower := strings.ToLower(up)
	for _, forbidden := range []string{
		"pgcrypto", "gen_random_uuid", "uuid_generate", "random()", "md5(",
		"insert into host_installation_identity",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("host installation identity migration must not contain %q", forbidden)
		}
	}
}

func TestHostInstallationIdentityMigrationDownPreservesInitializedAuthority(t *testing.T) {
	body, err := fs.ReadFile(Files(), hostInstallationIdentityMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("host installation identity migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE host_installation_identity IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM host_installation_identity)",
		"RAISE EXCEPTION 'cannot remove initialized host installation identity'",
		"DROP TABLE IF EXISTS host_installation_identity",
		"DROP FUNCTION IF EXISTS reject_host_installation_identity_mutation()",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("host installation identity Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM host_installation_identity", "TRUNCATE host_installation_identity"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("host installation identity Down contains %q", forbidden)
		}
	}
}
