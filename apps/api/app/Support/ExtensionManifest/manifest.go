package extensionmanifest

import (
	"encoding/json"
	"errors"
	"net/mail"
	"net/url"
	"path"
	"regexp"
	"strings"

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
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	URL           string                 `json:"url"`
	Author        ManifestAuthor         `json:"author"`
	Version       string                 `json:"version"`
	Type          string                 `json:"type"`
	SForumVersion string                 `json:"sforumVersion"`
	Permissions   []string               `json:"permissions"`
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

type ManifestSetting struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
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

type TopicActionContributionPayload struct {
	Type    string `json:"type"`
	Method  string `json:"method"`
	Path    string `json:"path"`
	Confirm bool   `json:"confirm,omitempty"`
}

type ContributionPointDefinition struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	PayloadType string `json:"payloadType"`
}

func ContributionPointDefinitions() []ContributionPointDefinition {
	return []ContributionPointDefinition{{
		ID:          "forum.topic.actions",
		Owner:       "forum",
		Kind:        ContributionPointKindDescriptor,
		Description: "Topic detail action descriptors rendered by the host UI.",
		PayloadType: "extensionRoute",
	}}
}

func Validate(manifest Manifest) error {
	return ValidateWithContributionPoints(manifest, ContributionPointDefinitions())
}

func validateManifest(manifest Manifest, points []ContributionPointDefinition) error {
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
		if setting.Key == "" || setting.Label == "" || setting.Type == "" || strings.Contains(setting.Key, " ") {
			return ErrInvalidManifest
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
	for index := range manifest.Settings {
		manifest.Settings[index].Key = strings.TrimSpace(manifest.Settings[index].Key)
		manifest.Settings[index].Label = strings.TrimSpace(manifest.Settings[index].Label)
		manifest.Settings[index].Description = strings.TrimSpace(manifest.Settings[index].Description)
		manifest.Settings[index].Type = strings.ToLower(strings.TrimSpace(manifest.Settings[index].Type))
		manifest.Settings[index].Default = strings.TrimSpace(manifest.Settings[index].Default)
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
		len(manifest.Migrations) == 0 &&
		len(manifest.Routes) == 0 &&
		len(manifest.Hooks) == 0 &&
		len(manifest.Events) == 0 &&
		len(manifest.Jobs) == 0 &&
		len(manifest.Providers) == 0 &&
		len(manifest.Contributions) == 0
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
			if definition.PayloadType != "extensionRoute" {
				return ErrInvalidManifest
			}
			if err := validateTopicActionContributionPayload(contribution.Payload); err != nil {
				return err
			}
		case ContributionPointKindComponent:
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

func validateTopicActionContributionPayload(raw json.RawMessage) error {
	if len(raw) == 0 {
		return ErrInvalidManifest
	}
	var payload TopicActionContributionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ErrInvalidManifest
	}
	if payload.Type != "extensionRoute" {
		return ErrInvalidManifest
	}
	switch payload.Method {
	case "POST", "PUT", "PATCH", "DELETE":
	default:
		return ErrInvalidManifest
	}
	if !safeContributionRoutePath(payload.Path) {
		return ErrInvalidManifest
	}
	return nil
}

func safeContributionRoutePath(value string) bool {
	if value == "" || !strings.HasPrefix(value, "/") || value == "/" {
		return false
	}
	if strings.Contains(value, "://") || strings.Contains(value, "..") {
		return false
	}
	return value != "/api" && !strings.HasPrefix(value, "/api/")
}

func knownProviderSlot(slot string) bool {
	switch slot {
	case "search.provider", "attachment.storage.provider", "human_verification.provider", "auth.risk.provider", "editor.sanitizer.provider":
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
