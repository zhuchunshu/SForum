package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func TestWebReleasePlannerProducesDeterministicComposition(t *testing.T) {
	theme := plannerThemeFixture(t)
	alpha := plannerPluginFixture(t, "alpha.plugin", SourceBuiltin, StatusEnabled)
	zeta := plannerPluginFixture(t, "zeta.plugin", SourceBuiltin, StatusEnabled)
	reader := &plannerExtensionReader{theme: theme, items: []Extension{zeta, alpha, theme}}
	planner := NewWebReleasePlanner(reader, &plannerGrantReader{}, plannerHostFixture())

	first, err := planner.Plan(context.Background(), PlanWebReleaseInput{TriggerKind: WebReleaseTriggerRebuild})
	if err != nil {
		t.Fatalf("plan first composition: %v", err)
	}
	slices.Reverse(reader.items)
	second, err := planner.Plan(context.Background(), PlanWebReleaseInput{TriggerKind: WebReleaseTriggerRebuild})
	if err != nil {
		t.Fatalf("plan second composition: %v", err)
	}
	if first.Hash != second.Hash || string(first.Snapshot) != string(second.Snapshot) {
		t.Fatalf("composition changed with input order:\nfirst=%s\nsecond=%s", first.Snapshot, second.Snapshot)
	}
	if got := []string{first.Composition.Extensions[0].ExtensionID, first.Composition.Extensions[1].ExtensionID}; !slices.Equal(got, []string{"alpha.plugin", "zeta.plugin"}) {
		t.Fatalf("extensions are not deterministic: %#v", got)
	}
	if first.Composition.WebSource != "source-sha" || first.Composition.WebLock != "lock-sha" || first.Composition.SDKVersion != 1 {
		t.Fatalf("host build identity is missing: %#v", first.Composition)
	}
}

func TestWebReleasePlannerAppliesTrustAndLifecycleRules(t *testing.T) {
	theme := plannerThemeFixture(t)
	trusted := plannerPluginFixture(t, "trusted.plugin", SourceUploaded, StatusEnabled)
	pending := plannerPluginFixture(t, "pending.plugin", SourceUploaded, StatusEnabled)
	untrusted := plannerPluginFixture(t, "untrusted.plugin", SourceUploaded, StatusEnabled)
	disabled := plannerPluginFixture(t, "disabled.plugin", SourceUploaded, StatusDisabled)
	grants := &plannerGrantReader{items: map[string]FrontendTrustGrant{}}
	grants.add(trusted, false)
	grants.add(pending, true)
	grants.add(disabled, false)
	planner := NewWebReleasePlanner(
		&plannerExtensionReader{theme: theme, items: []Extension{theme, pending, disabled, untrusted, trusted}},
		grants,
		plannerHostFixture(),
	)

	ordinary, err := planner.Plan(context.Background(), PlanWebReleaseInput{TriggerKind: WebReleaseTriggerRebuild})
	if err != nil {
		t.Fatalf("plan trusted composition: %v", err)
	}
	if got := plannerExtensionIDs(ordinary.Composition.Extensions); !slices.Equal(got, []string{"trusted.plugin"}) {
		t.Fatalf("unexpected trusted composition: %#v", got)
	}

	enabling, err := planner.Plan(context.Background(), PlanWebReleaseInput{
		TriggerKind:        WebReleaseTriggerPluginEnable,
		TriggerExtensionID: disabled.ID,
	})
	if err != nil {
		t.Fatalf("plan plugin enable: %v", err)
	}
	if got := plannerExtensionIDs(enabling.Composition.Extensions); !slices.Equal(got, []string{"disabled.plugin", "trusted.plugin"}) {
		t.Fatalf("enable target was not included: %#v", got)
	}

	disabling, err := planner.Plan(context.Background(), PlanWebReleaseInput{
		TriggerKind:        WebReleaseTriggerPluginDisable,
		TriggerExtensionID: trusted.ID,
	})
	if err != nil {
		t.Fatalf("plan plugin disable: %v", err)
	}
	if len(disabling.Composition.Extensions) != 0 {
		t.Fatalf("disable target remained in composition: %#v", disabling.Composition.Extensions)
	}
}

