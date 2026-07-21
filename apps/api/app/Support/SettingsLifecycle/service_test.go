package settingslifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"

	secretstore "github.com/zhuchunshu/sforum/apps/api/app/Support/SecretStore"
)

func TestPutPreviewResetExportImportAndMigration(t *testing.T) {
	ctx := context.Background()
	secrets, err := secretstore.New(secretstore.NewMemoryStore(), nil)
	if err != nil {
		t.Fatal(err)
	}
	docs := NewMemoryDocumentStore()
	svc := NewWithStore(docs, secrets)
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

	// Seed v1 document then migrate on put.
	docs.Seed("demo.settings", Document{
		SchemaVersion: SchemaVersion, ExtensionID: "demo.settings", DataVersion: 1,
		Values: map[string]string{"legacy_name": "safe"}, SecretRefs: map[string]string{}, SecretSet: map[string]bool{},
	})

	doc, err := svc.Put(ctx, "demo.settings", "admin", map[string]string{
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

	preview, err := svc.Preview(ctx, "demo.settings", map[string]string{"mode": "safe"})
	if err != nil || !preview.OK {
		t.Fatalf("preview = %#v err=%v", preview, err)
	}
	for _, name := range preview.VisibleFields {
		if name == "advanced_only" {
			t.Fatal("advanced_only should be hidden when mode=safe")
		}
	}
	preview2, err := svc.Preview(ctx, "demo.settings", map[string]string{"mode": "advanced", "advanced_only": ""})
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
	doc2, err := svc.Put(ctx, "demo.settings", "admin", map[string]string{"mode": "safe", "token": ""}, true)
	if err != nil || !doc2.SecretSet["token"] {
		t.Fatalf("preserve secret = %#v err=%v", doc2, err)
	}

	exported, err := svc.Export(ctx, "demo.settings")
	if err != nil || !exported.SecretsNeverIncluded || exported.Values["token"] != "" {
		t.Fatalf("export = %#v err=%v", exported, err)
	}
	if exported.SecretRefs["token"] == "s3cret" {
		t.Fatal("export must not include plaintext secret")
	}

	// Reset 明确不保留密钥。
	reset, err := svc.ResetDefaults(ctx, "demo.settings", "admin", ResetOptions{PreserveSecrets: false})
	if err != nil || reset.Values["mode"] != "safe" || reset.SecretSet["token"] {
		t.Fatalf("reset clear secrets = %#v err=%v", reset, err)
	}

	// Import restores values + secret refs (no plaintext).
	imported, err := svc.Import(ctx, "demo.settings", "admin", exported)
	if err != nil || imported.SecretRefs["token"] == "" {
		t.Fatalf("import = %#v err=%v", imported, err)
	}
	if _, err := svc.Import(ctx, "demo.settings", "admin", ExportBundle{
		ExtensionID: "demo.settings", Values: map[string]string{"mode": "enc::deadbeef"},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("ciphertext import = %v", err)
	}

	// Reset 保留密钥（beginner-friendly 推荐路径）。
	if _, err := svc.Put(ctx, "demo.settings", "admin", map[string]string{"mode": "advanced", "token": "again"}, true); err != nil {
		t.Fatal(err)
	}
	kept, err := svc.ResetDefaults(ctx, "demo.settings", "admin", ResetOptions{PreserveSecrets: true})
	if err != nil || kept.Values["mode"] != "safe" || !kept.SecretSet["token"] {
		t.Fatalf("reset preserve secrets = %#v err=%v", kept, err)
	}
}

func TestFailedMigrationDoesNotPersist(t *testing.T) {
	ctx := context.Background()
	docs := NewMemoryDocumentStore()
	svc := NewWithStore(docs, nil)
	if err := svc.RegisterSchema("demo.mig", 2, []FieldSchema{
		{Name: "mode", Type: "string", Default: "a"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterMigration("demo.mig", Migration{
		From: 1, To: 2,
		Apply: func(values map[string]string) (map[string]string, error) {
			return nil, errors.New("boom")
		},
	}); err != nil {
		t.Fatal(err)
	}
	docs.Seed("demo.mig", Document{
		SchemaVersion: SchemaVersion, ExtensionID: "demo.mig", DataVersion: 1,
		Values: map[string]string{"mode": "old"}, SecretRefs: map[string]string{}, SecretSet: map[string]bool{},
	})
	if _, err := svc.Put(ctx, "demo.mig", "admin", map[string]string{"mode": "new"}, true); !errors.Is(err, ErrMigration) {
		t.Fatalf("expected migration error, got %v", err)
	}
	// 权威存储仍是 v1。
	doc, _, err := docs.Load(ctx, "demo.mig")
	if err != nil || doc.DataVersion != 1 || doc.Values["mode"] != "old" {
		t.Fatalf("store after failed migration = %#v err=%v", doc, err)
	}
}

func TestSettingsKVStoreRoundTripAndCAS(t *testing.T) {
	ctx := context.Background()
	kv := NewMemorySettingsKV()
	store, err := NewSettingsKVStore(kv)
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := secretstore.New(secretstore.NewMemoryStore(), nil)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewWithStore(store, secrets)
	if err := svc.RegisterSchema("demo.kv", 1, []FieldSchema{
		{Name: "title", Type: "string", Default: "hi"},
		{Name: "token", Type: "secret", Secret: true},
	}); err != nil {
		t.Fatal(err)
	}
	doc, err := svc.Put(ctx, "demo.kv", "admin", map[string]string{"title": "hello", "token": "s3cret"}, true)
	if err != nil {
		t.Fatal(err)
	}
	// 新 Service 实例模拟重启。
	svc2 := NewWithStore(store, secrets)
	_ = svc2.RegisterSchema("demo.kv", 1, []FieldSchema{
		{Name: "title", Type: "string", Default: "hi"},
		{Name: "token", Type: "secret", Secret: true},
	})
	got, err := svc2.Get(ctx, "demo.kv")
	if err != nil || got.Values["title"] != "hello" || got.SecretRefs["token"] != doc.SecretRefs["token"] {
		t.Fatalf("restart get = %#v err=%v", got, err)
	}
	// CAS conflict.
	_, rev, err := store.Load(ctx, "demo.kv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(ctx, "demo.kv", got, rev); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(ctx, "demo.kv", got, rev); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale CAS = %v", err)
	}

	// 并发 Put：最终有一致 revision，无 panic。
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			_, _ = svc2.Put(ctx, "demo.kv", "admin", map[string]string{"title": "t"}, true)
		}(i)
	}
	wait.Wait()
	final, err := svc2.Get(ctx, "demo.kv")
	if err != nil || final.Values["title"] == "" {
		t.Fatalf("after concurrent put = %#v err=%v", final, err)
	}
}

func TestActorRequired(t *testing.T) {
	ctx := context.Background()
	svc := New(nil)
	_ = svc.RegisterSchema("x", 1, []FieldSchema{{Name: "a", Type: "string", Default: ""}})
	if _, err := svc.Put(ctx, "x", "", map[string]string{"a": "1"}, true); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("put actor = %v", err)
	}
	if _, err := svc.Put(nil, "x", "admin", map[string]string{"a": "1"}, true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil ctx = %v", err)
	}
}
