package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestProviderSlotSelectionMigrationPreservesExactAuditEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), "202607150023_provider_slot_selections.sql")
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.Join(strings.Fields(string(body)), " ")
	for _, required := range []string{
		"CREATE TABLE extension_provider_slot_selections",
		"contract_extension_version_id BIGINT NOT NULL",
		"contract_package_digest TEXT NOT NULL",
		"provider_extension_version_id BIGINT NOT NULL",
		"provider_package_digest TEXT NOT NULL",
		"CREATE TABLE extension_provider_slot_selection_events",
		"action IN ('select', 'reset', 'invalidate')",
		"cannot remove provider slot selection evidence",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("provider slot selection migration missing %q", required)
		}
	}
}
