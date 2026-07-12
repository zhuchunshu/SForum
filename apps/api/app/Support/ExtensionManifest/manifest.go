package extensionmanifest

import (
	"encoding/json"
	"errors"
	"net/mail"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

const (
	ManifestFileName = "sforum.extension.json"

	TypePlugin = "plugin"
	TypeTheme  = "theme"

	RouteAccessPublic     = "public"
	RouteAccessLogin      = "login"
	RouteAccessPermission = "permission"
)

var (
	ErrInvalidManifest = errors.New("extensions: invalid manifest")

	manifestIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,80}$`)
	// manifestVersionPattern 约束 Version 字段，防止 "../../../tmp/evil" 之类的路径穿越
	// 在 filepath.Join(extensionRoot, id, version) 时逃逸出 extensionRoot（C1）。
	// 允许语义化版本常见字符，禁止 / \ 与纯 . / .. 。
	manifestVersionPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+\-~]{0,63}$`)
)

type Manifest struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	URL           string         `json:"url"`
	Author        ManifestAuthor `json:"author"`
	Version       string         `json:"version"`
	Type          string         `json:"type"`
	SForumVersion string         `json:"sforumVersion"`
	// Langs 是可选的本地化覆盖。顶层 name/description/author 为默认英文；
	// 未声明 langs 时无需翻译，直接使用顶层字段。
	Langs       map[string]ManifestLocale `json:"langs,omitempty"`
	Permissions []string                  `json:"permissions"`
	// Capabilities 为插件声明的 Host 能力（F2.1）。主题必须为空。
	// 未声明时宿主仍会按 jobs/settings/providers/backend 推断最小集。
	Capabilities  []string               `json:"capabilities,omitempty"`
	Settings      []ManifestSetting      `json:"settings"`
	Migrations    []ManifestMigration    `json:"migrations"`
	Backend       ManifestBackend        `json:"backend"`
	Frontend      ManifestFrontend       `json:"frontend"`
	Admin         ManifestAdmin          `json:"admin"`
	AdminPages    []ManifestAdminPage    `json:"adminPages"`
	Routes        []ManifestRoute        `json:"routes"`
	Hooks         []ManifestHook         `json:"hooks"`
	Events        []ManifestEvent        `json:"events"`
	Jobs          []ManifestJob          `json:"jobs"`
	Providers     []ManifestProvider     `json:"providers"`
	Contributions []ManifestContribution `json:"contributions"`
}

type ManifestAuthor struct {
	Name  string `json:"name"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// ManifestLocale 覆盖可展示文案。字段均可选，有值才覆盖顶层默认。
type ManifestLocale struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	URL         string         `json:"url,omitempty"`
	Author      ManifestAuthor `json:"author,omitempty"`
}

type ManifestSetting struct {
	Key              string                  `json:"key"`
	Label            LocalizedText           `json:"label"`
	Description      LocalizedText           `json:"description,omitempty"`
	Type             string                  `json:"type"`
	Default          string                  `json:"default,omitempty"`
	Placeholder      LocalizedText           `json:"placeholder,omitempty"`
	RecommendedValue string                  `json:"recommendedValue,omitempty"`
	Group            LocalizedText           `json:"group,omitempty"`
	Options          []ManifestSettingOption `json:"options,omitempty"`
}

type ManifestSettingOption struct {
	Value       string        `json:"value"`
	Label       LocalizedText `json:"label"`
	Description LocalizedText `json:"description,omitempty"`
}

type ManifestMigration struct {
	Path string `json:"path"`
}

type ManifestBackend struct {
	Entry           string `json:"entry"`
	RPC             string `json:"rpc"`
	ProtocolVersion int    `json:"protocolVersion,omitempty"`
}

type ManifestFrontend struct {
	Layer string                 `json:"layer"`
	Admin *ManifestAdminFrontend `json:"admin,omitempty"`
}

type ManifestAdmin struct {
	Entry string              `json:"entry,omitempty"`
	Pages []ManifestAdminPage `json:"pages,omitempty"`
}

type ManifestAdminPage struct {
	Path        string `json:"path"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	View        string `json:"view,omitempty"`
	Menu        bool   `json:"menu,omitempty"`
	Order       int    `json:"order,omitempty"`
	Permission  string `json:"permission,omitempty"`
}

type ManifestRoute struct {
	Path       string   `json:"path"`
	Methods    []string `json:"methods"`
	Access     string   `json:"access,omitempty"`
	Permission string   `json:"permission,omitempty"`
	TimeoutMS  int      `json:"timeoutMs,omitempty"`
}

type ManifestHook struct {
	Name string `json:"name"`
}

type ManifestEvent struct {
	Name      string `json:"name"`
	Kind      string `json:"kind,omitempty"`
	TimeoutMS int    `json:"timeoutMs,omitempty"`
}

type ManifestJob struct {
	Name string `json:"name"`
}

