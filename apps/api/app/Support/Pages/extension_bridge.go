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
	// 仅注册候选贡献；核心页 replace 必须由 super_admin 通过 ApproveReplace 明确批准。
	// 主题激活不得静默批准、不得 ApprovedBy=0、不得忽略审批错误。
	if err := b.Registry.RegisterContributions(ext.ID, contribs); err != nil {
		return err
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
