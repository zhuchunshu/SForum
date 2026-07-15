package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func TestPublicFrontendRequiresLiveExactArtifactTrust(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	store := &memoryExecutableTrustStore{}
	exactTrust := NewExecutableTrustService(reader, store)
	service := newAdmittedPublicFrontendService(reader, exactTrust)

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
		descriptor.Entry.ContractVersion != descriptor.Entry.Handle+"@1" ||
		len(descriptor.Assets) != 1 || descriptor.Assets[0].Handle != extension.Manifest.Assets[0].Handle {
		t.Fatalf("unexpected public descriptor: %#v", descriptor)
	}
	if descriptor.Entry.Integrity == "" || descriptor.Assets[0].Integrity == "" ||
		!strings.Contains(descriptor.Entry.AssetPath, "/packages/"+extension.PackageDigest+"/frontend/public/card.mjs") {
		t.Fatalf("descriptor is not immutable/integrity-bound: %#v", descriptor)
	}
}

func TestPublicL2EntryAssetContractTracksComponentMajor(t *testing.T) {
	component := ManifestComponent{
		ID:              "demo.public.component.card",
		ContractVersion: "demo.public.component.card@2",
	}
	if got, want := publicL2EntryContractVersion(component), "demo.public.component.card.l2.entry@2"; got != want {
		t.Fatalf("entry asset contract=%q want=%q", got, want)
	}
}

func TestPublicFrontendReplaceAllIsIndependentOfFirstRequestAndCatalogOrder(t *testing.T) {
	owner := publicFrontendFixtureFor(t, "owner.assets", nil)
	ownerHandle := owner.Manifest.Assets[0].Handle
	consumer := publicFrontendFixtureFor(t, "consumer.assets", []string{ownerHandle})
	consumer.Manifest.Dependencies = []ManifestDependency{{
		ID: owner.ID, Version: "^1.0.0", Kind: "required",
	}}
	reader := &fakeFrontendExtensionReader{items: []Extension{consumer, owner}}
	trustStore := &memoryExecutableTrustStore{}
	trust := NewExecutableTrustService(reader, trustStore)
	grantPublicFrontend(t, trust, owner)
	grantPublicFrontend(t, trust, consumer)

	first := newAdmittedPublicFrontendService(reader, trust)
	descriptor, err := first.PublicComponent(t.Context(), consumer.ID, consumer.Manifest.Components[0].ID)
	if err != nil || len(descriptor.Assets) != 2 || descriptor.Assets[0].Handle != ownerHandle {
		t.Fatalf("consumer-first dependency plan=%#v err=%v", descriptor.Assets, err)
	}
	firstSnapshot := first.publicAssets.Snapshot()

	reader.items = []Extension{owner, consumer}
	restarted := newAdmittedPublicFrontendService(reader, trust)
	if _, err := restarted.PublicComponent(t.Context(), owner.ID, owner.Manifest.Components[0].ID); err != nil {
		t.Fatal(err)
	}
	if secondSnapshot := restarted.publicAssets.Snapshot(); !reflect.DeepEqual(firstSnapshot, secondSnapshot) {
		t.Fatalf("catalog order changed snapshot: first=%#v second=%#v", firstSnapshot, secondSnapshot)
	}
}

func TestPublicFrontendServesExactPackageLocalModuleCSSAndBinaryResources(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	trust := NewExecutableTrustService(reader, &memoryExecutableTrustStore{})
	service := newAdmittedPublicFrontendService(reader, trust)
	grantPublicFrontend(t, trust, extension)

	tests := []struct {
		path        string
		contentType string
	}{
		{path: "frontend/public/card.mjs", contentType: "application/javascript; charset=utf-8"},
		{path: "frontend/public/chunk.mjs", contentType: "application/javascript; charset=utf-8"},
		{path: "frontend/public/card.css", contentType: "text/css; charset=utf-8"},
		{path: "frontend/public/nested.css", contentType: "text/css; charset=utf-8"},
		{path: "frontend/public/font.woff2", contentType: "font/woff2"},
	}
	for _, test := range tests {
		asset, err := service.PublicPackageAsset(t.Context(), extension.ID, extension.PackageDigest, test.path)
		if err != nil || asset.ContentType != test.contentType || asset.Digest == "" || asset.Integrity == "" {
			t.Fatalf("%s asset=%#v err=%v", test.path, asset, err)
		}
	}
	for _, rejected := range []string{
		"frontend/public/unsafe.html", "frontend/public/unsafe.svg", "frontend/public/missing.mjs", "../card.mjs",
	} {
		if _, err := service.PublicPackageAsset(t.Context(), extension.ID, extension.PackageDigest, rejected); !errors.Is(err, ErrPublicFrontendUnavailable) {
			t.Fatalf("unsafe package resource %q error=%v", rejected, err)
		}
	}
	if _, err := service.PublicPackageAsset(
		t.Context(), extension.ID, strings.Repeat("f", 64), "frontend/public/card.mjs",
	); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("stale package resource error=%v", err)
	}
}