type ManifestProvider struct {
	Slot      string `json:"slot"`
	Label     string `json:"label"`
	TimeoutMS int    `json:"timeoutMs,omitempty"`
}

type ManifestContribution struct {
	Point   string            `json:"point"`
	ID      string            `json:"id"`
	Order   int               `json:"order,omitempty"`
	Label   map[string]string `json:"label,omitempty"`
	Icon    string            `json:"icon,omitempty"`
	Payload json.RawMessage   `json:"payload,omitempty"`
}

// 稳定贡献点 ID（F4.3 起与目录同步；新增点必须改此处 + OpenAPI + 文档 regenerate）。
const (
	PointForumTopicActions     = "forum.topic.actions"
	PointForumComposerToolbar  = "forum.composer.toolbar"
	PointForumProfileTabs      = "forum.profile.tabs"
	PointAdminDashboardWidgets = "admin.dashboard.widgets"
	PointSystemHealthChecks    = "system.health.checks"
)

// 宿主拥有的 payloadType；禁止可执行 JSON。
const (
	PayloadTypeExtensionRoute   = "extensionRoute"
	PayloadTypeAdminComponent   = "adminComponent"
	PayloadTypeProfileSection   = "profileSection"
	PayloadTypeDashboardLink    = "dashboardLink"
	PayloadTypeHealthDescriptor = "healthDescriptor"
)

// TopicActionContributionPayload 用于 forum.topic.actions（extensionRoute）。
type TopicActionContributionPayload struct {
	Type    string `json:"type"`
	Method  string `json:"method"`
	Path    string `json:"path"`
	Confirm bool   `json:"confirm,omitempty"`
}

// ComposerToolbarContributionPayload 用于 forum.composer.toolbar（extensionRoute）。
// 与主题操作相同：宿主渲染按钮，执行走扩展路由代理。
type ComposerToolbarContributionPayload = TopicActionContributionPayload

// ProfileTabContributionPayload 用于 forum.profile.tabs（profileSection）。
// type=extensionRoute：走扩展路由；type=hostLink：仅允许站内相对路径（非 /api）。
type ProfileTabContributionPayload struct {
	Type   string `json:"type"`
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Href   string `json:"href,omitempty"`
}

// DashboardWidgetContributionPayload 用于 admin.dashboard.widgets（dashboardLink）。
// 仅允许管理端相对路由（以 / 开头，禁止外链与 /api）。
type DashboardWidgetContributionPayload struct {
	Type     string `json:"type"`
	Route    string `json:"route"`
	Severity string `json:"severity,omitempty"`
}

// HealthCheckContributionPayload 用于 system.health.checks（healthDescriptor）。
// 宿主在 /ready 中根据插件运行时状态贡献组件项；不在 ready 路径上调用插件 RPC。
// type=extensionRuntime：按该扩展 runtime.state 映射 ok/degraded/error。
// type=static：扩展已启用则固定 ok（仅声明「插件已加载」）。
type HealthCheckContributionPayload struct {
	Type      string `json:"type"`
	Component string `json:"component"`
	Required  bool   `json:"required,omitempty"`
}

type ContributionPointDefinition struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	PayloadType string `json:"payloadType"`
}

func ContributionPointDefinitions() []ContributionPointDefinition {
	return []ContributionPointDefinition{
		{ID: PointForumTopicActions, Owner: "forum", Kind: ContributionPointKindDescriptor, Description: "Topic detail action descriptors rendered by the host UI.", PayloadType: PayloadTypeExtensionRoute},
		{ID: PointForumComposerToolbar, Owner: "forum", Kind: ContributionPointKindDescriptor, Description: "Composer/editor toolbar actions rendered by the host UI; payload is an extensionRoute only.", PayloadType: PayloadTypeExtensionRoute},
		{ID: PointForumProfileTabs, Owner: "forum", Kind: ContributionPointKindDescriptor, Description: "Public profile tabs/sections rendered by the host UI (extensionRoute or hostLink).", PayloadType: PayloadTypeProfileSection},
		{ID: PointAdminDashboardWidgets, Owner: "admin", Kind: ContributionPointKindDescriptor, Description: "Admin dashboard link widgets; host-owned routes only, no executable payloads.", PayloadType: PayloadTypeDashboardLink},
		{ID: PointSystemHealthChecks, Owner: "system", Kind: ContributionPointKindDescriptor, Description: "Plugin readiness components merged into GET /ready without invoking plugin RPC.", PayloadType: PayloadTypeHealthDescriptor},
		{ID: "admin.jobs.table.columns", Owner: "jobs", Kind: ContributionPointKindComponent, Description: "Trusted client components rendered as job table columns.", PayloadType: PayloadTypeAdminComponent},
		{ID: "admin.jobs.row.actions", Owner: "jobs", Kind: ContributionPointKindComponent, Description: "Trusted client components rendered beside core job actions.", PayloadType: PayloadTypeAdminComponent},
		{ID: "admin.jobs.detail.sections", Owner: "jobs", Kind: ContributionPointKindComponent, Description: "Trusted client components rendered in job detail.", PayloadType: PayloadTypeAdminComponent},
		// 扩展设置页：默认由宿主渲染 manifest settings；插件可替换整页或注入页眉/页脚。
		{ID: "admin.extension.settings.page", Owner: "extensions", Kind: ContributionPointKindComponent, Description: "Trusted client component that replaces the host-rendered extension settings form for the owning extension.", PayloadType: PayloadTypeAdminComponent},
		{ID: "admin.extension.settings.header", Owner: "extensions", Kind: ContributionPointKindComponent, Description: "Trusted client components rendered above the host-rendered extension settings form.", PayloadType: PayloadTypeAdminComponent},
		{ID: "admin.extension.settings.footer", Owner: "extensions", Kind: ContributionPointKindComponent, Description: "Trusted client components rendered below the host-rendered extension settings form.", PayloadType: PayloadTypeAdminComponent},
	}
}

