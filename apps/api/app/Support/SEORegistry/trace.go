package seoregistry

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	TraceOutcomeApplied        = "applied"
	TraceOutcomeInvalid        = "invalid"
	TraceOutcomeProviderFailed = "provider_failed"
	TraceOutcomeRuntimeStale   = "runtime_stale"
	TraceOutcomeSnapshotStale  = "snapshot_stale"
	TraceOutcomeDeadline       = "deadline"
	TraceOutcomeCancelled      = "cancelled"
	TraceOutcomePolicyDenied   = "policy_denied"
	TraceOutcomeOutputInvalid  = "output_invalid"
	TraceOutcomeOutputTooLarge = "output_too_large"
	TraceOutcomeInternal       = "internal"

	TraceCallApplied        = "applied"
	TraceCallUnavailable    = "provider_unavailable"
	TraceCallRuntimeStale   = "runtime_stale"
	TraceCallFailed         = "failed"
	TraceCallDeadline       = "deadline"
	TraceCallOutputInvalid  = "output_invalid"
	TraceCallMutationDenied = "mutation_denied"
	TraceCallSnapshotStale  = "snapshot_stale"

	defaultTraceCapacity = 256
	maximumTraceCapacity = 2048
	maximumTraceRead     = 200
)

// ExecutionTrace is bounded metadata only. SEO values, URLs, titles, actor
// data, error strings, and provider output are never retained.
type ExecutionTrace struct {
	RecordedAt     time.Time           `json:"recordedAt"`
	Duration       time.Duration       `json:"duration"`
	Scope          string              `json:"scope"`
	Revision       uint64              `json:"revision,omitempty"`
	SnapshotDigest string              `json:"snapshotDigest,omitempty"`
	Stage          string              `json:"stage"`
	Outcome        string              `json:"outcome"`
	Applied        int                 `json:"applied,omitempty"`
	Fallbacks      int                 `json:"fallbacks,omitempty"`
	Calls          []ProviderCallTrace `json:"calls,omitempty"`
}

type ProviderCallTrace struct {
	ID               string        `json:"id"`
	ExtensionID      string        `json:"extensionId"`
	ExtensionVersion string        `json:"extensionVersion"`
	ArtifactDigest   string        `json:"artifactDigest"`
	ProviderDigest   string        `json:"providerDigest,omitempty"`
	Kind             string        `json:"kind"`
	Action           string        `json:"action"`
	Priority         int           `json:"priority"`
	FailurePolicy    string        `json:"failurePolicy"`
	Duration         time.Duration `json:"duration"`
	Outcome          string        `json:"outcome"`
}

type ExecutionTraceSink interface {
	AppendSEOExecutionTrace(ExecutionTrace)
}

type ExecutionTraceReader interface {
	SEOExecutionTraces(int) []ExecutionTrace
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
		capacity = defaultTraceCapacity
	}
	if capacity > maximumTraceCapacity {
		capacity = maximumTraceCapacity
	}
	return &ExecutionTraceRing{capacity: capacity, records: make([]ExecutionTrace, capacity)}
}

func (r *ExecutionTraceRing) AppendSEOExecutionTrace(trace ExecutionTrace) {
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

func (r *ExecutionTraceRing) SEOExecutionTraces(limit int) []ExecutionTrace {
	if r == nil {
		return []ExecutionTrace{}
	}
	if limit <= 0 || limit > maximumTraceRead {
		limit = maximumTraceRead
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

func providerCallTrace(contribution Contribution) ProviderCallTrace {
	return ProviderCallTrace{
		ID: contribution.ID, ExtensionID: contribution.Artifact.ExtensionID,
		ExtensionVersion: contribution.Artifact.ExtensionVersion, ArtifactDigest: contribution.Artifact.PackageDigest,
		Kind: contribution.Kind, Action: contribution.Action, Priority: contribution.Priority,
		FailurePolicy: contribution.FailurePolicy,
	}
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
	trace.Scope = boundedTraceString(trace.Scope, 121)
	trace.SnapshotDigest = boundedTraceString(trace.SnapshotDigest, 64)
	trace.Stage = boundedTraceString(trace.Stage, 32)
	trace.Outcome = boundedTraceString(trace.Outcome, 32)
	if len(trace.Calls) > maxContributions {
		trace.Calls = trace.Calls[:maxContributions]
	}
	trace.Calls = append([]ProviderCallTrace(nil), trace.Calls...)
	for index := range trace.Calls {
		call := &trace.Calls[index]
		call.ID = boundedTraceString(call.ID, 121)
		call.ExtensionID = boundedTraceString(call.ExtensionID, 121)
		call.ExtensionVersion = boundedTraceString(call.ExtensionVersion, 128)
		call.ArtifactDigest = boundedTraceString(call.ArtifactDigest, 64)
		call.ProviderDigest = boundedTraceString(call.ProviderDigest, 64)
		call.Kind = boundedTraceString(call.Kind, 16)
		call.Action = boundedTraceString(call.Action, 16)
		call.FailurePolicy = boundedTraceString(call.FailurePolicy, 16)
		call.Outcome = boundedTraceString(call.Outcome, 32)
		if call.Duration < 0 {
			call.Duration = 0
		}
	}
	return trace
}

func cloneExecutionTrace(trace ExecutionTrace) ExecutionTrace {
	trace.Calls = append([]ProviderCallTrace(nil), trace.Calls...)
	return trace
}

func boundedTraceString(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}

func traceOutcome(err error) string {
	switch {
	case err == nil:
		return TraceOutcomeApplied
	case errors.Is(err, context.DeadlineExceeded):
		return TraceOutcomeDeadline
	case errors.Is(err, context.Canceled):
		return TraceOutcomeCancelled
	case errors.Is(err, ErrSnapshotStale):
		return TraceOutcomeSnapshotStale
	case errors.Is(err, ErrArtifactUnavailable):
		return TraceOutcomeRuntimeStale
	case errors.Is(err, ErrProviderFailed), errors.Is(err, ErrProviderUnavailable):
		return TraceOutcomeProviderFailed
	case errors.Is(err, ErrOutputTooLarge):
		return TraceOutcomeOutputTooLarge
	case errors.Is(err, ErrPolicyDenied):
		return TraceOutcomePolicyDenied
	case errors.Is(err, ErrOutputInvalid), errors.Is(err, ErrMutationDenied):
		return TraceOutcomeOutputInvalid
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrExecutionInvalid):
		return TraceOutcomeInvalid
	default:
		return TraceOutcomeInternal
	}
}
