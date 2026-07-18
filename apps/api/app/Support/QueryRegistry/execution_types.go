package queryregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
)

const (
	ProviderFailureFailClosed = "fail_closed"
	ResultFilterFailClosed    = "fail_closed"
	ResultFilterFailOpen      = "fail_open"

	defaultExecutionTimeout = 5 * time.Second
	maximumExecutionTimeout = 30 * time.Second
	defaultFilterTimeout    = time.Second
	maximumFilterTimeout    = 5 * time.Second
	defaultResultBytes      = 2 << 20
	maximumResultBytes      = 8 << 20
	maximumExecutionRows    = maximumPageLimit + 1
	maximumResultFilters    = 64
)

var (
	ErrExecutionInvalid    = errors.New("query registry execution configuration is invalid")
	ErrProviderUnavailable = errors.New("query registry executable provider is unavailable")
	ErrProviderFailed      = errors.New("query registry executable provider failed")
	ErrDependencyDenied    = errors.New("query registry executable dependency is unavailable or incompatible")
	ErrResultInvalid       = errors.New("query registry result failed its release schema")
	ErrResultTooLarge      = errors.New("query registry result exceeds Host bounds")
	ErrCachePoisoned       = errors.New("query registry cached result failed its release fence")
)

// QueryRow is the JSON-compatible typed record exchanged at the provider and
// result-filter boundaries. The runtime always clones it before ownership
// crosses a boundary.
type QueryRow map[string]any

type ProviderExecutionRequest struct {
	Plan       QueryPlan `json:"plan"`
	FetchLimit int       `json:"fetchLimit"`
}

type ProviderExecutionResult struct {
	Rows []QueryRow `json:"rows"`
}

type ExecutableProvider interface {
	// Production adapters must stop when ctx is cancelled. The Host deliberately
	// does not detach an in-flight call and release its exact runtime lease while
	// extension code may still be executing.
	ExecuteQuery(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error)
}

type ExecutableProviderFunc func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error)

func (f ExecutableProviderFunc) ExecuteQuery(ctx context.Context, request ProviderExecutionRequest) (ProviderExecutionResult, error) {
	if f == nil {
		return ProviderExecutionResult{}, ErrProviderUnavailable
	}
	return f(ctx, request)
}

// ExecutableProviderBinding freezes a callable implementation to the exact
// query declaration and artifact. Query providers have no declared fallback in
// ManifestQuery, so only fail_closed is accepted.
type ExecutableProviderBinding struct {
	QueryID         string
	ContractVersion string
	PlanVersion     string
	ResultSchema    string
	Artifact        Artifact
	// ProviderDigest identifies the immutable executable mapping behind this
	// exact query binding. Empty input derives the artifact-bound default;
	// adapters that bridge another catalog must digest that mapping explicitly.
	ProviderDigest string
	FailurePolicy  string
	Provider       ExecutableProvider
}

type ExecutableProviderResolver interface {
	// resolveQueryProvider is deliberately package-private. Callers may construct
	// or pass a resolver into ExecutionRuntime, but only the runtime may recover
	// the raw executable provider from a Host-owned snapshot.
	resolveQueryProvider(context.Context, QueryPlan) (ExecutableProviderBinding, error)
}

type executableProviderResolverFunc func(context.Context, QueryPlan) (ExecutableProviderBinding, error)

func (f executableProviderResolverFunc) resolveQueryProvider(ctx context.Context, plan QueryPlan) (ExecutableProviderBinding, error) {
	if f == nil {
		return ExecutableProviderBinding{}, ErrProviderUnavailable
	}
	return f(ctx, cloneQueryPlan(plan))
}

// NewProviderResolverFunc wraps an external resolve callback as an
// ExecutableProviderResolver. The package-private resolve method stays
// unexportable; adapters outside this package must use this factory.
func NewProviderResolverFunc(
	resolve func(context.Context, QueryPlan) (ExecutableProviderBinding, error),
) ExecutableProviderResolver {
	if resolve == nil {
		return executableProviderResolverFunc(nil)
	}
	return executableProviderResolverFunc(resolve)
}

