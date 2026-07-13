package extensions

import (
	"context"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
)

// PageRegistryAdapter 将 pages.ExtensionBridge 接到 Service.PageRegistry。
type PageRegistryAdapter struct {
	Bridge *pages.ExtensionBridge
}

func NewPageRegistryAdapter(registry *pages.Registry) *PageRegistryAdapter {
	return &PageRegistryAdapter{Bridge: pages.NewExtensionBridge(registry)}
}

func (a *PageRegistryAdapter) PreflightThemePackage(ctx context.Context, extension Extension, previousActiveThemeID string) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	_, err := a.Bridge.PreflightThemePackage(toThemeExt(extension), previousActiveThemeID)
	return err
}

func (a *PageRegistryAdapter) RegisterThemePackage(ctx context.Context, extension Extension) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	return a.Bridge.RegisterThemePackage(ctx, toThemeExt(extension))
}

func (a *PageRegistryAdapter) RegisterThemePackageReplacing(ctx context.Context, extension Extension, previousActiveThemeID string) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	return a.Bridge.RegisterThemePackageReplacing(ctx, toThemeExt(extension), previousActiveThemeID)
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
