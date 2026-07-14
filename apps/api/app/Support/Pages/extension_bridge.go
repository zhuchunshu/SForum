package pages

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ThemeExtension 是 extensions.Extension 的最小视图，避免循环依赖。
type ThemeExtension struct {
	ID            string
	Version       string
	PackagePath   string
	PackageDigest string
	Source        string
}

// ExtensionBridge 把主题/插件包接到 Registry（供 extensions.Service 注入）。
type ExtensionBridge struct {
	Registry *Registry
}

func NewExtensionBridge(registry *Registry) *ExtensionBridge {
	return &ExtensionBridge{Registry: registry}
}

// PreflightThemePackage 激活前完整预检：manifest、theme.json、模板、CSS、资源、routes。
// previousActiveThemeID 非空时按「用新主题替换旧主题」的最终状态校验 add 路径。
// 不修改 Registry 状态。
func (b *ExtensionBridge) PreflightThemePackage(ext ThemeExtension, previousActiveThemeID string) ([]PageContribution, error) {
	root := strings.TrimSpace(ext.PackagePath)
	if root == "" {
		return nil, fmt.Errorf("pages: theme package path empty")
	}
	pkg, err := LoadThemePackage(root)
	if err != nil {
		return nil, err
	}
	// 校验皮肤 CSS
	for _, rel := range pkg.Skin.CSS {
		full, err := ResolveThemeAsset(root, rel)
		if err != nil {
			return nil, fmt.Errorf("pages: skin css %s: %w", rel, err)
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		if err := ValidateCSS(string(raw)); err != nil {
			return nil, fmt.Errorf("pages: skin css %s: %w", rel, err)
		}
	}
	if tok := strings.TrimSpace(pkg.Skin.Tokens); tok != "" {
		full, err := ResolveThemeAsset(root, tok)
		if err != nil {
			return nil, err
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		if err := ValidateCSS(string(raw)); err != nil {
			return nil, err
		}
	}
	// 校验 L1 模板
	for _, p := range pkg.Pages {
		if strings.TrimSpace(p.Template) == "" {
			continue
		}
		if _, err := LoadTemplate(root, p.Template); err != nil {
			return nil, fmt.Errorf("pages: template %s: %w", p.Template, err)
		}
	}
	// L2 widgets：声明存在时仅记录拒绝加载（不阻断 L0/L1 激活）
	contribs := ContributionsFromTheme(ext.ID, ext.Version, ext.PackageDigest, pkg)
	if b != nil && b.Registry != nil {
		ignore := []string{ext.ID}
		if previousActiveThemeID != "" && previousActiveThemeID != ext.ID {
			ignore = append(ignore, previousActiveThemeID)
		}
		if err := b.Registry.PreflightContributionsReplacing(ext.ID, contribs, ignore...); err != nil {
			return nil, err
		}
	} else {
		if _, err := prepareContributions(ext.ID, contribs); err != nil {
			return nil, err
		}
	}
	return contribs, nil
}

// RegisterThemePackage 加载 theme.json、校验 L0 CSS，并注册页面贡献（仅候选，不批准 replace）。
func (b *ExtensionBridge) RegisterThemePackage(ctx context.Context, ext ThemeExtension) error {
	if b == nil || b.Registry == nil {
		return nil
	}
	contribs, err := b.PreflightThemePackage(ext, "")
	if err != nil {
		return err
	}
	// 仅注册候选；核心页 replace 必须 super_admin 明确批准。
	if err := b.Registry.RegisterContributions(ext.ID, contribs); err != nil {
		return err
	}
	return nil
}

// RegisterThemePackageReplacing 原子主题切换：用新主题贡献替换旧活动主题。
func (b *ExtensionBridge) RegisterThemePackageReplacing(ctx context.Context, ext ThemeExtension, previousActiveThemeID string) error {
	if b == nil || b.Registry == nil {
		return nil
	}
	contribs, err := b.PreflightThemePackage(ext, previousActiveThemeID)
	if err != nil {
		return err
	}
	return b.Registry.ReplaceThemeContributions(ext.ID, contribs, previousActiveThemeID)
}

// RegisterPluginPackage 从 theme.json（统一页面 manifest 契约）注册插件页面贡献。
// 插件与主题使用同一 theme.json pages 声明，不另立格式。
func (b *ExtensionBridge) RegisterPluginPackage(ctx context.Context, ext ThemeExtension) error {
	if b == nil || b.Registry == nil {
		return nil
	}
	contribs, err := b.PreflightPluginPackage(ext)
	if err != nil {
		return err
	}
	return b.Registry.RegisterContributions(ext.ID, contribs)
}

// PreflightPluginPackage performs all package I/O before a plugin page or its
// compiled template snapshot becomes visible.
func (b *ExtensionBridge) PreflightPluginPackage(ext ThemeExtension) ([]PageContribution, error) {
	// 插件可选 theme.json；无则无页面贡献
	pkg, err := LoadThemePackage(ext.PackagePath)
	if err != nil {
		return nil, err
	}
	if len(pkg.Pages) == 0 {
		return nil, nil
	}
	// 校验模板
	for _, p := range pkg.Pages {
		if strings.TrimSpace(p.Template) == "" {
			continue
		}
		if _, err := LoadTemplate(ext.PackagePath, p.Template); err != nil {
			return nil, fmt.Errorf("pages: plugin template %s: %w", p.Template, err)
		}
	}
	contribs := ContributionsFromTheme(ext.ID, ext.Version, ext.PackageDigest, pkg)
	if err := b.Registry.PreflightContributions(ext.ID, contribs); err != nil {
		return nil, err
	}
	return contribs, nil
}

// ClearExtension 实现 extensions.PageRegistry。
func (b *ExtensionBridge) ClearExtension(extensionID string) {
	if b == nil || b.Registry == nil {
		return
	}
	b.Registry.ClearExtension(extensionID)
}

// ActiveSkinPublic 返回可公开注入的皮肤资源相对 URL 路径（由 API 静态路由服务）。
type ActiveSkinPublic struct {
	ExtensionID   string   `json:"extensionId"`
	Version       string   `json:"version"`
	PackageDigest string   `json:"packageDigest,omitempty"`
	CSS           []string `json:"css"`
	Tokens        string   `json:"tokens,omitempty"`
}

// SkinFromPackage 读取主题包皮肤清单；URL 携带 package digest 以支持 immutable cache。
func SkinFromPackage(extensionID, version, packageDigest, packageRoot string) (ActiveSkinPublic, error) {
	pkg, err := LoadThemePackage(packageRoot)
	if err != nil {
		return ActiveSkinPublic{}, err
	}
	q := ""
	if d := strings.TrimSpace(packageDigest); d != "" {
		q = "?v=" + d
	}
	css := make([]string, 0, len(pkg.Skin.CSS))
	for _, rel := range pkg.Skin.CSS {
		css = append(css, "/api/v1/site/theme-assets/"+extensionID+"/"+filepath.ToSlash(rel)+q)
	}
	tokens := ""
	if t := strings.TrimSpace(pkg.Skin.Tokens); t != "" {
		tokens = "/api/v1/site/theme-assets/" + extensionID + "/" + filepath.ToSlash(t) + q
	}
	return ActiveSkinPublic{
		ExtensionID:   extensionID,
		Version:       version,
		PackageDigest: packageDigest,
		CSS:           css,
		Tokens:        tokens,
	}, nil
}
