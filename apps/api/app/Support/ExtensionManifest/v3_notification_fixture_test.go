package extensionmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNotificationReferenceFixtureFreezesNamespacedInertType(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve fixture test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../extensions/fixtures/plugins/sforum-notification-reference"))
	templateBody, err := os.ReadFile(filepath.Join(root, "sforum.extension.json.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	schemaBody, err := os.ReadFile(filepath.Join(root, "schemas/order-ready.json"))
	if err != nil {
		t.Fatal(err)
	}
	binaryBody := []byte("notification-reference-fixture-binary")
	binaryDigest := sha256.Sum256(binaryBody)
	schemaDigest := sha256.Sum256(schemaBody)
	manifestBody := strings.ReplaceAll(string(templateBody), "__BACKEND_DIGEST__", hex.EncodeToString(binaryDigest[:]))
	manifestBody = strings.ReplaceAll(manifestBody, "__PAYLOAD_SCHEMA_DIGEST__", hex.EncodeToString(schemaDigest[:]))
	pkg := FileMapFS{
		ManifestFileName: []byte(manifestBody), "backend/plugin": binaryBody,
		"schemas/order-ready.json": schemaBody,
	}
	manifest, err := LoadRootBytes([]byte(manifestBody), pkg)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "sforum.notification-reference" || len(manifest.NotificationTypes) != 1 {
		t.Fatalf("notification fixture manifest = %#v", manifest)
	}
	declaration := manifest.NotificationTypes[0]
	if declaration.ID != "sforum.notification-reference.order_ready" || declaration.Required ||
		declaration.PayloadSchema != "sforum.notification-reference.schema.order-ready@1" ||
		!strings.HasPrefix(declaration.ID, manifest.ID+".") {
		t.Fatalf("notification declaration = %#v", declaration)
	}
	if !HasDatabaseGrant(manifest.Database, DatabaseGrantHostCommands) || len(manifest.Database.Grants) != 1 {
		t.Fatalf("notification fixture must request only the Host Command grant: %#v", manifest.Database.Grants)
	}
}

func TestNotificationTypeDeclarationRejectsRequiredAndForeignNamespace(t *testing.T) {
	base := Manifest{
		ManifestVersion: ManifestVersionV3, ID: "fixture.notifications", Name: "Notifications",
		Description: "Notification declaration validation fixture.", URL: "https://example.com",
		Author: ManifestAuthor{Name: "SForum"}, Version: "1.0.0", Type: TypePlugin, SForumVersion: ">=0.1.0",
		NotificationTypes: []ManifestNotificationType{{
			ID: "fixture.notifications.notice", ContractVersion: "fixture.notifications.notice@1",
			PayloadVersion: 1, Category: "system", PayloadSchema: "fixture.notifications.schema.notice@1",
			Label: LocalizedText{Default: "Notice"}, Body: LocalizedText{Default: "Notice body"},
			TargetKind: "none", Channels: []string{"in_app"},
		}},
		PackageFiles: []ManifestPackageFile{{ID: "fixture.notifications.schema.notice", Kind: "schema", Path: "schemas/notice.json", Digest: strings.Repeat("a", 64), Version: "1"}},
	}
	for _, mutate := range []func(*Manifest){
		func(value *Manifest) { value.NotificationTypes[0].Required = true },
		func(value *Manifest) {
			value.NotificationTypes[0].ID = "other.notifications.notice"
			value.NotificationTypes[0].ContractVersion = "other.notifications.notice@1"
		},
	} {
		candidate := base
		candidate.NotificationTypes = append([]ManifestNotificationType(nil), base.NotificationTypes...)
		mutate(&candidate)
		if err := Validate(candidate); err == nil {
			t.Fatal("unsafe notification declaration accepted")
		}
	}
}

func TestWebPushBuiltinManifestLoadsWithExactPackageFiles(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve fixture test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../storage/builtin-dev/plugins/sforum-web-push"))
	if _, err := os.Stat(filepath.Join(root, "backend/plugin")); err != nil {
		t.Skip("built-in staging artifact has not been built")
	}
	manifest, err := LoadPackage(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Providers) != 1 || manifest.Providers[0].Slot != "notification.channel.web_push" {
		t.Fatalf("Web Push provider declaration = %#v", manifest.Providers)
	}
}
