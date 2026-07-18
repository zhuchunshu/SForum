package queryregistry

import (
	"context"
	"errors"
)

// SchemaVersion is the stable identity of this registry snapshot contract.
const SchemaVersion = "sforum.query-registry@1"

// Pagination modes mirror frozen ManifestQuery.pagination values.
const (
	PaginationNone   = "none"
	PaginationOffset = "offset"
	PaginationCursor = "cursor"
)

// PermissionPolicyPublic / Login are Host-recognized policy tokens. Any other
// non-empty PermissionPolicy is treated as a Host permission key identity.
const (
	PermissionPolicyPublic = "public"
	PermissionPolicyLogin  = "login"
)

// Provider kinds identify which declaration slot an exact artifact owns in a
// composed plan. ManifestQuery has no field-level contribution action, so each
// slot is always owned by the single complete query declaration.
const (
	ProviderKindQuery    = "query"
	ProviderKindField    = "field"
	ProviderKindRelation = "relation"
	ProviderKindFilter   = "filter"
	ProviderKindSort     = "sort"
)

var (
	ErrInvalid             = errors.New("query registry declaration is invalid")
	ErrConflict            = errors.New("query registry conflicts with the active graph")
	ErrArtifactConflict    = errors.New("query registry artifact does not own the active publication")
	ErrRevisionConflict    = errors.New("query registry revision changed during replacement")
	ErrSafeMode            = errors.New("query registry rejects third-party publication in safe mode")
	ErrNotFound            = errors.New("query registry declaration is not found")
	ErrArtifactUnavailable = errors.New("query registry exact runtime artifact is unavailable")
	ErrDenied              = errors.New("query registry permission recheck denied")
	ErrCostExceeded        = errors.New("query registry plan exceeds maximum deterministic cost")
	// ErrContractInsufficient is returned when a requested semantic cannot be
	// expressed with frozen ManifestQuery fields. Callers must not invent handlers.
	ErrContractInsufficient = errors.New("query registry frozen ManifestQuery contract is insufficient")
)

// Artifact binds every contribution to the exact inert package that declared
// it. Core publications use Core=true but still carry stable version/digests.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	// coreSeal is deliberately not serializable or constructible by callers in
	// another package. Core authority must enter through NewCoreArtifact, never
	// through decoded extension metadata's Core flag or core.* prefix.
	coreSeal [32]byte
}

// QueryDeclaration is the frozen ManifestQuery surface plus optional executable
// opt-in metadata. Handler never invents a provider by naming convention.
type QueryDeclaration struct {
	ID              string   `json:"id"`
	ContractVersion string   `json:"contractVersion"`
	Entity          string   `json:"entity"`
	PlanVersion     string   `json:"planVersion"`
	Fields          []string `json:"fields"`
	Relations       []string `json:"relations,omitempty"`
	Filters         []string `json:"filters,omitempty"`
	Sort            []string `json:"sort,omitempty"`
	Pagination      string   `json:"pagination"`
	ResultSchema    string   `json:"resultSchema"`
	// ResultSchemaDigest is Host-derived publication metadata. Manifest authors
	// cannot set it directly: a non-empty value is accepted only when the private
	// compiled material was produced by BindResultSchemas.
	ResultSchemaDigest string   `json:"resultSchemaDigest,omitempty"`
	PermissionPolicy   string   `json:"permissionPolicy"`
	CacheTags          []string `json:"cacheTags,omitempty"`
	// Handler explicitly opts this declaration into Host-to-plugin execution.
	// Empty remains the compatible inspect/plan-only contract.
	Handler        string      `json:"handler,omitempty"`
	IdentityFields []string    `json:"identityFields,omitempty"`
	DefaultSort    []SortValue `json:"defaultSort,omitempty"`
	// ProviderDigest is Host-derived executable mapping identity. A non-empty
	// value is accepted only with private material from BindExecutableRuntime.
	ProviderDigest string `json:"providerDigest,omitempty"`

	boundResultSchema *compiledResultSchema
	// boundProvider holds non-serializable executable material. JSON encode
	// drops it so round-trips fail closed without re-binding.
	boundProvider *boundExecutableProvider
}

