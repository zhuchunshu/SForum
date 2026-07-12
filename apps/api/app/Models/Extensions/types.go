package extensions

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

const (
	ManifestFileName = extensionmanifest.ManifestFileName
	DefaultThemeID   = "sforum.default-theme"

	TypePlugin = extensionmanifest.TypePlugin
	TypeTheme  = extensionmanifest.TypeTheme

	StatusInstalled = "installed"
	StatusEnabled   = "enabled"
	StatusDisabled  = "disabled"

	RuntimeStopped  = "stopped"
	RuntimeStarting = "starting"
	RuntimeRunning  = "running"
	RuntimeFailed   = "failed"

	ThemeReleaseQueued     = "queued"
	ThemeReleaseBuilding   = "building"
	ThemeReleaseBuilt      = "built"
	ThemeReleaseActivating = "activating"
	ThemeReleaseActive     = "active"
	ThemeReleaseFailed     = "failed"
	ThemeReleaseRolledBack = "rolled_back"

	RouteAccessPublic     = extensionmanifest.RouteAccessPublic
	RouteAccessLogin      = extensionmanifest.RouteAccessLogin
	RouteAccessPermission = extensionmanifest.RouteAccessPermission

	EventInstalled             = "installed"
	EventBuiltinSynced         = "builtin_synced"
	EventVerified              = "verified"
	EventEnabled               = "enabled"
	EventEnableFailed          = "enable_failed"
	EventDisabled              = "disabled"
	EventThemeActivated        = "theme_activated"
	EventThemeActivationQueued = "theme_activation_queued"

	DeliveryQueued    = "queued"
	DeliveryRunning   = "running"
	DeliverySucceeded = "succeeded"
	DeliveryFailed    = "failed"
	DeliverySkipped   = "skipped"

	CodeInvalidArchive             = "extension.archive_invalid"
	CodeInvalidManifest            = "extension.manifest_invalid"
	CodeNotFound                   = "extension.not_found"
	CodePreflightFailed            = "extension.preflight_failed"
	CodeBuildFailed                = "extension.build_failed"
	CodeThemeActivationRequired    = "extension.theme_activation_required"
	CodeThemeRuntimeUnavailable    = "extension.theme_runtime_unavailable"
	CodeRouteNotFound              = "extension.route_not_found"
	CodeRouteMethodNotAllowed      = "extension.route_method_not_allowed"
	CodeRuntimeUnavailable         = "extension.runtime_unavailable"
	CodeRuntimeFailed              = "extension.runtime_failed"
	CodeFrontendRuntimeUnavailable = "extension.frontend_runtime_unavailable"
	CodeFrontendDigestInvalid      = "extension.frontend_digest_invalid"
	CodeFrontendTrustNotFound      = "extension.frontend_trust_not_found"
	CodeWebReleaseNotFound         = "extension.web_release_not_found"
	CodeWebReleaseConflict         = "extension.web_release_conflict"
	// 插件已禁用：设置读写、自定义管理页等功能性能力不可用。
	CodeExtensionDisabled = "extension.disabled"
	// 启用前需运营确认 capability 授权（F2.1）。
	CodeCapabilityConfirmationRequired = "extension.capability_confirmation_required"
	// 运行时缺少已授权 capability。
	CodeCapabilityDenied = "extension.capability_denied"

	SourceBuiltin  = "builtin"
	SourceUploaded = "uploaded"
)

var (
	ErrInvalidArchive          = errors.New("extensions: invalid archive")
	ErrInvalidManifest         = extensionmanifest.ErrInvalidManifest
	ErrExtensionNotFound       = errors.New("extensions: not found")
	ErrExtensionDisabled       = errors.New("extensions: disabled")
	ErrPreflightFailed         = errors.New("extensions: preflight failed")
	ErrBuildFailed             = errors.New("extensions: build failed")
	ErrThemeActivationRequired = errors.New("extensions: themes must be activated")
	ErrThemeRuntimeUnavailable = errors.New("extensions: theme activation runtime unavailable")
	ErrRuntimeFailed           = errors.New("extensions: runtime failed")
	ErrRouteNotFound           = errors.New("extensions: route not found")
	ErrRouteMethodNotAllowed   = errors.New("extensions: route method not allowed")
	ErrRuntimeUnavailable      = errors.New("extensions: runtime unavailable")
	// ErrCapabilityConfirmationRequired 启用插件前需 confirmCapabilities=true。
	ErrCapabilityConfirmationRequired = errors.New("extensions: capability confirmation required")
	ErrCapabilityDenied               = errors.New("extensions: capability denied")
)

