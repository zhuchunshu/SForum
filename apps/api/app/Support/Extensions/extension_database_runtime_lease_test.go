package extensionsruntime

import (
	"context"
	"strings"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestExtensionDatabaseRuntimeLeaseSearchPathFollowsExactPowers(t *testing.T) {
	tests := []struct {
		name   string
		powers []string
		want   string
	}{
		{"own schema", []string{extensionmanifest.DatabaseGrantOwnSchema}, `"plugin_schema", pg_catalog`},
		{"core views", []string{extensionmanifest.DatabaseGrantCoreViews}, `"sforum_core_v1", pg_catalog`},
		{"raw core", []string{extensionmanifest.DatabaseGrantRawCore}, `"public", pg_catalog`},
		{"all", []string{
			extensionmanifest.DatabaseGrantOwnSchema, extensionmanifest.DatabaseGrantCoreViews,
			extensionmanifest.DatabaseGrantHostCommands, extensionmanifest.DatabaseGrantRawCore,
		}, `"plugin_schema", "sforum_core_v1", "public", pg_catalog`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := extensionDatabaseRuntimeLeaseSearchPath("plugin_schema", test.powers); got != test.want {
				t.Fatalf("search path = %q, want %q", got, test.want)
			}
		})
	}
}

func TestExtensionDatabaseRuntimeLeaseInputFailsClosed(t *testing.T) {
	registry := &PostgresExtensionDatabaseRegistry{}
	artifact := ExtensionDatabaseArtifact{
		ExtensionID: "demo.plugin", Version: "1.0.0", VersionID: 1,
		PackageDigest: strings.Repeat("a", 64),
	}
	valid := ExtensionDatabaseRuntimeLeaseIssue{
		Artifact: artifact, RuntimeInstanceID: "runtime-1",
		Authority: ExtensionDatabaseLeaseAuthority{
			Kind: ExtensionDatabaseLeaseIssuerActor, ActorUserID: 42, AuditEventID: 43,
		},
	}
	if err := registry.validateRuntimeLeaseIssue(context.Background(), valid); err == nil {
		t.Fatal("registry without a PostgreSQL pool accepted runtime lease issue")
	}
	for _, authority := range []ExtensionDatabaseLeaseAuthority{
		{},
		{Kind: ExtensionDatabaseLeaseIssuerActor, AuditEventID: 1},
		{Kind: ExtensionDatabaseLeaseIssuerHost, ActorUserID: 1, AuditEventID: 2},
		{Kind: "plugin", ActorUserID: 1, AuditEventID: 2},
	} {
		if validExtensionDatabaseLeaseAuthority(authority) {
			t.Fatalf("invalid lease authority accepted: %#v", authority)
		}
	}
	if !validExtensionDatabaseLeaseAuthority(ExtensionDatabaseLeaseAuthority{
		Kind: ExtensionDatabaseLeaseIssuerHost, AuditEventID: 2,
	}) {
		t.Fatal("Host lease authority was rejected")
	}
}