// ResultFilterDeclaration is one independent post-provider filter owned by the
// same exact publication revision as its target query metadata.
type ResultFilterDeclaration struct {
	ID                   string                  `json:"id"`
	ContractVersion      string                  `json:"contractVersion"`
	QueryID              string                  `json:"queryId"`
	QueryContractVersion string                  `json:"queryContractVersion"`
	QueryPlanVersion     string                  `json:"queryPlanVersion"`
	Handler              string                  `json:"handler"`
	Priority             int                     `json:"priority,omitempty"`
	FailurePolicy        string                  `json:"failurePolicy,omitempty"`
	TimeoutMS            int                     `json:"timeoutMs,omitempty"`
	Dependency           *ResultFilterDependency `json:"dependency,omitempty"`
	// IdentityFields are Host-copied from the target query owner. Filter authors
	// cannot self-select decorative identity.
	IdentityFields []string `json:"identityFields,omitempty"`
	// FilterDigest is Host-derived executable mapping identity. A non-empty
	// value is accepted only with private material from BindExecutableRuntime.
	FilterDigest string `json:"filterDigest,omitempty"`

	boundFilter *boundExecutableFilter
}

// Publication is one exact-artifact owner. Empty query lists remain valid for
// Host-owned catalogs even though lifecycle publication omits queryless plugins.
// Executable providers, independent result filters, and result Schemas join the
// same immutable Registry revision when bound.
type Publication struct {
	Artifact      Artifact                  `json:"artifact"`
	Queries       []QueryDeclaration        `json:"queries,omitempty"`
	ResultFilters []ResultFilterDeclaration `json:"resultFilters,omitempty"`
}

// QueryContribution is a frozen declaration plus its exact owning artifact.
type QueryContribution struct {
	QueryDeclaration
	Artifact Artifact `json:"artifact"`
}

// Snapshot is an immutable, inspectable view of the active registry graph.
type Snapshot struct {
	SchemaVersion string              `json:"schemaVersion"`
	Revision      uint64              `json:"revision"`
	Digest        string              `json:"digest"`
	SafeMode      bool                `json:"safeMode,omitempty"`
	Publications  []Publication       `json:"publications"`
	Queries       []QueryContribution `json:"queries"`
}

// CacheState fences cached plans with local revision plus restart-stable digest.
type CacheState struct {
	Revision uint64 `json:"revision"`
	Digest   string `json:"digest"`
	SafeMode bool   `json:"safeMode,omitempty"`
}

// ProviderRef names one exact provider slot used by a plan.
type ProviderRef struct {
	Kind           string   `json:"kind"`
	Name           string   `json:"name"`
	ContributionID string   `json:"contributionId"`
	Artifact       Artifact `json:"artifact"`
}

// FilterValue is one requested filter against a declared filter slot.
// Values are Host-normalized opaque strings so the registry never executes them.
type FilterValue struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// SortValue is one requested sort against a declared sort slot.
type SortValue struct {
	Field      string `json:"field"`
	Descending bool   `json:"descending,omitempty"`
}

