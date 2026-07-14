package extensions

import (
	"context"
	"errors"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
)

// PageRegistryAdapter 将 pages.ExtensionBridge 接到 Service.PageRegistry。
type PageRegistryAdapter struct {
	Bridge       *pages.ExtensionBridge
	ThemeRuntime *pages.ThemeRuntimeRegistry
	SiteName     string
	Locales      []string
}

func (a *PageRegistryAdapter) WithThemeRuntime(runtime *pages.ThemeRuntimeRegistry, siteName string, locales []string) *PageRegistryAdapter {
	if a != nil {
		a.ThemeRuntime = runtime
		a.SiteName = siteName
		a.Locales = append([]string(nil), locales...)
	}
	return a
}

func NewPageRegistryAdapter(registry *pages.Registry) *PageRegistryAdapter {
	return &PageRegistryAdapter{Bridge: pages.NewExtensionBridge(registry)}
}

func (a *PageRegistryAdapter) PreflightThemePackage(ctx context.Context, extension Extension, previousActiveThemeID string) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	contributions, err := a.Bridge.PreflightThemePackage(toThemeExt(extension), previousActiveThemeID)
	if err != nil || a.ThemeRuntime == nil {
		return err
	}
	_, err = a.buildThemeRuntime(extension, contributions)
	return err
}

func (a *PageRegistryAdapter) RegisterThemePackage(ctx context.Context, extension Extension) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	contributions, err := a.Bridge.PreflightThemePackage(toThemeExt(extension), "")
	if err != nil {
		return err
	}
	snapshot, err := a.buildThemeRuntime(extension, contributions)
	if err != nil {
		return err
	}
	staged, err := a.stageThemeRuntime(snapshot)
	if err != nil {
		return err
	}
	if err := a.Bridge.Registry.RegisterContributions(extension.ID, contributions); err != nil {
		a.rollbackStagedThemeRuntime(snapshot, staged)
		return err
	}
	if err := a.activateThemeRuntime(snapshot); err != nil {
		a.Bridge.Registry.ClearExtension(extension.ID)
		a.rollbackStagedThemeRuntime(snapshot, staged)
		return err
	}
	return nil
}

func (a *PageRegistryAdapter) RegisterThemePackageReplacing(ctx context.Context, extension Extension, previousActiveThemeID string) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	contributions, err := a.Bridge.PreflightThemePackage(toThemeExt(extension), previousActiveThemeID)
	if err != nil {
		return err
	}
	snapshot, err := a.buildThemeRuntime(extension, contributions)
	if err != nil {
		return err
	}
	var previous pages.RuntimeArtifact
	if active, _, ok := a.ThemeRuntime.Active(); ok {
		previous = active.Artifact()
	}
	staged, err := a.stageThemeRuntime(snapshot)
	if err != nil {
		return err
	}
	if err := a.Bridge.Registry.ReplaceThemeContributions(extension.ID, contributions, previousActiveThemeID); err != nil {
		a.rollbackStagedThemeRuntime(snapshot, staged)
		return err
	}
	if err := a.activateThemeRuntime(snapshot); err != nil {
		a.rollbackStagedThemeRuntime(snapshot, staged)
		return err
	}
	if previous.ExtensionID != "" && (snapshot == nil || previous != snapshot.Artifact()) {
		_, _ = a.ThemeRuntime.RemoveExact(previous)
	}
	return nil
}

func (a *PageRegistryAdapter) RegisterPluginPackage(ctx context.Context, extension Extension) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	return a.Bridge.RegisterPluginPackage(ctx, toThemeExt(extension))
}

func (a *PageRegistryAdapter) ClearExtension(extensionID string) {
	if a == nil || a.Bridge == nil {
		return
	}
	a.Bridge.ClearExtension(extensionID)
	if a.ThemeRuntime != nil {
		a.ThemeRuntime.ClearExtension(extensionID)
	}
}

func (a *PageRegistryAdapter) buildThemeRuntime(extension Extension, contributions []pages.PageContribution) (*pages.ThemeRuntimeSnapshot, error) {
	if a == nil || a.ThemeRuntime == nil {
		return nil, nil
	}
	snapshot, err := pages.BuildThemeRuntimeSnapshot(pages.ThemeRuntimeBuildInput{
		Artifact: pages.RuntimeArtifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version, PackageDigest: extension.PackageDigest,
		},
		PackageRoot: PackageContentRoot(extension), Contributions: contributions,
		SiteName: a.SiteName, Locales: a.Locales,
	})
	if errors.Is(err, pages.ErrThemeRuntimeMissing) {
		return nil, nil
	}
	return snapshot, err
}

func (a *PageRegistryAdapter) stageThemeRuntime(snapshot *pages.ThemeRuntimeSnapshot) (bool, error) {
	if snapshot == nil || a.ThemeRuntime == nil {
		return false, nil
	}
	_, staged, err := a.ThemeRuntime.Stage(snapshot)
	return staged, err
}

func (a *PageRegistryAdapter) activateThemeRuntime(snapshot *pages.ThemeRuntimeSnapshot) error {
	if snapshot == nil || a.ThemeRuntime == nil {
		return nil
	}
	_, err := a.ThemeRuntime.ActivateExact(snapshot.Artifact())
	return err
}

func (a *PageRegistryAdapter) rollbackStagedThemeRuntime(snapshot *pages.ThemeRuntimeSnapshot, staged bool) {
	if !staged || snapshot == nil || a.ThemeRuntime == nil {
		return
	}
	_, _ = a.ThemeRuntime.RemoveExact(snapshot.Artifact())
}

func toThemeExt(extension Extension) pages.ThemeExtension {
	return pages.ThemeExtension{
		ID:      extension.ID,
		Version: extension.Version,
		// 旧上传包 PackagePath 可能是 package.zip；L0/L1 必须读 files/ 内容根。
		PackagePath:   PackageContentRoot(extension),
		PackageDigest: extension.PackageDigest,
		Source:        string(extension.Source),
	}
}
