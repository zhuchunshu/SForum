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

// Publish atomically adds one extension publication or replays the exact active
// artifact idempotently. Artifact changes must use PublishIfArtifact so a stale
// runtime cannot overwrite a newer publication without an exact CAS check.
func (r *Registry) Publish(publication Publication) (uint64, error) {
	return r.publish(nil, publication)
}

// PublishIfArtifact replaces one extension publication only while expected is
// still the exact active artifact. Package rollback uses the same CAS contract;
// artifact version ordering is deliberately not inferred here.
func (r *Registry) PublishIfArtifact(expected Artifact, publication Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	normalizedExpected, err := normalizeArtifact(expected)
	if err != nil {
		return r.load().revision, ErrInvalid
	}
	return r.publish(&normalizedExpected, publication)
}

func (r *Registry) publish(expected *Artifact, publication Publication) (uint64, error) {
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
	active, found := current.publications[normalized.Artifact.ExtensionID]
	if expected != nil {
		if expected.ExtensionID != normalized.Artifact.ExtensionID || !found || active.Artifact != *expected {
			return current.revision, ErrArtifactConflict
		}
	} else if found && active.Artifact != normalized.Artifact {
		return current.revision, ErrArtifactConflict
	}
	// 同 exact artifact 仅允许语义等价重放；声明漂移 fail-closed。
	if found && active.Artifact == normalized.Artifact && !equalPublications(active, normalized) {
		return current.revision, ErrArtifactConflict
	}
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications))
	if err != nil {
		return current.revision, err
	}
	if equalPublicationMaps(current.publications, next.publications) {
		return current.revision, nil
	}
	r.state.Store(next)
	return next.revision, nil
}

// ReplaceAll replaces the complete graph only if no writer advances the
// revision while the candidate graph is being built.
func (r *Registry) ReplaceAll(publications []Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	return r.ReplaceAllIfRevision(r.load().revision, publications)
}

// ReplaceAllIfRevision publishes one fully validated graph while the expected
// revision is still current. Exact-artifact declarations are immutable; a
// revision-fenced batch may add, remove, upgrade, or roll back artifacts.
func (r *Registry) ReplaceAllIfRevision(expectedRevision uint64, publications []Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	// 完整图先在锁外构建，锁内只做 revision CAS、artifact 漂移校验和一次发布。
	next, err := buildState(0, publications)
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != expectedRevision {
		return current.revision, ErrRevisionConflict
	}
	if err := validateExactPublicationReplay(current.publications, next.publications); err != nil {
		return current.revision, err
	}
	if equalPublicationMaps(current.publications, next.publications) {
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

func equalPublicationMaps(left, right map[string]Publication) bool {
	return reflect.DeepEqual(left, right)
}

func equalPublications(left, right Publication) bool {
	return reflect.DeepEqual(left, right)
}

func validateExactPublicationReplay(current, next map[string]Publication) error {
	for extensionID, active := range current {
		candidate, found := next[extensionID]
		if !found || active.Artifact != candidate.Artifact {
			continue
		}
		if !equalPublications(active, candidate) {
			return fmt.Errorf("%w: exact artifact %s changed declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
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