// NewFallbackProviderResolver tries primary first. On ErrProviderUnavailable it
// falls back for non-Core plans only; Core seals never leave the primary path.
func NewFallbackProviderResolver(
	primary ExecutableProviderResolver,
	fallback ExecutableProviderResolver,
) (ExecutableProviderResolver, error) {
	if primary == nil || fallback == nil {
		return nil, ErrExecutionInvalid
	}
	return executableProviderResolverFunc(func(ctx context.Context, plan QueryPlan) (ExecutableProviderBinding, error) {
		binding, err := primary.resolveQueryProvider(ctx, plan)
		if err == nil {
			return binding, nil
		}
		if plan.Query.Artifact.Core || !errors.Is(err, ErrProviderUnavailable) {
			return ExecutableProviderBinding{}, err
		}
		return fallback.resolveQueryProvider(ctx, plan)
	}), nil
}

// ExecutionAdmission holds the Host runtime's exact-artifact admission lease
// across provider or filter code. Core artifacts bypass this process lease but
// remain snapshot-fenced by Registry.
type ExecutionAdmission interface {
	AcquireQueryExecution(context.Context, Artifact) (release func(), err error)
}

type ExecutionAdmissionFunc func(context.Context, Artifact) (func(), error)

func (f ExecutionAdmissionFunc) AcquireQueryExecution(ctx context.Context, artifact Artifact) (func(), error) {
	if f == nil {
		return nil, ErrArtifactUnavailable
	}
	return f(ctx, artifact)
}

type staticProviderResolver struct {
	bindings map[string]ExecutableProviderBinding
}

type executableProviderInspector interface {
	inspectQueryProvider(QueryContribution) (ExecutableProviderBinding, bool)
}

// NewStaticProviderResolver creates an immutable exact binding catalog suitable
// for Host bootstrap snapshots and tests.
func NewStaticProviderResolver(bindings []ExecutableProviderBinding) (ExecutableProviderResolver, error) {
	result := &staticProviderResolver{bindings: make(map[string]ExecutableProviderBinding, len(bindings))}
	for _, raw := range bindings {
		binding, err := normalizeProviderBinding(raw)
		if err != nil {
			return nil, err
		}
		if _, exists := result.bindings[binding.QueryID]; exists {
			return nil, fmt.Errorf("%w: duplicate executable provider %s", ErrExecutionInvalid, binding.QueryID)
		}
		result.bindings[binding.QueryID] = binding
	}
	return result, nil
}

func (r *staticProviderResolver) resolveQueryProvider(_ context.Context, plan QueryPlan) (ExecutableProviderBinding, error) {
	if r == nil {
		return ExecutableProviderBinding{}, ErrProviderUnavailable
	}
	binding, ok := r.bindings[plan.Query.ID]
	if !ok || !providerBindingMatchesPlan(binding, plan) {
		return ExecutableProviderBinding{}, ErrProviderUnavailable
	}
	return binding, nil
}

func (r *staticProviderResolver) inspectQueryProvider(query QueryContribution) (ExecutableProviderBinding, bool) {
	if r == nil {
		return ExecutableProviderBinding{}, false
	}
	binding, ok := r.bindings[query.ID]
	if !ok || binding.QueryID != query.ID || binding.ContractVersion != query.ContractVersion ||
		binding.PlanVersion != query.PlanVersion || binding.ResultSchema != query.ResultSchema ||
		binding.Artifact != query.Artifact || !digestPattern.MatchString(binding.ProviderDigest) {
		return ExecutableProviderBinding{}, false
	}
	return binding, true
}

