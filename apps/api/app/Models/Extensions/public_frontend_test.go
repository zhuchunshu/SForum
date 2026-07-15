package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func TestPublicFrontendRequiresLiveExactArtifactTrust(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	store := &memoryExecutableTrustStore{}
	exactTrust := NewExecutableTrustService(reader, store)
	service := NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(exactTrust, true).
		WithPublicL2(true)

	if _, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("untrusted L2 component must be unavailable, got %v", err)
	}
	grantPublicFrontend(t, exactTrust, extension)
	descriptor, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.SchemaVersion != PublicFrontendSchemaV1 || descriptor.APIVersion != PublicFrontendAPIVersion ||
		descriptor.TrustNotice != PublicFrontendTrustNotice || descriptor.PackageDigest != extension.PackageDigest ||
		descriptor.Entry.Handle != extension.Manifest.Components[0].ID+publicL2EntrySuffix || !descriptor.Entry.Module ||
		len(descriptor.Assets) != 1 || descriptor.Assets[0].Handle != extension.Manifest.Assets[0].Handle {
		t.Fatalf("unexpected public descriptor: %#v", descriptor)
	}
	if descriptor.Entry.Integrity == "" || descriptor.Assets[0].Integrity == "" ||
		!strings.HasPrefix(descriptor.Entry.AssetPath, "/extensions/runtime/") {
		t.Fatalf("descriptor is not immutable/integrity-bound: %#v", descriptor)
	}
}

func TestPublicFrontendServesOnlyDeclaredImmutableBytes(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	store := &memoryExecutableTrustStore{}
	exactTrust := NewExecutableTrustService(reader, store)
	service := NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(exactTrust, true).
		WithPublicL2(true)
	grantPublicFrontend(t, exactTrust, extension)
	descriptor, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	entry, err := service.PublicAsset(context.Background(), extension.ID, extension.PackageDigest, descriptor.Entry.Digest, descriptor.Entry.Handle)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ContentType != "application/javascript; charset=utf-8" || entry.ETag != `"`+descriptor.Entry.Digest+`"` ||
		entry.Integrity != descriptor.Entry.Integrity || !strings.Contains(string(entry.Body), "export async function mount") {
		t.Fatalf("unexpected entry asset: %#v", entry)
	}
	styleRef := descriptor.Assets[0]
	style, err := service.PublicAsset(context.Background(), extension.ID, extension.PackageDigest, styleRef.Digest, styleRef.Handle)
	if err != nil || style.ContentType != "text/css; charset=utf-8" || !strings.Contains(string(style.Body), ".demo-l2") {
		t.Fatalf("style asset=%#v err=%v", style, err)
	}
	if _, err := service.PublicAsset(context.Background(), extension.ID, strings.Repeat("f", 64), descriptor.Entry.Digest, descriptor.Entry.Handle); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("stale package digest must fail, got %v", err)
	}
	if _, err := service.PublicAsset(context.Background(), extension.ID, extension.PackageDigest, strings.Repeat("f", 64), descriptor.Entry.Handle); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("stale file digest must fail, got %v", err)
	}
	if _, err := service.PublicAsset(context.Background(), extension.ID, extension.PackageDigest, descriptor.Entry.Digest, "demo.public.undeclared"); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("undeclared handle must fail, got %v", err)
	}
}

func TestPublicFrontendRebuildsAssetSnapshotAfterHostRestart(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	store := &memoryExecutableTrustStore{}
	exactTrust := NewExecutableTrustService(reader, store)
	first := NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(exactTrust, true).
		WithPublicL2(true)
	grantPublicFrontend(t, exactTrust, extension)
	descriptor, err := first.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	restarted := NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(exactTrust, true).
		WithPublicL2(true)
	asset, err := restarted.PublicAsset(context.Background(), extension.ID, extension.PackageDigest, descriptor.Entry.Digest, descriptor.Entry.Handle)
	if err != nil || len(asset.Body) == 0 || restarted.publicAssets.Snapshot().Revision != 1 {
		t.Fatalf("restart did not rebuild immutable asset snapshot: asset=%#v revision=%d err=%v", asset, restarted.publicAssets.Snapshot().Revision, err)
	}
}

