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
	Pages []ThemePageDecl `json:"pages"`
	// Shell 是插件公开页外壳的 L1 模板相对路径（可选）。
	// 外壳承担插件正文之外的 chrome，因此与页面模板一样在激活前预检可加载性；
	// 消费方（resolve-path 组装）在后续层引入，未声明时插件页回到宿主外壳。
	Shell               string            `json:"shell,omitempty"`
	Skin                ThemeSkin         `json:"skin"`
	NavigationLocations map[string]string `json:"navigationLocations,omitempty"`
	Widgets             []ThemeWidget     `json:"widgets,omitempty"` // L2 声明保留解析，运行时不加载
}

var themeNavigationLocationBindings = map[string]string{
	"public.topbar.primary":  "sf-navbar",
	"public.sidebar.primary": "sf-home-navigation",
	"public.mobile.primary":  "sf-navbar",
	"public.footer.primary":  "sf-footer",
}

type ThemePageDecl struct {
	ID         string `json:"id"`
	Action     string `json:"action"` // add | replace
	Target     string `json:"target,omitempty"`
	Path       string `json:"path,omitempty"`
	Template   string `json:"template,omitempty"`
	// Label 是页面的展示标题（可选）：resolve-path 透传给前端，用于
	// document.title 兜底，避免模板刻意不输出 h1 时标题退回路径。
	Label string `json:"label,omitempty"`
	Contract   string `json:"contract,omitempty"`
	Access     string `json:"access,omitempty"`
	Permission string `json:"permission,omitempty"` // access=permission 时的权限键
	// Layout: site | bare。空值 = 平台默认 site（见 NormalizeLayout）；未知非空值在
	// 预检时 fail closed。预检阶段保留声明原样，归一化由 prepareContributions 完成。
	Layout string         `json:"layout,omitempty"`
	Data   *ThemePageData `json:"data,omitempty"`
	// View: template | component。空值等价 template（历史行为）。
	// component 表示正文由包内已声明的公开 L2 组件提供（manifest.components[].l2Component），
	// 此时 template 必须为空；未知非空值在预检时 fail closed。
	View      string              `json:"view,omitempty"`
	Component *ThemePageComponent `json:"component,omitempty"`
	// Schema: view=schema 时的有界页面描述符声明。
	Schema *ThemePageSchema `json:"schema,omitempty"`
}

// ThemePageComponent 指向 manifest.components[] 中一个公开 L2 组件。
// 只声明组件 id：入口、摘要、信任与准入仍由既有的公开前端管线裁决
// （GET /extensions/runtime/{extensionId}/components/{componentId}），
// 页面声明不重复一份信任判断。
type ThemePageComponent struct {
	ID string `json:"id"`
}

// ThemePageSchema 声明一个 view=schema 公开页的正文来源。
//
// 描述符（descriptor）是包内 kind=schema 的 packageFiles[] id，内容是**有界**页面描述符
// JSON（list | detail | form | notice）；契约（contract）是包内精确 JSON Schema 引用
// （<schemaFileId>@<version>），描述符必须满足它。两者都只声明 id/引用，文件是否存在、
// kind 是否匹配、schema 能否编译、描述符是否满足契约与有界规则，全部由扩展层的包预检
// 裁决（Pages 包不引入扩展注册表，避免反向依赖）。
type ThemePageSchema struct {
	Descriptor string `json:"descriptor"`
	Contract   string `json:"contract"`
}

// PageView 是公开页正文的提供方式。
type PageView string

const (
	// PageViewTemplate：正文由包内 L1 Go 模板渲染（历史行为，空值等价于此）。
	PageViewTemplate PageView = "template"
	// PageViewComponent：正文由包内公开 L2 组件挂载。
	PageViewComponent PageView = "component"
	// PageViewSchema：正文由包内声明的有界页面描述符渲染（list | detail | form | notice）。
	PageViewSchema PageView = "schema"
)

