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

	RouteAccessPublic            = extensionmanifest.RouteAccessPublic
	RouteAccessLogin             = extensionmanifest.RouteAccessLogin
	RouteAccessPermission        = extensionmanifest.RouteAccessPermission
	AdminSurfaceOperationQuery   = extensionmanifest.AdminSurfaceOperationQuery
	AdminSurfaceOperationCommand = extensionmanifest.AdminSurfaceOperationCommand

	EventInstalled             = "installed"
	EventUpgraded              = "upgraded"
	EventUninstalled           = "uninstalled"
	EventBuiltinSynced         = "builtin_synced"
	EventVerified              = "verified"
	EventEnabled               = "enabled"
	EventEnableFailed          = "enable_failed"
	EventDisabled              = "disabled"
	EventRolledBack            = "rolled_back"
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
	CodeThemePreviewStale          = "extension.theme_preview_stale"
	CodeRouteNotFound              = "extension.route_not_found"
	CodeRouteMethodNotAllowed      = "extension.route_method_not_allowed"
	CodeRuntimeUnavailable         = "extension.runtime_unavailable"
	CodeRuntimeFailed              = "extension.runtime_failed"
	CodeFrontendRuntimeUnavailable = "extension.frontend_runtime_unavailable"
	CodeFrontendDigestInvalid      = "extension.frontend_digest_invalid"
	CodeFrontendPackageChanged     = "extension.frontend_package_changed"
	CodeFrontendTrustNotFound      = "extension.frontend_trust_not_found"
	// 插件已禁用：设置读写、自定义管理页等功能性能力不可用。
	CodeExtensionDisabled = "extension.disabled"
	// 设置已写入但插件重启失败，且旧设置恢复也失败，需要运营介入。
	CodeSettingsRollbackFailed    = "extension.settings_rollback_failed"
	CodeSettingsActionInvalid     = "extension.settings_action_invalid"
	CodeSettingsActionUnavailable = "extension.settings_action_unavailable"
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
	CodeMigrationFailed        = "extension.migration_failed"
	CodeSafeModeActive         = "extension.safe_mode_active"
	CodeLifecycleInvalid       = "extension.lifecycle_invalid"
	CodeLifecycleUnavailable   = "extension.lifecycle_unavailable"
	CodeLifecycleConflict      = "extension.lifecycle_conflict"
	CodeLifecycleActionFailed  = "extension.lifecycle_action_failed"
	CodeLifecycleAuthorityGone = "extension.lifecycle_authority_not_found"
	CodeLifecycleCleanupFailed = "extension.lifecycle_cleanup_failed"
	CodeStagedVersionNotFound  = "extension.staged_version_not_found"
	CodeVersionNotFound        = "extension.version_not_found"
	// CodeUntrustedBackendRestricted 见 backend_trust.go（非 super_admin 执行非内置后端）。

	SourceBuiltin  = "builtin"
	SourceUploaded = "uploaded"
)

var (
	ErrInvalidArchive            = errors.New("extensions: invalid archive")
	ErrInvalidManifest           = extensionmanifest.ErrInvalidManifest
	ErrExtensionNotFound         = errors.New("extensions: not found")
	ErrExtensionDisabled         = errors.New("extensions: disabled")
	ErrSettingsRollbackFailed    = errors.New("extensions: settings rollback failed")
	ErrSettingsActionInvalid     = errors.New("extensions: settings action invalid")
	ErrSettingsActionUnavailable = errors.New("extensions: settings action unavailable")
	ErrPreflightFailed           = errors.New("extensions: preflight failed")
	ErrBuildFailed               = errors.New("extensions: build failed")
	ErrThemeActivationRequired   = errors.New("extensions: themes must be activated")
	ErrThemeRuntimeUnavailable   = errors.New("extensions: theme activation runtime unavailable")
	ErrThemePreviewStale         = errors.New("extensions: theme activation preview is stale")
	ErrRuntimeFailed             = errors.New("extensions: runtime failed")
	ErrRouteNotFound             = errors.New("extensions: route not found")
	ErrRouteMethodNotAllowed     = errors.New("extensions: route method not allowed")
	ErrRuntimeUnavailable        = errors.New("extensions: runtime unavailable")
	// ErrCapabilityConfirmationRequired 启用插件前需 confirmCapabilities=true。
	ErrCapabilityConfirmationRequired = errors.New("extensions: capability confirmation required")
	ErrCapabilityDenied               = errors.New("extensions: capability denied")
	// ErrFeaturesRequired 站点产品开关未满足 manifest requiresFeatures（F4.5）。
	ErrFeaturesRequired = errors.New("extensions: required features disabled")
	ErrNotDeletable     = errors.New("extensions: not deletable")
	ErrMustDisableFirst = errors.New("extensions: must disable before uninstall")
	ErrMigrationFailed  = errors.New("extensions: migration failed")
	ErrSafeModeActive   = errors.New("extensions: safe mode active")
)

