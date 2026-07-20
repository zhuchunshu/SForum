package extensions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestP9JoinedPublicL2TrustMatrix covers the public L2 half of the P9 joined
// production matrix: admit → page CSP policy → revoke → byte-drift quarantine →
// Safe Mode fail-closed. Mount/unmount/CSS cleanup live in Nuxt visual tests.
// Digest-upgrade selection is covered by the Host component matrix (package-local
// SSR path); this row uses real fixture bytes only — no synthetic digests.
func TestP9JoinedPublicL2TrustMatrix(t *testing.T) {
	ctx := context.Background()
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	store := &memoryExecutableTrustStore{}
	trust := NewExecutableTrustService(reader, store)
	service := newAdmittedPublicFrontendService(reader, trust)
	trust.WithPublicAssetRegistry(service.publicAssets)

	// --- 1) Admit: exact grant + publication + descriptor ---
	grantPublicFrontend(t, trust, extension)
	publishTrustedPublicAssets(t, service, extension)
	descriptor, err := service.PublicComponent(ctx, extension.ID, extension.Manifest.Components[0].ID)
	if err != nil {
		t.Fatalf("admit PublicComponent: %v", err)
	}
	if descriptor.PackageDigest != extension.PackageDigest || descriptor.TrustNotice != PublicFrontendTrustNotice {
		t.Fatalf("descriptor=%#v", descriptor)
	}

	// page policy aggregates admitted soft refs into Host CSP document policy
	policy, err := service.PublicPagePolicyForComponents(ctx, []PublicFrontendComponentRef{{
		ExtensionID: extension.ID, ComponentID: extension.Manifest.Components[0].ID,
	}})
	if err != nil || policy.DocumentPolicy.HeaderValue == "" {
		t.Fatalf("page policy: %#v err=%v", policy, err)
	}
	if !strings.Contains(policy.DocumentPolicy.HeaderValue, "script-src 'self'") {
		t.Fatalf("host baseline missing: %q", policy.DocumentPolicy.HeaderValue)
	}
	if len(policy.AdmittedComponents) != 1 {
		t.Fatalf("admitted components: %#v", policy.AdmittedComponents)
	}

	// --- 2) Trust revoke: descriptor + assets + soft-ref policy all fail closed ---
	if err := trust.RevokeAllForExtension(ctx, extension.ID, 1, "p9-matrix"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PublicComponent(ctx, extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("revoked component: %v", err)
	}
	if len(service.publicAssets.Snapshot().Assets) != 0 {
		t.Fatal("revoke left assets published")
	}
	if _, err := service.PublicPagePolicyForComponents(ctx, []PublicFrontendComponentRef{{
		ExtensionID: extension.ID, ComponentID: extension.Manifest.Components[0].ID,
	}}); err == nil {
		t.Fatal("page policy must fail after revoke")
	}

	// --- 3) Byte drift: re-admit, then mutate package file → PublicAsset quarantine ---
	grantPublicFrontend(t, trust, extension)
	publishTrustedPublicAssets(t, service, extension)
	live, err := service.PublicComponent(ctx, extension.ID, extension.Manifest.Components[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(extension.PackagePath, extension.Manifest.PackageFiles[0].Path)
	if err := os.WriteFile(entryPath, []byte("changed-for-p9-matrix"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PublicAsset(
		ctx, extension.ID, extension.PackageDigest, live.Entry.Digest, live.Entry.Handle,
	); !errors.Is(err, ErrPublicFrontendUnavailable) || !errors.Is(err, ErrFrontendPackageChanged) {
		t.Fatalf("byte drift must invalidate exact public runtime, got %v", err)
	}
	if _, ok := service.publicAssets.SnapshotPublication(extension.ID); ok {
		t.Fatal("byte drift must quarantine exact publication")
	}

	// --- 4) Safe Mode: L2 component + page policy gates fail closed ---
	// 重新用干净 fixture 发布，再打开 Safe Mode（避免漂移文件干扰 restore）。
	extension = publicFrontendFixture(t)
	reader.item = extension
	grantPublicFrontend(t, trust, extension)
	publishTrustedPublicAssets(t, service, extension)
	safe := newAdmittedPublicFrontendService(reader, trust).
		WithPublicAssetRegistry(service.PublicAssetRegistry()).
		WithSafeMode(true)
	if err := safe.RestorePublicAssetPublications(ctx, []Extension{extension}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := safe.PublicComponent(ctx, extension.ID, extension.Manifest.Components[0].ID); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("Safe Mode must close L2: %v", err)
	}
	// 基线（nil soft refs）同样走 publicPagePolicyGates；Safe Mode 必须整页失败关闭。
	if _, err := safe.PublicPagePolicyForComponents(ctx, nil); err == nil {
		t.Fatal("Safe Mode baseline page policy must fail closed when public L2 gates fail")
	}
}
