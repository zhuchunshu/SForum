package extensions

import (
	"errors"
	"time"
)

const (
	ManifestFileName = "sforum.extension.json"
	DefaultThemeID   = "sforum.default-theme"

	TypePlugin = "plugin"
	TypeTheme  = "theme"

	StatusInstalled = "installed"
	StatusEnabled   = "enabled"
	StatusDisabled  = "disabled"

	RuntimeStopped  = "stopped"
	RuntimeStarting = "starting"
	RuntimeRunning  = "running"
	RuntimeFailed   = "failed"

	RouteAccessPublic     = "public"
	RouteAccessLogin      = "login"
	RouteAccessPermission = "permission"

	EventInstalled      = "installed"
	EventBuiltinSynced  = "builtin_synced"
	EventVerified       = "verified"
	EventEnabled        = "enabled"
	EventEnableFailed   = "enable_failed"
	EventDisabled       = "disabled"
	EventThemeActivated = "theme_activated"

	CodeInvalidArchive          = "extension.archive_invalid"
	CodeInvalidManifest         = "extension.manifest_invalid"
	CodeNotFound                = "extension.not_found"
	CodePreflightFailed         = "extension.preflight_failed"
	CodeBuildFailed             = "extension.build_failed"
	CodeThemeActivationRequired = "extension.theme_activation_required"
	CodeThemeRuntimeUnavailable = "extension.theme_runtime_unavailable"
	CodeRouteNotFound           = "extension.route_not_found"
	CodeRouteMethodNotAllowed   = "extension.route_method_not_allowed"
	CodeRuntimeUnavailable      = "extension.runtime_unavailable"
	CodeRuntimeFailed           = "extension.runtime_failed"

	SourceBuiltin  = "builtin"
	SourceUploaded = "uploaded"
)

var (
	ErrInvalidArchive          = errors.New("extensions: invalid archive")
	ErrInvalidManifest         = errors.New("extensions: invalid manifest")
	ErrExtensionNotFound       = errors.New("extensions: not found")
	ErrPreflightFailed         = errors.New("extensions: preflight failed")
	ErrBuildFailed             = errors.New("extensions: build failed")
	ErrThemeActivationRequired = errors.New("extensions: themes must be activated")
	ErrThemeRuntimeUnavailable = errors.New("extensions: theme activation runtime unavailable")
	ErrRuntimeFailed           = errors.New("extensions: runtime failed")
	ErrRouteNotFound           = errors.New("extensions: route not found")
	ErrRouteMethodNotAllowed   = errors.New("extensions: route method not allowed")
	ErrRuntimeUnavailable      = errors.New("extensions: runtime unavailable")
)

type Manifest struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	Type          string              `json:"type"`
	SForumVersion string              `json:"sforumVersion"`
	Permissions   []string            `json:"permissions"`
	Settings      []ManifestSetting   `json:"settings"`
	Migrations    []ManifestMigration `json:"migrations"`
	Backend       ManifestBackend     `json:"backend"`
	Frontend      ManifestFrontend    `json:"frontend"`
	AdminPages    []ManifestAdminPage `json:"adminPages"`
	Routes        []ManifestRoute     `json:"routes"`
	Hooks         []ManifestHook      `json:"hooks"`
	Jobs          []ManifestJob       `json:"jobs"`
	Providers     []ManifestProvider  `json:"providers"`
}

type ManifestSetting struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Default string `json:"default,omitempty"`
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
	Layer string `json:"layer"`
}

type ManifestAdminPage struct {
	Path       string `json:"path"`
	Label      string `json:"label"`
	Permission string `json:"permission"`
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

type ManifestJob struct {
	Name string `json:"name"`
}

type ManifestProvider struct {
	Slot      string `json:"slot"`
	Label     string `json:"label"`
	TimeoutMS int    `json:"timeoutMs,omitempty"`
}

type RuntimeStatus struct {
	State         string     `json:"state"`
	LastError     string     `json:"lastError,omitempty"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	RouteCount    int        `json:"routeCount"`
	HookCount     int        `json:"hookCount"`
	ProviderCount int        `json:"providerCount"`
}

type MatchedRoute struct {
	Extension Extension
	Route     ManifestRoute
	Path      string
}

type Extension struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	Source      string         `json:"source"`
	IsSystem    bool           `json:"isSystem"`
	IsDeletable bool           `json:"isDeletable"`
	Manifest    Manifest       `json:"manifest"`
	Runtime     *RuntimeStatus `json:"runtime,omitempty"`
	PackagePath string         `json:"packagePath"`
	InstalledAt time.Time      `json:"installedAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type ExtensionEvent struct {
	ID          int64     `json:"id"`
	ExtensionID string    `json:"extensionId"`
	ActorUserID int64     `json:"actorUserId"`
	Action      string    `json:"action"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ArchiveInput struct {
	FileName string
	Data     []byte
}

type SaveInstalledInput struct {
	Manifest    Manifest
	PackagePath string
}

type SaveBuiltinInput struct {
	Manifest    Manifest
	PackagePath string
}

type EventInput struct {
	ExtensionID string
	ActorUserID int64
	Action      string
	Message     string
}