type ResultSchemaClaim struct {
	QueryID         string
	ContractVersion string
	PlanVersion     string
	ResultSchema    string
	ShapeDigest     string
	Artifact        Artifact
	Fields          []string
	Relations       []string
	RowIndex        int
}

type ResultSchemaValidator interface {
	// Production validators are Host-owned and must honor cancellation where the
	// implementation performs work beyond an in-memory compiled schema check.
	ValidateQueryResult(context.Context, ResultSchemaClaim, QueryRow) error
}

type ResultSchemaValidatorFunc func(context.Context, ResultSchemaClaim, QueryRow) error

func (f ResultSchemaValidatorFunc) ValidateQueryResult(ctx context.Context, claim ResultSchemaClaim, row QueryRow) error {
	if f == nil {
		return ErrResultInvalid
	}
	return f(ctx, claim, row)
}

type ResultFilterDependency struct {
	ExtensionID       string `json:"extensionId"`
	VersionConstraint string `json:"versionConstraint"`
}

type ResultFilterRequest struct {
	Plan QueryPlan  `json:"plan"`
	Rows []QueryRow `json:"rows"`
}

type ResultFilterResult struct {
	Rows []QueryRow `json:"rows"`
}

type ResultFilter interface {
	// Production adapters must stop on cancellation for the same lease-safety
	// reason as ExecutableProvider.
	FilterQueryResult(context.Context, ResultFilterRequest) (ResultFilterResult, error)
}

type ResultFilterFunc func(context.Context, ResultFilterRequest) (ResultFilterResult, error)

func (f ResultFilterFunc) FilterQueryResult(ctx context.Context, request ResultFilterRequest) (ResultFilterResult, error) {
	if f == nil {
		return ResultFilterResult{}, ErrProviderUnavailable
	}
	return f(ctx, request)
}

// ResultFilterRegistration is a Host-bound execution registration. Frozen
// ManifestQuery cannot publish these registrations yet; callers must not derive
// them from undeclared naming conventions.
type ResultFilterRegistration struct {
	ID                   string
	ContractVersion      string
	QueryID              string
	QueryContractVersion string
	QueryPlanVersion     string
	Priority             int
	Artifact             Artifact
	Dependency           ResultFilterDependency
	IdentityFields       []string
	FailurePolicy        string
	Timeout              time.Duration
	Filter               ResultFilter
}

// ResultFilterSource materializes independent result-filter registrations from
// an immutable Registry snapshot or other Host-owned catalog at match time.
// Callables may resolve live exact runtimes; the source itself must not capture
// process-start-only instances.
type ResultFilterSource interface {
	ResultFiltersFor(QueryContribution) ([]ResultFilterRegistration, error)
}

type ResultFilterSourceFunc func(QueryContribution) ([]ResultFilterRegistration, error)

func (f ResultFilterSourceFunc) ResultFiltersFor(query QueryContribution) ([]ResultFilterRegistration, error) {
	if f == nil {
		return nil, ErrExecutionInvalid
	}
	return f(query)
}

type preparedResultFilter struct {
	registration ResultFilterRegistration
	constraint   *semver.Constraints
}

type QueryResultPage struct {
	Mode       string `json:"mode"`
	Offset     int    `json:"offset,omitempty"`
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"hasMore"`
	NextOffset int    `json:"nextOffset,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type QueryResult struct {
	Rows           []QueryRow      `json:"rows"`
	Page           QueryResultPage `json:"page"`
	CacheKey       string          `json:"cacheKey"`
	CacheTags      []string        `json:"cacheTags,omitempty"`
	CacheHit       bool            `json:"cacheHit,omitempty"`
	Revision       uint64          `json:"revision"`
	Digest         string          `json:"digest"`
	FilterPlan     string          `json:"filterPlan"`
	ProviderDigest string          `json:"providerDigest"`
}

