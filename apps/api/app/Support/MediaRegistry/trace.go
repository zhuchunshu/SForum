package mediaregistry

import (
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	defaultTraceCapacity  = 256
	maxTraceCapacity      = 4096
	externalTraceCapacity = 64
	TraceSucceeded        = "succeeded"
	TraceRetry            = "retry"
	TraceFallback         = "fallback_original"
	TraceSkipped          = "skipped"
	TraceDenied           = "denied"
	TraceFailedClosed     = "failed_closed"
)

type TraceEvent struct {
	Revision     uint64
	OperationKey string
	PlanKind     string
	Stage        string
	StepID       string
	Outcome      string
	Reason       string
	Duration     time.Duration
	Artifact     Artifact
}

// TraceRecord deliberately excludes actor ids, permissions, filenames, MIME
// candidates, URLs, metadata, provider errors, and storage handles.
type TraceRecord struct {
	Sequence       uint64    `json:"sequence"`
	ObservedAt     time.Time `json:"observedAt"`
	Revision       uint64    `json:"revision"`
	OperationKey   string    `json:"operationKey"`
	PlanKind       string    `json:"planKind"`
	Stage          string    `json:"stage"`
	StepID         string    `json:"stepId"`
	Outcome        string    `json:"outcome"`
	Reason         string    `json:"reason,omitempty"`
	DurationMicros int64     `json:"durationMicros"`
	Artifact       Artifact  `json:"artifact"`
}

type TraceSink interface{ AppendMediaTrace(TraceEvent) }
type TraceReader interface{ MediaTraces(int) []TraceRecord }

// hostSynchronousTraceSink is package-sealed: only the bounded Host ring may
// execute inline after a media call. External sinks are isolated asynchronously.
type hostSynchronousTraceSink interface {
	TraceSink
	hostSynchronousMediaTrace()
}

type externalTraceCall struct {
	sink  TraceSink
	event TraceEvent
}

var (
	externalTraceOnce  sync.Once
	externalTraceQueue = make(chan externalTraceCall, externalTraceCapacity)
)

func enqueueExternalMediaTrace(sink TraceSink, event TraceEvent) {
	if sink == nil {
		return
	}
	externalTraceOnce.Do(func() {
		// This single process-owned worker deliberately has no Close lifecycle.
		// A non-cooperative sink can retain only this goroutine; the fixed queue
		// then drops observations instead of delaying execution or terminals.
		go func() {
			for call := range externalTraceQueue {
				appendMediaTraceSafely(call.sink, call.event)
			}
		}()
	})
	select {
	case externalTraceQueue <- externalTraceCall{sink: sink, event: event}:
	default:
	}
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

func (*TraceRing) hostSynchronousMediaTrace() {}

func (r *TraceRing) AppendMediaTrace(event TraceEvent) {
	if r == nil || !validTraceEvent(event) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	record := TraceRecord{Sequence: r.next, ObservedAt: time.Now().UTC(), Revision: event.Revision, OperationKey: event.OperationKey, PlanKind: event.PlanKind, Stage: event.Stage, StepID: event.StepID, Outcome: event.Outcome, Reason: event.Reason, DurationMicros: event.Duration.Microseconds(), Artifact: event.Artifact}
	if r.size < r.capacity {
		index := (r.start + r.size) % r.capacity
		r.records[index] = record
		r.size++
		return
	}
	r.records[r.start] = record
	r.start = (r.start + 1) % r.capacity
}

func (r *TraceRing) MediaTraces(limit int) []TraceRecord {
	if r == nil {
		return []TraceRecord{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > r.size {
		limit = r.size
	}
	result := make([]TraceRecord, limit)
	logical := r.size - limit
	for i := range result {
		result[i] = r.records[(r.start+logical+i)%r.capacity]
	}
	return result
}

func validTraceEvent(event TraceEvent) bool {
	if event.Revision == 0 || !digestPattern.MatchString(event.OperationKey) || !validPlanKind(event.PlanKind) || !validStage(event.Stage) || event.StepID == "" || !validPlainString(event.StepID, maxStringBytes) || event.Duration < 0 || !validTraceOutcome(event.Outcome) {
		return false
	}
	if event.Reason != "" && (!validPlainString(event.Reason, maxReasonCodeBytes) || strings.ContainsAny(event.Reason, " /")) {
		return false
	}
	if event.Artifact.Core {
		return validCoreArtifactSeal(event.Artifact)
	}
	_, err := normalizeArtifact(event.Artifact)
	return err == nil
}
func validTraceOutcome(value string) bool {
	switch value {
	case TraceSucceeded, TraceRetry, TraceFallback, TraceSkipped, TraceDenied, TraceFailedClosed:
		return true
	default:
		return false
	}
}

type Inspection struct {
	Snapshot Snapshot      `json:"snapshot"`
	Traces   []TraceRecord `json:"traces"`
}

func Inspect(registry *Registry, traces TraceReader, limit int) Inspection {
	result := Inspection{}
	if registry != nil {
		result.Snapshot = registry.Snapshot()
	}
	if traces != nil {
		result.Traces = slices.Clone(traces.MediaTraces(limit))
	} else {
		result.Traces = []TraceRecord{}
	}
	return result
}
