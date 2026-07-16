package contentregistry

import (
	"sync"
	"sync/atomic"
	"time"
)

const contentTraceQueueCapacity = 256

type TraceOutcome string

const (
	TraceSucceeded      TraceOutcome = "succeeded"
	TraceDenied         TraceOutcome = "denied"
	TraceStale          TraceOutcome = "stale"
	TraceSchemaRejected TraceOutcome = "schema_rejected"
	TraceTimedOut       TraceOutcome = "timed_out"
	TracePanicked       TraceOutcome = "panicked"
	TraceFailed         TraceOutcome = "failed"
	TraceFallback       TraceOutcome = "fallback"
)

// ContentTraceEvent excludes source, rendered output, actor identity, and raw
// errors. Inspector attribution stays useful without becoming a content leak.
type ContentTraceEvent struct {
	Revision        uint64
	TargetID        string
	ContentID       string
	ContractVersion string
	Action          string
	Operation       string
	Artifact        Artifact
	Outcome         TraceOutcome
	Fallback        string
	Duration        time.Duration
}

type ContentTraceRecord struct {
	Sequence        uint64       `json:"sequence"`
	ObservedAt      time.Time    `json:"observedAt"`
	Revision        uint64       `json:"revision"`
	TargetID        string       `json:"targetId"`
	ContentID       string       `json:"contentId"`
	ContractVersion string       `json:"contractVersion"`
	Action          string       `json:"action"`
	Operation       string       `json:"operation"`
	Artifact        Artifact     `json:"artifact"`
	Outcome         TraceOutcome `json:"outcome"`
	Fallback        string       `json:"fallback,omitempty"`
	DurationMicros  int64        `json:"durationMicros"`
}

type ContentTraceSink interface {
	AppendContentTrace(ContentTraceEvent)
}

// appendContentTraceSafely keeps optional diagnostics from changing content
// authorization or release behavior. Production sinks must still be bounded
// and non-blocking; this boundary contains sink panics without exposing them.
func appendContentTraceSafely(sink ContentTraceSink, event ContentTraceEvent) {
	if sink == nil {
		return
	}
	defer func() { _ = recover() }()
	sink.AppendContentTrace(event)
}

// contentTraceDispatcher contains arbitrary Host diagnostics behind one fixed
// worker and a bounded queue. Request paths never wait for custom sinks and a
// stalled sink cannot cause per-event goroutine growth.
type contentTraceDispatcher struct {
	sink      ContentTraceSink
	events    chan ContentTraceEvent
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	dropped   atomic.Uint64
}

func newContentTraceDispatcher(sink ContentTraceSink) *contentTraceDispatcher {
	dispatcher := &contentTraceDispatcher{
		sink: sink, events: make(chan ContentTraceEvent, contentTraceQueueCapacity),
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	go dispatcher.run()
	return dispatcher
}

func (d *contentTraceDispatcher) run() {
	defer close(d.done)
	for {
		// Prioritize shutdown after a stalled sink returns instead of draining a
		// queue that the caller has explicitly abandoned.
		select {
		case <-d.stop:
			return
		default:
		}
		select {
		case <-d.stop:
			return
		case event := <-d.events:
			appendContentTraceSafely(d.sink, event)
		}
	}
}

func (d *contentTraceDispatcher) append(event ContentTraceEvent) {
	if d == nil {
		return
	}
	select {
	case <-d.stop:
		return
	default:
	}
	select {
	case <-d.stop:
	case d.events <- event:
	default:
		d.dropped.Add(1)
	}
}

func (d *contentTraceDispatcher) Close() {
	if d == nil {
		return
	}
	d.closeOnce.Do(func() { close(d.stop) })
	<-d.done
}

func (e *Executor) appendTrace(event ContentTraceEvent) {
	if e == nil || e.trace == nil {
		return
	}
	if ring, ok := e.trace.(*ContentTraceRing); ok {
		appendContentTraceSafely(ring, event)
		return
	}
	e.traceDispatch.append(event)
}

type ContentTraceReader interface {
	ContentTraces(limit int) []ContentTraceRecord
}

type ContentTargetTraceReader interface {
	ContentTracesForTarget(targetID string, limit int) []ContentTraceRecord
}

type ContentTraceRing struct {
	mu       sync.RWMutex
	capacity int
	next     uint64
	start    int
	size     int
	records  []ContentTraceRecord
}

func NewContentTraceRing(capacity int) *ContentTraceRing {
	if capacity <= 0 {
		capacity = 256
	}
	if capacity > 4096 {
		capacity = 4096
	}
	return &ContentTraceRing{capacity: capacity, records: make([]ContentTraceRecord, capacity)}
}

func (r *ContentTraceRing) AppendContentTrace(event ContentTraceEvent) {
	if r == nil || !validContentTraceEvent(event) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	record := ContentTraceRecord{
		Sequence: r.next, ObservedAt: time.Now().UTC(), Revision: event.Revision,
		TargetID: event.TargetID, ContentID: event.ContentID, ContractVersion: event.ContractVersion,
		Action: event.Action, Operation: event.Operation, Artifact: event.Artifact,
		Outcome: event.Outcome, Fallback: event.Fallback, DurationMicros: event.Duration.Microseconds(),
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

func (r *ContentTraceRing) ContentTraces(limit int) []ContentTraceRecord {
	if r == nil {
		return []ContentTraceRecord{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > r.size {
		limit = r.size
	}
	result := make([]ContentTraceRecord, limit)
	logicalStart := r.size - limit
	for index := range result {
		result[index] = r.records[(r.start+logicalStart+index)%r.capacity]
	}
	return result
}

func (r *ContentTraceRing) ContentTracesForTarget(targetID string, limit int) []ContentTraceRecord {
	if r == nil || !idPattern.MatchString(targetID) || limit <= 0 {
		return []ContentTraceRecord{}
	}
	if limit > r.capacity {
		limit = r.capacity
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContentTraceRecord, 0, limit)
	for logical := r.size - 1; logical >= 0 && len(result) < limit; logical-- {
		record := r.records[(r.start+logical)%r.capacity]
		if record.TargetID == targetID {
			result = append(result, record)
		}
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result
}

func validContentTraceEvent(event ContentTraceEvent) bool {
	if event.Revision == 0 || !idPattern.MatchString(event.TargetID) || !idPattern.MatchString(event.ContentID) ||
		!contractPattern.MatchString(event.ContractVersion) || !validExecutionAction(event.Action) ||
		!validExecutionOperation(event.Operation) || !IsExactArtifact(event.Artifact) ||
		event.Duration < 0 || event.Duration > 24*time.Hour {
		return false
	}
	if event.Outcome == TraceFallback {
		if !validExecutionFallback(event.Action, event.Fallback) {
			return false
		}
	} else if event.Fallback != "" {
		return false
	}
	switch event.Outcome {
	case TraceSucceeded, TraceDenied, TraceStale, TraceSchemaRejected, TraceTimedOut,
		TracePanicked, TraceFailed, TraceFallback:
		return true
	default:
		return false
	}
}

func validExecutionOperation(operation string) bool {
	switch operation {
	case OperationEditor, OperationValidator, OperationSerializer, OperationRenderer,
		OperationFilter, OperationHide, OperationSource, OperationRelease:
		return true
	default:
		return false
	}
}

var _ ContentTraceSink = (*ContentTraceRing)(nil)
var _ ContentTraceReader = (*ContentTraceRing)(nil)
var _ ContentTargetTraceReader = (*ContentTraceRing)(nil)
