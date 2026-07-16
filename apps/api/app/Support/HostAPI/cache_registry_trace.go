package hostapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	HostCacheDefaultSlowThreshold = 250 * time.Millisecond
	hostCacheTraceIdentityLimit   = 255
	hostCacheTraceVersionLimit    = 128
	hostCacheTraceDigestLimit     = 128
	hostCacheInspectorDefaultSize = 512
	hostCacheInspectorMaximumSize = 4096
)

type HostCacheTraceOutcome string

const (
	HostCacheTraceHit      HostCacheTraceOutcome = "hit"
	HostCacheTraceMiss     HostCacheTraceOutcome = "miss"
	HostCacheTraceAllowed  HostCacheTraceOutcome = "allowed"
	HostCacheTraceDenied   HostCacheTraceOutcome = "denied"
	HostCacheTraceStale    HostCacheTraceOutcome = "stale"
	HostCacheTraceConflict HostCacheTraceOutcome = "conflict"
	HostCacheTraceError    HostCacheTraceOutcome = "error"
	HostCacheTraceCancel   HostCacheTraceOutcome = "cancel"
	HostCacheTraceDeadline HostCacheTraceOutcome = "deadline"
)

// HostCacheTrace contains only bounded Host-owned attribution. User keys,
// values, tag names, lock tokens, request payloads, and error strings are never
// recorded.
type HostCacheTrace struct {
	Sequence          uint64
	ExtensionID       string
	ExtensionVersion  string
	ArtifactDigest    string
	RuntimeInstanceID string
	VersionID         int64
	CacheID           string
	ContractVersion   string
	RegistryRevision  uint64
	ProviderRevision  uint64
	ProviderID        string
	ProviderExtension string
	ProviderArtifact  string
	ProviderRuntime   string
	ProviderVersionID int64
	Operation         string
	TagDigest         string
	TagCount          int
	InvalidatorID     string
	Outcome           HostCacheTraceOutcome
	Duration          time.Duration
	Attempts          int
	Hit               bool
	Affected          uint64
	Slow              bool
}

type HostCacheTraceSink interface {
	RecordHostCacheTrace(HostCacheTrace)
}

type multiHostCacheTraceSink []HostCacheTraceSink

