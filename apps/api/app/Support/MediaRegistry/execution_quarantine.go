package mediaregistry

import (
	"errors"
	"sync"
)

// runtimeQuarantine is local to one Executor. This kernel cannot terminate a
// subprocess, so a callback that ignores cancellation remains quarantined for
// its exact artifact even after it eventually returns.
type runtimeQuarantine struct {
	mu        sync.RWMutex
	artifacts map[Artifact]struct{}
}

func (q *runtimeQuarantine) mark(artifact Artifact) {
	q.mu.Lock()
	if q.artifacts == nil {
		q.artifacts = make(map[Artifact]struct{})
	}
	q.artifacts[artifact] = struct{}{}
	q.mu.Unlock()
}

func (q *runtimeQuarantine) contains(artifact Artifact) bool {
	q.mu.RLock()
	_, found := q.artifacts[artifact]
	q.mu.RUnlock()
	return found
}

func (e *Executor) IsQuarantined(artifact Artifact) bool {
	if e == nil {
		return false
	}
	normalized, err := normalizeArtifact(artifact)
	if err != nil || normalized.Core && !validCoreArtifactSeal(artifact) {
		return false
	}
	return e.quarantine.contains(normalized)
}

func (e *Executor) quarantineError(artifact Artifact) error {
	if e != nil && e.quarantine.contains(artifact) {
		return errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
	}
	return nil
}
