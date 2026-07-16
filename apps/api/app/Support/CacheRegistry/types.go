package cacheregistry

import "errors"

// SchemaVersion is the stable identity of the immutable registry snapshot.
const SchemaVersion = "sforum.cache-registry@1"

const (
	PolicyPrivate    = "private"
	PolicyActor      = "actor"
	PolicyPermission = "permission"
	PolicyPublic     = "public"
)

var (
	ErrInvalid             = errors.New("cache registry declaration is invalid")
	ErrConflict            = errors.New("cache registry declaration conflicts with the active graph")
	ErrArtifactConflict    = errors.New("cache registry artifact does not own the active publication")
	ErrRevisionConflict    = errors.New("cache registry revision changed during replacement")
	ErrSafeMode            = errors.New("cache registry rejects third-party publication in safe mode")
	ErrNotFound            = errors.New("cache registry declaration is not found")
	ErrArtifactUnavailable = errors.New("cache registry exact runtime artifact is unavailable")
	ErrPlanStale           = errors.New("cache registry execution plan is stale")
	ErrIsolationRequired   = errors.New("cache registry policy isolation is required")
)

// Artifact binds declarations to one exact package and runtime instance.
// Core artifacts carry a package-private seal created by NewCoreArtifact.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	coreSeal          [32]byte
}

// Declaration mirrors the frozen ExtensionManifest ManifestCache contract.
type Declaration struct {
	ID              string   `json:"id"`
	ContractVersion string   `json:"contractVersion"`
	Namespace       string   `json:"namespace"`
	Policy          string   `json:"policy"`
	Tags            []string `json:"tags,omitempty"`
	Provider        string   `json:"provider,omitempty"`
	Invalidators    []string `json:"invalidators,omitempty"`
}

type Publication struct {
	Artifact Artifact      `json:"artifact"`
	Caches   []Declaration `json:"caches,omitempty"`
}

// Contribution exposes a frozen declaration together with its exact owner.
type Contribution struct {
	Declaration
	Artifact Artifact `json:"artifact"`
}

type Snapshot struct {
	SchemaVersion string         `json:"schemaVersion"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	SafeMode      bool           `json:"safeMode,omitempty"`
	Publications  []Publication  `json:"publications"`
	Caches        []Contribution `json:"caches"`
}

type CacheState struct {
	Revision uint64 `json:"revision"`
	Digest   string `json:"digest"`
	SafeMode bool   `json:"safeMode,omitempty"`
}

// PlanRequest contains only Host-derived isolation projections. The Registry
// does not accept cache keys, values, TTLs, Redis commands, or provider choices.
type PlanRequest struct {
	CacheID               string
	Namespace             string
	ActorFingerprint      string
	PermissionFingerprint string
	LocaleFingerprint     string
}

// IsolationMetadata is sufficient for a later Host CacheService to isolate a
// namespace without this leaf executing or selecting a cache provider.
type IsolationMetadata struct {
	CacheID               string   `json:"cacheId"`
	Namespace             string   `json:"namespace"`
	Policy                string   `json:"policy"`
	SegmentDigest         string   `json:"segmentDigest"`
	Artifact              Artifact `json:"artifact"`
	ActorFingerprint      string   `json:"actorFingerprint,omitempty"`
	PermissionFingerprint string   `json:"permissionFingerprint,omitempty"`
	LocaleFingerprint     string   `json:"localeFingerprint"`
}

// Plan is immutable declaration and isolation metadata. It is not a Redis
// operation, a provider selection, or authority to execute cache calls.
type Plan struct {
	SchemaVersion string            `json:"schemaVersion"`
	Revision      uint64            `json:"revision"`
	Digest        string            `json:"digest"`
	SafeMode      bool              `json:"safeMode,omitempty"`
	Cache         Contribution      `json:"cache"`
	Isolation     IsolationMetadata `json:"isolation"`
}