type Manifest = extensionmanifest.Manifest
type ManifestAuthor = extensionmanifest.ManifestAuthor
type ManifestSetting = extensionmanifest.ManifestSetting
type ManifestSettingOption = extensionmanifest.ManifestSettingOption
type LocalizedText = extensionmanifest.LocalizedText
type ManifestMigration = extensionmanifest.ManifestMigration
type ManifestBackend = extensionmanifest.ManifestBackend
type ManifestFrontend = extensionmanifest.ManifestFrontend
type ManifestAdminFrontend = extensionmanifest.ManifestAdminFrontend
type ManifestAdmin = extensionmanifest.ManifestAdmin
type ManifestAdminPage = extensionmanifest.ManifestAdminPage
type ManifestRoute = extensionmanifest.ManifestRoute
type ManifestHook = extensionmanifest.ManifestHook
type ManifestEvent = extensionmanifest.ManifestEvent
type ManifestJob = extensionmanifest.ManifestJob
type ManifestProvider = extensionmanifest.ManifestProvider
type ManifestContribution = extensionmanifest.ManifestContribution
type TopicActionContributionPayload = extensionmanifest.TopicActionContributionPayload
type AdminComponentContributionPayload = extensionmanifest.AdminComponentContributionPayload
type ContributionPointDefinition = extensionmanifest.ContributionPointDefinition
type Dependency = extensionpackage.Dependency
type DependencySummary = extensionpackage.DependencySummary

