package navigationregistry

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

type registryState struct {
	revision uint64
	digest   string

	publications map[string]Publication
	navigation   map[string]NavigationContribution
	regions      map[string]RegionContribution

	navigationTargets  map[string]NavigationContribution
	regionTargets      map[string]RegionContribution
	navigationByTarget map[string][]NavigationContribution
	regionsByTarget    map[string][]RegionContribution
}

// Registry publishes a complete immutable navigation/region graph. Writers
// validate and compose off to the side; readers observe the old or new graph,
// never a half-disabled provider or a partially rebuilt theme region set.
type Registry struct {
	mu    sync.Mutex
	state atomic.Pointer[registryState]
}

func New() *Registry {
	r := &Registry{}
	r.state.Store(emptyState())
	return r
}

// Publish atomically replaces one extension's entire publication.
func (r *Registry) Publish(publication Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	normalized, err := normalizePublication(publication)
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications))
	if err != nil {
		return current.revision, err
	}
	if reflect.DeepEqual(current.publications, next.publications) {
		return current.revision, nil
	}
	r.state.Store(next)
	return next.revision, nil
}

// ReplaceAll is the startup/restart path. Input order cannot change provider
// winners, target ordering, or the restart-stable digest.
func (r *Registry) ReplaceAll(publications []Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	next, err := buildState(0, publications)
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if reflect.DeepEqual(current.publications, next.publications) {
		return current.revision, nil
	}
	next.revision = current.revision + 1
	r.state.Store(next)
	return next.revision, nil
}

// Remove disables one exact artifact publication. A stale shutdown cannot
// remove a replacement artifact, and required consumers prevent unsafe removal.
func (r *Registry) Remove(artifact Artifact) (uint64, bool, error) {
	if r == nil {
		return 0, false, ErrInvalid
	}
	normalized, err := normalizeArtifact(artifact)
	if err != nil {
		return r.load().revision, false, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	publication, found := current.publications[normalized.ExtensionID]
	if !found {
		return current.revision, false, nil
	}
	if publication.Artifact != normalized {
		return current.revision, false, ErrArtifactConflict
	}
	publications := clonePublicationMap(current.publications)
	delete(publications, normalized.ExtensionID)
	next, err := buildState(current.revision+1, publicationValues(publications))
	if err != nil {
		return current.revision, false, err
	}
	r.state.Store(next)
	return next.revision, true, nil
}

func (r *Registry) Revision() uint64 {
	return r.load().revision
}

func (r *Registry) CacheState() CacheState {
	state := r.load()
	return CacheState{Revision: state.revision, Digest: state.digest}
}

// CacheInvalidated lets callers fence a cached resolution with both the local
// monotonic revision and restart-stable graph digest.
func (r *Registry) CacheInvalidated(previous CacheState) bool {
	current := r.CacheState()
	return previous.Revision != current.Revision || previous.Digest != current.Digest
}

func (r *Registry) Snapshot() Snapshot {
	state := r.load()
	return cloneSnapshot(snapshotFromState(state))
}

func (r *Registry) load() *registryState {
	if r != nil {
		if state := r.state.Load(); state != nil {
			return state
		}
	}
	return emptyState()
}

func (r *Registry) String() string {
	state := r.load()
	return fmt.Sprintf("NavigationRegistry(revision=%d,digest=%s)", state.revision, state.digest)
}
