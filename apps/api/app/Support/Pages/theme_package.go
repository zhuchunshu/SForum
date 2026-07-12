package pages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ThemePackage 描述 L0/L1 运行时主题包（theme.json + assets + templates）。
type ThemePackage struct {
	Pages   []ThemePageDecl `json:"pages"`
	Skin    ThemeSkin       `json:"skin"`
	Widgets []ThemeWidget   `json:"widgets,omitempty"` // L2 声明保留解析，运行时不加载
}

type ThemePageDecl struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"` // add | replace
	Target     string         `json:"target,omitempty"`
	Path       string         `json:"path,omitempty"`
	Template   string         `json:"template,omitempty"`
	Contract   string         `json:"contract,omitempty"`
	Access     string         `json:"access,omitempty"`
	Permission string         `json:"permission,omitempty"` // access=permission 时的权限键
	Data       *ThemePageData `json:"data,omitempty"`
}

type ThemePageData struct {
	Source string `json:"source"` // plugin | core
	Route  string `json:"route,omitempty"`
	Schema string `json:"schema,omitempty"` // 可选 JSON Schema 相对路径
}

type ThemeSkin struct {
	CSS    []string `json:"css"`
	Tokens string   `json:"tokens,omitempty"`
}

// ThemeWidget L2 声明；当前宿主拒绝加载可执行 widget。
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
			Permission:    p.Permission,
			ExtensionID:   extensionID,
			Version:       version,
			PackageDigest: digest,
		}
		if p.Data != nil {
			c.DataSource = p.Data.Source
			c.DataRoute = p.Data.Route
			c.DataSchema = p.Data.Schema
		}
		out = append(out, c)
	}
	return out
}

// AllowedThemeAssetExt 普通主题资源允许的扩展名（禁止 SVG / JS / HTML）。
var AllowedThemeAssetExt = map[string]string{
	".css":   "text/css; charset=utf-8",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
}

// ResolveThemeAsset 安全解析主题包内相对资源路径。
// 使用 EvalSymlinks 防止 symlink 逃逸出包根。
func ResolveThemeAsset(packageRoot, rel string) (string, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") || strings.Contains(rel, "\\") {
		return "", fmt.Errorf("pages: unsafe asset path")
	}
	// 拒绝空段与隐藏绝对化
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Errorf("pages: unsafe asset path segment")
		}
	}
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return "", err
	}
	// 解析包根真实路径（含 symlink）
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		// 包根必须存在
		return "", fmt.Errorf("pages: package root unavailable: %w", err)
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	// 目标必须存在才能 EvalSymlinks
	info, err := os.Lstat(fullAbs)
	if err != nil {
		return "", fmt.Errorf("pages: asset not found")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		// 显式拒绝资源路径上的 symlink（防止逃逸）
		return "", fmt.Errorf("pages: asset symlink forbidden")
	}
	// 若中间目录是 symlink，EvalSymlinks 后校验仍在 root 下
	realFull, err := filepath.EvalSymlinks(fullAbs)
	if err != nil {
		return "", fmt.Errorf("pages: asset path resolve failed")
	}
	if realFull != rootReal && !strings.HasPrefix(realFull, rootReal+string(os.PathSeparator)) {
		return "", fmt.Errorf("pages: asset escapes package root")
	}
	return realFull, nil
}

// FileDigestSHA256 计算文件 sha256 hex。
func FileDigestSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