type Manifest = extensionmanifest.Manifest
type ManifestAuthor = extensionmanifest.ManifestAuthor
type ManifestSetting = extensionmanifest.ManifestSetting
type ManifestSettingOption = extensionmanifest.ManifestSettingOption
type SettingsDocument = extensionmanifest.SettingsDocument
type SettingsUI = extensionmanifest.SettingsUI
type SettingsTab = extensionmanifest.SettingsTab
type SettingsGroup = extensionmanifest.SettingsGroup
type SettingsCallout = extensionmanifest.SettingsCallout
type SettingsComponent = extensionmanifest.SettingsComponent
type SettingsAction = extensionmanifest.SettingsAction
type LocalizedText = extensionmanifest.LocalizedText
type ManifestMigration = extensionmanifest.ManifestMigration
type ManifestBackend = extensionmanifest.ManifestBackend
type ManifestAdmin = extensionmanifest.ManifestAdmin
type ManifestAdminPage = extensionmanifest.ManifestAdminPage
type ManifestRoute = extensionmanifest.ManifestRoute
type ManifestHook = extensionmanifest.ManifestHook
type ManifestEvent = extensionmanifest.ManifestEvent
type ManifestJob = extensionmanifest.ManifestJob
type ManifestProvider = extensionmanifest.ManifestProvider
type ManifestContribution = extensionmanifest.ManifestContribution
type ManifestGuard = extensionmanifest.ManifestGuard
type ManifestSchedule = extensionmanifest.ManifestSchedule
type ManifestComponent = extensionmanifest.ManifestComponent
type ManifestTemplate = extensionmanifest.ManifestTemplate
type ManifestAsset = extensionmanifest.ManifestAsset
type ManifestContent = extensionmanifest.ManifestContent
type ManifestDatabase = extensionmanifest.ManifestDatabase
type ManifestCache = extensionmanifest.ManifestCache
type ManifestService = extensionmanifest.ManifestService
type ManifestCommand = extensionmanifest.ManifestCommand
type ManifestAdminSurface = extensionmanifest.ManifestAdminSurface
type ManifestQuery = extensionmanifest.ManifestQuery
type ManifestIdentity = extensionmanifest.ManifestIdentity
type ManifestPermissionDefinition = extensionmanifest.ManifestPermissionDefinition
type ManifestMediaPipeline = extensionmanifest.ManifestMediaPipeline
type ManifestNavigation = extensionmanifest.ManifestNavigation
type ManifestRegion = extensionmanifest.ManifestRegion
type ManifestDependency = extensionmanifest.ManifestDependency
type ManifestLifecycle = extensionmanifest.ManifestLifecycle
type ManifestOpenAPIFragment = extensionmanifest.ManifestOpenAPIFragment
type ManifestPackageFile = extensionmanifest.ManifestPackageFile
type TopicActionContributionPayload = extensionmanifest.TopicActionContributionPayload
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
	// P3：协议迁移观测；v1 保持可用但必须明确标记 deprecated。
	ProtocolVersion    int        `json:"protocolVersion,omitempty"`
	ProtocolTransport  string     `json:"protocolTransport,omitempty"`
	ProtocolDeprecated bool       `json:"protocolDeprecated,omitempty"`
	ProtocolStartCount uint64     `json:"protocolStartCount,omitempty"`
	ProtocolCallCount  uint64     `json:"protocolCallCount,omitempty"`
	ProtocolLastCallAt *time.Time `json:"protocolLastCallAt,omitempty"`
}

