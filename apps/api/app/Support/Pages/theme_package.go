package pages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ThemePackage 描述 L0/L1 运行时主题包（theme.json + assets + templates）。
type ThemePackage struct {
	Pages []ThemePageDecl `json:"pages"`
	Skin  ThemeSkin       `json:"skin"`
	Widgets []ThemeWidget `json:"widgets,omitempty"`
}

type ThemePageDecl struct {
	ID       string `json:"id"`
	Action   string `json:"action"` // add | replace
	Target   string `json:"target,omitempty"`
	Path     string `json:"path,omitempty"`
	Template string `json:"template,omitempty"`
	Contract string `json:"contract,omitempty"`
	Access   string `json:"access,omitempty"`
	Data     *ThemePageData `json:"data,omitempty"`
}

type ThemePageData struct {
	Source string `json:"source"` // plugin | core
	Route  string `json:"route,omitempty"`
}

type ThemeSkin struct {
	CSS    []string `json:"css"`
	Tokens string   `json:"tokens,omitempty"`
}

type ThemeWidget struct {
	ID        string `json:"id"`
	Entry     string `json:"entry"`
	CSS       string `json:"css,omitempty"`
	Integrity string `json:"integrity,omitempty"`
}

// LoadThemePackage 从扩展包根目录读取 theme.json；不存在时返回空包（仅 L0 可选）。
func LoadThemePackage(packageRoot string) (ThemePackage, error) {
	path := filepath.Join(packageRoot, "theme.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 兼容：无 theme.json 时尝试默认 skin 路径
			return ThemePackage{
				Skin: ThemeSkin{CSS: defaultSkinCSSIfPresent(packageRoot)},
			}, nil
		}
		return ThemePackage{}, err
	}
	var pkg ThemePackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return ThemePackage{}, fmt.Errorf("pages: invalid theme.json: %w", err)
	}
	return pkg, nil
}

func defaultSkinCSSIfPresent(packageRoot string) []string {
	candidates := []string{
		"assets/theme.css",
		"assets/css/theme.css",
		"skin/theme.css",
	}
	var out []string
	for _, rel := range candidates {
		if st, err := os.Stat(filepath.Join(packageRoot, rel)); err == nil && !st.IsDir() {
			out = append(out, rel)
		}
	}
	return out
}

// ContributionsFromTheme 将 theme.json 转为 Registry 贡献。
func ContributionsFromTheme(extensionID, version, digest string, pkg ThemePackage) []PageContribution {
	out := make([]PageContribution, 0, len(pkg.Pages))
	for _, p := range pkg.Pages {
		action := ContributionAction(strings.ToLower(strings.TrimSpace(p.Action)))
		if action != ActionAdd && action != ActionReplace {
			continue
		}
		c := PageContribution{
			ID:            p.ID,
			Action:        action,
			Target:        p.Target,
			Path:          p.Path,
			Template:      p.Template,
			Contract:      p.Contract,
			Access:        Access(p.Access),
			ExtensionID:   extensionID,
			Version:       version,
			PackageDigest: digest,
		}
		if p.Data != nil {
			c.DataSource = p.Data.Source
			c.DataRoute = p.Data.Route
		}
		out = append(out, c)
	}
	return out
}

// ResolveThemeAsset 安全解析主题包内相对资源路径。
func ResolveThemeAsset(packageRoot, rel string) (string, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("pages: unsafe asset path")
	}
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return "", err
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	full, err = filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("pages: asset escapes package root")
	}
	return full, nil
}
