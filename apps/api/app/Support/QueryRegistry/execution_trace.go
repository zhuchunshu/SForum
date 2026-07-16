package queryregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	TraceOutcomeAllowed                 = "allowed"
	TraceOutcomeDenied                  = "denied"
	TraceOutcomeInvalid                 = "invalid"
	TraceOutcomeCostExceeded            = "cost_exceeded"
	TraceOutcomeProviderMissing         = "provider_unavailable"
	TraceOutcomeProviderFailure         = "provider_failure"
	TraceOutcomeDependencyDenied        = "dependency_denied"
	TraceOutcomeSchemaInvalid           = "schema_invalid"
	TraceOutcomeCachePoisoned           = "cache_poisoned"
	TraceOutcomeSnapshotStale           = "snapshot_stale"
	TraceOutcomeRuntimeStale            = "runtime_stale"
	TraceOutcomeDeadline                = "deadline"
	TraceOutcomeCancelled               = "cancelled"
	TraceOutcomeResultTooLarge          = "result_too_large"
	TraceOutcomeInternal                = "internal"
	ResultFilterTraceApplied            = "applied"
	ResultFilterTraceSkipped            = "skipped"
	ResultFilterTraceFailed             = "failed"
	ResultFilterTraceUnavailable        = "unavailable"
	ResultFilterTraceContractMismatch   = "contract_mismatch"
	ResultFilterTraceDependencyMismatch = "dependency_mismatch"
	defaultExecutionTraceCapacity       = 256
	maximumExecutionTraceCapacity       = 2048
	maximumExecutionTraceRead           = 200
	unplannedExecutionTracePrefix       = "unplanned:"
)

// ExecutionTrace contains bounded metadata only. It deliberately excludes
// filter values, actor fingerprints, rows, cache keys, cursors, and error text.
type ExecutionTrace struct {
	RecordedAt       time.Time                    `json:"recordedAt"`
	Duration         time.Duration                `json:"duration"`
	QueryID          string                       `json:"queryId"`
	ContractVersion  string                       `json:"contractVersion,omitempty"`
	PlanVersion      string                       `json:"planVersion,omitempty"`
	ResultSchema     string                       `json:"resultSchema,omitempty"`
	ExtensionID      string                       `json:"extensionId,omitempty"`
	ExtensionVersion string                       `json:"extensionVersion,omitempty"`
	ArtifactDigest   string                       `json:"artifactDigest,omitempty"`
	Revision         uint64                       `json:"revision,omitempty"`
	SnapshotDigest   string                       `json:"snapshotDigest,omitempty"`
	ShapeDigest      string                       `json:"shapeDigest,omitempty"`
	ProviderDigest   string                       `json:"providerDigest,omitempty"`
	FilterPlan       string                       `json:"filterPlan,omitempty"`
	FilterCount      int                          `json:"filterCount,omitempty"`
	CostUnits        int                          `json:"costUnits,omitempty"`
	CostMaximum      int                          `json:"costMaximum,omitempty"`
	PageMode         string                       `json:"pageMode,omitempty"`
	PageLimit        int                          `json:"pageLimit,omitempty"`
	Rows             int                          `json:"rows,omitempty"`
	CacheStatus      string                       `json:"cacheStatus,omitempty"`
	ResultFilters    []ResultFilterExecutionTrace `json:"resultFilters,omitempty"`
	Stage            string                       `json:"stage"`
	Outcome          string                       `json:"outcome"`
}

// ResultFilterExecutionTrace is bounded execution metadata only. It never
// retains rows, actor data, filter values, cache identities, or error strings.
type ResultFilterExecutionTrace struct {
	ID               string        `json:"id"`
	ExtensionID      string        `json:"extensionId"`
	ExtensionVersion string        `json:"extensionVersion"`
	ArtifactDigest   string        `json:"artifactDigest"`
	Priority         int           `json:"priority"`
	FailurePolicy    string        `json:"failurePolicy"`
	Duration         time.Duration `json:"duration"`
	Outcome          string        `json:"outcome"`
}

type ExecutionTraceSink interface {
	AppendExecutionTrace(ExecutionTrace)
}