type MatchedRoute struct {
	Extension Extension
	Route     ManifestRoute
	Path      string
}

// CapabilityGrant 是启用审查用的能力条目（与 capabilities.Grant 对齐）。
type CapabilityGrant = capabilities.Grant

// ExtensionVersion 是不可变包快照。活动版本和待执行升级版本必须分开表示，
// 静态上传不得把候选制品伪装成已经生效的运行版本。
type ExtensionVersion struct {
	ID                  int64     `json:"-"`
	Version             string    `json:"version"`
	Manifest            Manifest  `json:"manifest"`
	PackageDigest       string    `json:"packageDigest"`
	AdminFrontendDigest string    `json:"adminFrontendDigest"`
	PackagePath         string    `json:"packagePath"`
	InstalledAt         time.Time `json:"installedAt"`
}

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
	CapabilityGrants    []CapabilityGrant `json:"capabilityGrants,omitempty"`
	Runtime             *RuntimeStatus    `json:"runtime,omitempty"`
	PackageDigest       string            `json:"packageDigest"`
	AdminFrontendDigest string            `json:"adminFrontendDigest,omitempty"`
	PackagePath         string            `json:"packagePath"`
	ActiveVersionID     int64             `json:"-"`
	StagedVersion       *ExtensionVersion `json:"stagedVersion,omitempty"`
	InstalledAt         time.Time         `json:"installedAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
}

// EnableInput 启用插件的可选请求体。
type EnableInput struct {
	// ConfirmCapabilities 为 true 时表示运营已审阅并确认 capability 授权。
	ConfirmCapabilities bool `json:"confirmCapabilities"`
	// ConfirmationToken 是 V3 P1 服务端签发的一次性 exact-artifact token。
	ConfirmationToken string `json:"confirmationToken,omitempty"`
	// IdempotencyKey 来自 HTTP Idempotency-Key；不接受 body 覆盖。
	IdempotencyKey string `json:"-"`
}

type ThemeActivationInput struct {
	Version             string `json:"version"`
	PackageDigest       string `json:"packageDigest"`
	CurrentThemeID      string `json:"currentThemeId"`
	CurrentThemeVersion string `json:"currentThemeVersion"`
	CurrentThemeDigest  string `json:"currentThemeDigest"`
	// ConfirmationToken 是上传主题包含 L2/其他可执行声明时的一次性 exact-artifact token。
	ConfirmationToken string `json:"confirmationToken,omitempty"`
	// ApproveCoreReplacements is an explicit approval from the exact visible
	// preview. It is effective only for a super_admin actor.
	ApproveCoreReplacements bool  `json:"approveCoreReplacements"`
	ActorUserID             int64 `json:"-"`
}

type ThemeRuntimePublicationState string

const (
	ThemeRuntimePublicationActive ThemeRuntimePublicationState = "active"
	ThemeRuntimePublicationNone   ThemeRuntimePublicationState = "none"
)

type ThemeRuntimePublicationReason string

const (
	ThemeRuntimePublicationActivation    ThemeRuntimePublicationReason = "activation"
	ThemeRuntimePublicationCompensation  ThemeRuntimePublicationReason = "compensation"
	ThemeRuntimePublicationStartupRepair ThemeRuntimePublicationReason = "startup_repair"
)

type ThemeRuntimePublication struct {
	Revision                       int64                         `json:"revision"`
	DesiredState                   ThemeRuntimePublicationState  `json:"desiredState"`
	ThemeID                        string                        `json:"themeId,omitempty"`
	ThemeVersion                   string                        `json:"themeVersion,omitempty"`
	PackageDigest                  string                        `json:"packageDigest,omitempty"`
	SourceThemeID                  string                        `json:"sourceThemeId,omitempty"`
	SourceThemeVersion             string                        `json:"sourceThemeVersion,omitempty"`
	SourcePackageDigest            string                        `json:"sourcePackageDigest,omitempty"`
	SourceCoreReplacementsApproved bool                          `json:"sourceCoreReplacementsApproved"`
	SourceActorUserID              int64                         `json:"sourceActorUserId,omitempty"`
	CoreReplacementsApproved       bool                          `json:"coreReplacementsApproved"`
	ActorUserID                    int64                         `json:"actorUserId,omitempty"`
	Reason                         ThemeRuntimePublicationReason `json:"reason"`
	CreatedAt                      time.Time                     `json:"createdAt"`
}

type ThemeActivationResult struct {
	Extension   Extension               `json:"extension"`
	Publication ThemeRuntimePublication `json:"publication"`
}

// LifecycleRequestInput 是无业务 body 的 V2 lifecycle 请求元数据。
type LifecycleRequestInput struct {
	IdempotencyKey string `json:"-"`
}

// UpgradeInput 激活当前不可变 staged candidate。
type UpgradeInput struct {
	ConfirmationToken string `json:"confirmationToken,omitempty"`
	IdempotencyKey    string `json:"-"`
}

// RollbackInput 只接受 exact historical version + digest，禁止“最近版本”一类可变指针。
type RollbackInput struct {
	TargetVersion       string `json:"targetVersion"`
	TargetPackageDigest string `json:"targetPackageDigest"`
	IdempotencyKey      string `json:"-"`
}

// UninstallInput 卸载扩展的请求体（F2.4）。
type UninstallInput struct {
	// RemovalMode 是 Lifecycle V2 的数据去向。空值使用安全默认 preserve。
	RemovalMode string `json:"removalMode,omitempty"`
	// RetainSettings 为 true 时保留 extension_settings（便于同 id 重装恢复配置）。
	// 仅供 Lifecycle V1 兼容路径使用。
	RetainSettings bool `json:"retainSettings"`
	// RetainPackage 为 true 时保留磁盘上的包快照目录；默认删除。
	// 仅供 Lifecycle V1 兼容路径使用。
	RetainPackage bool `json:"retainPackage"`
	// IdempotencyKey 来自 HTTP Idempotency-Key；不接受 body 覆盖。
	IdempotencyKey string `json:"-"`
}

// LifecycleCleanupFinalization 是 Host exact-receipt finalizer 的最小跨层结果。
type LifecycleCleanupFinalization struct {
	OperationID           int64  `json:"operationId"`
	Status                string `json:"status"`
	PhysicalPurgeComplete bool   `json:"physicalPurgeComplete"`
}

type UninstallResult struct {
	Uninstalled bool                          `json:"uninstalled"`
	ExtensionID string                        `json:"extensionId"`
	OperationID int64                         `json:"operationId,omitempty"`
	RemovalMode string                        `json:"removalMode,omitempty"`
	Replayed    bool                          `json:"replayed,omitempty"`
	Cleanup     *LifecycleCleanupFinalization `json:"cleanup,omitempty"`
}

// LifecycleRecoveryInput 精确恢复原 operation，不会创建新幂等域。
type LifecycleRecoveryInput struct {
	Decision       string `json:"decision"`
	Reason         string `json:"reason,omitempty"`
	EscalateForced bool   `json:"escalateForced,omitempty"`
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
	// ActivationPending 表示候选已惰性保存，活动制品仍在服务，尚未执行受信任升级事务。
	ActivationPending bool `json:"activationPending,omitempty"`
}

// MigrationRecord 账本中的一条迁移。
type MigrationRecord struct {
	Path      string    `json:"path"`
	Checksum  string    `json:"checksum"`
	Status    string    `json:"status"`
	AppliedAt time.Time `json:"appliedAt"`
	Message   string    `json:"message,omitempty"`
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
	Key              string `json:"key"`
	Label            string `json:"label"`
	Description      string `json:"description"`
	Type             string `json:"type"`
	Default          string `json:"default"`
	Value            string `json:"value"`
	Placeholder      string `json:"placeholder,omitempty"`
	RecommendedValue string `json:"recommendedValue,omitempty"`
	// Width 为 Schema UI 控件宽度：default 或 full；省略时前端按 default。
	Width     string                   `json:"width,omitempty"`
	Group     string                   `json:"group,omitempty"`
	GroupID   string                   `json:"groupId,omitempty"`
	Column    int                      `json:"column,omitempty"`
	Options   []ExtensionSettingOption `json:"options,omitempty"`
	SecretSet bool                     `json:"secretSet,omitempty"`
}

type ExtensionSettingsRenderer struct {
	Mode      string                      `json:"mode"`
	Layout    string                      `json:"layout"`
	Source    string                      `json:"source"`
	Fallback  string                      `json:"fallback"`
	Component *ExtensionSettingsComponent `json:"component,omitempty"`
}

type ExtensionSettingsComponent struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	APIVersion int    `json:"apiVersion"`
	Entry      string `json:"entry,omitempty"`
	CSS        string `json:"css,omitempty"`
}

type ExtensionSettingsTab struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Groups      []string `json:"groups,omitempty"`
}

type ExtensionSettingsGroup struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Columns     int    `json:"columns,omitempty"`
}

type ExtensionSettingsCallout struct {
	ID    string `json:"id"`
	Tone  string `json:"tone"`
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
	Tab   string `json:"tab,omitempty"`
	Group string `json:"group,omitempty"`
}

type ExtensionSettingsAction struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Label             string   `json:"label"`
	Description       string   `json:"description,omitempty"`
	Placement         string   `json:"placement"`
	UseDraftValues    bool     `json:"useDraftValues"`
	Fields            []string `json:"fields,omitempty"`
	Available         bool     `json:"available"`
	UnavailableReason string   `json:"unavailableReason,omitempty"`
}

type ExtensionSettings struct {
	ExtensionID      string                     `json:"extensionId"`
	ExtensionType    string                     `json:"extensionType"`
	ExtensionVersion string                     `json:"extensionVersion"`
	ExtensionStatus  string                     `json:"extensionStatus"`
	Renderer         ExtensionSettingsRenderer  `json:"renderer"`
	Tabs             []ExtensionSettingsTab     `json:"tabs,omitempty"`
	Groups           []ExtensionSettingsGroup   `json:"groups,omitempty"`
	Callouts         []ExtensionSettingsCallout `json:"callouts,omitempty"`
	Items            []ExtensionSettingValue    `json:"items"`
	Actions          []ExtensionSettingsAction  `json:"actions,omitempty"`
}

// PublicActiveThemeSettings 当前激活主题的非 secret 运行时设置（前台可读）。
type PublicActiveThemeSettings struct {
	ThemeID  string            `json:"themeId"`
	Settings map[string]string `json:"settings"`
}

type UpdateSettingsInput struct {
	Values map[string]string `json:"values"`
}

type SettingsActionSecretInput struct {
	Mode  string `json:"mode"`
	Value string `json:"value,omitempty"`
}

type ExecuteSettingsActionInput struct {
	Values  map[string]string                    `json:"values,omitempty"`
	Secrets map[string]SettingsActionSecretInput `json:"secrets,omitempty"`
}

type SettingsActionResult struct {
	Success     bool              `json:"success"`
	Reason      string            `json:"reason"`
	Message     string            `json:"message"`
	Details     map[string]string `json:"details,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
	DurationMS  int64             `json:"durationMs"`
}

type SettingsActionProbeResult struct {
	OK          bool
	Reason      string
	Message     string
	Details     map[string]string
	Suggestions []string
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
	Manifest            Manifest
	PackagePath         string
	PackageDigest       string
	AdminFrontendDigest string
}

type SaveBuiltinInput struct {
	Manifest            Manifest
	PackagePath         string
	PackageDigest       string
	AdminFrontendDigest string
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