type RuntimeStatus struct {
	State         string     `json:"state"`
	LastError     string     `json:"lastError,omitempty"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	RouteCount    int        `json:"routeCount"`
	HookCount     int        `json:"hookCount"`
	EventCount    int        `json:"eventCount"`
	ProviderCount int        `json:"providerCount"`
}

type MatchedRoute struct {
	Extension Extension
	Route     ManifestRoute
	Path      string
}

// CapabilityGrant 是启用审查用的能力条目（与 capabilities.Grant 对齐）。
type CapabilityGrant = capabilities.Grant

type Extension struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Type          string         `json:"type"`
	Status        string         `json:"status"`
	Source        string         `json:"source"`
	IsSystem      bool           `json:"isSystem"`
	IsDeletable   bool           `json:"isDeletable"`
	Manifest      Manifest       `json:"manifest"`
	// CapabilityGrants 有效 Host 能力（显式 + 推断），供启用审查 UI（F2.1）。
	CapabilityGrants []CapabilityGrant `json:"capabilityGrants,omitempty"`
	Runtime       *RuntimeStatus    `json:"runtime,omitempty"`
	ThemeRelease  *ThemeRelease     `json:"themeRelease,omitempty"`
	// WebRelease 为插件启停/信任变更排队的 live 或失败发布进度（主题仍用 themeRelease）。
	WebRelease    *WebReleaseSummary `json:"webRelease,omitempty"`
	PackageDigest string            `json:"packageDigest"`
	PackagePath   string            `json:"packagePath"`
	InstalledAt   time.Time         `json:"installedAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// EnableInput 启用插件的可选请求体。
type EnableInput struct {
	// ConfirmCapabilities 为 true 时表示运营已审阅并确认 capability 授权。
	ConfirmCapabilities bool `json:"confirmCapabilities"`
}

type ThemeRelease struct {
	ID               int64      `json:"id"`
	ExtensionID      string     `json:"extensionId"`
	ExtensionVersion string     `json:"extensionVersion"`
	Status           string     `json:"status"`
	LayerPath        string     `json:"layerPath"`
	ArtifactPath     string     `json:"artifactPath"`
	ServerEntry      string     `json:"serverEntry"`
	Message          string     `json:"message"`
	BuildLog         string     `json:"buildLog,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	ActivatedAt      *time.Time `json:"activatedAt,omitempty"`
}

type ThemeReleaseInput struct {
	ExtensionID string
	Version     string
	LayerPath   string
}

type ThemeReleaseUpdate struct {
	ID           int64
	Status       string
	ArtifactPath string
	ServerEntry  string
	Message      string
	BuildLog     string
}

type ExtensionEvent struct {
	ID          int64     `json:"id"`
	ExtensionID string    `json:"extensionId"`
	ActorUserID int64     `json:"actorUserId"`
	Action      string    `json:"action"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ExtensionAdminNavigationItem struct {
	ExtensionID     string `json:"extensionId"`
	ExtensionName   string `json:"extensionName"`
	ExtensionType   string `json:"extensionType"`
	ExtensionStatus string `json:"extensionStatus"`
	Path            string `json:"path"`
	Label           string `json:"label"`
	Description     string `json:"description"`
	Icon            string `json:"icon"`
	View            string `json:"view"`
	Order           int    `json:"order"`
}

type EffectiveContribution struct {
	ExtensionID   string            `json:"extensionId"`
	ExtensionName string            `json:"extensionName"`
	ExtensionType string            `json:"extensionType"`
	Point         string            `json:"point"`
	ID            string            `json:"id"`
	Order         int               `json:"order"`
	Label         map[string]string `json:"label,omitempty"`
	Icon          string            `json:"icon,omitempty"`
	Payload       json.RawMessage   `json:"payload,omitempty"`
}

// ExtensionSettingOption 是按请求 locale 解析后的选项展示值。
type ExtensionSettingOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type ExtensionSettingValue struct {
	Key              string                   `json:"key"`
	Label            string                   `json:"label"`
	Description      string                   `json:"description"`
	Type             string                   `json:"type"`
	Default          string                   `json:"default"`
	Value            string                   `json:"value"`
	Placeholder      string                   `json:"placeholder,omitempty"`
	RecommendedValue string                   `json:"recommendedValue,omitempty"`
	Group            string                   `json:"group,omitempty"`
	Options          []ExtensionSettingOption `json:"options,omitempty"`
	SecretSet        bool                     `json:"secretSet,omitempty"`
}

type ExtensionSettings struct {
	ExtensionID string                  `json:"extensionId"`
	Items       []ExtensionSettingValue `json:"items"`
}

type UpdateSettingsInput struct {
	Values map[string]string `json:"values"`
}

type ExtensionEventDelivery struct {
	ID            int64      `json:"id"`
	ExtensionID   string     `json:"extensionId"`
	EventName     string     `json:"eventName"`
	EventKind     string     `json:"eventKind"`
	Status        string     `json:"status"`
	Reason        string     `json:"reason"`
	Message       string     `json:"message"`
	CorrelationID string     `json:"correlationId"`
	AttemptCount  int        `json:"attemptCount"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}

type ArchiveInput struct {
	FileName string
	Data     []byte
}

type SaveInstalledInput struct {
	Manifest      Manifest
	PackagePath   string
	PackageDigest string
}

type SaveBuiltinInput struct {
	Manifest      Manifest
	PackagePath   string
	PackageDigest string
}

type EventInput struct {
	ExtensionID string
	ActorUserID int64
	Action      string
	Message     string
}

type EventDeliveryInput struct {
	ExtensionID   string
	EventName     string
	EventKind     string
	Status        string
	Reason        string
	Message       string
	CorrelationID string
}

type EventDeliveryUpdateInput struct {
	ID           int64
	Status       string
	Reason       string
	Message      string
	AttemptCount int
	Completed    bool
}

type EventDeliveryListInput struct {
	ExtensionID string
	EventName   string
	Status      string
	Limit       int
}
