package cachepolicy

import (
	"errors"
	"time"
)

// SchemaVersion is the Host cache policy contract identity.
const SchemaVersion = "sforum.cache-policy@1"

const (
	ProviderMemory = "memory"
	ProviderRedis  = "redis"
	ProviderNoop   = "noop"

	DirectiveStore   = "store"
	DirectiveBypass  = "bypass"
	DirectiveNoStore = "no-store"

	DefaultTTL = 30 * time.Second
	MaxTTL     = 24 * time.Hour
	MinTTL     = time.Second
)

var (
	ErrInvalid          = errors.New("cache policy input is invalid")
	ErrProviderDenied   = errors.New("cache policy provider is not selected")
	ErrBypass           = errors.New("cache policy bypassed")
	ErrNoStore          = errors.New("cache policy forbids store")
	ErrPermissionDenied = errors.New("cache policy permission denied")
	ErrNotFound         = errors.New("cache policy declaration is not found")
)

// ProviderSelection is Host-owned selection of the process-local cache backend.
// Plugins may declare preferred providers on Cache Registry contributions, but
// Host selection remains authoritative.
type ProviderSelection struct {
	Provider string `json:"provider"`
	// SelectedAt is wall time for inspector/audit only.
	SelectedAt time.Time `json:"selectedAt,omitempty"`
}

// KeyRequest builds a route/page/entity-aware cache key under a declared
// Cache Registry namespace. Host supplies all isolation fingerprints.
type KeyRequest struct {
	CacheID               string
	Namespace             string
	RouteID               string
	PageID                string
	EntityEvent           string
	ActorFingerprint      string
	PermissionFingerprint string
	LocaleFingerprint     string
	// ThemeRevision and PluginRevision force miss when presentation/runtime
	// artifacts change without a full process restart.
	ThemeRevision  string
	PluginRevision string
	// Directive is store|bypass|no-store. Empty defaults to store.
	Directive string
	// TTL is the requested TTL; Host clamps to [MinTTL, MaxTTL].
	TTL time.Duration
}

// KeyPlan is the Host-resolved key material. Values are never stored here.
type KeyPlan struct {
	SchemaVersion string        `json:"schemaVersion"`
	CacheID       string        `json:"cacheId"`
	Namespace     string        `json:"namespace"`
	Key           string        `json:"key"`
	Directive     string        `json:"directive"`
	TTL           time.Duration `json:"ttl"`
	Provider      string        `json:"provider"`
	Tags          []string      `json:"tags,omitempty"`
}

// InvalidateRequest removes keys by exact key and/or declared tags.
type InvalidateRequest struct {
	CacheID   string
	Namespace string
	Keys      []string
	Tags      []string
	// Actor is required for audit; empty denies.
	Actor  string
	Reason string
}

// InvalidateResult is the Host audit outcome for one invalidation.
type InvalidateResult struct {
	DeletedKeys int       `json:"deletedKeys"`
	Tags        []string  `json:"tags,omitempty"`
	AuditID     string    `json:"auditId"`
	Actor       string    `json:"actor"`
	Reason      string    `json:"reason,omitempty"`
	At          time.Time `json:"at"`
}

// Metrics is a process-local hit/miss/latency inspector snapshot.
type Metrics struct {
	Hits           uint64        `json:"hits"`
	Misses         uint64        `json:"misses"`
	Stores         uint64        `json:"stores"`
	Deletes        uint64        `json:"deletes"`
	Bypasses       uint64        `json:"bypasses"`
	Errors         uint64        `json:"errors"`
	Invalidations  uint64        `json:"invalidations"`
	AvgGetLatency  time.Duration `json:"avgGetLatency"`
	AvgSetLatency  time.Duration `json:"avgSetLatency"`
	Provider       string        `json:"provider"`
	ThemeRevision  string        `json:"themeRevision,omitempty"`
	PluginRevision string        `json:"pluginRevision,omitempty"`
}

// InspectorSnapshot is the operator-facing cache policy view.
type InspectorSnapshot struct {
	SchemaVersion string             `json:"schemaVersion"`
	Provider      ProviderSelection  `json:"provider"`
	Metrics       Metrics            `json:"metrics"`
	RecentAudit   []InvalidateResult `json:"recentAudit,omitempty"`
}
