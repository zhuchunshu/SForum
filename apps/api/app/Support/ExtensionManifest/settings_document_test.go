package extensionmanifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestSettingsDocumentLegacyArrayNormalizesAndStaysCanonical(t *testing.T) {
	body := manifestWithSettings(t, readSettingsFixture(t, "legacy-array.json"))
	manifest, err := LoadRootBytes(body, settingsTestPackageFS())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SettingsDocument.Explicit || manifest.SettingsDocument.SchemaVersion != 1 || manifest.SettingsDocument.UI.Layout != SettingsLayoutForm {
		t.Fatalf("unexpected normalized legacy document: %#v", manifest.SettingsDocument)
	}
	canonical, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	_ = json.Unmarshal(canonical, &object)
	if len(object["settings"]) == 0 || object["settings"][0] != '[' {
		t.Fatalf("legacy canonical settings must remain an array: %s", object["settings"])
	}
}

func TestSettingsDocumentObjectLoadsAndStaysCanonical(t *testing.T) {
	body := manifestWithSettings(t, readSettingsFixture(t, "document-tabs-actions.json"))
	manifest, err := LoadRootBytes(body, settingsTestPackageFS())
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.SettingsDocument.Explicit || manifest.SettingsDocument.UI.Layout != SettingsLayoutTabs || len(manifest.SettingsDocument.Actions) != 1 {
		t.Fatalf("unexpected settings document: %#v", manifest.SettingsDocument)
	}
	canonical, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	_ = json.Unmarshal(canonical, &object)
	if len(object["settings"]) == 0 || object["settings"][0] != '{' {
		t.Fatalf("document canonical settings must remain an object: %s", object["settings"])
	}
}

func TestSettingsDocumentRejectsInvalidPresentation(t *testing.T) {
	cases := []string{
		`{"schemaVersion":2,"ui":{"mode":"schema","layout":"form"},"fields":[{"key":"x","label":"X","type":"text"}]}`,
		`{"schemaVersion":1,"ui":{"mode":"schema","layout":"tabs","tabs":[{"id":"a","label":"A","groups":["missing"]}]},"fields":[{"key":"x","label":"X","type":"text"}]}`,
		`{"schemaVersion":1,"ui":{"mode":"schema","layout":"form"},"fields":[{"key":"x","label":"X","type":"text"}],"actions":[{"id":"x","kind":"remote_url","label":"X"}]}`,
		`{"schemaVersion":1,"ui":{"mode":"component","layout":"form","component":{"id":"x","apiVersion":1,"entry":"https://example.com/x.mjs"}},"fields":[{"key":"x","label":"X","type":"text"}]}`,
	}
	for _, settings := range cases {
		if _, err := LoadRootBytes(manifestWithSettings(t, []byte(settings)), settingsTestPackageFS()); !errors.Is(err, ErrInvalidManifest) {
			t.Fatalf("expected invalid settings document for %s, got %v", settings, err)
		}
	}
}

func TestSettingsDocumentRejectsExecutableManifestContribution(t *testing.T) {
	manifest := validBaseManifest()
	manifest.Settings = []ManifestSetting{{Key: "x", Label: LocalizedText{Default: "X"}, Type: "text"}}
	manifest.SettingsDocument = defaultSettingsDocument(manifest.Settings)
	manifest.SettingsDocument.Explicit = true
	manifest.Contributions = []ManifestContribution{{
		Point: "admin.extension.settings.page", ID: "settings", Payload: json.RawMessage(`{"component":"settings"}`),
	}}
	if err := Validate(Normalize(manifest)); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected executable contribution rejection, got %v", err)
	}
}

func manifestWithSettings(t *testing.T, settings []byte) []byte {
	t.Helper()
	base := validBaseManifest()
	if bytes.Contains(settings, []byte(`"provider_probe"`)) {
		const emptyDigest = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		base.Backend = ManifestBackend{
			Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 2,
			Digest: emptyDigest, HostAPIVersion: "sforum.host@2",
		}
		base.PackageFiles = []ManifestPackageFile{{
			ID: "demo.plugin.file.backend", Kind: "executable", Path: "backend/plugin", Digest: emptyDigest,
		}}
		base.Providers = []ManifestProvider{{
			ID: "demo.plugin.provider.mail", ContractVersion: "demo.plugin.provider.mail@1",
			Slot: "mail.provider", Label: "Mail", Handler: "mail.handle",
		}}
	}
	body, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	_ = json.Unmarshal(body, &object)
	object["settings"] = append(json.RawMessage(nil), settings...)
	body, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func settingsTestPackageFS() FileMapFS {
	return FileMapFS{"backend/plugin": []byte{}}
}
