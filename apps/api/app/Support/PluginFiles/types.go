// Package pluginfiles is the Host-owned namespaced filesystem for plugin private
// data, temporary files, and user-owned files with quotas and cleanup.
//
// Plugins never receive absolute host paths outside their namespace root. Path
// traversal, symlink escape, and quota overflow fail closed.
package pluginfiles

import (
	"errors"
	"time"
)

// SchemaVersion is the Host plugin file contract identity.
const SchemaVersion = "sforum.plugin-files@1"

const (
	// KindPrivate is durable plugin-owned data under the package namespace.
	// Uninstall default: delete unless CleanupOptions.RetainPrivate.
	KindPrivate = "private"
	// KindTemp is short-lived data cleaned on disable/uninstall or TTL.
	// Uninstall default: always delete.
	KindTemp = "temp"
	// KindStatic is Host-served package static assets (not user uploads).
	// Uninstall default: delete with package.
	KindStatic = "static"
	// KindUser is user-owned files under extension+user isolation.
	// Uninstall default: retain (user data) unless CleanupOptions.DeleteUser.
	KindUser = "user"

	DefaultPrivateQuotaBytes = 64 * 1024 * 1024  // 64 MiB
	DefaultTempQuotaBytes    = 32 * 1024 * 1024  // 32 MiB
	DefaultStaticQuotaBytes  = 64 * 1024 * 1024  // 64 MiB
	DefaultUserQuotaBytes    = 128 * 1024 * 1024 // 128 MiB
	DefaultTempTTL           = 24 * time.Hour
	MaxRelativePathLen       = 512
	MaxWriteBytes            = 16 * 1024 * 1024 // single write cap
)

var (
	ErrInvalid          = errors.New("plugin files input is invalid")
	ErrPermissionDenied = errors.New("plugin files permission denied")
	ErrNotFound         = errors.New("plugin files entry is not found")
	ErrTraversal        = errors.New("plugin files path escapes namespace")
	ErrQuotaExceeded    = errors.New("plugin files quota exceeded")
	ErrTooLarge         = errors.New("plugin files write exceeds limit")
	ErrSymlink          = errors.New("plugin files rejects symlink escape")
)

// Namespace is one extension's Host-managed file space.
type Namespace struct {
	ExtensionID string `json:"extensionId"`
	// Root is the Host absolute directory for this extension (not plugin-visible).
	Root string `json:"-"`
	// PrivateQuotaBytes defaults to DefaultPrivateQuotaBytes.
	PrivateQuotaBytes int64 `json:"privateQuotaBytes,omitempty"`
	TempQuotaBytes    int64 `json:"tempQuotaBytes,omitempty"`
	StaticQuotaBytes  int64 `json:"staticQuotaBytes,omitempty"`
	UserQuotaBytes    int64 `json:"userQuotaBytes,omitempty"`
}

// CleanupOptions controls uninstall retention policy.
type CleanupOptions struct {
	// RetainPrivate keeps KindPrivate when true (operator opt-in).
	RetainPrivate bool
	// DeleteUser removes KindUser when true (default retain user data).
	DeleteUser bool
	// RetainStatic keeps KindStatic when true (rare; package assets usually go).
	RetainStatic bool
}

// WriteRequest writes bytes under a relative path.
type WriteRequest struct {
	ExtensionID string
	Kind        string
	// RelativePath is package-relative without leading slash (e.g. "cache/a.json").
	RelativePath string
	// UserID is required for KindUser.
	UserID string
	Data   []byte
	// Actor is required for audit.
	Actor string
}

// ReadRequest reads a relative path.
type ReadRequest struct {
	ExtensionID  string
	Kind         string
	RelativePath string
	UserID       string
}

// DeleteRequest removes a relative path.
type DeleteRequest struct {
	ExtensionID  string
	Kind         string
	RelativePath string
	UserID       string
	Actor        string
}

// FileInfo is operator/plugin-safe metadata (no absolute host path).
type FileInfo struct {
	SchemaVersion string    `json:"schemaVersion"`
	ExtensionID   string    `json:"extensionId"`
	Kind          string    `json:"kind"`
	RelativePath  string    `json:"relativePath"`
	UserID        string    `json:"userId,omitempty"`
	Size          int64     `json:"size"`
	ModTime       time.Time `json:"modTime"`
}

// Usage is per-namespace quota accounting.
type Usage struct {
	ExtensionID string `json:"extensionId"`
	PrivateUsed int64  `json:"privateUsed"`
	PrivateMax  int64  `json:"privateMax"`
	TempUsed    int64  `json:"tempUsed"`
	TempMax     int64  `json:"tempMax"`
	StaticUsed  int64  `json:"staticUsed"`
	StaticMax   int64  `json:"staticMax"`
	UserUsed    int64  `json:"userUsed"`
	UserMax     int64  `json:"userMax"`
}

// CleanupResult reports uninstall/disable cleanup.
type CleanupResult struct {
	ExtensionID  string   `json:"extensionId"`
	RemovedFiles int      `json:"removedFiles"`
	RemovedBytes int64    `json:"removedBytes"`
	Kinds        []string `json:"kinds,omitempty"`
}
