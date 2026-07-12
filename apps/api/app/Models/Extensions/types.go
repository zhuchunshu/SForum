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
	// RuntimeDegraded 进程仍在，但熔断打开或近期连续失败（F2.3）。
	RuntimeDegraded = "degraded"
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
	EventUpgraded              = "upgraded"
	EventUninstalled           = "uninstalled"
	EventBuiltinSynced         = "builtin_synced"
	EventVerified              = "verified"
	EventEnabled               = "enabled"
	EventEnableFailed          = "enable_failed"
	EventDisabled              = "disabled"
	EventThemeActivated        = "theme_activated"
	EventThemeActivationQueued = "theme_activation_queued"
	EventMigrationsApplied     = "migrations_applied"

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
	// 设置已写入但插件重启失败，且旧设置恢复也失败，需要运营介入。
	CodeSettingsRollbackFailed = "extension.settings_rollback_failed"
	// 启用前需运营确认 capability 授权（F2.1）。
	CodeCapabilityConfirmationRequired = "extension.capability_confirmation_required"
	// CodeFeaturesRequired F4.5：站点产品开关未满足 requiresFeatures。
	CodeFeaturesRequired = "extension.features_required"
	// 运行时缺少已授权 capability。
	CodeCapabilityDenied = "extension.capability_denied"
	// 系统/内置扩展不可卸载。
	CodeNotDeletable = "extension.not_deletable"
	// 卸载前必须先禁用（或 drain 失败）。
	CodeMustDisableFirst = "extension.must_disable_first"
	// 插件迁移执行失败。
	CodeMigrationFailed = "extension.migration_failed"
	// CodeUntrustedBackendRestricted 见 backend_trust.go（非 super_admin 执行非内置后端）。

	SourceBuiltin  = "builtin"
	SourceUploaded = "uploaded"
)

var (
	ErrInvalidArchive          = errors.New("extensions: invalid archive")
	ErrInvalidManifest         = extensionmanifest.ErrInvalidManifest
	ErrExtensionNotFound       = errors.New("extensions: not found")
	ErrExtensionDisabled       = errors.New("extensions: disabled")
	ErrSettingsRollbackFailed  = errors.New("extensions: settings rollback failed")
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
	// ErrFeaturesRequired 站点产品开关未满足 manifest requiresFeatures（F4.5）。
	ErrFeaturesRequired = errors.New("extensions: required features disabled")
	ErrNotDeletable     = errors.New("extensions: not deletable")
	ErrMustDisableFirst = errors.New("extensions: must disable before uninstall")
	ErrMigrationFailed  = errors.New("extensions: migration failed")
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
	// F2.3 韧性观测字段。
	CircuitOpen         bool       `json:"circuitOpen,omitempty"`
	CircuitOpenUntil    *time.Time `json:"circuitOpenUntil,omitempty"`
	ConsecutiveFailures int        `json:"consecutiveFailures,omitempty"`
	LastFailureReason   string     `json:"lastFailureReason,omitempty"`
	LastFailureAt       *time.Time `json:"lastFailureAt,omitempty"`
	ActiveRPCCalls      int        `json:"activeRpcCalls,omitempty"`
	MaxConcurrentRPC    int        `json:"maxConcurrentRpc,omitempty"`
}

type MatchedRoute struct {
	Extension Extension
	Route     ManifestRoute
	Path      string
}

// CapabilityGrant 是启用审查用的能力条目（与 capabilities.Grant 对齐）。
type CapabilityGrant = capabilities.Grant

type Extension struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	Source      string   `json:"source"`
	IsSystem    bool     `json:"isSystem"`
	IsDeletable bool     `json:"isDeletable"`
	Manifest    Manifest `json:"manifest"`
	// CapabilityGrants 有效 Host 能力（显式 + 推断），供启用审查 UI（F2.1）。
	CapabilityGrants []CapabilityGrant `json:"capabilityGrants,omitempty"`
	Runtime          *RuntimeStatus    `json:"runtime,omitempty"`
	ThemeRelease     *ThemeRelease     `json:"themeRelease,omitempty"`
	// WebRelease 为插件启停/信任变更排队的 live 或失败发布进度（主题仍用 themeRelease）。
	WebRelease    *WebReleaseSummary `json:"webRelease,omitempty"`
	PackageDigest string             `json:"packageDigest"`
	PackagePath   string             `json:"packagePath"`
	InstalledAt   time.Time          `json:"installedAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

// EnableInput 启用插件的可选请求体。
type EnableInput struct {
	// ConfirmCapabilities 为 true 时表示运营已审阅并确认 capability 授权。
	ConfirmCapabilities bool `json:"confirmCapabilities"`
}

// UninstallInput 卸载扩展的请求体（F2.4）。
type UninstallInput struct {
	// RetainSettings 为 true 时保留 extension_settings（便于同 id 重装恢复配置）。
	// 默认 false：随扩展行 CASCADE 删除。
	RetainSettings bool `json:"retainSettings"`
	// RetainPackage 为 true 时保留磁盘上的包快照目录；默认删除。
	RetainPackage bool `json:"retainPackage"`
}

// InstallResult 安装/升级结果。
type InstallResult struct {
	Extension Extension `json:"extension"`
	// Upgraded 为 true 表示同 id 覆盖安装（digest/version 变更路径）。
	Upgraded bool `json:"upgraded"`
	// PreviousVersion 升级前版本（若有）。
	PreviousVersion string `json:"previousVersion,omitempty"`
	// PreviousDigest 升级前包摘要。
	PreviousDigest string `json:"previousDigest,omitempty"`
	// TrustRevoked 表示升级使前端信任失效，需重新授权。
	TrustRevoked bool `json:"trustRevoked,omitempty"`
	// RequiredReEnable 升级后状态回到 installed，需重新启用。
	RequiredReEnable bool `json:"requiredReEnable,omitempty"`
}

// MigrationRecord 账本中的一条迁移。
type MigrationRecord struct {
	Path      string    `json:"path"`
	Checksum  string    `json:"checksum"`
	Status    string    `json:"status"`
	AppliedAt time.Time `json:"appliedAt"`
	Message   string    `json:"message,omitempty"`
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

// PublicActiveThemeSettings 当前激活主题的非 secret 运行时设置（前台可读）。
type PublicActiveThemeSettings struct {
	ThemeID  string            `json:"themeId"`
	Settings map[string]string `json:"settings"`
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
