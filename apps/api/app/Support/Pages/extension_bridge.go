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
	ID          string
	Version     string
	PackagePath string
	PackageDigest string
	Source      string
}

// ExtensionBridge 把主题包接到 Registry（供 extensions.Service 注入）。
type ExtensionBridge struct {
	Registry *Registry
}

func NewExtensionBridge(registry *Registry) *ExtensionBridge {
	return &ExtensionBridge{Registry: registry}
}

// RegisterThemePackage 加载 theme.json、校验 L0 CSS，并注册页面贡献。
func (b *ExtensionBridge) RegisterThemePackage(ctx context.Context, ext ThemeExtension) error {
	if b == nil || b.Registry == nil {
		return nil
	}
	root := strings.TrimSpace(ext.PackagePath)
	if root == "" {
		return fmt.Errorf("pages: theme package path empty")
	}
	pkg, err := LoadThemePackage(root)
	if err != nil {
		return err
	}
	// 校验皮肤 CSS
	for _, rel := range pkg.Skin.CSS {
		full, err := ResolveThemeAsset(root, rel)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		if err := ValidateCSS(string(raw)); err != nil {
			return err
		}
	}
	if tok := strings.TrimSpace(pkg.Skin.Tokens); tok != "" {
		full, err := ResolveThemeAsset(root, tok)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		if err := ValidateCSS(string(raw)); err != nil {
			return err
		}
	}
	// 校验 L1 模板存在且安全
	for _, p := range pkg.Pages {
		if strings.TrimSpace(p.Template) == "" {
			continue
		}
		if _, err := LoadTemplate(root, p.Template); err != nil {
			return fmt.Errorf("pages: template %s: %w", p.Template, err)
		}
	}
	contribs := ContributionsFromTheme(ext.ID, ext.Version, ext.PackageDigest, pkg)
	if err := b.Registry.RegisterContributions(ext.ID, contribs); err != nil {
		return err
	}
	// 主题激活时自动绑定该主题声明的 replace（操作者已选择激活该主题）。
	// 冲突多候选时仍只绑定本主题贡献；插件 replace 仍需 super_admin 审批。
	for _, c := range contribs {
		if c.Action != ActionReplace {
			continue
		}
		_ = b.Registry.ApproveReplace(ctx, ProviderBinding{
			PageID:         c.Target,
			ExtensionID:    c.ExtensionID,
			ContributionID: c.ID,
			Version:        c.Version,
			PackageDigest:  c.PackageDigest,
			TemplatePath:   c.Template,
		})
	}
	return nil
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
	ExtensionID string   `json:"extensionId"`
	Version     string   `json:"version"`
	CSS         []string `json:"css"`
	Tokens      string   `json:"tokens,omitempty"`
}

// SkinFromPackage 读取主题包皮肤清单。
func SkinFromPackage(extensionID, version, packageRoot string) (ActiveSkinPublic, error) {
	pkg, err := LoadThemePackage(packageRoot)
	if err != nil {
		return ActiveSkinPublic{}, err
	}
	css := make([]string, 0, len(pkg.Skin.CSS))
	for _, rel := range pkg.Skin.CSS {
		css = append(css, "/api/v1/site/theme-assets/"+extensionID+"/"+filepath.ToSlash(rel))
	}
	tokens := ""
	if t := strings.TrimSpace(pkg.Skin.Tokens); t != "" {
		tokens = "/api/v1/site/theme-assets/" + extensionID + "/" + filepath.ToSlash(t)
	}
	return ActiveSkinPublic{
		ExtensionID: extensionID,
		Version:     version,
		CSS:         css,
		Tokens:      tokens,
	}, nil
}