type ExecutionTraceReader interface {
	ExecutionTraces(int) []ExecutionTrace
}

type ExecutionTraceRing struct {
	mu       sync.Mutex
	capacity int
	records  []ExecutionTrace
	next     int
	full     bool
}

func NewExecutionTraceRing(capacity int) *ExecutionTraceRing {
	if capacity <= 0 {
		capacity = defaultExecutionTraceCapacity
	}
	if capacity > maximumExecutionTraceCapacity {
		capacity = maximumExecutionTraceCapacity
	}
	return &ExecutionTraceRing{capacity: capacity, records: make([]ExecutionTrace, capacity)}
}

func (r *ExecutionTraceRing) AppendExecutionTrace(trace ExecutionTrace) {
	if r == nil || r.capacity <= 0 {
		return
	}
	trace = boundExecutionTrace(trace)
	r.mu.Lock()
	r.records[r.next] = trace
	r.next = (r.next + 1) % r.capacity
	if r.next == 0 {
		r.full = true
	}
	r.mu.Unlock()
}

func (r *ExecutionTraceRing) ExecutionTraces(limit int) []ExecutionTrace {
	if r == nil {
		return []ExecutionTrace{}
	}
	if limit <= 0 || limit > maximumExecutionTraceRead {
		limit = maximumExecutionTraceRead
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	count := r.next
	if r.full {
		count = r.capacity
	}
	if limit < count {
		count = limit
	}
	result := make([]ExecutionTrace, 0, count)
	for index := 0; index < count; index++ {
		position := r.next - 1 - index
		if position < 0 {
			position += r.capacity
		}
		result = append(result, cloneExecutionTrace(r.records[position]))
	}
	return result
}

func boundExecutionTrace(trace ExecutionTrace) ExecutionTrace {
	if trace.RecordedAt.IsZero() {
		trace.RecordedAt = time.Now().UTC()
	} else {
		trace.RecordedAt = trace.RecordedAt.UTC()
	}
	if trace.Duration < 0 {
		trace.Duration = 0
	}
	trace.QueryID = boundedExecutionTraceQueryID(trace.QueryID)
	trace.ContractVersion = boundedTraceString(trace.ContractVersion, maxSchemaRefLength)
	trace.PlanVersion = boundedTraceString(trace.PlanVersion, maxSchemaRefLength)
	trace.ResultSchema = boundedTraceString(trace.ResultSchema, maxSchemaRefLength)
	trace.ExtensionID = boundedTraceString(trace.ExtensionID, maxIDLength)
	trace.ExtensionVersion = boundedTraceString(trace.ExtensionVersion, maxExtensionVersionLength)
	trace.ArtifactDigest = boundedTraceString(trace.ArtifactDigest, 64)
	trace.SnapshotDigest = boundedTraceString(trace.SnapshotDigest, 64)
	trace.ShapeDigest = boundedTraceString(trace.ShapeDigest, 64)
	trace.ProviderDigest = boundedTraceString(trace.ProviderDigest, 64)
	trace.FilterPlan = boundedTraceString(trace.FilterPlan, 64)
	trace.CacheStatus = boundedTraceString(trace.CacheStatus, 32)
	trace.Stage = boundedTraceString(trace.Stage, 32)
	trace.Outcome = boundedTraceString(trace.Outcome, 32)
	if len(trace.ResultFilters) > maximumResultFilters {
		trace.ResultFilters = trace.ResultFilters[:maximumResultFilters]
	}
	trace.ResultFilters = append([]ResultFilterExecutionTrace(nil), trace.ResultFilters...)
	for index := range trace.ResultFilters {
		item := &trace.ResultFilters[index]
		item.ID = boundedTraceString(item.ID, maxIDLength)
		item.ExtensionID = boundedTraceString(item.ExtensionID, maxIDLength)
		item.ExtensionVersion = boundedTraceString(item.ExtensionVersion, maxExtensionVersionLength)
		item.ArtifactDigest = boundedTraceString(item.ArtifactDigest, 64)
		item.FailurePolicy = boundedTraceString(item.FailurePolicy, 32)
		item.Outcome = boundedTraceString(item.Outcome, 32)
		if item.Duration < 0 {
			item.Duration = 0
		}
	}
	sort.SliceStable(trace.ResultFilters, func(i, j int) bool {
		left, right := trace.ResultFilters[i], trace.ResultFilters[j]
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if left.ExtensionID != right.ExtensionID {
			return left.ExtensionID < right.ExtensionID
		}
		if left.ExtensionVersion != right.ExtensionVersion {
			return left.ExtensionVersion < right.ExtensionVersion
		}
		if left.ArtifactDigest != right.ArtifactDigest {
			return left.ArtifactDigest < right.ArtifactDigest
		}
		return left.ID < right.ID
	})
	if trace.Rows < 0 {
		trace.Rows = 0
	} else if trace.Rows > maximumPageLimit {
		trace.Rows = maximumPageLimit
	}
	if trace.FilterCount < 0 {
		trace.FilterCount = 0
	} else if trace.FilterCount > maximumResultFilters {
		trace.FilterCount = maximumResultFilters
	}
	return trace
}

func cloneExecutionTrace(trace ExecutionTrace) ExecutionTrace {
	trace.ResultFilters = append([]ResultFilterExecutionTrace(nil), trace.ResultFilters...)
	return trace
}

func resultFilterExecutionTrace(
	registration ResultFilterRegistration,
	outcome string,
	duration time.Duration,
) ResultFilterExecutionTrace {
	return ResultFilterExecutionTrace{
		ID: registration.ID, ExtensionID: registration.Artifact.ExtensionID,
		ExtensionVersion: registration.Artifact.ExtensionVersion,
		ArtifactDigest:   registration.Artifact.PackageDigest, Priority: registration.Priority,
		FailurePolicy: registration.FailurePolicy, Duration: duration, Outcome: outcome,
	}
}

func boundedTraceString(value string, limit int) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return -1
		}
		return character
	}, value)
	if len(value) > limit {
		return value[:limit]
	}
	return value
}

