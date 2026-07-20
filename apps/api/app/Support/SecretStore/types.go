// Package secretstore is the Host-owned namespaced Secret Store for V3 P11.
//
// Plugins never read ciphertext or raw storage. They Resolve declared secrets
// inside their own namespace (or a Host-granted core ref) for an audited purpose.
// Admin surfaces only see Meta (secretSet / version / timestamps), never values.
package secretstore

import (
	"errors"
	"time"
)

// SchemaVersion is the stable Host Secret Store contract identity.
const SchemaVersion = "sforum.secret-store@1"

// ReferenceScheme is the settings/document reference prefix without plaintext.
// Example: sforum.secret://demo.mail/smtp.password
const ReferenceScheme = "sforum.secret://"

const (
	// NamespaceCore holds Host site secrets (options, infrastructure adapters).
	NamespaceCore = "core"
	// NamespaceWebhooks holds webhook signing secrets.
	NamespaceWebhooks = "webhooks"

	DefaultResolveTTL = 30 * time.Second
	MaxResolveTTL     = 5 * time.Minute
	MinResolveTTL     = time.Second
	MaxPlaintextBytes = 64 * 1024
	MaxSecretIDLen    = 128
	MaxNamespaceLen   = 81
	MaxPurposeLen     = 128
	MaxAuditRing      = 128
)

var (
	ErrInvalid          = errors.New("secret store input is invalid")
	ErrNotFound         = errors.New("secret store entry is not found")
	ErrPermissionDenied = errors.New("secret store permission denied")
	ErrNamespaceDenied  = errors.New("secret store namespace is not admitted")
	ErrPurposeDenied    = errors.New("secret store purpose is not allowed")
	ErrAlreadyExists    = errors.New("secret store entry already exists")
	ErrRevoked          = errors.New("secret store entry is revoked")
	ErrCipher           = errors.New("secret store cipher failed")
)

// Ref is the stable identity of one secret. Version 0 means latest non-revoked.
type Ref struct {
	Namespace string `json:"namespace"`
	SecretID  string `json:"secretId"`
	Version   int64  `json:"version,omitempty"`
}

// Meta is the operator-facing view. Value is never included.
type Meta struct {
	SchemaVersion string    `json:"schemaVersion"`
	Namespace     string    `json:"namespace"`
	SecretID      string    `json:"secretId"`
	Version       int64     `json:"version"`
	SecretSet     bool      `json:"secretSet"`
	MediaType     string    `json:"mediaType,omitempty"`
	Purposes      []string  `json:"purposes,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
	UpdatedBy     string    `json:"updatedBy,omitempty"`
	Revoked       bool      `json:"revoked,omitempty"`
	// Reference is the opaque document value for settings import/export.
	Reference string `json:"reference"`
}

// PutOptions control write semantics.
type PutOptions struct {
	// Actor is required for audit. Empty denies.
	Actor string
	// MediaType defaults to text/plain.
	MediaType string
	// Purposes is the allowlist for Resolve. Empty means any purpose in-namespace.
	Purposes []string
	// PreserveEmpty skips the write when plaintext is empty (settings UX).
	PreserveEmpty bool
}

// Caller is Host-attested extension identity for Resolve admission.
type Caller struct {
	// ExtensionID is empty for core Host callers (NamespaceCore / system).
	ExtensionID string
	// Actor is optional human/system actor for audit (never logged with value).
	Actor string
}

// Lease is a short-lived resolve result. Value must not be cached beyond ExpiresAt
// by callers without a fresh Resolve.
type Lease struct {
	LeaseID   string    `json:"leaseId"`
	Namespace string    `json:"namespace"`
	SecretID  string    `json:"secretId"`
	Version   int64     `json:"version"`
	Value     []byte    `json:"-"` // never JSON-serialized by Host APIs
	MediaType string    `json:"mediaType,omitempty"`
	ExpiresAt time.Time `json:"expiresAt"`
	Purpose   string    `json:"purpose"`
}

// AuditEvent records secret lifecycle without values.
type AuditEvent struct {
	AuditID   string    `json:"auditId"`
	Action    string    `json:"action"` // put|rotate|clear|resolve|revoke
	Namespace string    `json:"namespace"`
	SecretID  string    `json:"secretId"`
	Version   int64     `json:"version,omitempty"`
	Actor     string    `json:"actor,omitempty"`
	Purpose   string    `json:"purpose,omitempty"`
	OK        bool      `json:"ok"`
	At        time.Time `json:"at"`
}

// InspectorSnapshot is the operator-facing audit/metrics view.
type InspectorSnapshot struct {
	SchemaVersion string       `json:"schemaVersion"`
	Puts          uint64       `json:"puts"`
	Rotates       uint64       `json:"rotates"`
	Clears        uint64       `json:"clears"`
	Resolves      uint64       `json:"resolves"`
	Denies        uint64       `json:"denies"`
	Errors        uint64       `json:"errors"`
	RecentAudit   []AuditEvent `json:"recentAudit,omitempty"`
}