// PaginationRequest carries offset/cursor/limit for plan validation.
type PaginationRequest struct {
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// PaginationPlan is the normalized pagination admitted for execution.
type PaginationPlan struct {
	Mode   string `json:"mode"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// QueryCost is a deterministic pre-execution budget estimate.
type QueryCost struct {
	Units   int `json:"units"`
	Maximum int `json:"maximum"`
}

// QueryCostInput is the complete normalized shape supplied to a Host-owned
// deterministic cost policy. The Registry deliberately ships no default cost
// weights because the V3 ADR leaves that product boundary open.
type QueryCostInput struct {
	Query            QueryContribution `json:"query"`
	Fields           []string          `json:"fields"`
	Relations        []string          `json:"relations,omitempty"`
	Filters          []FilterValue     `json:"filters,omitempty"`
	Sorts            []SortValue       `json:"sorts,omitempty"`
	Pagination       PaginationPlan    `json:"pagination"`
	RequestedMaximum int               `json:"requestedMaximum,omitempty"`
}

// CostPolicy must return the same result for the same normalized input across
// nodes and repeated release checks. Production planning remains closed until
// the Host installs an accepted policy.
type CostPolicy interface {
	EstimateQueryCost(QueryCostInput) (QueryCost, error)
}

type CostPolicyFunc func(QueryCostInput) (QueryCost, error)

func (f CostPolicyFunc) EstimateQueryCost(input QueryCostInput) (QueryCost, error) {
	if f == nil {
		return QueryCost{}, ErrContractInsufficient
	}
	return f(input)
}

// PermissionClaim is the exact Host-owned recheck payload. Callers never supply
// the permission policy; it is always taken from the frozen declaration.
type PermissionClaim struct {
	QueryID          string   `json:"queryId"`
	ContractVersion  string   `json:"contractVersion"`
	PlanVersion      string   `json:"planVersion"`
	Entity           string   `json:"entity"`
	PermissionPolicy string   `json:"permissionPolicy"`
	ResultSchema     string   `json:"resultSchema"`
	ShapeDigest      string   `json:"shapeDigest"`
	Artifact         Artifact `json:"artifact"`
	Locale           string   `json:"locale,omitempty"`
	Scope            string   `json:"scope,omitempty"`
}

// PermissionRecheck is the Host-owned authorization callback. A non-nil
// implementation is required for every non-public plan; policy fingerprints and
// caller-supplied permission strings alone never authorize.
type PermissionRecheck interface {
	// For the login policy this callback must revalidate current Host session
	// authority; Authenticated is only the request projection checked alongside it.
	AuthorizeQuery(ctx context.Context, claim PermissionClaim) error
}

// PermissionInput carries Host-derived actor isolation metadata plus the
// mandatory recheck callback. Permissions lists are intentionally absent.
type PermissionInput struct {
	ActorFingerprint  string
	Authenticated     bool
	PolicyFingerprint string
	Recheck           PermissionRecheck
}

// PlanRequest is the typed query plan request validated before any execution.
type PlanRequest struct {
	QueryID      string
	PlanVersion  string
	ResultSchema string
	Fields       []string
	Relations    []string
	Filters      []FilterValue
	Sorts        []SortValue
	Pagination   PaginationRequest
	// ResultFilters is intentionally unsupported by frozen ManifestQuery.
	// Non-empty values fail closed with ErrContractInsufficient.
	ResultFilters []string
	Permission    PermissionInput
	Locale        string
	Scope         string
	// MaxCost is Host-derived input to the configured policy; it never installs
	// or overrides a Registry policy by itself.
	MaxCost int
}

// QueryPlan is immutable Host-owned material ready for a Host executor. It is
// not a bearer token and must never be reconstructed from untrusted input.
// CacheKey is an unkeyed derived cache identity, not an authenticity MAC; final
// release authority comes only from RecheckBeforeRelease and its Host callback.
// This package never executes the plan.
type QueryPlan struct {
	SchemaVersion    string            `json:"schemaVersion"`
	Revision         uint64            `json:"revision"`
	Digest           string            `json:"digest"`
	ShapeDigest      string            `json:"shapeDigest"`
	CacheKey         string            `json:"cacheKey"`
	CacheTags        []string          `json:"cacheTags"`
	Query            QueryContribution `json:"query"`
	Fields           []string          `json:"fields"`
	Relations        []string          `json:"relations,omitempty"`
	Filters          []FilterValue     `json:"filters,omitempty"`
	Sorts            []SortValue       `json:"sorts,omitempty"`
	Pagination       PaginationPlan    `json:"pagination"`
	Cost             QueryCost         `json:"cost"`
	RequestedMaximum int               `json:"requestedMaximum,omitempty"`
	// Recheck is the exact claim required for a second Host recheck before
	// result release. Callers must not authorize from CacheKey alone.
	Recheck   PermissionClaim `json:"recheck"`
	Providers []ProviderRef   `json:"providers"`
	Locale    string          `json:"locale,omitempty"`
	Scope     string          `json:"scope,omitempty"`
	// ActorFingerprint / PolicyFingerprint are retained so cache isolation
	// cannot be reconstructed without the Host-derived actor projection.
	ActorFingerprint  string `json:"actorFingerprint,omitempty"`
	PolicyFingerprint string `json:"policyFingerprint,omitempty"`
}

// PermissionRecheckFunc adapts a function to PermissionRecheck.
type PermissionRecheckFunc func(ctx context.Context, claim PermissionClaim) error

func (f PermissionRecheckFunc) AuthorizeQuery(ctx context.Context, claim PermissionClaim) error {
	if f == nil {
		return ErrDenied
	}
	return f(ctx, claim)
}