func Validate(manifest Manifest) error {
	return ValidateWithContributionPoints(manifest, ContributionPointDefinitions())
}

func validateManifest(manifest Manifest, points []ContributionPointDefinition) error {
	// langs 在 Normalize 丢弃空项前先校验，避免无效语言码被静默忽略。
	if err := validateManifestLangs(manifest.Langs); err != nil {
		return err
	}
	manifest = Normalize(manifest)
	if !manifestIDPattern.MatchString(manifest.ID) {
		return ErrInvalidManifest
	}
	if manifest.Name == "" || manifest.Description == "" || manifest.URL == "" || manifest.Author.Name == "" || manifest.Version == "" || manifest.SForumVersion == "" {
		return ErrInvalidManifest
	}
	// C1：Version 严格约束，防止路径穿越逃逸 extensionRoot。
	if !manifestVersionPattern.MatchString(manifest.Version) {
		return ErrInvalidManifest
	}
	if !validHTTPURL(manifest.URL) || (manifest.Author.URL != "" && !validHTTPURL(manifest.Author.URL)) {
		return ErrInvalidManifest
	}
	if manifest.Author.Email != "" {
		if _, err := mail.ParseAddress(manifest.Author.Email); err != nil {
			return ErrInvalidManifest
		}
	}
	if manifest.Type != TypePlugin && manifest.Type != TypeTheme {
		return ErrInvalidManifest
	}
	for _, setting := range manifest.Settings {
		// label 支持纯字符串或多语言 map，校验时只要求解析后非空。
		if setting.Key == "" || setting.Label.IsEmpty() || setting.Type == "" || strings.Contains(setting.Key, " ") {
			return ErrInvalidManifest
		}
		optionValues := make(map[string]struct{}, len(setting.Options))
		for _, option := range setting.Options {
			if option.Value == "" || option.Label.IsEmpty() {
				return ErrInvalidManifest
			}
			if _, exists := optionValues[option.Value]; exists {
				return ErrInvalidManifest
			}
			optionValues[option.Value] = struct{}{}
		}
		if setting.RecommendedValue != "" && len(optionValues) > 0 {
			if _, exists := optionValues[setting.RecommendedValue]; !exists {
				return ErrInvalidManifest
			}
		}
	}
	if err := validateAdminDeclaration(manifest); err != nil {
		return err
	}
	if manifest.Type == TypeTheme && !isThemeManifestSupported(manifest) {
		return ErrInvalidManifest
	}
	if manifest.Backend.Entry != "" {
		if _, ok := SafeArchivePath(manifest.Backend.Entry); !ok {
			return ErrInvalidManifest
		}
	}
	if manifest.Backend.RPC != "" && manifest.Backend.RPC != "hashicorp-go-plugin" {
		return ErrInvalidManifest
	}
	if manifest.Backend.ProtocolVersion < 0 || manifest.Backend.ProtocolVersion > 1 {
		return ErrInvalidManifest
	}
	if manifest.Frontend.Layer != "" {
		if _, ok := SafeArchivePath(manifest.Frontend.Layer); !ok {
			return ErrInvalidManifest
		}
	}
	if err := validateAdminFrontend(manifest); err != nil {
		return err
	}
	for _, migration := range manifest.Migrations {
		if _, ok := SafeArchivePath(migration.Path); !ok || !strings.HasSuffix(migration.Path, ".sql") {
			return ErrInvalidManifest
		}
	}
	for _, route := range manifest.Routes {
		if route.Path == "" || !strings.HasPrefix(route.Path, "/") || strings.Contains(route.Path, "..") {
			return ErrInvalidManifest
		}
		access := route.Access
		if access == "" {
			access = RouteAccessLogin
		}
		if access != RouteAccessPublic && access != RouteAccessLogin && access != RouteAccessPermission {
			return ErrInvalidManifest
		}
		if len(route.Methods) == 0 {
			return ErrInvalidManifest
		}
		for _, method := range route.Methods {
			switch method {
			case "GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE":
			default:
				return ErrInvalidManifest
			}
			if access == RouteAccessPublic && method != "GET" && method != "HEAD" && method != "OPTIONS" {
				return ErrInvalidManifest
			}
		}
		if access == RouteAccessPermission && (route.Permission == "" || !manifestHasPermission(manifest, route.Permission)) {
			return ErrInvalidManifest
		}
		if route.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
	for _, hook := range manifest.Hooks {
		if !appevents.Known(hook.Name) {
			return ErrInvalidManifest
		}
	}
	seenEvents := map[string]bool{}
	for _, event := range DeclaredEvents(manifest) {
		definition, ok := appevents.FindDefinition(event.Name)
		if !ok {
			return ErrInvalidManifest
		}
		kind := event.Kind
		if kind == "" {
			kind = definition.Kind
		}
		if kind != definition.Kind {
			return ErrInvalidManifest
		}
		if event.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
		key := event.Name + ":" + kind
		if seenEvents[key] {
			return ErrInvalidManifest
		}
		seenEvents[key] = true
	}
	for _, provider := range manifest.Providers {
		if provider.Label == "" || !knownProviderSlot(provider.Slot) || provider.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
	// F2.1：capabilities 必须落在宿主目录内；主题禁止声明。
	if len(manifest.Capabilities) > 0 {
		if manifest.Type == TypeTheme {
			return ErrInvalidManifest
		}
		if err := capabilities.ValidateKeys(manifest.Capabilities); err != nil {
			return ErrInvalidManifest
		}
	}
	if err := validateContributions(manifest, points); err != nil {
		return err
	}
	return nil
}

func Normalize(manifest Manifest) Manifest {
	manifest.ID = NormalizeID(manifest.ID)
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Description = strings.TrimSpace(manifest.Description)
	manifest.URL = strings.TrimSpace(manifest.URL)
	manifest.Author.Name = strings.TrimSpace(manifest.Author.Name)
	manifest.Author.URL = strings.TrimSpace(manifest.Author.URL)
	manifest.Author.Email = strings.TrimSpace(manifest.Author.Email)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Type = strings.ToLower(strings.TrimSpace(manifest.Type))
	manifest.SForumVersion = strings.TrimSpace(manifest.SForumVersion)
	manifest.Langs = normalizeManifestLangs(manifest.Langs)
	manifest.Capabilities = capabilities.NormalizeKeys(manifest.Capabilities)
	for index := range manifest.Settings {
		manifest.Settings[index].Key = strings.TrimSpace(manifest.Settings[index].Key)
		manifest.Settings[index].Label = manifest.Settings[index].Label.normalized()
		manifest.Settings[index].Description = manifest.Settings[index].Description.normalized()
		manifest.Settings[index].Type = strings.ToLower(strings.TrimSpace(manifest.Settings[index].Type))
		manifest.Settings[index].Default = strings.TrimSpace(manifest.Settings[index].Default)
		manifest.Settings[index].Placeholder = manifest.Settings[index].Placeholder.normalized()
		manifest.Settings[index].RecommendedValue = strings.TrimSpace(manifest.Settings[index].RecommendedValue)
		manifest.Settings[index].Group = manifest.Settings[index].Group.normalized()
		for optionIndex := range manifest.Settings[index].Options {
			option := &manifest.Settings[index].Options[optionIndex]
			option.Value = strings.TrimSpace(option.Value)
			option.Label = option.Label.normalized()
			option.Description = option.Description.normalized()
		}
	}
	manifest.Backend.Entry = strings.TrimSpace(manifest.Backend.Entry)
	manifest.Backend.RPC = strings.TrimSpace(manifest.Backend.RPC)
	if manifest.Backend.ProtocolVersion == 0 && manifest.Backend.RPC != "" {
		manifest.Backend.ProtocolVersion = 1
	}
	manifest.Frontend.Layer = strings.TrimSpace(manifest.Frontend.Layer)
	manifest.Frontend.Admin = normalizeAdminFrontend(manifest.Frontend.Admin)
	manifest.Admin.Entry = NormalizeRoutePath(manifest.Admin.Entry)
	normalizeAdminPageSlice(manifest.Admin.Pages)
	normalizeAdminPageSlice(manifest.AdminPages)
	for index := range manifest.Routes {
		manifest.Routes[index].Path = NormalizeRoutePath(manifest.Routes[index].Path)
		manifest.Routes[index].Access = strings.ToLower(strings.TrimSpace(manifest.Routes[index].Access))
		manifest.Routes[index].Permission = strings.TrimSpace(manifest.Routes[index].Permission)
		for methodIndex := range manifest.Routes[index].Methods {
			manifest.Routes[index].Methods[methodIndex] = strings.ToUpper(strings.TrimSpace(manifest.Routes[index].Methods[methodIndex]))
		}
	}
	for index := range manifest.Hooks {
		manifest.Hooks[index].Name = strings.TrimSpace(manifest.Hooks[index].Name)
	}
	for index := range manifest.Events {
		manifest.Events[index].Name = strings.TrimSpace(manifest.Events[index].Name)
		manifest.Events[index].Kind = strings.ToLower(strings.TrimSpace(manifest.Events[index].Kind))
		if manifest.Events[index].Kind == "" {
			if definition, ok := appevents.FindDefinition(manifest.Events[index].Name); ok {
				manifest.Events[index].Kind = definition.Kind
			}
		}
	}
	for index := range manifest.Providers {
		manifest.Providers[index].Slot = strings.TrimSpace(manifest.Providers[index].Slot)
		manifest.Providers[index].Label = strings.TrimSpace(manifest.Providers[index].Label)
	}
	for index := range manifest.Contributions {
		manifest.Contributions[index] = normalizeContribution(manifest.Contributions[index])
	}
	return manifest
}

func normalizeAdminPageSlice(pages []ManifestAdminPage) {
	for index := range pages {
		pages[index].Path = NormalizeRoutePath(pages[index].Path)
		pages[index].Label = strings.TrimSpace(pages[index].Label)
		pages[index].Description = strings.TrimSpace(pages[index].Description)
		pages[index].Icon = strings.TrimSpace(pages[index].Icon)
		pages[index].View = strings.ToLower(strings.TrimSpace(pages[index].View))
		if pages[index].View == "" {
			pages[index].View = "about"
		}
		pages[index].Permission = strings.TrimSpace(pages[index].Permission)
	}
}

func validateAdminDeclaration(manifest Manifest) error {
	pages := EffectiveAdminPages(manifest)
	for _, page := range pages {
		if page.Path == "" || !strings.HasPrefix(page.Path, "/") || strings.Contains(page.Path, "..") || page.Label == "" {
			return ErrInvalidManifest
		}
		if page.View != "" && page.View != "about" && page.View != "settings" {
			return ErrInvalidManifest
		}
		if page.Order < 0 {
			return ErrInvalidManifest
		}
	}
	if manifest.Admin.Entry == "" {
		return nil
	}
	if strings.Contains(manifest.Admin.Entry, "://") || !strings.HasPrefix(manifest.Admin.Entry, "/") || strings.Contains(manifest.Admin.Entry, "..") {
		return ErrInvalidManifest
	}
	if manifest.Admin.Entry == "/about" {
		return nil
	}
	for _, page := range pages {
		if page.Path == manifest.Admin.Entry {
			return nil
		}
	}
	return ErrInvalidManifest
}

func EffectiveAdminPages(manifest Manifest) []ManifestAdminPage {
	manifest = Normalize(manifest)
	if len(manifest.Admin.Pages) > 0 {
		return manifest.Admin.Pages
	}
	return manifest.AdminPages
}

func MenuAdminPages(manifest Manifest) []ManifestAdminPage {
	pages := EffectiveAdminPages(manifest)
	menuPages := make([]ManifestAdminPage, 0, len(pages))
	for _, page := range pages {
		if page.Menu {
			menuPages = append(menuPages, page)
		}
	}
	return menuPages
}

func AdminManagePath(manifest Manifest) string {
	manifest = Normalize(manifest)
	pages := EffectiveAdminPages(manifest)
	if manifest.Admin.Entry != "" {
		return manifest.Admin.Entry
	}
	for _, page := range pages {
		if page.Path == "/settings" {
			return page.Path
		}
	}
	if len(pages) > 0 {
		return pages[0].Path
	}
	return "/about"
}

func NormalizeID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

// LocalizedDisplay 按 locale 解析可展示字段；无 langs 或未命中时回退顶层默认英文。
func LocalizedDisplay(manifest Manifest, locale string) ManifestLocale {
	manifest = Normalize(manifest)
	display := ManifestLocale{
		Name:        manifest.Name,
		Description: manifest.Description,
		URL:         manifest.URL,
		Author:      manifest.Author,
	}
	override, ok := lookupManifestLocale(manifest.Langs, locale)
	if !ok {
		return display
	}
	if override.Name != "" {
		display.Name = override.Name
	}
	if override.Description != "" {
		display.Description = override.Description
	}
	if override.URL != "" {
		display.URL = override.URL
	}
	if override.Author.Name != "" {
		display.Author.Name = override.Author.Name
	}
	if override.Author.URL != "" {
		display.Author.URL = override.Author.URL
	}
	if override.Author.Email != "" {
		display.Author.Email = override.Author.Email
	}
	return display
}

func normalizeManifestLangs(langs map[string]ManifestLocale) map[string]ManifestLocale {
	if len(langs) == 0 {
		return nil
	}
	normalized := make(map[string]ManifestLocale, len(langs))
	for key, locale := range langs {
		code := normalizeLocaleKey(key)
		if code == "" {
			continue
		}
		locale.Name = strings.TrimSpace(locale.Name)
		locale.Description = strings.TrimSpace(locale.Description)
		locale.URL = strings.TrimSpace(locale.URL)
		locale.Author.Name = strings.TrimSpace(locale.Author.Name)
		locale.Author.URL = strings.TrimSpace(locale.Author.URL)
		locale.Author.Email = strings.TrimSpace(locale.Author.Email)
		// 空覆盖无意义，直接丢弃。
		if locale.Name == "" && locale.Description == "" && locale.URL == "" && locale.Author.Name == "" && locale.Author.URL == "" && locale.Author.Email == "" {
			continue
		}
		normalized[code] = locale
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func validateManifestLangs(langs map[string]ManifestLocale) error {
	if len(langs) == 0 {
		return nil
	}
	for key, locale := range langs {
		if normalizeLocaleKey(key) == "" {
			return ErrInvalidManifest
		}
		name := strings.TrimSpace(locale.Name)
		description := strings.TrimSpace(locale.Description)
		localeURL := strings.TrimSpace(locale.URL)
		authorName := strings.TrimSpace(locale.Author.Name)
		authorURL := strings.TrimSpace(locale.Author.URL)
		authorEmail := strings.TrimSpace(locale.Author.Email)
		// 局部字段可选；整段覆盖不能全空。
		if name == "" && description == "" && localeURL == "" && authorName == "" && authorURL == "" && authorEmail == "" {
			return ErrInvalidManifest
		}
		if localeURL != "" && !validHTTPURL(localeURL) {
			return ErrInvalidManifest
		}
		if authorURL != "" && !validHTTPURL(authorURL) {
			return ErrInvalidManifest
		}
		if authorEmail != "" {
			if _, err := mail.ParseAddress(authorEmail); err != nil {
				return ErrInvalidManifest
			}
		}
	}
	return nil
}

func lookupManifestLocale(langs map[string]ManifestLocale, locale string) (ManifestLocale, bool) {
	if len(langs) == 0 {
		return ManifestLocale{}, false
	}
	for _, candidate := range localeLookupCandidates(locale) {
		if item, ok := langs[candidate]; ok {
			return item, true
		}
	}
	return ManifestLocale{}, false
}

func localeLookupCandidates(locale string) []string {
	code := normalizeLocaleKey(locale)
	if code == "" {
		return nil
	}
	candidates := []string{code}
	if primary, _, ok := strings.Cut(code, "-"); ok && primary != "" && primary != code {
		candidates = append(candidates, primary)
	}
	return candidates
}

func normalizeLocaleKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "_", "-")
	parts := strings.Split(value, "-")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	for index := 1; index < len(parts); index++ {
		// 地区码常见为大写（CN/US），语言码小写。
		if len(parts[index]) == 2 {
			parts[index] = strings.ToUpper(parts[index])
		} else {
			parts[index] = strings.ToLower(parts[index])
		}
	}
	return strings.Join(parts, "-")
}

func NormalizeRoutePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ""
	}
	if strings.Contains(value, "..") {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return path.Clean(value)
}

func SafeArchivePath(name string) (string, bool) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || strings.HasPrefix(name, "/") {
		return "", false
	}
	clean := path.Clean(name)
	if clean == "." || clean == ManifestFileName {
		return clean, true
	}
	if strings.HasPrefix(clean, "../") || clean == ".." || strings.Contains(clean, "/../") {
		return "", false
	}
	return clean, true
}

