// Package settingslifecycle owns versioned plugin settings document migrations,
// default/reset policy, import/export with secret masking, and conditional fields.
package settingslifecycle

import (
	"errors"
	"time"
)

// SchemaVersion is the Host settings lifecycle contract.
const SchemaVersion = "sforum.settings-lifecycle@1"

var (
	ErrInvalid          = errors.New("settings lifecycle input is invalid")
	ErrNotFound         = errors.New("settings lifecycle document is not found")
	ErrMigration        = errors.New("settings lifecycle migration failed")
	ErrValidation       = errors.New("settings lifecycle validation failed")
	ErrPermissionDenied = errors.New("settings lifecycle permission denied")
	// ErrConflict is a CAS revision mismatch on durable document save.
	ErrConflict = errors.New("settings lifecycle revision conflict")
)

// Document is a versioned settings payload for one extension.
type Document struct {
	SchemaVersion string `json:"schemaVersion"`
	ExtensionID   string `json:"extensionId"`
	// DataVersion is the settings data schema version (migrated by Host).
	DataVersion int               `json:"dataVersion"`
	Values      map[string]string `json:"values"`
	// SecretRefs maps secret field names to sforum.secret:// references.
	SecretRefs map[string]string `json:"secretRefs,omitempty"`
	// SecretSet marks which secret fields have values without exposing them.
	SecretSet map[string]bool `json:"secretSet,omitempty"`
	UpdatedAt time.Time       `json:"updatedAt,omitempty"`
	UpdatedBy string          `json:"updatedBy,omitempty"`
}

// FieldSchema describes one settings field for validation and conditionals.
type FieldSchema struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // string|number|boolean|secret|select
	Required bool     `json:"required,omitempty"`
	Default  string   `json:"default,omitempty"`
	Options  []string `json:"options,omitempty"`
	// VisibleWhen is "field=value" simple condition; empty = always visible.
	VisibleWhen string `json:"visibleWhen,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

// Migration upgrades or downgrades values between data versions.
type Migration struct {
	From int
	To   int
	// Apply mutates values; must be pure and reversible for downgrade pair.
	Apply func(values map[string]string) (map[string]string, error)
}

// ValidationPreview is a non-persisting validation result for admin UI.
type ValidationPreview struct {
	OK       bool              `json:"ok"`
	Errors   map[string]string `json:"errors,omitempty"`
	Warnings map[string]string `json:"warnings,omitempty"`
	// VisibleFields lists fields after conditional evaluation.
	VisibleFields []string `json:"visibleFields,omitempty"`
}

// ExportBundle is a masked export safe for backup/transfer.
type ExportBundle struct {
	SchemaVersion string            `json:"schemaVersion"`
	ExtensionID   string            `json:"extensionId"`
	DataVersion   int               `json:"dataVersion"`
	Values        map[string]string `json:"values"`
	SecretRefs    map[string]string `json:"secretRefs,omitempty"`
	SecretSet     map[string]bool   `json:"secretSet,omitempty"`
	// SecretsNeverIncluded is always true in honest exports.
	SecretsNeverIncluded bool `json:"secretsNeverIncluded"`
}