func TestPublicFrontendServesOnlyDeclaredImmutableBytes(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	store := &memoryExecutableTrustStore{}
	exactTrust := NewExecutableTrustService(reader, store)
	service := newAdmittedPublicFrontendService(reader, exactTrust)
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
	first := newAdmittedPublicFrontendService(reader, exactTrust)
	grantPublicFrontend(t, exactTrust, extension)
	descriptor, err := first.PublicComponent(context.Background(), extension.ID, extension.Manifest.Components[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	restarted := newAdmittedPublicFrontendService(reader, exactTrust)
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
	service := newAdmittedPublicFrontendService(reader, exactTrust)
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
	safe := newAdmittedPublicFrontendService(reader, exactTrust).WithSafeMode(true)
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

func TestPublicFrontendComponentAdmissionDefaultsClosed(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	exactTrust := NewExecutableTrustService(reader, &memoryExecutableTrustStore{})
	grantPublicFrontend(t, exactTrust, extension)
	service := NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(exactTrust, true).
		WithPublicL2(true)
	if _, err := service.PublicComponent(t.Context(), extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("public L2 must require Component Registry admission, got %v", err)
	}
}

type allowPublicComponentAdmission struct{}

func (allowPublicComponentAdmission) AdmitPublicComponent(Extension, ManifestComponent) bool {
	return true
}

func newAdmittedPublicFrontendService(
	reader FrontendExtensionReader,
	trust *ExecutableTrustService,
) *FrontendService {
	return NewFrontendService(reader, &fakeFrontendTrustStore{}).
		WithExecutableTrust(trust, true).
		WithPublicL2(true).
		WithPublicComponentAdmission(allowPublicComponentAdmission{})
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
	return publicFrontendFixtureFor(t, "demo.public", nil)
}

func publicFrontendFixtureFor(t *testing.T, id string, assetDependencies []string) Extension {
	t.Helper()
	root := t.TempDir()
	entryPath := "frontend/public/card.mjs"
	stylePath := "frontend/public/card.css"
	files := map[string][]byte{
		entryPath:                     []byte("import { marker } from './chunk.mjs'\nexport const apiVersion = 1\nexport const moduleURL = import.meta.url\nexport async function mount(target) { target.dataset.mounted = marker; return () => target.replaceChildren() }\n"),
		stylePath:                     []byte("@import './nested.css';\n.demo-l2 { font-family: demo; src: url('./font.woff2'); }\n"),
		"frontend/public/chunk.mjs":   []byte("export const marker = '1'\n"),
		"frontend/public/nested.css":  []byte(".nested { color: var(--sf-accent); }\n"),
		"frontend/public/font.woff2":  []byte("test-font"),
		"frontend/public/unsafe.html": []byte("<script>top.location='https://example.invalid'</script>"),
		"frontend/public/unsafe.svg":  []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>"),
	}
	for name, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entryBody := files[entryPath]
	styleBody := files[stylePath]
	entryDigest := bytesDigest(entryBody)
	styleDigest := bytesDigest(styleBody)
	manifest := Manifest{
		ManifestVersion: 3, ID: id, Name: "Public L2", Version: "1.0.0", Type: TypePlugin,
		PackageFiles: []ManifestPackageFile{
			{ID: id + ".file.card", Kind: "frontend", Path: entryPath, Digest: entryDigest},
			{ID: id + ".file.style", Kind: "asset", Path: stylePath, Digest: styleDigest},
			{ID: id + ".file.chunk", Kind: "frontend", Path: "frontend/public/chunk.mjs", Digest: bytesDigest(files["frontend/public/chunk.mjs"])},
			{ID: id + ".file.nested", Kind: "asset", Path: "frontend/public/nested.css", Digest: bytesDigest(files["frontend/public/nested.css"])},
			{ID: id + ".file.font", Kind: "asset", Path: "frontend/public/font.woff2", Digest: bytesDigest(files["frontend/public/font.woff2"])},
			{ID: id + ".file.html", Kind: "asset", Path: "frontend/public/unsafe.html", Digest: bytesDigest(files["frontend/public/unsafe.html"])},
			{ID: id + ".file.svg", Kind: "asset", Path: "frontend/public/unsafe.svg", Digest: bytesDigest(files["frontend/public/unsafe.svg"])},
		},
		Assets: []ManifestAsset{{
			Handle: id + ".asset.style", ContractVersion: id + ".asset.style@1",
			Type: "style", Path: stylePath, Digest: styleDigest,
			Dependencies: append([]string(nil), assetDependencies...),
			Scope:        []string{id + ".component.card"}, Loading: "blocking", CSP: []string{"connect-src 'self'"},
		}},
		Components: []ManifestComponent{{
			ID: id + ".component.card", ContractVersion: id + ".component.card@1",
			Action: "add", L2Component: id + ".file.card", PropsSchema: id + ".component.card.props@1",
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