func DeclaredEvents(manifest Manifest) []ManifestEvent {
	items := []ManifestEvent{}
	seen := map[string]bool{}
	for _, event := range manifest.Events {
		name := strings.TrimSpace(event.Name)
		if name == "" {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(event.Kind))
		if kind == "" {
			if definition, ok := appevents.FindDefinition(name); ok {
				kind = definition.Kind
			}
		}
		key := name + ":" + kind
		if seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, ManifestEvent{Name: name, Kind: kind, TimeoutMS: event.TimeoutMS})
	}
	// 旧 hooks 字段作为 events 的兼容别名保留，统一转换给运行时消费。
	for _, hook := range manifest.Hooks {
		name := strings.TrimSpace(hook.Name)
		if name == "" {
			continue
		}
		definition, ok := appevents.FindDefinition(name)
		if !ok {
			continue
		}
		key := name + ":" + definition.Kind
		if seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, ManifestEvent{Name: name, Kind: definition.Kind})
	}
	return items
}

func isThemeManifestSupported(manifest Manifest) bool {
	if strings.TrimSpace(manifest.Frontend.Layer) == "" {
		return false
	}
	return manifest.Backend == (ManifestBackend{}) &&
		len(manifest.Permissions) == 0 &&
		len(manifest.Capabilities) == 0 &&
		len(manifest.Migrations) == 0 &&
		len(manifest.Routes) == 0 &&
		len(manifest.Hooks) == 0 &&
		len(manifest.Events) == 0 &&
		len(manifest.Jobs) == 0 &&
		len(manifest.Providers) == 0 &&
		len(manifest.Contributions) == 0
}

