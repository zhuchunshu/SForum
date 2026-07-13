package extensionmanifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsContractInventoryFixtures(t *testing.T) {
	legacy := readSettingsFixture(t, "legacy-array.json")
	var fields []ManifestSetting
	if err := json.Unmarshal(legacy, &fields); err != nil {
		t.Fatalf("decode legacy settings fixture: %v", err)
	}
	types := map[string]bool{}
	for _, field := range fields {
		types[field.Type] = true
	}
	for _, fieldType := range []string{"text", "string", "number", "boolean", "select", "secret", "textarea"} {
		if !types[fieldType] {
			t.Fatalf("legacy fixture must inventory field type %q", fieldType)
		}
	}
	hasFullWidth := false
	for _, field := range fields {
		if field.Type == "textarea" && field.Width == "full" {
			hasFullWidth = true
		}
	}
	if !hasFullWidth {
		t.Fatal("legacy fixture must inventory textarea with width=full")
	}

	for _, name := range []string{"document-tabs-actions.json", "document-component.json"} {
		var document map[string]json.RawMessage
		if err := json.Unmarshal(readSettingsFixture(t, name), &document); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		if len(document["schemaVersion"]) == 0 || len(document["ui"]) == 0 || len(document["fields"]) == 0 {
			t.Fatalf("%s must capture the versioned settings document shape", name)
		}
	}
}

func readSettingsFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "settings", name))
	if err != nil {
		t.Fatalf("read settings fixture %s: %v", name, err)
	}
	return body
}
