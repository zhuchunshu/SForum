package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestRouteProviderSelectionMigrationPreservesExactAuditEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), "202607140018_route_provider_selections.sql")
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.Join(strings.Fields(string(body)), " ")
	for _, required := range []string{
		"CREATE TABLE extension_route_provider_selections",
		"PRIMARY KEY (target_route_id, method, path_signature)",
		"provider_extension_version_id BIGINT NOT NULL",
		"provider_package_digest TEXT NOT NULL",
		"CREATE TABLE extension_route_provider_selection_events",
		"action IN ('select', 'reset', 'invalidate')",
		"cannot remove route provider selection evidence",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("route provider selection migration missing %q", required)
		}
	}
}