// CapabilityResolveInput 将 manifest 转为 capabilities 解析输入（F2.1）。
func CapabilityResolveInput(manifest Manifest) capabilities.ResolveInput {
	slots := make([]string, 0, len(manifest.Providers))
	for _, provider := range manifest.Providers {
		slots = append(slots, provider.Slot)
	}
	return capabilities.ResolveInput{
		Explicit:      manifest.Capabilities,
		HasJobs:       len(manifest.Jobs) > 0,
		HasSettings:   len(manifest.Settings) > 0,
		ProviderSlots: slots,
		HasBackend:    strings.TrimSpace(manifest.Backend.Entry) != "",
	}
}

// ResolvedCapabilities 返回有效能力 key 与 implied 标记。
func ResolvedCapabilities(manifest Manifest) (keys []string, implied map[string]bool) {
	return capabilities.Resolve(CapabilityResolveInput(manifest))
}

// CapabilityGrants 返回启用审查用的能力列表。
func CapabilityGrants(manifest Manifest) []capabilities.Grant {
	return capabilities.GrantsFor(CapabilityResolveInput(manifest))
}

func normalizeContribution(contribution ManifestContribution) ManifestContribution {
	contribution.Point = strings.TrimSpace(contribution.Point)
	contribution.ID = NormalizeID(contribution.ID)
	contribution.Icon = strings.TrimSpace(contribution.Icon)
	if contribution.Label != nil {
		labels := make(map[string]string, len(contribution.Label))
		for locale, value := range contribution.Label {
			locale = strings.TrimSpace(locale)
			value = strings.TrimSpace(value)
			if locale != "" && value != "" {
				labels[locale] = value
			}
		}
		contribution.Label = labels
	}
	if contribution.Point == "forum.topic.actions" && len(contribution.Payload) > 0 {
		var payload TopicActionContributionPayload
		if err := json.Unmarshal(contribution.Payload, &payload); err == nil {
			payload.Type = strings.TrimSpace(payload.Type)
			payload.Method = strings.ToUpper(strings.TrimSpace(payload.Method))
			payload.Path = strings.TrimSpace(strings.ReplaceAll(payload.Path, "\\", "/"))
			if !strings.Contains(payload.Path, "://") {
				payload.Path = NormalizeRoutePath(payload.Path)
			}
			if normalized, err := json.Marshal(payload); err == nil {
				contribution.Payload = normalized
			}
		}
	}
	if normalized, ok := normalizeAdminComponentPayload(contribution.Payload); ok {
		contribution.Payload = normalized
	}
	return contribution
}