func unplannedExecutionTraceQueryID(value string) string {
	digest := sha256.Sum256([]byte(value))
	return unplannedExecutionTracePrefix + hex.EncodeToString(digest[:16])
}

func boundedExecutionTraceQueryID(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if idPattern.MatchString(normalized) {
		return normalized
	}
	if strings.HasPrefix(normalized, unplannedExecutionTracePrefix) &&
		len(normalized) == len(unplannedExecutionTracePrefix)+32 {
		if _, err := hex.DecodeString(strings.TrimPrefix(normalized, unplannedExecutionTracePrefix)); err == nil {
			return normalized
		}
	}
	return unplannedExecutionTraceQueryID(value)
}

func knownExecutionTraceQueryID(registry *Registry, value string) string {
	queryID := strings.ToLower(strings.TrimSpace(value))
	if registry != nil && idPattern.MatchString(queryID) {
		if contribution, exists := registry.load().queries[queryID]; exists {
			return contribution.ID
		}
	}
	return unplannedExecutionTraceQueryID(value)
}

type ResultFilterInspection struct {
	ID                   string   `json:"id"`
	ContractVersion      string   `json:"contractVersion"`
	Priority             int      `json:"priority"`
	FailurePolicy        string   `json:"failurePolicy"`
	TimeoutMS            int64    `json:"timeoutMs"`
	ExtensionID          string   `json:"extensionId"`
	ExtensionVersion     string   `json:"extensionVersion"`
	ArtifactDigest       string   `json:"artifactDigest"`
	DependencyExtension  string   `json:"dependencyExtension,omitempty"`
	DependencyConstraint string   `json:"dependencyConstraint,omitempty"`
	IdentityFields       []string `json:"identityFields,omitempty"`
	Admitted             bool     `json:"admitted"`
	Status               string   `json:"status"`
}

