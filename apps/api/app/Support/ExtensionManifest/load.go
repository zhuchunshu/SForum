package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestIncludes 声明可选的分文件 partial 路径（仅存在于入口 JSON，不进入运行时 Manifest）。
type ManifestIncludes struct {
	Langs                 json.RawMessage `json:"langs,omitempty"`
	Settings              json.RawMessage `json:"settings,omitempty"`
	Contributions         json.RawMessage `json:"contributions,omitempty"`
	Admin                 json.RawMessage `json:"admin,omitempty"`
	Events                json.RawMessage `json:"events,omitempty"`
	Routes                json.RawMessage `json:"routes,omitempty"`
	Jobs                  json.RawMessage `json:"jobs,omitempty"`
	Migrations            json.RawMessage `json:"migrations,omitempty"`
	Permissions           json.RawMessage `json:"permissions,omitempty"`
	Hooks                 json.RawMessage `json:"hooks,omitempty"`
	Providers             json.RawMessage `json:"providers,omitempty"`
	AdminPages            json.RawMessage `json:"adminPages,omitempty"`
	Guards                json.RawMessage `json:"guards,omitempty"`
	Schedules             json.RawMessage `json:"schedules,omitempty"`
	Components            json.RawMessage `json:"components,omitempty"`
	Templates             json.RawMessage `json:"templates,omitempty"`
	Assets                json.RawMessage `json:"assets,omitempty"`
	Content               json.RawMessage `json:"content,omitempty"`
	Database              json.RawMessage `json:"database,omitempty"`
	Cache                 json.RawMessage `json:"cache,omitempty"`
	Services              json.RawMessage `json:"services,omitempty"`
	Commands              json.RawMessage `json:"commands,omitempty"`
	AdminSurfaces         json.RawMessage `json:"adminSurfaces,omitempty"`
	Queries               json.RawMessage `json:"queries,omitempty"`
	Identity              json.RawMessage `json:"identity,omitempty"`
	PermissionDefinitions json.RawMessage `json:"permissionDefinitions,omitempty"`
	Media                 json.RawMessage `json:"media,omitempty"`
	Navigation            json.RawMessage `json:"navigation,omitempty"`
	Regions               json.RawMessage `json:"regions,omitempty"`
	Dependencies          json.RawMessage `json:"dependencies,omitempty"`
	Lifecycle             json.RawMessage `json:"lifecycle,omitempty"`
	OpenAPI               json.RawMessage `json:"openapi,omitempty"`
	PackageFiles          json.RawMessage `json:"packageFiles,omitempty"`
}

// rootManifestFile 解析入口文件：Manifest 字段 + includes 索引。
type rootManifestFile struct {
	Includes *ManifestIncludes `json:"includes,omitempty"`
}

// PackageFS 抽象包内相对路径读取（磁盘包或 ZIP 内存映射）。
type PackageFS interface {
	// ReadFile 读取相对路径（正斜杠）。不存在时返回 fs.ErrNotExist。
	ReadFile(rel string) ([]byte, error)
	// Stat 返回是否为目录；不存在时返回 fs.ErrNotExist。
	Stat(rel string) (isDir bool, err error)
	// ReadDir 列出目录下直接子项名称（非递归）；不存在时返回 fs.ErrNotExist。
	ReadDir(rel string) ([]string, error)
}

// LoadPackage 从包根目录加载入口 manifest，解析 includes，合并后 Validate。
func LoadPackage(root string) (Manifest, error) {
	return LoadPackageFS(diskPackageFS{root: root})
}

// LoadPackageFS 从任意 PackageFS 加载并合并 manifest。
func LoadPackageFS(pkg PackageFS) (Manifest, error) {
	body, err := pkg.ReadFile(ManifestFileName)
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: missing %s", ErrInvalidManifest, ManifestFileName)
	}
	return LoadRootBytes(body, pkg)
}