func TestWebReleasePlannerRejectsTamperedTrustedPackage(t *testing.T) {
	theme := plannerThemeFixture(t)
	plugin := plannerPluginFixture(t, "tampered.plugin", SourceUploaded, StatusEnabled)
	grants := &plannerGrantReader{items: map[string]FrontendTrustGrant{}}
	grants.add(plugin, false)
	if err := os.WriteFile(filepath.Join(plugin.PackagePath, "frontend", "admin", "components", "Cell.vue"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	planner := NewWebReleasePlanner(
		&plannerExtensionReader{theme: theme, items: []Extension{theme, plugin}},
		grants,
		plannerHostFixture(),
	)

	_, err := planner.Plan(context.Background(), PlanWebReleaseInput{TriggerKind: WebReleaseTriggerRebuild})
	if !errors.Is(err, ErrWebReleasePackageChanged) {
		t.Fatalf("expected package changed error, got %v", err)
	}
}

type plannerExtensionReader struct {
	items []Extension
	theme Extension
}

func (r *plannerExtensionReader) List(context.Context) ([]Extension, error) {
	return append([]Extension(nil), r.items...), nil
}

func (r *plannerExtensionReader) Get(_ context.Context, id string) (Extension, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return Extension{}, ErrExtensionNotFound
}

func (r *plannerExtensionReader) ActiveTheme(context.Context) (Extension, error) {
	return r.theme, nil
}

type plannerGrantReader struct {
	items map[string]FrontendTrustGrant
}

func (r *plannerGrantReader) FrontendGrant(_ context.Context, extensionID string, version string, digest string) (FrontendTrustGrant, error) {
	grant, ok := r.items[plannerGrantKey(extensionID, version, digest)]
	if !ok {
		return FrontendTrustGrant{}, ErrFrontendGrantNotFound
	}
	return grant, nil
}

func (r *plannerGrantReader) add(extension Extension, pending bool) {
	if r.items == nil {
		r.items = map[string]FrontendTrustGrant{}
	}
	grant := FrontendTrustGrant{
		ID:               int64(len(r.items) + 1),
		ExtensionID:      extension.ID,
		ExtensionVersion: extension.Version,
		PackageDigest:    extension.PackageDigest,
		APIVersion:       extension.Manifest.Frontend.Admin.APIVersion,
	}
	if pending {
		now := time.Now()
		grant.RevocationRequestedAt = &now
	}
	r.items[plannerGrantKey(extension.ID, extension.Version, extension.PackageDigest)] = grant
}

func plannerGrantKey(extensionID string, version string, digest string) string {
	return extensionID + "\x00" + version + "\x00" + digest
}

func plannerHostFixture() WebCompositionHost {
	return WebCompositionHost{
		WebSource:  "source-sha",
		WebLock:    "lock-sha",
		SDKVersion: 1,
		BunVersion: "1.3.0",
		Contract:   1,
		HostPeers: extensionpackage.HostPeers{
			"vue":               "3.5.39",
			"nuxt":              "4.4.8",
			"@nuxt/ui":          "4.9.0",
			"vue-router":        "5.1.0",
			"@sforum/admin-sdk": "1.0.0",
		},
	}
}

func plannerThemeFixture(t *testing.T) Extension {
	t.Helper()
	root := t.TempDir()
	writePlannerFile(t, root, "layer/app.vue", "<template><div /></template>")
	writePlannerFile(t, root, ManifestFileName, `{"id":"sforum.default-theme"}`)
	digest, err := extensionpackage.DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	return Extension{
		ID:            DefaultThemeID,
		Name:          "Default",
		Version:       "1.0.0",
		Type:          TypeTheme,
		Status:        StatusEnabled,
		Source:        SourceBuiltin,
		IsSystem:      true,
		IsDeletable:   false,
		PackagePath:   root,
		PackageDigest: digest,
		Manifest: Manifest{
			ID:       DefaultThemeID,
			Version:  "1.0.0",
			Type:     TypeTheme,
			Frontend: ManifestFrontend{Layer: "layer"},
		},
	}
}

func plannerPluginFixture(t *testing.T, id string, source string, status string) Extension {
	t.Helper()
	root := t.TempDir()
	components := map[string]string{"cell": "components/Cell.vue"}
	locales := map[string]string{"zh-CN": "locales/zh-CN.json", "en-US": "locales/en-US.json"}
	writePlannerFile(t, root, ManifestFileName, `{"id":"`+id+`"}`)
	writePlannerFile(t, root, "frontend/admin/components/Cell.vue", "<template><span /></template>")
	writePlannerFile(t, root, "frontend/admin/locales/zh-CN.json", `{"label":"列"}`)
	writePlannerFile(t, root, "frontend/admin/locales/en-US.json", `{"label":"Column"}`)
	writePlannerFile(t, root, "frontend/admin/package.json", plannerPackageJSON)
	writePlannerFile(t, root, "frontend/admin/bun.lock", plannerBunLock)
	digest, err := extensionpackage.DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(AdminComponentContributionPayload{Component: "cell"})
	if err != nil {
		t.Fatal(err)
	}
	return Extension{
		ID:            id,
		Name:          id,
		Version:       "1.0.0",
		Type:          TypePlugin,
		Status:        status,
		Source:        source,
		IsSystem:      source == SourceBuiltin,
		IsDeletable:   source != SourceBuiltin,
		PackagePath:   root,
		PackageDigest: digest,
		Manifest: Manifest{
			ID:      id,
			Version: "1.0.0",
			Type:    TypePlugin,
			Frontend: ManifestFrontend{Admin: &ManifestAdminFrontend{
				Root:       "frontend/admin",
				APIVersion: 1,
				Components: components,
				Locales:    locales,
			}},
			Contributions: []ManifestContribution{{
				Point:   "admin.test.fixture",
				ID:      "cell",
				Order:   10,
				Payload: payload,
			}},
		},
	}
}

func plannerExtensionIDs(items []WebExtensionSnapshot) []string {
	result := make([]string, len(items))
	for index := range items {
		result[index] = items[index].ExtensionID
	}
	return result
}

func writePlannerFile(t *testing.T, root string, relative string, body string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const plannerPackageJSON = `{
  "name": "planner-fixture",
  "private": true,
  "peerDependencies": {
    "vue": "^3.5.0",
    "nuxt": "^4.4.0",
    "@nuxt/ui": "^4.9.0",
    "vue-router": "^5.0.0",
    "@sforum/admin-sdk": "^1.0.0"
  }
}`

const plannerBunLock = `{
  "lockfileVersion": 1,
  "configVersion": 1,
  "workspaces": {
    "": {
      "name": "planner-fixture",
      "peerDependencies": {
        "vue": "^3.5.0",
        "nuxt": "^4.4.0",
        "@nuxt/ui": "^4.9.0",
        "vue-router": "^5.0.0",
        "@sforum/admin-sdk": "^1.0.0"
      }
    }
  },
  "packages": {}
}`