// NormalizePageView 归一化 view 声明：空值等价 template。
func NormalizePageView(value string) PageView {
	normalized := PageView(strings.ToLower(strings.TrimSpace(value)))
	if normalized == "" {
		return PageViewTemplate
	}
	return normalized
}

// ValidateThemePageViews 校验 view 与 template/component 的互斥关系。
// 未知 view 一律 fail closed：作者写错时要立刻失败，而不是静默退化成模板页。
func ValidateThemePageViews(pkg ThemePackage) error {
	for _, page := range pkg.Pages {
		view := NormalizePageView(page.View)
		template := strings.TrimSpace(page.Template)
		switch view {
		case PageViewTemplate:
			if page.Component != nil && strings.TrimSpace(page.Component.ID) != "" {
				return fmt.Errorf("pages: page %s declares view=template together with a component", page.ID)
			}
		case PageViewComponent:
			if template != "" {
				return fmt.Errorf("pages: page %s declares view=component together with a template", page.ID)
			}
			if page.Component == nil || strings.TrimSpace(page.Component.ID) == "" {
				return fmt.Errorf("pages: page %s declares view=component without a component id", page.ID)
			}
			if page.Schema != nil {
				return fmt.Errorf("pages: page %s declares view=component together with a schema descriptor", page.ID)
			}
		case PageViewSchema:
			if template != "" {
				return fmt.Errorf("pages: page %s declares view=schema together with a template", page.ID)
			}
			if page.Component != nil && strings.TrimSpace(page.Component.ID) != "" {
				return fmt.Errorf("pages: page %s declares view=schema together with a component", page.ID)
			}
			if page.Schema == nil || strings.TrimSpace(page.Schema.Descriptor) == "" ||
				strings.TrimSpace(page.Schema.Contract) == "" {
				return fmt.Errorf(
					"pages: page %s declares view=schema without a descriptor and contract", page.ID,
				)
			}
		default:
			return fmt.Errorf("pages: page %s declares unknown view %q", page.ID, page.View)
		}
	}
	return nil
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
	locations, err := normalizeThemeNavigationLocations(pkg.NavigationLocations)
	if err != nil {
		return ThemePackage{}, err
	}
	pkg.NavigationLocations = locations
	// view 与 template/component 的互斥在这里统一裁决：CLI（extension validate/test）
	// 与运行时激活都经由此读取点，避免两套校验再次分叉。
	if err := ValidateThemePageViews(pkg); err != nil {
		return ThemePackage{}, err
	}
	return pkg, nil
}

func normalizeThemeNavigationLocations(input map[string]string) (map[string]string, error) {
	if len(input) == 0 {
		return nil, nil
	}
	result := make(map[string]string, len(input))
	bindings := productionThemeIslandBindings()
	for location, island := range input {
		location = strings.TrimSpace(location)
		island = strings.TrimSpace(strings.ToLower(island))
		expected, known := themeNavigationLocationBindings[location]
		if !known || island != expected {
			return nil, fmt.Errorf("pages: invalid theme navigation location binding %q=%q", location, island)
		}
		if _, allowed := allowedHostIslands[island]; !allowed || bindings[island].ComponentID == "" {
			return nil, fmt.Errorf("pages: unreviewed theme navigation island %q", island)
		}
		result[location] = island
	}
	return result, nil
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
			Label:         strings.TrimSpace(p.Label),
			View:          NormalizePageView(p.View),
			Contract:      p.Contract,
			Access:        Access(p.Access),
			Permission:    p.Permission,
			Layout:        PageLayout(p.Layout),
			ExtensionID:   extensionID,
			Version:       version,
			PackageDigest: digest,
		}
		if p.Component != nil {
			c.ComponentID = strings.TrimSpace(p.Component.ID)
		}
		if p.Schema != nil {
			c.SchemaDescriptorFile = strings.TrimSpace(p.Schema.Descriptor)
			c.SchemaContractRef = strings.TrimSpace(p.Schema.Contract)
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