// LoadRootBytes 解析入口 JSON 字节，并按 includes 从 pkg 读取 partials。
func LoadRootBytes(rootBody []byte, pkg PackageFS) (Manifest, error) {
	var root rootManifestFile
	if err := json.Unmarshal(rootBody, &root); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(rootBody, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	if root.Includes != nil {
		if err := applyIncludes(&manifest, *root.Includes, pkg); err != nil {
			return Manifest{}, err
		}
	}
	manifest = Normalize(manifest)
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	if EffectiveManifestVersion(manifest) == ManifestVersionV3 {
		if err := ValidatePackageFiles(manifest, pkg); err != nil {
			return Manifest{}, err
		}
	}
	return manifest, nil
}

// FileMapFS 将 path -> body 映射为 PackageFS（ZIP 安装用）。
// 键必须是正斜杠相对路径；目录通过键前缀推断。
type FileMapFS map[string][]byte

func (m FileMapFS) ReadFile(rel string) ([]byte, error) {
	rel = cleanRel(rel)
	if rel == "" {
		return nil, fs.ErrNotExist
	}
	body, ok := m[rel]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return body, nil
}

func (m FileMapFS) Stat(rel string) (bool, error) {
	rel = cleanRel(rel)
	if rel == "" {
		return true, nil
	}
	if _, ok := m[rel]; ok {
		return false, nil
	}
	prefix := rel + "/"
	for key := range m {
		if key == rel || strings.HasPrefix(key, prefix) {
			// 仅当存在子路径时视为目录
			if strings.HasPrefix(key, prefix) {
				return true, nil
			}
		}
	}
	// 精确文件已处理；无前缀则不存在
	for key := range m {
		if strings.HasPrefix(key, prefix) {
			return true, nil
		}
	}
	return false, fs.ErrNotExist
}

func (m FileMapFS) ReadDir(rel string) ([]string, error) {
	rel = cleanRel(rel)
	prefix := ""
	if rel != "" && rel != "." {
		isDir, err := m.Stat(rel)
		if err != nil {
			return nil, err
		}
		if !isDir {
			return nil, fmt.Errorf("%w: not a directory: %s", ErrInvalidManifest, rel)
		}
		prefix = rel + "/"
	}
	children := map[string]struct{}{}
	for key := range m {
		if prefix != "" {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			rest := strings.TrimPrefix(key, prefix)
			name, _, _ := strings.Cut(rest, "/")
			if name != "" {
				children[name] = struct{}{}
			}
			continue
		}
		name, _, _ := strings.Cut(key, "/")
		if name != "" {
			children[name] = struct{}{}
		}
	}
	if len(children) == 0 && rel != "" && rel != "." {
		// 空目录：map 中无子项
		return nil, nil
	}
	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

type diskPackageFS struct {
	root string
}

func (d diskPackageFS) ReadFile(rel string) ([]byte, error) {
	safe, err := safePackageRel(rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(d.root, filepath.FromSlash(safe)))
}

func (d diskPackageFS) Stat(rel string) (bool, error) {
	safe, err := safePackageRel(rel)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(filepath.Join(d.root, filepath.FromSlash(safe)))
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (d diskPackageFS) ReadDir(rel string) ([]string, error) {
	safe, err := safePackageRel(rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(d.root, filepath.FromSlash(safe)))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func safePackageRel(rel string) (string, error) {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "" || rel == "." {
		return ".", nil
	}
	clean, ok := SafeArchivePath(rel)
	if !ok {
		return "", fmt.Errorf("%w: unsafe include path %q", ErrInvalidManifest, rel)
	}
	return clean, nil
}

func cleanRel(rel string) string {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, "./")
	rel = path.Clean("/" + rel)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." {
		return ""
	}
	return rel
}

func applyIncludes(manifest *Manifest, includes ManifestIncludes, pkg PackageFS) error {
	type includeSlot struct {
		name   string
		raw    json.RawMessage
		apply  func(raw json.RawMessage) error
		filled func() bool
	}
	slots := []includeSlot{
		{
			name:   "langs",
			raw:    includes.Langs,
			filled: func() bool { return len(manifest.Langs) > 0 },
			apply: func(raw json.RawMessage) error {
				langs, err := loadLangsInclude(raw, pkg)
				if err != nil {
					return err
				}
				manifest.Langs = langs
				return nil
			},
		},
		{
			name:   "settings",
			raw:    includes.Settings,
			filled: func() bool { return len(manifest.Settings) > 0 || manifest.SettingsDocument.Explicit },
			apply: func(raw json.RawMessage) error {
				document, err := loadSettingsInclude(raw, pkg)
				if err != nil {
					return err
				}
				if err := ensureUniqueSettingKeys(document.Fields); err != nil {
					return err
				}
				manifest.Settings = document.Fields
				manifest.SettingsDocument = document
				return nil
			},
		},
		{
			name:   "contributions",
			raw:    includes.Contributions,
			filled: func() bool { return len(manifest.Contributions) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestContribution](raw, pkg, "contributions")
				if err != nil {
					return err
				}
				if err := ensureUniqueContributionIDs(items); err != nil {
					return err
				}
				manifest.Contributions = items
				return nil
			},
		},
		{
			name: "admin",
			raw:  includes.Admin,
			filled: func() bool {
				return strings.TrimSpace(manifest.Admin.Entry) != "" || len(manifest.Admin.Pages) > 0
			},
			apply: func(raw json.RawMessage) error {
				var admin ManifestAdmin
				if err := decodeIncludeObject(raw, pkg, &admin); err != nil {
					return err
				}
				manifest.Admin = admin
				return nil
			},
		},
		{
			name:   "events",
			raw:    includes.Events,
			filled: func() bool { return len(manifest.Events) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestEvent](raw, pkg, "events")
				if err != nil {
					return err
				}
				manifest.Events = items
				return nil
			},
		},
		{
			name:   "routes",
			raw:    includes.Routes,
			filled: func() bool { return len(manifest.Routes) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestRoute](raw, pkg, "routes")
				if err != nil {
					return err
				}
				manifest.Routes = items
				return nil
			},
		},
		{
			name:   "jobs",
			raw:    includes.Jobs,
			filled: func() bool { return len(manifest.Jobs) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestJob](raw, pkg, "jobs")
				if err != nil {
					return err
				}
				manifest.Jobs = items
				return nil
			},
		},
		{
			name:   "migrations",
			raw:    includes.Migrations,
			filled: func() bool { return len(manifest.Migrations) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestMigration](raw, pkg, "migrations")
				if err != nil {
					return err
				}
				manifest.Migrations = items
				return nil
			},
		},
		{
			name:   "permissions",
			raw:    includes.Permissions,
			filled: func() bool { return len(manifest.Permissions) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[string](raw, pkg, "permissions")
				if err != nil {
					return err
				}
				manifest.Permissions = items
				return nil
			},
		},
		{
			name:   "hooks",
			raw:    includes.Hooks,
			filled: func() bool { return len(manifest.Hooks) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestHook](raw, pkg, "hooks")
				if err != nil {
					return err
				}
				manifest.Hooks = items
				return nil
			},
		},
		{
			name:   "providers",
			raw:    includes.Providers,
			filled: func() bool { return len(manifest.Providers) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestProvider](raw, pkg, "providers")
				if err != nil {
					return err
				}
				manifest.Providers = items
				return nil
			},
		},
		{
			name:   "adminPages",
			raw:    includes.AdminPages,
			filled: func() bool { return len(manifest.AdminPages) > 0 },
			apply: func(raw json.RawMessage) error {
				items, err := loadJSONShardList[ManifestAdminPage](raw, pkg, "adminPages")
				if err != nil {
					return err
				}
				manifest.AdminPages = items
				return nil
			},
		},
	}
	for _, slot := range slots {
		if len(slot.raw) == 0 || isJSONNull(slot.raw) {
			continue
		}
		if slot.filled() {
			return fmt.Errorf("%w: dual source for %s (root and includes)", ErrInvalidManifest, slot.name)
		}
		if err := slot.apply(slot.raw); err != nil {
			return fmt.Errorf("%w: includes.%s: %v", ErrInvalidManifest, slot.name, err)
		}
	}
	return applyV3Includes(manifest, includes, pkg)
}