func validateContributions(manifest Manifest, definitions []ContributionPointDefinition) error {
	points := make(map[string]ContributionPointDefinition, len(definitions))
	for _, definition := range definitions {
		if definition.ID == "" || (definition.Kind != ContributionPointKindDescriptor && definition.Kind != ContributionPointKindComponent) {
			return ErrInvalidManifest
		}
		if _, duplicate := points[definition.ID]; duplicate {
			return ErrInvalidManifest
		}
		points[definition.ID] = definition
	}

	seen := map[string]bool{}
	componentReferences := map[string]int{}
	for _, contribution := range manifest.Contributions {
		definition, known := points[contribution.Point]
		if contribution.Point == "" || contribution.ID == "" || !known {
			return ErrInvalidManifest
		}
		key := contribution.Point + ":" + contribution.ID
		if seen[key] {
			return ErrInvalidManifest
		}
		seen[key] = true
		if contribution.Order < 0 {
			return ErrInvalidManifest
		}
		if contribution.Icon != "" && !allowedContributionIcon(contribution.Icon) {
			return ErrInvalidManifest
		}
		if len(contribution.Label) == 0 {
			return ErrInvalidManifest
		}
		for locale, label := range contribution.Label {
			if strings.TrimSpace(locale) == "" || strings.TrimSpace(label) == "" {
				return ErrInvalidManifest
			}
		}
		switch definition.Kind {
		case ContributionPointKindDescriptor:
			if err := validateDescriptorContributionPayload(definition.PayloadType, contribution.Payload); err != nil {
				return err
			}
		case ContributionPointKindComponent:
			if definition.PayloadType != PayloadTypeAdminComponent {
				return ErrInvalidManifest
			}
			component, err := adminComponentBinding(contribution.Payload)
			if err != nil || manifest.Frontend.Admin == nil {
				return ErrInvalidManifest
			}
			if _, exists := manifest.Frontend.Admin.Components[component]; !exists {
				return ErrInvalidManifest
			}
			componentReferences[component]++
			if componentReferences[component] > 1 {
				return ErrInvalidManifest
			}
		default:
			return ErrInvalidManifest
		}
	}
	return validateAdminComponentReferences(manifest, componentReferences)
}