type CachedQueryResult struct {
	SchemaVersion    string
	CacheKey         string
	RegistryRevision uint64
	RegistryDigest   string
	ShapeDigest      string
	FilterPlan       string
	QueryID          string
	ContractVersion  string
	PlanVersion      string
	ResultSchema     string
	Artifact         Artifact
	ProviderDigest   string
	Page             QueryResultPage
	Rows             []QueryRow
	CacheTags        []string
}

type QueryResultCache interface {
	LoadQueryResult(context.Context, string) (CachedQueryResult, bool, error)
	StoreQueryResult(context.Context, string, CachedQueryResult, []string) error
}

type ExecutionConfig struct {
	Registry           *Registry
	Providers          ExecutableProviderResolver
	Admission          ExecutionAdmission
	Schemas            ResultSchemaValidator
	Cache              QueryResultCache
	ResultFilters      []ResultFilterRegistration
	ResultFilterSource ResultFilterSource
	Trace              ExecutionTraceSink
	Timeout            time.Duration
	MaxResultBytes     int
}

type ExecutionRuntime struct {
	registry       *Registry
	providers      ExecutableProviderResolver
	admission      ExecutionAdmission
	schemas        ResultSchemaValidator
	cache          QueryResultCache
	filters        []preparedResultFilter
	filterSource   ResultFilterSource
	trace          ExecutionTraceSink
	timeout        time.Duration
	maxResultBytes int
}

func NewExecutionRuntime(config ExecutionConfig) (*ExecutionRuntime, error) {
	if config.Registry == nil || config.Providers == nil || config.Schemas == nil ||
		len(config.ResultFilters) > maximumResultFilters {
		return nil, ErrExecutionInvalid
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = defaultExecutionTimeout
	}
	maximumBytes := config.MaxResultBytes
	if maximumBytes == 0 {
		maximumBytes = defaultResultBytes
	}
	if timeout < time.Millisecond || timeout > maximumExecutionTimeout || maximumBytes < 1024 || maximumBytes > maximumResultBytes {
		return nil, ErrExecutionInvalid
	}
	filters, err := prepareResultFilters(config.ResultFilters)
	if err != nil {
		return nil, err
	}
	return &ExecutionRuntime{
		registry: config.Registry, providers: config.Providers, admission: config.Admission, schemas: config.Schemas,
		cache: config.Cache, filters: filters, filterSource: config.ResultFilterSource, trace: config.Trace,
		timeout: timeout, maxResultBytes: maximumBytes,
	}, nil
}