func loadSettingsInclude(raw json.RawMessage, pkg PackageFS) (SettingsDocument, error) {
	paths, err := includePaths(raw, pkg, "settings")
	if err != nil {
		return SettingsDocument{}, err
	}
	documents := make([]SettingsDocument, 0, len(paths))
	for _, includePath := range paths {
		body, err := pkg.ReadFile(includePath)
		if err != nil {
			return SettingsDocument{}, fmt.Errorf("%w: read settings include %s: %v", ErrInvalidManifest, includePath, err)
		}
		document, err := decodeSettingsDocument(body)
		if err != nil {
			return SettingsDocument{}, fmt.Errorf("%w: decode settings include %s: %v", ErrInvalidManifest, includePath, err)
		}
		documents = append(documents, document)
	}
	return mergeSettingsDocuments(documents)
}

func includePaths(raw json.RawMessage, pkg PackageFS, label string) ([]string, error) {
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		if len(list) == 0 {
			return nil, fmt.Errorf("%s file list is empty", label)
		}
		paths := make([]string, 0, len(list))
		for _, item := range list {
			safe, err := safePackageRel(item)
			if err != nil {
				return nil, err
			}
			paths = append(paths, safe)
		}
		return paths, nil
	}
	var ref string
	if err := json.Unmarshal(raw, &ref); err != nil {
		return nil, fmt.Errorf("%s include must be a path string or path array", label)
	}
	safe, err := safePackageRel(ref)
	if err != nil {
		return nil, err
	}
	isDir, err := pkg.Stat(safe)
	if err != nil {
		return nil, fmt.Errorf("%s path %q: %w", label, safe, err)
	}
	if !isDir {
		return []string{safe}, nil
	}
	names, err := pkg.ReadDir(safe)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(names))
	for _, name := range names {
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			return nil, fmt.Errorf("%s directory %q contains non-json file %q", label, safe, name)
		}
		files = append(files, path.Join(safe, name))
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s directory %q is empty", label, safe)
	}
	sort.Strings(files)
	return files, nil
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