func allowedContributionIcon(icon string) bool {
	return strings.HasPrefix(icon, "i-lucide-") || strings.HasPrefix(icon, "i-tabler-")
}

// validateDescriptorContributionPayload 按 payloadType 校验宿主拥有的描述符（F4.3）。
func validateDescriptorContributionPayload(payloadType string, raw json.RawMessage) error {
	switch payloadType {
	case PayloadTypeExtensionRoute:
		return validateTopicActionContributionPayload(raw)
	case PayloadTypeProfileSection:
		return validateProfileTabContributionPayload(raw)
	case PayloadTypeDashboardLink:
		return validateDashboardWidgetContributionPayload(raw)
	case PayloadTypeHealthDescriptor:
		return validateHealthCheckContributionPayload(raw)
	default:
		return ErrInvalidManifest
	}
}

func validateTopicActionContributionPayload(raw json.RawMessage) error {
	if len(raw) == 0 {
		return ErrInvalidManifest
	}
	var payload TopicActionContributionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ErrInvalidManifest
	}
	if payload.Type != PayloadTypeExtensionRoute {
		return ErrInvalidManifest
	}
	// 主题动作与 composer 工具均允许安全写方法（GET 禁止，避免 CSRF 风格误触发）。
	switch strings.ToUpper(strings.TrimSpace(payload.Method)) {
	case "POST", "PUT", "PATCH", "DELETE":
	default:
		return ErrInvalidManifest
	}
	if !safeContributionRoutePath(payload.Path) {
		return ErrInvalidManifest
	}
	return nil
}

