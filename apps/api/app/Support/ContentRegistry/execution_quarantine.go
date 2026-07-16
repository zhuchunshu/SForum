package contentregistry

import (
	"fmt"
	"sync"
)

// runtimeQuarantine is intentionally local to one immutable Executor. The
// current admission contract cannot stop or globally quarantine a subprocess,
// so a timed-out non-cooperative call permanently fails closed for its exact
// artifact in this Executor. Late completion never clears the mark and does not
// prove recovery; only lifecycle replacement with a newly reviewed Executor can
// admit that identity again.
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
	_, quarantined := q.artifacts[artifact]
	q.mu.RUnlock()
	return quarantined
}

func (e *Executor) runtimeQuarantineError(artifact Artifact) error {
	if e != nil && e.quarantine.contains(artifact) {
		return fmt.Errorf("%w: %w", ErrRuntimeUnavailable, ErrRuntimeQuarantined)
	}
	return nil
}
