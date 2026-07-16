package seoregistry

import (
	"context"
	"errors"
	"time"
)

const SchemaVersion = "sforum.seo-registry@1"

const (
	ActionAdd     = "add"
	ActionFilter  = "filter"
	ActionReplace = "replace"
)

const (
	KindTitle     = "title"
	KindMeta      = "meta"
	KindCanonical = "canonical"
	KindRobots    = "robots"
	KindHreflang  = "hreflang"
	KindSitemap   = "sitemap"
	KindJSONLD    = "jsonld"
)

const (
	FailurePolicyFailClosed = "fail_closed"
	FailurePolicyFallback   = "fallback"
	GlobalScope             = "global"
)

const (
	RobotsIndex    = "index"
	RobotsNoIndex  = "noindex"
	RobotsFollow   = "follow"
	RobotsNoFollow = "nofollow"
)

const (
	SitemapAlways  = "always"
	SitemapHourly  = "hourly"
	SitemapDaily   = "daily"
	SitemapWeekly  = "weekly"
	SitemapMonthly = "monthly"
	SitemapYearly  = "yearly"
	SitemapNever   = "never"
)

var (
	ErrInvalid             = errors.New("seo registry declaration is invalid")
	ErrConflict            = errors.New("seo registry declaration conflicts with the active graph")
	ErrArtifactConflict    = errors.New("seo registry artifact does not own the active publication")
	ErrRevisionConflict    = errors.New("seo registry revision changed during replacement")
	ErrSafeMode            = errors.New("seo registry rejects publication in safe mode")
	ErrNotFound            = errors.New("seo registry scope has no contributions")
	ErrExecutionInvalid    = errors.New("seo registry execution configuration is invalid")
	ErrProviderUnavailable = errors.New("seo registry executable provider is unavailable")
	ErrProviderFailed      = errors.New("seo registry executable provider failed")
	ErrProviderDeadline    = errors.New("seo registry executable provider exceeded its deadline")
	ErrArtifactUnavailable = errors.New("seo registry exact runtime artifact is unavailable")
	ErrSnapshotStale       = errors.New("seo registry snapshot changed during execution")
	ErrMutationDenied      = errors.New("seo registry provider mutated an undeclared field or action")
	ErrPolicyDenied        = errors.New("seo registry Host final policy rejected output")
	ErrOutputInvalid       = errors.New("seo registry output failed validation")
	ErrOutputTooLarge      = errors.New("seo registry output exceeds Host bounds")
)

// Artifact binds every declaration and executable provider to one exact
// package and runtime. The unexported seal prevents decoded extension input
// from claiming core authority.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	ImpactDigest      string `json:"impactDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	coreSeal          [32]byte
}

// Declaration grants one provider authority over one SEO family in either a
// stable page/route scope or GlobalScope. Failure policy is mandatory: omitted
// policy never silently becomes fail-open.
type Declaration struct {
	ID              string        `json:"id"`
	ContractVersion string        `json:"contractVersion"`
	Scope           string        `json:"scope"`
	Kind            string        `json:"kind"`
	Action          string        `json:"action"`
	Handler         string        `json:"handler"`
	Priority        int           `json:"priority,omitempty"`
	FailurePolicy   string        `json:"failurePolicy"`
	Timeout         time.Duration `json:"timeout"`
}

type Publication struct {
	Artifact      Artifact      `json:"artifact"`
	Contributions []Declaration `json:"contributions,omitempty"`
}

type Contribution struct {
	Declaration
	Artifact Artifact `json:"artifact"`
}

type Conflict struct {
	Scope      string         `json:"scope"`
	Kind       string         `json:"kind"`
	Action     string         `json:"action"`
	Candidates []Contribution `json:"candidates"`
	Winner     Contribution   `json:"winner"`
}

