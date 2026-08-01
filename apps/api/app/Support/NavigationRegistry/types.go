package navigationregistry

import (
	"context"
	"errors"
)

const SchemaVersion = "sforum.navigation-region-registry@1"

const (
	ActionAdd     = "add"
	ActionBefore  = "before"
	ActionAfter   = "after"
	ActionWrap    = "wrap"
	ActionReplace = "replace"
	ActionHide    = "hide"
	ActionFilter  = "filter"
)

const (
	VisibilityPublic        = "public"
	VisibilityAnonymous     = "anonymous"
	VisibilityAuthenticated = "authenticated"
)

const (
	ProviderFamilyNavigation = "navigation"
	ProviderFamilyRegion     = "region"
)

const (
	NavigationKindMenu       = "menu"
	NavigationKindItem       = "item"
	NavigationKindBreadcrumb = "breadcrumb"
	NavigationKindHeader     = "header"
	NavigationKindFooter     = "footer"
	NavigationKindSidebar    = "sidebar"
	// NavigationKindAccountSettings is a Host-owned personal settings slot.
	NavigationKindAccountSettings = "account-settings"
)

const (
	RegionKindMenu    = "menu"
	RegionKindWidget  = "widget"
	RegionKindHeader  = "header"
	RegionKindFooter  = "footer"
	RegionKindSidebar = "sidebar"
	RegionKindContent = "content"
)

const (
	DependencyRequired = "required"
	DependencyOptional = "optional"
	DependencyConflict = "conflict"
	DependencyProvides = "provides"
)

var (
	ErrInvalid            = errors.New("navigation registry declaration is invalid")
	ErrConflict           = errors.New("navigation registry conflicts with the active graph")
	ErrDependency         = errors.New("navigation registry dependency is unavailable, ambiguous, or cyclic")
	ErrArtifactConflict   = errors.New("navigation registry artifact does not own the active publication")
	ErrRevisionConflict   = errors.New("navigation registry revision changed during replacement")
	ErrSafeMode           = errors.New("navigation registry rejects third-party publication in safe mode")
	ErrProviderSelection  = errors.New("navigation registry provider selection is invalid, stale, or unavailable")
	ErrRuntimeUnavailable = errors.New("navigation registry exact runtime is unavailable")
	ErrTrustedReplace     = errors.New("navigation registry trusted replacement failed closed")
	ErrLimitExceeded      = errors.New("navigation registry composition exceeds a bounded limit")
)

// Artifact binds every contribution to the exact inert package that declared
// it. Core publications use Core=true but still carry stable version/digests so
// inspectors and multi-node snapshots do not lose provenance.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	ImpactDigest      string `json:"impactDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	// coreSeal prevents decoded extension metadata from claiming core.*
	// authority. Only NewCoreArtifact can construct a valid Core artifact.
	coreSeal [32]byte
}

// Dependency mirrors Manifest V3 package-graph semantics. Exactly one of
// ExtensionID and Capability is set for required/optional/conflict entries;
// provides entries set only Capability.
type Dependency struct {
	ExtensionID string `json:"extensionId,omitempty"`
	Capability  string `json:"capability,omitempty"`
	Version     string `json:"version"`
	Kind        string `json:"kind"`
}

type NavigationDeclaration struct {
	ID              string            `json:"id"`
	ContractVersion string            `json:"contractVersion"`
	Kind            string            `json:"kind"`
	Action          string            `json:"action"`
	TargetID        string            `json:"targetId,omitempty"`
	Label           string            `json:"label"`
	Labels          map[string]string `json:"labels,omitempty"`
	Href            string            `json:"href,omitempty"`
	Permission      string            `json:"permission,omitempty"`
	// OwnerResource is a Host-defined retained-resource key. The contribution
	// is visible when the actor owns this key even without the permission.
	OwnerResource string `json:"ownerResource,omitempty"`
	Visibility    string `json:"visibility,omitempty"`
	Handler       string `json:"handler,omitempty"`
	Order         int    `json:"order,omitempty"`
	// Priority orders competing providers/modifiers. Manifest V3 currently
	// defaults this field to zero; callers may supply Host policy priority.
	Priority int `json:"priority,omitempty"`
}

