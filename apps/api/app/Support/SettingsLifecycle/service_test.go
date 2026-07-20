package settingslifecycle

import (
	"errors"
	"testing"

	secretstore "github.com/zhuchunshu/sforum/apps/api/app/Support/SecretStore"
)

func TestPutPreviewResetExportImportAndMigration(t *testing.T) {
	secrets, err := secretstore.New(secretstore.NewMemoryStore(), nil)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(secrets)
	if err := svc.RegisterSchema("demo.settings", 2, []FieldSchema{
		{Name: "mode", Type: "select", Required: true, Default: "safe", Options: []string{"safe", "advanced"}},
		{Name: "token", Type: "secret", Secret: true},
		{Name: "advanced_only", Type: "string", VisibleWhen: "mode=advanced"},
		{Name: "legacy_name", Type: "string", Default: ""},
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterMigration("demo.settings", Migration{
		From: 1, To: 2,
		Apply: func(values map[string]string) (map[string]string, error) {
			if v, ok := values["legacy_name"]; ok {
				values["mode"] = v
				delete(values, "legacy_name")
			}
			return values, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Seed v1 document by direct store then migrate on put.
	svc.mu.Lock()
	svc.documents["demo.settings"] = Document{
		SchemaVersion: SchemaVersion, ExtensionID: "demo.settings", DataVersion: 1,
		Values: map[string]string{"legacy_name": "safe"}, SecretRefs: map[string]string{}, SecretSet: map[string]bool{},
	}
	svc.mu.Unlock()

	doc, err := svc.Put("demo.settings", "admin", map[string]string{
		"mode": "safe", "token": "s3cret",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if doc.DataVersion != 2 || !doc.SecretSet["token"] || doc.SecretRefs["token"] == "" {
		t.Fatalf("doc = %#v", doc)
	}
	if _, ok := doc.Values["token"]; ok {
		t.Fatal("secret plaintext leaked into values")
	}

	preview, err := svc.Preview("demo.settings", map[string]string{"mode": "safe"})
	if err != nil || !preview.OK {
		t.Fatalf("preview = %#v err=%v", preview, err)
	}
	for _, name := range preview.VisibleFields {
		if name == "advanced_only" {
			t.Fatal("advanced_only should be hidden when mode=safe")
		}
	}
	preview2, err := svc.Preview("demo.settings", map[string]string{"mode": "advanced", "advanced_only": ""})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, name := range preview2.VisibleFields {
		if name == "advanced_only" {
			found = true
		}
	}
	if !found {
		t.Fatalf("advanced_only not visible: %#v", preview2)
	}

	// Preserve secret on empty.
	doc2, err := svc.Put("demo.settings", "admin", map[string]string{"mode": "safe", "token": ""}, true)
	if err != nil || !doc2.SecretSet["token"] {
		t.Fatalf("preserve secret = %#v err=%v", doc2, err)
	}

	exported, err := svc.Export("demo.settings")
	if err != nil || !exported.SecretsNeverIncluded || exported.Values["token"] != "" {
		t.Fatalf("export = %#v err=%v", exported, err)
	}

	reset, err := svc.ResetDefaults("demo.settings", "admin")
	if err != nil || reset.Values["mode"] != "safe" || reset.SecretSet["token"] {
		t.Fatalf("reset = %#v err=%v", reset, err)
	}

	// Import restores values + secret refs (no plaintext).
	imported, err := svc.Import("demo.settings", "admin", exported)
	if err != nil || imported.SecretRefs["token"] == "" {
		t.Fatalf("import = %#v err=%v", imported, err)
	}
	if _, err := svc.Import("demo.settings", "admin", ExportBundle{
		ExtensionID: "demo.settings", Values: map[string]string{"mode": "enc::deadbeef"},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("ciphertext import = %v", err)
	}
}

func TestActorRequired(t *testing.T) {
	svc := New(nil)
	_ = svc.RegisterSchema("x", 1, []FieldSchema{{Name: "a", Type: "string", Default: ""}})
	if _, err := svc.Put("x", "", map[string]string{"a": "1"}, true); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("put actor = %v", err)
	}
}