type Snapshot struct {
	SchemaVersion string         `json:"schemaVersion"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	SafeMode      bool           `json:"safeMode,omitempty"`
	Publications  []Publication  `json:"publications"`
	Contributions []Contribution `json:"contributions"`
	Conflicts     []Conflict     `json:"conflicts,omitempty"`
}

type CacheState struct {
	Revision uint64 `json:"revision"`
	Digest   string `json:"digest"`
	SafeMode bool   `json:"safeMode,omitempty"`
}

// Document is the complete typed SEO boundary. It intentionally contains no
// arbitrary attributes, maps, HTML, or raw JSON.
type Document struct {
	Title        string           `json:"title,omitempty"`
	Meta         []MetaTag        `json:"meta,omitempty"`
	CanonicalURL string           `json:"canonicalUrl,omitempty"`
	Robots       RobotsDirectives `json:"robots,omitempty"`
	Hreflang     []HreflangLink   `json:"hreflang,omitempty"`
	Sitemap      []SitemapEntry   `json:"sitemap,omitempty"`
	JSONLD       []JSONLDDocument `json:"jsonLd,omitempty"`
}

// MetaTag permits only ordinary name/property metadata. HTTP-equiv, charset,
// event handlers, and arbitrary attributes are deliberately outside the type.
type MetaTag struct {
	Attribute string `json:"attribute"`
	Key       string `json:"key"`
	Content   string `json:"content"`
}

type RobotsDirectives struct {
	Indexing     string `json:"indexing,omitempty"`
	Following    string `json:"following,omitempty"`
	NoArchive    bool   `json:"noArchive,omitempty"`
	NoImageIndex bool   `json:"noImageIndex,omitempty"`
	NoSnippet    bool   `json:"noSnippet,omitempty"`
}

type HreflangLink struct {
	Locale string `json:"locale"`
	URL    string `json:"url"`
}

type SitemapEntry struct {
	URL             string   `json:"url"`
	LastModified    string   `json:"lastModified,omitempty"`
	ChangeFrequency string   `json:"changeFrequency,omitempty"`
	Priority        *float64 `json:"priority,omitempty"`
}

// JSONLDDocument is a fixed-shape schema.org graph node. Supporting another
// schema property requires a versioned type change instead of accepting raw
// maps from executable extensions.
type JSONLDDocument struct {
	Context       string             `json:"@context"`
	Type          string             `json:"@type"`
	ID            string             `json:"@id,omitempty"`
	URL           string             `json:"url,omitempty"`
	Name          string             `json:"name,omitempty"`
	Headline      string             `json:"headline,omitempty"`
	Description   string             `json:"description,omitempty"`
	ImageURLs     []string           `json:"image,omitempty"`
	DatePublished string             `json:"datePublished,omitempty"`
	DateModified  string             `json:"dateModified,omitempty"`
	Author        []JSONLDParty      `json:"author,omitempty"`
	Publisher     *JSONLDParty       `json:"publisher,omitempty"`
	Breadcrumbs   []JSONLDBreadcrumb `json:"itemListElement,omitempty"`
}

type JSONLDParty struct {
	Type    string `json:"@type"`
	ID      string `json:"@id,omitempty"`
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	LogoURL string `json:"logo,omitempty"`
}

type JSONLDBreadcrumb struct {
	Type     string `json:"@type"`
	Position int    `json:"position"`
	Name     string `json:"name"`
	URL      string `json:"item"`
}

type ExecuteRequest struct {
	Scope string   `json:"scope"`
	Base  Document `json:"base"`
}

type ExecuteResult struct {
	SchemaVersion string            `json:"schemaVersion"`
	Revision      uint64            `json:"revision"`
	Digest        string            `json:"digest"`
	Document      Document          `json:"document"`
	Applied       []ContributionRef `json:"applied,omitempty"`
	Fallbacks     []FallbackRecord  `json:"fallbacks,omitempty"`
}

type ContributionRef struct {
	ID       string   `json:"id"`
	Action   string   `json:"action"`
	Kind     string   `json:"kind"`
	Artifact Artifact `json:"artifact"`
}

type FallbackRecord struct {
	Contribution ContributionRef `json:"contribution"`
	Reason       string          `json:"reason"`
}

type ProviderRequest struct {
	Scope        string       `json:"scope"`
	Contribution Contribution `json:"contribution"`
	Current      Document     `json:"current"`
}

type ProviderResult struct {
	Document Document `json:"document"`
}

type Provider interface {
	// Implementations must stop on cancellation. The Host keeps the exact
	// runtime lease until the final snapshot fence, so it never detaches a live
	// callback and releases its runtime underneath executing extension code.
	ApplySEO(context.Context, ProviderRequest) (ProviderResult, error)
}

type ProviderFunc func(context.Context, ProviderRequest) (ProviderResult, error)

func (f ProviderFunc) ApplySEO(ctx context.Context, request ProviderRequest) (ProviderResult, error) {
	if f == nil {
		return ProviderResult{}, ErrProviderUnavailable
	}
	return f(ctx, request)
}

type ProviderBinding struct {
	ContributionID  string
	ContractVersion string
	Handler         string
	Artifact        Artifact
	ProviderDigest  string
	Provider        Provider
}

// ProviderResolver resolves one exact Registry contribution to its executable
// transport binding. Production uses a Manager-backed resolver so lifecycle
// snapshot swaps do not leave a startup-only callback catalog behind.
type ProviderResolver interface {
	ResolveSEOProvider(context.Context, Contribution) (ProviderBinding, error)
}

type ProviderResolverFunc func(context.Context, Contribution) (ProviderBinding, error)

func (f ProviderResolverFunc) ResolveSEOProvider(ctx context.Context, contribution Contribution) (ProviderBinding, error) {
	if f == nil {
		return ProviderBinding{}, ErrProviderUnavailable
	}
	return f(ctx, cloneContribution(contribution))
}

// AdmissionLease must represent the exact Artifact passed to AcquireSEOExecution.
// Context cancellation is the Host drain signal; Release ends the lease.
type AdmissionLease interface {
	Context() context.Context
	Release()
}

type ExecutionAdmission interface {
	AcquireSEOExecution(context.Context, Artifact) (AdmissionLease, error)
}

type ExecutionAdmissionFunc func(context.Context, Artifact) (AdmissionLease, error)

func (f ExecutionAdmissionFunc) AcquireSEOExecution(ctx context.Context, artifact Artifact) (AdmissionLease, error) {
	if f == nil {
		return nil, ErrArtifactUnavailable
	}
	return f(ctx, artifact)
}

// FinalPolicy is the non-overridable Host SEO policy. Production adapters use
// it to enforce site origins, robots/options policy, and route ownership after
// all extension contributions have been composed.
type FinalPolicy interface {
	ValidateSEO(context.Context, FinalPolicyRequest) error
}

type FinalPolicyFunc func(context.Context, FinalPolicyRequest) error

func (f FinalPolicyFunc) ValidateSEO(ctx context.Context, request FinalPolicyRequest) error {
	if f == nil {
		return ErrPolicyDenied
	}
	return f(ctx, request)
}

type FinalPolicyRequest struct {
	Scope    string   `json:"scope"`
	Base     Document `json:"base"`
	Document Document `json:"document"`
}

type ExecutionConfig struct {
	Registry     *Registry
	Admission    ExecutionAdmission
	FinalPolicy  FinalPolicy
	Providers    []ProviderBinding
	Resolver     ProviderResolver
	Timeout      time.Duration
	MaximumBytes int
	Trace        ExecutionTraceSink
}

type ScopeInspection struct {
	SchemaVersion string         `json:"schemaVersion"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	SafeMode      bool           `json:"safeMode,omitempty"`
	Scope         string         `json:"scope"`
	Contributions []Contribution `json:"contributions"`
	Conflicts     []Conflict     `json:"conflicts,omitempty"`
}

type ProviderInspection struct {
	Contribution   Contribution `json:"contribution"`
	Bound          bool         `json:"bound"`
	ProviderDigest string       `json:"providerDigest,omitempty"`
}

type RuntimeInspection struct {
	Scope     ScopeInspection      `json:"scope"`
	Providers []ProviderInspection `json:"providers"`
}