type RegionDeclaration struct {
	ID              string            `json:"id"`
	ContractVersion string            `json:"contractVersion"`
	Action          string            `json:"action"`
	TargetID        string            `json:"targetId,omitempty"`
	Kind            string            `json:"kind"`
	Label           string            `json:"label"`
	Labels          map[string]string `json:"labels,omitempty"`
	Permission      string            `json:"permission,omitempty"`
	Visibility      string            `json:"visibility,omitempty"`
	Handler         string            `json:"handler,omitempty"`
	Multiple        bool              `json:"multiple,omitempty"`
	// Order and Priority are Host policy only. ManifestRegion currently has
	// neither field; both default to zero when projecting from Manifest V3.
	Order    int `json:"order,omitempty"`
	Priority int `json:"priority,omitempty"`
}

type Publication struct {
	Artifact     Artifact                `json:"artifact"`
	Dependencies []Dependency            `json:"dependencies,omitempty"`
	Navigation   []NavigationDeclaration `json:"navigation,omitempty"`
	Regions      []RegionDeclaration     `json:"regions,omitempty"`
}

type NavigationContribution struct {
	NavigationDeclaration
	Artifact Artifact `json:"artifact"`
}

type RegionContribution struct {
	RegionDeclaration
	Artifact Artifact `json:"artifact"`
}

type NavigationProviderConflict struct {
	TargetID   string                   `json:"targetId"`
	Candidates []NavigationContribution `json:"candidates"`
	Winner     NavigationContribution   `json:"winner"`
}

type RegionProviderConflict struct {
	TargetID   string               `json:"targetId"`
	Candidates []RegionContribution `json:"candidates"`
	Winner     RegionContribution   `json:"winner"`
}

type Snapshot struct {
	SchemaVersion string `json:"schemaVersion"`
	Revision      uint64 `json:"revision"`
	Digest        string `json:"digest"`
	SafeMode      bool   `json:"safeMode,omitempty"`
	// Publications retains every exact-artifact owner, including empty graphs
	// that only declare capability/lifecycle identity for dependency consumers.
	Publications        []Publication                `json:"publications"`
	Navigation          []NavigationContribution     `json:"navigation"`
	Regions             []RegionContribution         `json:"regions"`
	NavigationConflicts []NavigationProviderConflict `json:"navigationConflicts,omitempty"`
	RegionConflicts     []RegionProviderConflict     `json:"regionConflicts,omitempty"`
	ProviderSelections  []ProviderSelection          `json:"providerSelections,omitempty"`
}

type CacheState struct {
	Revision uint64 `json:"revision"`
	Digest   string `json:"digest"`
	SafeMode bool   `json:"safeMode,omitempty"`
}

type ProviderRef struct {
	ContributionID string   `json:"contributionId"`
	Artifact       Artifact `json:"artifact"`
}

// VisibilityInput is presentation state supplied by the Host. Permissions are
// a copied actor projection used only to omit UI; this API never authorizes the
// linked route or operation. HiddenIDs carries other Host visibility decisions,
// while DisabledProviders enables exact-artifact request-local fallback after
// quarantine.
type VisibilityInput struct {
	Authenticated     bool          `json:"authenticated,omitempty"`
	Permissions       []string      `json:"permissions,omitempty"`
	OwnedResources    []string      `json:"ownedResources,omitempty"`
	HiddenIDs         []string      `json:"hiddenIds,omitempty"`
	DisabledProviders []ProviderRef `json:"disabledProviders,omitempty"`
}

type NavigationResolveRequest struct {
	Kinds      []string        `json:"kinds,omitempty"`
	Locale     string          `json:"locale,omitempty"`
	Visibility VisibilityInput `json:"visibility"`
}

type RegionResolveRequest struct {
	Kinds      []string        `json:"kinds,omitempty"`
	Locale     string          `json:"locale,omitempty"`
	Visibility VisibilityInput `json:"visibility"`
}

