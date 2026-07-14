package extensionsruntime

import (
	"strings"
	"testing"
)

func TestExtensionDatabaseIdentifiersAreDeterministicSafeAndNamespaced(t *testing.T) {
	first, err := ExtensionDatabaseIdentifiersFor("vendor.really-long_plugin-name.with-segments")
	if err != nil {
		t.Fatalf("derive identifiers: %v", err)
	}
	second, err := ExtensionDatabaseIdentifiersFor("vendor.really-long_plugin-name.with-segments")
	if err != nil {
		t.Fatalf("derive identifiers again: %v", err)
	}
	if first != second {
		t.Fatalf("identifier derivation is not deterministic: first=%#v second=%#v", first, second)
	}
	for label, value := range map[string]string{
		"schema": first.Schema, "owner": first.OwnerRole, "runtime": first.RuntimeRole,
	} {
		if !strings.HasPrefix(value, extensionDatabaseNamespace+"_") || !validPostgresIdentifier(value) {
			t.Fatalf("%s identifier is unsafe: %q", label, value)
		}
		if len(value) > postgresIdentifierMaximumBytes {
			t.Fatalf("%s identifier exceeds PostgreSQL limit: %d", label, len(value))
		}
	}
	if first.Schema == first.OwnerRole || first.Schema == first.RuntimeRole || first.OwnerRole == first.RuntimeRole {
		t.Fatalf("physical identities collide: %#v", first)
	}
}

func TestExtensionDatabaseIdentifiersKeepCollisionResistantSuffix(t *testing.T) {
	left, err := ExtensionDatabaseIdentifiersFor("vendor.plugin-name")
	if err != nil {
		t.Fatal(err)
	}
	right, err := ExtensionDatabaseIdentifiersFor("vendor.plugin_name")
	if err != nil {
		t.Fatal(err)
	}
	if extensionDatabaseSlug("vendor.plugin-name", extensionDatabaseSlugBytes) !=
		extensionDatabaseSlug("vendor.plugin_name", extensionDatabaseSlugBytes) {
		t.Fatal("test inputs must normalize to the same readable slug")
	}
	if left.Schema == right.Schema || left.OwnerRole == right.OwnerRole || left.LockKey == right.LockKey {
		t.Fatalf("hash suffix did not disambiguate normalized ids: left=%#v right=%#v", left, right)
	}
}

func TestExtensionDatabaseMigrationRoleBindsExactPlan(t *testing.T) {
	firstDigest := strings.Repeat("a", 64)
	secondDigest := strings.Repeat("b", 64)
	first, err := ExtensionDatabaseMigrationRoleFor("vendor.plugin", firstDigest)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ExtensionDatabaseMigrationRoleFor("vendor.plugin", firstDigest)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ExtensionDatabaseMigrationRoleFor("vendor.plugin", secondDigest)
	if err != nil {
		t.Fatal(err)
	}
	if first != again || first == second || !validPostgresIdentifier(first) {
		t.Fatalf("migration role is not exact-plan deterministic: first=%q again=%q second=%q", first, again, second)
	}
}

func TestExtensionDatabaseRuntimeLeaseRoleBindsExactRuntimeAndLease(t *testing.T) {
	leaseID := strings.Repeat("a", 64)
	first, err := ExtensionDatabaseRuntimeLeaseRoleFor("vendor.plugin", "runtime-1", leaseID)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ExtensionDatabaseRuntimeLeaseRoleFor("vendor.plugin", "runtime-1", leaseID)
	if err != nil {
		t.Fatal(err)
	}
	otherRuntime, err := ExtensionDatabaseRuntimeLeaseRoleFor("vendor.plugin", "runtime-2", leaseID)
	if err != nil {
		t.Fatal(err)
	}
	otherLease, err := ExtensionDatabaseRuntimeLeaseRoleFor("vendor.plugin", "runtime-1", strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	if first != again || first == otherRuntime || first == otherLease || !validPostgresIdentifier(first) {
		t.Fatalf("runtime lease role is not exact: first=%q again=%q runtime=%q lease=%q", first, again, otherRuntime, otherLease)
	}
}

func TestExtensionDatabaseIdentifiersRejectInvalidInputs(t *testing.T) {
	for _, value := range []string{"", "a", "Vendor.Plugin", "vendor/plugin", " vendor.plugin", strings.Repeat("a", 82)} {
		if _, err := ExtensionDatabaseIdentifiersFor(value); err == nil {
			t.Fatalf("expected invalid extension id %q to fail", value)
		}
	}
	if _, err := ExtensionDatabaseMigrationRoleFor("vendor.plugin", "not-a-digest"); err == nil {
		t.Fatal("invalid plan digest must fail")
	}
	for _, input := range []struct {
		runtime string
		lease   string
	}{
		{"", strings.Repeat("a", 64)},
		{" runtime", strings.Repeat("a", 64)},
		{strings.Repeat("r", 513), strings.Repeat("a", 64)},
		{"runtime", "not-a-digest"},
	} {
		if _, err := ExtensionDatabaseRuntimeLeaseRoleFor("vendor.plugin", input.runtime, input.lease); err == nil {
			t.Fatalf("invalid runtime lease role input %#v must fail", input)
		}
	}
}
