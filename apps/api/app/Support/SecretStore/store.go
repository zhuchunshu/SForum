package secretstore

import (
	"context"
	"time"
)

// Row is one immutable secret version at rest (ciphertext only).
type Row struct {
	Namespace string
	SecretID  string
	Version   int64
	Cipher    string
	MediaType string
	Purposes  []string
	UpdatedAt time.Time
	UpdatedBy string
	Revoked   bool
}

// Store is the durable Secret Store backend.
// Append must assign a unique version atomically under concurrency.
// Implementations must never log or return plaintext.
type Store interface {
	// Append inserts a new version. Version on input is ignored; the store
	// assigns the next positive version for (namespace, secret_id).
	Append(ctx context.Context, row Row) (Row, error)
	// Latest returns the highest version. When includeRevoked is false, skips
	// revoked tip rows and returns the latest non-revoked version if any.
	// When includeRevoked is true, returns the tip including a revoke tombstone.
	Latest(ctx context.Context, namespace, secretID string, includeRevoked bool) (Row, bool, error)
	// GetVersion returns a specific version (including revoked).
	GetVersion(ctx context.Context, namespace, secretID string, version int64) (Row, bool, error)
	// ListNamespace returns the latest non-revoked row per secret_id.
	ListNamespace(ctx context.Context, namespace string) ([]Row, error)
}

// AuditStore optionally persists lifecycle events across restarts.
// Values must never appear in Action metadata.
type AuditStore interface {
	AppendAudit(ctx context.Context, event AuditEvent) error
	// ListRecentAudit returns newest-first events (limit capped by MaxAuditRing).
	ListRecentAudit(ctx context.Context, limit int) ([]AuditEvent, error)
}