// NavigationTargetPlan is one stable add target plus its selected provider and
// ordered composition chain. Provider falls back to Target when every replace
// candidate is hidden, denied, disabled, or absent from the active snapshot.
type NavigationTargetPlan struct {
	Target              NavigationContribution   `json:"target"`
	ParentID            string                   `json:"parentId,omitempty"`
	Provider            NavigationContribution   `json:"provider"`
	ReplaceCandidates   []NavigationContribution `json:"replaceCandidates,omitempty"`
	Before              []NavigationContribution `json:"before,omitempty"`
	After               []NavigationContribution `json:"after,omitempty"`
	Wrap                []NavigationContribution `json:"wrap,omitempty"`
	Filters             []NavigationContribution `json:"filters,omitempty"`
	UsingFallback       bool                     `json:"usingFallback"`
	SelectionConfigured bool                     `json:"selectionConfigured,omitempty"`
	SelectedProvider    bool                     `json:"selectedProvider,omitempty"`
	hides               []NavigationContribution
}

type RegionTargetPlan struct {
	Target              RegionContribution   `json:"target"`
	ParentID            string               `json:"parentId,omitempty"`
	Provider            RegionContribution   `json:"provider"`
	ReplaceCandidates   []RegionContribution `json:"replaceCandidates,omitempty"`
	Before              []RegionContribution `json:"before,omitempty"`
	After               []RegionContribution `json:"after,omitempty"`
	Wrap                []RegionContribution `json:"wrap,omitempty"`
	Filters             []RegionContribution `json:"filters,omitempty"`
	UsingFallback       bool                 `json:"usingFallback"`
	SelectionConfigured bool                 `json:"selectionConfigured,omitempty"`
	SelectedProvider    bool                 `json:"selectedProvider,omitempty"`
	hides               []RegionContribution
}

type NavigationResolution struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Revision      uint64                 `json:"revision"`
	Digest        string                 `json:"digest"`
	SafeMode      bool                   `json:"safeMode,omitempty"`
	Locale        string                 `json:"locale"`
	CacheKey      string                 `json:"cacheKey"`
	Targets       []NavigationTargetPlan `json:"targets"`
}

type RegionResolution struct {
	SchemaVersion string             `json:"schemaVersion"`
	Revision      uint64             `json:"revision"`
	Digest        string             `json:"digest"`
	SafeMode      bool               `json:"safeMode,omitempty"`
	Locale        string             `json:"locale"`
	CacheKey      string             `json:"cacheKey"`
	Targets       []RegionTargetPlan `json:"targets"`
}

// ProviderSelection is an exact, revisioned administrator choice. Resetting a
// selection restores deterministic priority/order/id resolution; artifact or
// runtime drift never silently retargets it.
type ProviderSelection struct {
	Family                  string      `json:"family"`
	TargetID                string      `json:"targetId"`
	TargetContractVersion   string      `json:"targetContractVersion"`
	ProviderID              string      `json:"providerId"`
	ProviderContractVersion string      `json:"providerContractVersion"`
	Provider                ProviderRef `json:"provider"`
}

type SelectProviderRequest struct {
	ExpectedRevision uint64      `json:"expectedRevision"`
	Family           string      `json:"family"`
	TargetID         string      `json:"targetId"`
	Provider         ProviderRef `json:"provider"`
}

type ResetProviderRequest struct {
	ExpectedRevision uint64 `json:"expectedRevision"`
	Family           string `json:"family"`
	TargetID         string `json:"targetId"`
}

// RuntimeAdmission holds the Host runtime gate across one contribution
// materialization. Static themes may implement the same interface with an
// exact lifecycle lease; executable plugins normally adapt Manager admission.
type RuntimeAdmission interface {
	Available(Artifact) bool
	Acquire(context.Context, Artifact) (RuntimeLease, error)
}

type RuntimeLease interface {
	Context() context.Context
	RuntimeInstanceID() string
	Release()
}