type ExecutionInspection struct {
	SchemaVersion    string                   `json:"schemaVersion"`
	Revision         uint64                   `json:"revision"`
	Digest           string                   `json:"digest"`
	SafeMode         bool                     `json:"safeMode"`
	Query            QueryContribution        `json:"query"`
	Provider         ProviderRef              `json:"provider"`
	ProviderBound    bool                     `json:"providerBound"`
	ProviderResolved bool                     `json:"providerResolved"`
	ProviderDigest   string                   `json:"providerDigest,omitempty"`
	AdmissionBound   bool                     `json:"admissionBound"`
	SchemaBound      bool                     `json:"schemaBound"`
	CacheBound       bool                     `json:"cacheBound"`
	CursorBound      bool                     `json:"cursorBound"`
	TimeoutMS        int64                    `json:"timeoutMs"`
	MaxResultBytes   int                      `json:"maxResultBytes"`
	ResultFilters    []ResultFilterInspection `json:"resultFilters"`
	FilterPlan       string                   `json:"filterPlan"`
	Traces           []ExecutionTrace         `json:"traces"`
}

// Inspect returns a detached execution snapshot without calling permission,
// provider, schema, or cache backends.
func (r *ExecutionRuntime) Inspect(queryID string, traces ExecutionTraceReader) (ExecutionInspection, error) {
	if r == nil || r.registry == nil {
		return ExecutionInspection{}, ErrExecutionInvalid
	}
	state := r.registry.load()
	query, ok := state.queries[strings.ToLower(strings.TrimSpace(queryID))]
	if !ok {
		return ExecutionInspection{}, ErrNotFound
	}
	query = cloneContribution(query)
	filters, matchEvidence, err := r.matchingFiltersWithEvidence(query)
	if err != nil {
		return ExecutionInspection{}, err
	}
	statusByID := make(map[string]string, len(r.filters))
	for _, filter := range filters {
		statusByID[filter.registration.ID] = "selected"
	}
	for _, evidence := range matchEvidence {
		statusByID[evidence.ID] = evidence.Outcome
	}
	result := ExecutionInspection{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode,
		Query: query, Provider: ProviderRef{Kind: ProviderKindQuery, Name: query.ID, ContributionID: query.ID, Artifact: query.Artifact},
		ProviderBound: r.providers != nil, AdmissionBound: r.admission != nil,
		SchemaBound: r.schemas != nil, CacheBound: r.cache != nil,
		CursorBound: r.registry.cursorCodec != nil, TimeoutMS: r.timeout.Milliseconds(), MaxResultBytes: r.maxResultBytes,
		ResultFilters: make([]ResultFilterInspection, 0, len(statusByID)), FilterPlan: resultFilterPlanDigest(filters),
	}
	if inspector, ok := r.providers.(executableProviderInspector); ok {
		if binding, resolved := inspector.inspectQueryProvider(query); resolved {
			result.ProviderResolved = true
			result.ProviderDigest = binding.ProviderDigest
		}
	}
	for _, filter := range r.filters {
		registration := filter.registration
		status, relevant := statusByID[registration.ID]
		if !relevant || registration.QueryID != query.ID {
			continue
		}
		result.ResultFilters = append(result.ResultFilters, ResultFilterInspection{
			ID: registration.ID, ContractVersion: registration.ContractVersion, Priority: registration.Priority,
			FailurePolicy: registration.FailurePolicy, TimeoutMS: registration.Timeout.Milliseconds(),
			ExtensionID: registration.Artifact.ExtensionID, ExtensionVersion: registration.Artifact.ExtensionVersion,
			ArtifactDigest:       registration.Artifact.PackageDigest,
			DependencyExtension:  registration.Dependency.ExtensionID,
			DependencyConstraint: registration.Dependency.VersionConstraint,
			IdentityFields:       append([]string(nil), registration.IdentityFields...),
			Admitted:             r.registry.artifactAdmitted(registration.Artifact),
			Status:               status,
		})
	}
	if traces != nil {
		for _, trace := range traces.ExecutionTraces(maximumExecutionTraceRead) {
			if trace.QueryID == query.ID {
				result.Traces = append(result.Traces, cloneExecutionTrace(trace))
			}
		}
	}
	if result.Traces == nil {
		result.Traces = []ExecutionTrace{}
	}
	sort.SliceStable(result.Traces, func(i, j int) bool {
		return result.Traces[i].RecordedAt.After(result.Traces[j].RecordedAt)
	})
	return result, nil
}