func validateProfileTabContributionPayload(raw json.RawMessage) error {
	if len(raw) == 0 {
		return ErrInvalidManifest
	}
	var payload ProfileTabContributionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ErrInvalidManifest
	}
	switch strings.TrimSpace(payload.Type) {
	case PayloadTypeExtensionRoute:
		method := strings.ToUpper(strings.TrimSpace(payload.Method))
		// 资料页 tab 导航以 GET 为主；仍允许写方法以便插件做「关注」类动作。
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			return ErrInvalidManifest
		}
		if !safeContributionRoutePath(payload.Path) {
			return ErrInvalidManifest
		}
		return nil
	case "hostLink":
		if !safeHostLinkPath(payload.Href) {
			return ErrInvalidManifest
		}
		return nil
	default:
		return ErrInvalidManifest
	}
}

func validateDashboardWidgetContributionPayload(raw json.RawMessage) error {
	if len(raw) == 0 {
		return ErrInvalidManifest
	}
	var payload DashboardWidgetContributionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ErrInvalidManifest
	}
	if strings.TrimSpace(payload.Type) != "adminLink" {
		return ErrInvalidManifest
	}
	if !safeAdminDashboardRoute(payload.Route) {
		return ErrInvalidManifest
	}
	switch strings.TrimSpace(payload.Severity) {
	case "", "info", "success", "warning", "danger":
	default:
		return ErrInvalidManifest
	}
	return nil
}

func validateHealthCheckContributionPayload(raw json.RawMessage) error {
	if len(raw) == 0 {
		return ErrInvalidManifest
	}
	var payload HealthCheckContributionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ErrInvalidManifest
	}
	switch strings.TrimSpace(payload.Type) {
	case "extensionRuntime", "static":
	default:
		return ErrInvalidManifest
	}
	component := strings.TrimSpace(payload.Component)
	// 组件名：小写字母数字与 ._-: ，避免与 core 名冲突时仍可读。
	if component == "" || len(component) > 80 {
		return ErrInvalidManifest
	}
	for _, r := range component {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == ':' {
			continue
		}
		return ErrInvalidManifest
	}
	return nil
}

func safeContributionRoutePath(value string) bool {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || !strings.HasPrefix(value, "/") || value == "/" {
		return false
	}
	if strings.Contains(value, "://") || strings.Contains(value, "..") {
		return false
	}
	return value != "/api" && !strings.HasPrefix(value, "/api/")
}

// safeHostLinkPath 仅允许站内相对路径（公开页），禁止协议相对 // 与 /api。
func safeHostLinkPath(value string) bool {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	if strings.Contains(value, "://") || strings.Contains(value, "..") {
		return false
	}
	return value != "/api" && !strings.HasPrefix(value, "/api/")
}

// safeAdminDashboardRoute 管理端相对路由（admin shell 内 path）。
func safeAdminDashboardRoute(value string) bool {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	if strings.Contains(value, "://") || strings.Contains(value, "..") {
		return false
	}
	// 禁止跳出到公开 API 或绝对 URL。
	if value == "/api" || strings.HasPrefix(value, "/api/") {
		return false
	}
	return true
}

func knownProviderSlot(slot string) bool {
	switch slot {
	case "mail.provider", "search.provider", "attachment.storage.provider", "human_verification.provider", "auth.risk.provider", "editor.sanitizer.provider":
		return true
	default:
		return false
	}
}

func manifestHasPermission(manifest Manifest, permission string) bool {
	for _, item := range manifest.Permissions {
		if strings.TrimSpace(item) == permission {
			return true
		}
	}
	return false
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
