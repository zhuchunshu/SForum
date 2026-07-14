package routes

import (
	"sync"
	"time"
)

const (
	defaultRouteTraceCapacity = 256
	maxRouteTraceCapacity     = 4096
)

type RouteTraceOutcome string

const (
	RouteTraceSucceeded       RouteTraceOutcome = "succeeded"
	RouteTraceDenied          RouteTraceOutcome = "denied"
	RouteTraceSchemaRejected  RouteTraceOutcome = "schema_rejected"
	RouteTraceTransportFailed RouteTraceOutcome = "transport_failed"
	RouteTraceFallbackUsed    RouteTraceOutcome = "fallback_used"
	RouteTraceCommitted       RouteTraceOutcome = "committed"
)

type RouteTraceEvent struct {
	Revision        uint64
	StepIndex       int
	Phase           RouteExecutionPhase
	Action          string
	RouteID         string
	ContractVersion string
	Method          string
	PathSignature   string
	Mode            string
	Fallback        string
	Outcome         RouteTraceOutcome
	Duration        time.Duration
	CommitState     RouteExecutionCommitState
	Provider        Provider
}

type RouteTraceRecord struct {
	Sequence        uint64                    `json:"sequence"`
	ObservedAt      time.Time                 `json:"observedAt"`
	Revision        uint64                    `json:"revision"`
	StepIndex       int                       `json:"stepIndex"`
	Phase           RouteExecutionPhase       `json:"phase"`
	Action          string                    `json:"action"`
	RouteID         string                    `json:"routeId"`
	ContractVersion string                    `json:"contractVersion"`
	Method          string                    `json:"method"`
	PathSignature   string                    `json:"pathSignature"`
	Mode            string                    `json:"mode"`
	Fallback        string                    `json:"fallback"`
	Outcome         RouteTraceOutcome         `json:"outcome"`
	DurationMicros  int64                     `json:"durationMicros"`
	CommitState     RouteExecutionCommitState `json:"commitState"`
	Provider        InspectorProvider         `json:"provider"`
}

// RouteTraceSink deliberately accepts no request, response, actor, header,
// query, body, secret, or raw error fields.
type RouteTraceSink interface {
	AppendRouteTrace(RouteTraceEvent)
}

type RouteTraceReader interface {
	RouteTraces(limit int) []RouteTraceRecord
}

type RouteTraceRing struct {
	mu       sync.RWMutex
	capacity int
	next     uint64
	start    int
	size     int
	records  []RouteTraceRecord
}

func NewRouteTraceRing(capacity int) *RouteTraceRing {
	if capacity <= 0 {
		capacity = defaultRouteTraceCapacity
	}
	if capacity > maxRouteTraceCapacity {
		capacity = maxRouteTraceCapacity
	}
	return &RouteTraceRing{capacity: capacity, records: make([]RouteTraceRecord, capacity)}
}

func (r *RouteTraceRing) AppendRouteTrace(event RouteTraceEvent) {
	if r == nil || !validRouteTraceEvent(event) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	record := RouteTraceRecord{
		Sequence: r.next, ObservedAt: time.Now().UTC(), Revision: event.Revision,
		StepIndex: event.StepIndex, Phase: event.Phase, Action: event.Action,
		RouteID: event.RouteID, ContractVersion: event.ContractVersion,
		Method: event.Method, PathSignature: event.PathSignature, Mode: event.Mode,
		Fallback: event.Fallback, Outcome: event.Outcome, DurationMicros: event.Duration.Microseconds(),
		CommitState: event.CommitState, Provider: inspectorProvider(event.Provider),
	}
	if r.size < r.capacity {
		index := (r.start + r.size) % r.capacity
		r.records[index] = record
		r.size++
		return
	}
	r.records[r.start] = record
	r.start = (r.start + 1) % r.capacity
}

func (r *RouteTraceRing) RouteTraces(limit int) []RouteTraceRecord {
	if r == nil {
		return []RouteTraceRecord{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > r.size {
		limit = r.size
	}
	result := make([]RouteTraceRecord, limit)
	logicalStart := r.size - limit
	for index := range result {
		result[index] = r.records[(r.start+logicalStart+index)%r.capacity]
	}
	for index := range result {
		if result[index].Provider.Artifact != nil {
			artifact := *result[index].Provider.Artifact
			result[index].Provider.Artifact = &artifact
		}
	}
	return result
}

func validRouteTraceEvent(event RouteTraceEvent) bool {
	if event.Revision == 0 || event.StepIndex < 0 || !routeIDPattern.MatchString(event.RouteID) ||
		!contractPattern.MatchString(event.ContractVersion) || !validMethod(event.Method) ||
		event.Method == "*" || event.Duration < 0 || event.Duration > 24*time.Hour ||
		!validTraceOutcome(event.Outcome) || !validTracePhase(event.Phase) ||
		!validTraceCommitState(event.CommitState) {
		return false
	}
	if event.Provider.Kind == ProviderCore {
		return event.Provider.Artifact == (PluginArtifact{})
	}
	return event.Provider.Kind == ProviderPlugin && validatePluginArtifact(event.Provider.Artifact) == nil
}

func validTraceOutcome(outcome RouteTraceOutcome) bool {
	switch outcome {
	case RouteTraceSucceeded, RouteTraceDenied, RouteTraceSchemaRejected,
		RouteTraceTransportFailed, RouteTraceFallbackUsed, RouteTraceCommitted:
		return true
	default:
		return false
	}
}

func validTracePhase(phase RouteExecutionPhase) bool {
	switch phase {
	case RoutePhaseGlobal, RoutePhaseBefore, RoutePhaseFilter, RoutePhaseWrap, RoutePhaseHandler, RoutePhaseAfter:
		return true
	default:
		return false
	}
}

func validTraceCommitState(state RouteExecutionCommitState) bool {
	switch state {
	case RouteCommitPristine, RouteCommitResponseStarted, RouteCommitSideEffectStarted, RouteCommitFinal:
		return true
	default:
		return false
	}
}

var _ RouteTraceSink = (*RouteTraceRing)(nil)
var _ RouteTraceReader = (*RouteTraceRing)(nil)