func NewMultiHostCacheTraceSink(sinks ...HostCacheTraceSink) HostCacheTraceSink {
	filtered := make(multiHostCacheTraceSink, 0, len(sinks))
	for _, sink := range sinks {
		if sink != nil {
			filtered = append(filtered, sink)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func (s multiHostCacheTraceSink) RecordHostCacheTrace(trace HostCacheTrace) {
	for _, sink := range s {
		sink.RecordHostCacheTrace(trace)
	}
}

type slogHostCacheTraceSink struct {
	logger *slog.Logger
}

func NewSlogHostCacheTraceSink(logger *slog.Logger) HostCacheTraceSink {
	if logger == nil {
		logger = slog.Default()
	}
	return &slogHostCacheTraceSink{logger: logger}
}

func (s *slogHostCacheTraceSink) RecordHostCacheTrace(trace HostCacheTrace) {
	if s == nil || s.logger == nil {
		return
	}
	trace = boundedHostCacheTrace(trace)
	level := slog.LevelInfo
	if trace.Slow || trace.Outcome == HostCacheTraceDenied || trace.Outcome == HostCacheTraceStale ||
		trace.Outcome == HostCacheTraceConflict || trace.Outcome == HostCacheTraceError {
		level = slog.LevelWarn
	}
	s.logger.LogAttrs(context.Background(), level, "Host Cache trace",
		slog.String("extension_id", trace.ExtensionID),
		slog.String("extension_version", trace.ExtensionVersion),
		slog.String("artifact_digest", trace.ArtifactDigest),
		slog.String("runtime_instance_id", trace.RuntimeInstanceID),
		slog.Int64("version_id", trace.VersionID),
		slog.String("cache_id", trace.CacheID),
		slog.String("contract_version", trace.ContractVersion),
		slog.Uint64("registry_revision", trace.RegistryRevision),
		slog.Uint64("provider_revision", trace.ProviderRevision),
		slog.String("provider_id", trace.ProviderID),
		slog.String("provider_extension", trace.ProviderExtension),
		slog.String("provider_artifact", trace.ProviderArtifact),
		slog.String("provider_runtime", trace.ProviderRuntime),
		slog.Int64("provider_version_id", trace.ProviderVersionID),
		slog.String("operation", trace.Operation),
		slog.String("tag_digest", trace.TagDigest),
		slog.Int("tag_count", trace.TagCount),
		slog.String("invalidator_id", trace.InvalidatorID),
		slog.String("outcome", string(trace.Outcome)),
		slog.Duration("duration", trace.Duration),
		slog.Int("attempts", trace.Attempts),
		slog.Bool("hit", trace.Hit),
		slog.Uint64("affected", trace.Affected),
		slog.Bool("slow", trace.Slow),
	)
}

type HostCacheInspector struct {
	mu       sync.RWMutex
	sequence uint64
	limit    int
	entries  []HostCacheTrace
}

func NewHostCacheInspector(limit int) *HostCacheInspector {
	if limit <= 0 {
		limit = hostCacheInspectorDefaultSize
	}
	if limit > hostCacheInspectorMaximumSize {
		limit = hostCacheInspectorMaximumSize
	}
	return &HostCacheInspector{limit: limit, entries: make([]HostCacheTrace, 0, limit)}
}

func (i *HostCacheInspector) RecordHostCacheTrace(trace HostCacheTrace) {
	if i == nil {
		return
	}
	trace = boundedHostCacheTrace(trace)
	i.mu.Lock()
	i.sequence++
	trace.Sequence = i.sequence
	if len(i.entries) == i.limit {
		copy(i.entries, i.entries[1:])
		i.entries[len(i.entries)-1] = trace
	} else {
		i.entries = append(i.entries, trace)
	}
	i.mu.Unlock()
}

func (i *HostCacheInspector) Snapshot() []HostCacheTrace {
	if i == nil {
		return []HostCacheTrace{}
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return slices.Clone(i.entries)
}

func (s *HostCacheService) recordHostCacheTrace(
	started time.Time,
	prepared preparedHostCache,
	candidate HostCacheProviderCandidate,
	operation string,
	err error,
	hit bool,
	affected uint64,
	attempts int,
) {
	if s == nil || s.traceSink == nil {
		return
	}
	trace := HostCacheTrace{
		ExtensionID:       prepared.plan.Cache.Artifact.ExtensionID,
		ExtensionVersion:  prepared.plan.Cache.Artifact.ExtensionVersion,
		ArtifactDigest:    prepared.plan.Cache.Artifact.PackageDigest,
		RuntimeInstanceID: prepared.plan.Cache.Artifact.RuntimeInstanceID,
		VersionID:         prepared.plan.Cache.Artifact.VersionID,
		CacheID:           prepared.plan.Cache.ID, ContractVersion: prepared.plan.Cache.ContractVersion,
		RegistryRevision: prepared.plan.Revision, ProviderRevision: prepared.providers.Revision,
		ProviderID: candidate.ProviderID, ProviderExtension: candidate.ExtensionID,
		ProviderArtifact: candidate.ArtifactDigest, ProviderRuntime: candidate.RuntimeInstance,
		ProviderVersionID: candidate.VersionID,
		Operation:         operation, Outcome: hostCacheTraceOutcome(err, hit, operation),
		Duration: time.Since(started), Attempts: attempts, Hit: hit, Affected: affected,
		TagDigest: prepared.traceTagDigest, TagCount: prepared.traceTagCount,
		InvalidatorID: prepared.traceInvalidator,
	}
	trace.Slow = trace.Duration >= HostCacheDefaultSlowThreshold
	s.traceSink.RecordHostCacheTrace(boundedHostCacheTrace(trace))
}

func (s *HostCacheService) recordRejectedHostCacheTrace(
	started time.Time,
	request HostCacheRequestBase,
	operation string,
	err error,
) {
	if s == nil || s.traceSink == nil {
		return
	}
	trace := HostCacheTrace{
		ExtensionID: request.Caller.ExtensionID, ExtensionVersion: request.Caller.ExtensionVersion,
		ArtifactDigest: request.Caller.ArtifactDigest, RuntimeInstanceID: request.Caller.RuntimeInstanceID,
		VersionID: request.Caller.VersionID,
		CacheID:   request.CacheID, Operation: operation, Outcome: hostCacheTraceOutcome(err, false, operation),
		Duration: time.Since(started),
	}
	trace.Slow = trace.Duration >= HostCacheDefaultSlowThreshold
	s.traceSink.RecordHostCacheTrace(boundedHostCacheTrace(trace))
}

func hostCacheTraceOutcome(err error, hit bool, operation string) HostCacheTraceOutcome {
	if err == nil {
		if operation == "get" || operation == "remember" {
			if hit {
				return HostCacheTraceHit
			}
			return HostCacheTraceMiss
		}
		return HostCacheTraceAllowed
	}
	switch {
	case errors.Is(err, context.Canceled):
		return HostCacheTraceCancel
	case errors.Is(err, context.DeadlineExceeded):
		return HostCacheTraceDeadline
	case errors.Is(err, ErrHostCacheDenied), errors.Is(err, ErrHostCacheScopeRequired):
		return HostCacheTraceDenied
	case errors.Is(err, ErrHostCacheStale):
		return HostCacheTraceStale
	case errors.Is(err, ErrHostCacheConflict):
		return HostCacheTraceConflict
	default:
		return HostCacheTraceError
	}
}

func boundedHostCacheTrace(trace HostCacheTrace) HostCacheTrace {
	trace.ExtensionID = boundedHostCacheTraceString(trace.ExtensionID, hostCacheTraceIdentityLimit)
	trace.ExtensionVersion = boundedHostCacheTraceString(trace.ExtensionVersion, hostCacheTraceVersionLimit)
	trace.ArtifactDigest = boundedHostCacheTraceString(trace.ArtifactDigest, hostCacheTraceDigestLimit)
	trace.RuntimeInstanceID = boundedHostCacheTraceString(trace.RuntimeInstanceID, hostCacheTraceIdentityLimit)
	trace.CacheID = boundedHostCacheTraceString(trace.CacheID, hostCacheTraceIdentityLimit)
	trace.ContractVersion = boundedHostCacheTraceString(trace.ContractVersion, hostCacheTraceVersionLimit)
	trace.ProviderID = boundedHostCacheTraceString(trace.ProviderID, hostCacheTraceIdentityLimit)
	trace.ProviderExtension = boundedHostCacheTraceString(trace.ProviderExtension, hostCacheTraceIdentityLimit)
	trace.ProviderArtifact = boundedHostCacheTraceString(trace.ProviderArtifact, hostCacheTraceDigestLimit)
	trace.ProviderRuntime = boundedHostCacheTraceString(trace.ProviderRuntime, hostCacheTraceIdentityLimit)
	trace.TagDigest = boundedHostCacheTraceString(trace.TagDigest, sha256.Size*2)
	trace.InvalidatorID = boundedHostCacheTraceString(trace.InvalidatorID, hostCacheTraceIdentityLimit)
	trace.Operation = boundedHostCacheTraceString(trace.Operation, 32)
	if trace.Duration < 0 {
		trace.Duration = 0
	}
	if trace.Attempts < 0 {
		trace.Attempts = 0
	} else if trace.Attempts > hostCacheMaximumProviderCount {
		trace.Attempts = hostCacheMaximumProviderCount
	}
	if trace.TagCount < 0 {
		trace.TagCount = 0
	} else if trace.TagCount > HostCacheMaximumTags {
		trace.TagCount = HostCacheMaximumTags
	}
	return trace
}

func (p *preparedHostCache) setTraceTags(tags []string) {
	if p == nil || len(tags) == 0 {
		return
	}
	values := slices.Clone(tags)
	slices.Sort(values)
	digest := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	p.traceTagDigest = hex.EncodeToString(digest[:])
	p.traceTagCount = len(values)
}

func boundedHostCacheTraceString(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) > limit {
		return value[:limit]
	}
	return value
}
