package coreauthority

import (
	"errors"
	"strings"
	"testing"
)

func TestOwnerRoleNameIsDeterministicAndDatabaseScoped(t *testing.T) {
	first, err := OwnerRoleName("sforum")
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := OwnerRoleName("sforum")
	if err != nil {
		t.Fatal(err)
	}
	other, err := OwnerRoleName("sforum_test")
	if err != nil {
		t.Fatal(err)
	}
	if first != repeated || first == other || !strings.HasPrefix(first, ownerRolePrefix) || len(first) > postgresNameMaxBytes {
		t.Fatalf("owner roles = %q, %q, %q", first, repeated, other)
	}
}

func TestOwnerRoleNameRejectsInvalidDatabaseIdentity(t *testing.T) {
	for _, value := range []string{"", strings.Repeat("a", 64), "bad\x00name"} {
		if _, err := OwnerRoleName(value); !errors.Is(err, ErrInvalidDatabaseIdentity) {
			t.Fatalf("database identity %q error = %v", value, err)
		}
	}
}

func TestCoreAndRiverObjectClassification(t *testing.T) {
	if !IsCoreSchema(PublicSchema) || !IsCoreSchema(StableCoreViewSchema) || IsCoreSchema("river") {
		t.Fatal("Core schema classification drifted")
	}
	for _, name := range []string{"river_job", "_river_job", "river_migration"} {
		if !IsRiverObjectName(name) {
			t.Fatalf("River object %q was not excluded", name)
		}
	}
	if IsRiverObjectName("goose_db_version") || IsRiverObjectName("forum_topics") {
		t.Fatal("Host Core object was classified as River-owned")
	}
}