func TestPublicFrontendRevocationDisableSafeModeAndByteDriftFailClosed(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	store := &memoryExecutableTrustStore{}
	exactTrust := NewExecutableTrustService(reader, store)
	service := NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(exactTrust, true).
		WithPublicL2(true)
	grantPublicFrontend(t, exactTrust, extension)
	if _, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID); err != nil {
		t.Fatal(err)
	}

	if err := exactTrust.RevokeAllForExtension(context.Background(), extension.ID, 1, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("revoked component must fail, got %v", err)
	}
	if len(service.publicAssets.Snapshot().Assets) != 0 {
		t.Fatal("revocation request did not clear process-local assets")
	}

	grantPublicFrontend(t, exactTrust, extension)
	reader.item.Status = StatusInstalled
	if _, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("disabled component must fail, got %v", err)
	}
	reader.item.Status = StatusEnabled
	safe := NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(exactTrust, true).
		WithPublicL2(true).
		WithSafeMode(true)
	if _, err := safe.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("safe mode must close L2, got %v", err)
	}

	entryPath := filepath.Join(extension.PackagePath, extension.Manifest.PackageFiles[0].Path)
	if err := os.WriteFile(entryPath, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) || !errors.Is(err, ErrFrontendPackageChanged) {
		t.Fatalf("byte drift must invalidate exact public runtime, got %v", err)
	}
}

func TestPublicFrontendGateDefaultsClosed(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	exactTrust := NewExecutableTrustService(reader, &memoryExecutableTrustStore{})
	service := NewFrontendService(reader, &fakeFrontendTrustStore{}).WithExecutableTrust(exactTrust, true)
	if _, err := service.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("public L2 migration gate must default closed, got %v", err)
	}
}

func grantPublicFrontend(t *testing.T, trust *ExecutableTrustService, extension Extension) {
	t.Helper()
	actor := frontendSuperAdmin()
	challenge, err := trust.Challenge(context.Background(), actor, extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := trust.ConfirmEnable(context.Background(), actor, extension, challenge.Token); err != nil {
		t.Fatal(err)
	}
}

func publicFrontendFixture(t *testing.T) Extension {
	t.Helper()
	root := t.TempDir()
	entryPath := "frontend/public/card.mjs"
	stylePath := "frontend/public/card.css"
	entryBody := []byte("export const apiVersion = 1\nexport async function mount(target) { target.dataset.mounted = '1'; return () => target.replaceChildren() }\n")
	styleBody := []byte(".demo-l2 { color: var(--sf-accent); }\n")
	for name, body := range map[string][]byte{entryPath: entryBody, stylePath: styleBody} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entryDigest := bytesDigest(entryBody)
	styleDigest := bytesDigest(styleBody)
	manifest := Manifest{
		ManifestVersion: 3, ID: "demo.public", Name: "Public L2", Version: "1.0.0", Type: TypePlugin,
		PackageFiles: []ManifestPackageFile{
			{ID: "demo.public.file.card", Kind: "frontend", Path: entryPath, Digest: entryDigest},
			{ID: "demo.public.file.style", Kind: "asset", Path: stylePath, Digest: styleDigest},
		},
		Assets: []ManifestAsset{{
			Handle: "demo.public.asset.style", ContractVersion: "demo.public.asset.style@1",
			Type: "style", Path: stylePath, Digest: styleDigest,
			Scope: []string{"demo.public.component.card"}, Loading: "blocking", CSP: []string{"connect-src 'self'"},
		}},
		Components: []ManifestComponent{{
			ID: "demo.public.component.card", ContractVersion: "demo.public.component.card@1",
			Action: "add", L2Component: "demo.public.file.card", PropsSchema: "demo.public.component.card.props@1",
		}},
	}
	packageDigest, err := extensionpackage.DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	return Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: manifest.Type,
		Status: StatusEnabled, Source: SourceUploaded, IsDeletable: true, Manifest: manifest,
		PackagePath: root, PackageDigest: packageDigest,
	}
}

func bytesDigest(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