func normalizeProviderBinding(input ExecutableProviderBinding) (ExecutableProviderBinding, error) {
	input.QueryID = strings.ToLower(strings.TrimSpace(input.QueryID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.PlanVersion = strings.TrimSpace(input.PlanVersion)
	input.ResultSchema = strings.TrimSpace(input.ResultSchema)
	input.ProviderDigest = normalizeDigest(input.ProviderDigest)
	input.FailurePolicy = strings.ToLower(strings.TrimSpace(input.FailurePolicy))
	if input.FailurePolicy == "" {
		input.FailurePolicy = ProviderFailureFailClosed
	}
	artifact, err := normalizeArtifact(input.Artifact)
	input.Artifact = artifact
	if input.ProviderDigest == "" && err == nil {
		input.ProviderDigest = defaultExecutableProviderDigest(input)
	}
	if err != nil || !idPattern.MatchString(input.QueryID) || !contractPattern.MatchString(input.ContractVersion) ||
		!contractPattern.MatchString(input.PlanVersion) || !validSchemaRef(input.ResultSchema) ||
		!digestPattern.MatchString(input.ProviderDigest) ||
		input.FailurePolicy != ProviderFailureFailClosed || input.Provider == nil {
		return ExecutableProviderBinding{}, ErrExecutionInvalid
	}
	return input, nil
}

func providerBindingMatchesPlan(binding ExecutableProviderBinding, plan QueryPlan) bool {
	return binding.QueryID == plan.Query.ID && binding.ContractVersion == plan.Query.ContractVersion &&
		binding.PlanVersion == plan.Query.PlanVersion && binding.ResultSchema == plan.Query.ResultSchema &&
		binding.Artifact == plan.Query.Artifact && binding.FailurePolicy == ProviderFailureFailClosed &&
		digestPattern.MatchString(binding.ProviderDigest) && binding.Provider != nil
}

func defaultExecutableProviderDigest(binding ExecutableProviderBinding) string {
	material := SchemaVersion + "\x00executable-provider\x00" + binding.QueryID + "\x00" +
		binding.ContractVersion + "\x00" + binding.PlanVersion + "\x00" + binding.ResultSchema + "\x00" +
		providerKey(ProviderRef{Kind: ProviderKindQuery, Name: binding.QueryID,
			ContributionID: binding.QueryID, Artifact: binding.Artifact})
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func prepareResultFilters(input []ResultFilterRegistration) ([]preparedResultFilter, error) {
	result := make([]preparedResultFilter, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		item, err := prepareResultFilter(raw)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[item.registration.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate result filter %s", ErrExecutionInvalid, item.registration.ID)
		}
		seen[item.registration.ID] = struct{}{}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i].registration, result[j].registration
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if left.Artifact != right.Artifact {
			return artifactBefore(left.Artifact, right.Artifact)
		}
		return left.ID < right.ID
	})
	return result, nil
}

func prepareResultFilter(raw ResultFilterRegistration) (preparedResultFilter, error) {
	raw.ID = strings.ToLower(strings.TrimSpace(raw.ID))
	raw.ContractVersion = strings.TrimSpace(raw.ContractVersion)
	raw.QueryID = strings.ToLower(strings.TrimSpace(raw.QueryID))
	raw.QueryContractVersion = strings.TrimSpace(raw.QueryContractVersion)
	raw.QueryPlanVersion = strings.TrimSpace(raw.QueryPlanVersion)
	raw.Dependency.ExtensionID = strings.ToLower(strings.TrimSpace(raw.Dependency.ExtensionID))
	raw.Dependency.VersionConstraint = strings.TrimSpace(raw.Dependency.VersionConstraint)
	raw.FailurePolicy = strings.ToLower(strings.TrimSpace(raw.FailurePolicy))
	if raw.FailurePolicy == "" {
		raw.FailurePolicy = ResultFilterFailClosed
	}
	if raw.Timeout == 0 {
		raw.Timeout = defaultFilterTimeout
	}
	artifact, err := normalizeArtifact(raw.Artifact)
	identityFields, identityErr := normalizeNameList(raw.IdentityFields, 8, false)
	if err != nil || !validContributionIdentity(artifact, raw.ID, raw.ContractVersion) ||
		!idPattern.MatchString(raw.QueryID) || !contractPattern.MatchString(raw.QueryContractVersion) ||
		!contractPattern.MatchString(raw.QueryPlanVersion) || raw.Filter == nil ||
		(raw.FailurePolicy != ResultFilterFailClosed && raw.FailurePolicy != ResultFilterFailOpen) ||
		raw.Timeout < time.Millisecond || raw.Timeout > maximumFilterTimeout || identityErr != nil {
		return preparedResultFilter{}, ErrExecutionInvalid
	}
	raw.Artifact = artifact
	raw.IdentityFields = identityFields
	var constraint *semver.Constraints
	if raw.Dependency.ExtensionID != "" || raw.Dependency.VersionConstraint != "" {
		if !idPattern.MatchString(raw.Dependency.ExtensionID) || raw.Dependency.VersionConstraint == "" {
			return preparedResultFilter{}, ErrExecutionInvalid
		}
		constraint, err = semver.NewConstraint(raw.Dependency.VersionConstraint)
		if err != nil {
			return preparedResultFilter{}, ErrExecutionInvalid
		}
	}
	return preparedResultFilter{registration: raw, constraint: constraint}, nil
}