// loadLangsInclude 支持：目录路径字符串、单文件 map 路径、字符串数组文件列表。
func loadLangsInclude(raw json.RawMessage, pkg PackageFS) (map[string]ManifestLocale, error) {
	// 字符串数组：显式文件列表
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		if len(list) == 0 {
			return nil, fmt.Errorf("langs file list is empty")
		}
		return loadLangsFromFiles(list, pkg)
	}
	// 字符串：目录或单文件
	var ref string
	if err := json.Unmarshal(raw, &ref); err != nil {
		return nil, fmt.Errorf("langs include must be a path string, path array, or is invalid JSON")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("langs include path is empty")
	}
	safe, err := safePackageRel(ref)
	if err != nil {
		return nil, err
	}
	isDir, err := pkg.Stat(safe)
	if err != nil {
		return nil, fmt.Errorf("langs path %q: %w", safe, err)
	}
	if isDir {
		names, err := pkg.ReadDir(safe)
		if err != nil {
			return nil, err
		}
		files := make([]string, 0, len(names))
		for _, name := range names {
			// 目录模式只允许 *.json；其它文件直接失败，避免静默忽略草稿。
			if strings.HasPrefix(name, ".") {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(name), ".json") {
				return nil, fmt.Errorf("langs directory %q contains non-json file %q", safe, name)
			}
			files = append(files, path.Join(safe, name))
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("langs directory %q is empty", safe)
		}
		sort.Strings(files)
		return loadLangsFromFiles(files, pkg)
	}
	// 单文件：map[string]ManifestLocale
	body, err := pkg.ReadFile(safe)
	if err != nil {
		return nil, err
	}
	var langs map[string]ManifestLocale
	if err := json.Unmarshal(body, &langs); err != nil {
		return nil, fmt.Errorf("langs file %q: %w", safe, err)
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("langs file %q is empty", safe)
	}
	return langs, nil
}

