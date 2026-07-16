package navigationregistry

import (
	"strings"
	"sync"
	"time"
)

const (
	defaultTraceCapacity = 256
	maxTraceCapacity     = 4096
)

const (
	TraceSucceeded    = "succeeded"
	TraceSkipped      = "skipped"
	TraceFallback     = "fallback"
	TraceDenied       = "denied"
	TraceFailedClosed = "failed_closed"
)

type TraceEvent struct {
	Revision        uint64
	Family          string
	TargetID        string
	ContributionID  string
	ContractVersion string
	Action          string
	Handler         string
	Locale          string
	Outcome         string
	FallbackReason  string
	Duration        time.Duration
	Artifact        Artifact
}

type TraceRecord struct {
	Sequence        uint64    `json:"sequence"`
	ObservedAt      time.Time `json:"observedAt"`
	Revision        uint64    `json:"revision"`
	Family          string    `json:"family"`
	TargetID        string    `json:"targetId"`
	ContributionID  string    `json:"contributionId"`
	ContractVersion string    `json:"contractVersion"`
	Action          string    `json:"action"`
	Handler         string    `json:"handler,omitempty"`
	Locale          string    `json:"locale"`
	Outcome         string    `json:"outcome"`
	FallbackReason  string    `json:"fallbackReason,omitempty"`
	DurationMicros  int64     `json:"durationMicros"`
	Artifact        Artifact  `json:"artifact"`
}

type TraceSink interface {
	AppendNavigationTrace(TraceEvent)
}

type TraceReader interface {
	NavigationTraces(limit int) []TraceRecord
}

type TraceRing struct {
	mu       sync.RWMutex
	capacity int
	next     uint64
	start    int
	size     int
	records  []TraceRecord
}

func NewTraceRing(capacity int) *TraceRing {
	if capacity <= 0 {
		capacity = defaultTraceCapacity
	}
	if capacity > maxTraceCapacity {
		capacity = maxTraceCapacity
	}
	return &TraceRing{capacity: capacity, records: make([]TraceRecord, capacity)}
}

func (r *TraceRing) AppendNavigationTrace(event TraceEvent) {
	if r == nil || !validTraceEvent(event) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	record := TraceRecord{
		Sequence: r.next, ObservedAt: time.Now().UTC(), Revision: event.Revision,
		Family: event.Family, TargetID: event.TargetID, ContributionID: event.ContributionID,
		ContractVersion: event.ContractVersion, Action: event.Action, Handler: event.Handler,
		Locale: event.Locale, Outcome: event.Outcome, FallbackReason: event.FallbackReason,
		DurationMicros: event.Duration.Microseconds(), Artifact: event.Artifact,
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

func (r *TraceRing) NavigationTraces(limit int) []TraceRecord {
	if r == nil {
		return []TraceRecord{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > r.size {
		limit = r.size
	}
	result := make([]TraceRecord, limit)
	logicalStart := r.size - limit
	for index := range result {
		result[index] = r.records[(r.start+logicalStart+index)%r.capacity]
	}
	return result
}

func validTraceEvent(event TraceEvent) bool {
	if event.Revision == 0 || !validProviderFamily(event.Family) || !idPattern.MatchString(event.TargetID) ||
		!idPattern.MatchString(event.ContributionID) || !contractPattern.MatchString(event.ContractVersion) ||
		!validAction(event.Action) || event.Duration < 0 {
		return false
	}
	switch event.Outcome {
	case TraceSucceeded, TraceSkipped, TraceFallback, TraceDenied, TraceFailedClosed:
		return true
	default:
		return false
	}
}

type Inspection struct {
	Snapshot Snapshot      `json:"snapshot"`
	Traces   []TraceRecord `json:"traces"`
}

type Inspector struct {
	registry *Registry
	traces   TraceReader
}

func NewInspector(registry *Registry, traces TraceReader) *Inspector {
	return &Inspector{registry: registry, traces: traces}
}

// Inspect returns immutable graph/conflict/selection evidence plus bounded
// exact attribution. HTTP authorization and artifact redaction remain the
// responsibility of the future admin controller.
func (i *Inspector) Inspect(targetID string, limit int) (Inspection, error) {
	if i == nil || i.registry == nil {
		return Inspection{}, ErrInvalid
	}
	targetID = normalizeInspectionTarget(targetID)
	if targetID == "!invalid" {
		return Inspection{}, ErrInvalid
	}
	traces := []TraceRecord{}
	if i.traces != nil {
		traceLimit := limit
		if targetID != "" {
			traceLimit = 0
		}
		for _, record := range i.traces.NavigationTraces(traceLimit) {
			if targetID == "" || record.TargetID == targetID {
				traces = append(traces, record)
			}
		}
		if limit > 0 && len(traces) > limit {
			traces = traces[len(traces)-limit:]
		}
	}
	return Inspection{Snapshot: i.registry.Snapshot(), Traces: traces}, nil
}

func normalizeInspectionTarget(value string) string {
	value = normalizeID(value)
	if value != "" && !idPattern.MatchString(value) {
		return "!invalid"
	}
	return value
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

var _ TraceSink = (*TraceRing)(nil)
var _ TraceReader = (*TraceRing)(nil)
