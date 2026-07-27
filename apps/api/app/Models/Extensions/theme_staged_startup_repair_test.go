package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Host 收紧宿主岛 allowlist 后，活动内置主题包可能无法预检，而 SyncBuiltins
// 只 stage 新包。启动恢复必须晋升 staged 并写入 theme runtime publication。
func TestRestoreActiveThemeRegistryPromotesStagedBuiltinAfterPreflightFailure(t *testing.T) {
	badDigest := strings.Repeat("a", 64)
	goodDigest := strings.Repeat("b", 64)

	theme := protectedBuiltinExtension(DefaultThemeID, TypeTheme)
	theme.PackageDigest = badDigest
	theme.PackagePath = "/tmp/theme-bad"
	theme.StagedVersion = &ExtensionVersion{
		ID:            2,
		Version:       theme.Version,
		Manifest:      theme.Manifest,
		PackageDigest: goodDigest,
		PackagePath:   "/tmp/theme-good",
	}

	store := &fakeExtensionStore{
		items:         map[string]Extension{theme.ID: theme},
		activeThemeID: theme.ID,
	}
	registry := &digestAwarePageRegistry{badDigest: badDigest}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", &countingRuntimeManager{}, WithPageRegistry(registry),
	)

	if err := service.RestoreActiveThemeRegistry(context.Background()); err != nil {
		t.Fatalf("restore: %v", err)
	}

	active, err := store.ActiveTheme(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(active.PackageDigest, goodDigest) {
		t.Fatalf("active digest=%q want=%q", active.PackageDigest, goodDigest)
	}
	if active.StagedVersion != nil {
		t.Fatalf("staged version should be cleared after promote: %#v", active.StagedVersion)
	}
	if registry.registeredDigest != goodDigest {
		t.Fatalf("registered digest=%q want=%q", registry.registeredDigest, goodDigest)
	}
	if store.activateThemeExactCalls != 1 {
		t.Fatalf("ActivateThemeExact calls=%d want=1", store.activateThemeExactCalls)
	}
	if store.latestThemePublication.PackageDigest != goodDigest ||
		store.latestThemePublication.ThemeID != DefaultThemeID {
		t.Fatalf("publication=%#v", store.latestThemePublication)
	}
}

func TestFailClosedThemeRuntimePromotesStagedDefaultWhenActiveBroken(t *testing.T) {
	badDigest := strings.Repeat("c", 64)
	goodDigest := strings.Repeat("d", 64)

	theme := protectedBuiltinExtension(DefaultThemeID, TypeTheme)
	theme.PackageDigest = badDigest
	theme.StagedVersion = &ExtensionVersion{
		ID: 3, Version: theme.Version, Manifest: theme.Manifest,
		PackageDigest: goodDigest, PackagePath: "/tmp/theme-good",
	}
	store := &fakeExtensionStore{
		items:         map[string]Extension{theme.ID: theme},
		activeThemeID: theme.ID,
	}
	registry := &digestAwarePageRegistry{badDigest: badDigest}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", &countingRuntimeManager{}, WithPageRegistry(registry),
	)

	if err := service.FailClosedThemeRuntime(context.Background()); err != nil {
		t.Fatalf("fail-closed: %v", err)
	}
	active, err := store.ActiveTheme(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(active.PackageDigest, goodDigest) {
		t.Fatalf("active digest=%q want=%q", active.PackageDigest, goodDigest)
	}
	if registry.registeredDigest != goodDigest {
		t.Fatalf("registered digest=%q want=%q", registry.registeredDigest, goodDigest)
	}
}

type digestAwarePageRegistry struct {
	badDigest        string
	registeredDigest string
	cleared          map[string]bool
}

func (r *digestAwarePageRegistry) PreflightThemePackage(_ context.Context, extension Extension, _ string) error {
	if strings.EqualFold(extension.PackageDigest, r.badDigest) {
		return errors.New(`pages: template host island "sf-my-home-page" is not allowed`)
	}
	return nil
}

func (r *digestAwarePageRegistry) RegisterPluginPackage(context.Context, Extension) error { return nil }

func (r *digestAwarePageRegistry) RegisterThemePackage(_ context.Context, extension Extension) error {
	if err := r.PreflightThemePackage(context.Background(), extension, ""); err != nil {
		return err
	}
	r.registeredDigest = extension.PackageDigest
	return nil
}

func (r *digestAwarePageRegistry) RegisterDefaultThemeFallback(context.Context, Extension) error {
	return nil
}

func (r *digestAwarePageRegistry) RegisterThemePackageRestoring(ctx context.Context, extension Extension, _ []string) error {
	return r.RegisterThemePackage(ctx, extension)
}

func (r *digestAwarePageRegistry) RegisterThemePackageReplacing(ctx context.Context, extension Extension, _ string) error {
	return r.RegisterThemePackage(ctx, extension)
}

func (r *digestAwarePageRegistry) RegisterThemePackageReplacingApproved(ctx context.Context, extension Extension, previous string, _ int64) error {
	return r.RegisterThemePackageReplacing(ctx, extension, previous)
}

func (r *digestAwarePageRegistry) ClearExtension(extensionID string) {
	if r.cleared == nil {
		r.cleared = map[string]bool{}
	}
	r.cleared[extensionID] = true
}
