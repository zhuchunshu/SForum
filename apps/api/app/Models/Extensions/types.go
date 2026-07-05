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
	Entry string `json:"entry"`
	RPC   string `json:"rpc"`
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
	Path    string   `json:"path"`
	Methods []string `json:"methods"`
}

type ManifestHook struct {
	Name string `json:"name"`
}

type ManifestJob struct {
	Name string `json:"name"`
}

type Extension struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Source      string    `json:"source"`
	IsSystem    bool      `json:"isSystem"`
	IsDeletable bool      `json:"isDeletable"`
	Manifest    Manifest  `json:"manifest"`
	PackagePath string    `json:"packagePath"`
	InstalledAt time.Time `json:"installedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
