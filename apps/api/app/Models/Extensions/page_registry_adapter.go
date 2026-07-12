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

func (a *PageRegistryAdapter) RegisterThemePackage(ctx context.Context, extension Extension) error {
	if a == nil || a.Bridge == nil {
		return nil
	}
	return a.Bridge.RegisterThemePackage(ctx, pages.ThemeExtension{
		ID:            extension.ID,
		Version:       extension.Version,
		PackagePath:   extension.PackagePath,
		PackageDigest: extension.PackageDigest,
		Source:        string(extension.Source),
	})
}

func (a *PageRegistryAdapter) ClearExtension(extensionID string) {
	if a == nil || a.Bridge == nil {
		return
	}
	a.Bridge.ClearExtension(extensionID)
}