func loadLangsFromFiles(files []string, pkg PackageFS) (map[string]ManifestLocale, error) {
	langs := make(map[string]ManifestLocale, len(files))
	for _, file := range files {
		safe, err := safePackageRel(file)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(strings.ToLower(safe), ".json") {
			return nil, fmt.Errorf("langs file %q must end with .json", safe)
		}
		locale := strings.TrimSuffix(path.Base(safe), path.Ext(safe))
		if normalizeLocaleKey(locale) == "" {
			return nil, fmt.Errorf("illegal locale filename %q", path.Base(safe))
		}
		// 用规范化 key 入库，与 normalizeManifestLangs 一致前先占位；Validate 会再规范化。
		key := strings.TrimSpace(locale)
		if _, exists := langs[key]; exists {
			return nil, fmt.Errorf("duplicate locale %q", key)
		}
		body, err := pkg.ReadFile(safe)
		if err != nil {
			return nil, err
		}
		var item ManifestLocale
		if err := json.Unmarshal(body, &item); err != nil {
			return nil, fmt.Errorf("langs file %q: %w", safe, err)
		}
		langs[key] = item
	}
	return langs, nil
}

// loadJSONShardList 支持：
// - 字符串路径 → 单文件 JSON 数组，或目录下多个 *.json 数组分片按文件名排序合并
// - 字符串数组 → 多个文件，每个文件为 JSON 数组，按顺序合并
func loadJSONShardList[T any](raw json.RawMessage, pkg PackageFS, label string) ([]T, error) {
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		if len(list) == 0 {
			return nil, fmt.Errorf("%s file list is empty", label)
		}
		return loadTypedArraysFromFiles[T](list, pkg, label)
	}
	var ref string
	if err := json.Unmarshal(raw, &ref); err != nil {
		return nil, fmt.Errorf("%s include must be a path string or path array", label)
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("%s include path is empty", label)
	}
	safe, err := safePackageRel(ref)
	if err != nil {
		return nil, err
	}
	isDir, err := pkg.Stat(safe)
	if err != nil {
		return nil, fmt.Errorf("%s path %q: %w", label, safe, err)
	}
	if isDir {
		names, err := pkg.ReadDir(safe)
		if err != nil {
			return nil, err
		}
		files := make([]string, 0, len(names))
		for _, name := range names {
			if strings.HasPrefix(name, ".") {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(name), ".json") {
				return nil, fmt.Errorf("%s directory %q contains non-json file %q", label, safe, name)
			}
			files = append(files, path.Join(safe, name))
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("%s directory %q is empty", label, safe)
		}
		sort.Strings(files)
		return loadTypedArraysFromFiles[T](files, pkg, label)
	}
	return loadTypedArraysFromFiles[T]([]string{safe}, pkg, label)
}

func loadTypedArraysFromFiles[T any](files []string, pkg PackageFS, label string) ([]T, error) {
	out := make([]T, 0)
	for _, file := range files {
		safe, err := safePackageRel(file)
		if err != nil {
			return nil, err
		}
		body, err := pkg.ReadFile(safe)
		if err != nil {
			return nil, fmt.Errorf("%s file %q: %w", label, safe, err)
		}
		var chunk []T
		if err := json.Unmarshal(body, &chunk); err != nil {
			return nil, fmt.Errorf("%s file %q: %w", label, safe, err)
		}
		out = append(out, chunk...)
	}
	return out, nil
}

func decodeIncludeObject(raw json.RawMessage, pkg PackageFS, target any) error {
	var ref string
	if err := json.Unmarshal(raw, &ref); err != nil {
		return fmt.Errorf("include must be a path string")
	}
	safe, err := safePackageRel(ref)
	if err != nil {
		return err
	}
	isDir, err := pkg.Stat(safe)
	if err != nil {
		return err
	}
	if isDir {
		return fmt.Errorf("expected file path, got directory %q", safe)
	}
	body, err := pkg.ReadFile(safe)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, target); err != nil {
		return err
	}
	return nil
}

func ensureUniqueSettingKeys(items []ManifestSetting) error {
	seen := map[string]struct{}{}
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate settings key %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func ensureUniqueContributionIDs(items []ManifestContribution) error {
	// 同一 point 下 id 必须唯一；跨 point 允许相同 id。
	seen := map[string]struct{}{}
	for _, item := range items {
		point := strings.TrimSpace(item.Point)
		id := strings.TrimSpace(item.ID)
		if point == "" || id == "" {
			continue
		}
		key := point + "\x00" + id
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate contribution %s/%s", point, id)
		}
		seen[key] = struct{}{}
	}
	return nil
}
