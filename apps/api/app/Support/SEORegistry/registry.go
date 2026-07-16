package seoregistry

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry publishes complete immutable SEO declaration graphs. It owns no
// lifecycle, HTTP, theme, sitemap storage, or extension process integration.
type Registry struct {
	mu      sync.Mutex
	state   atomic.Pointer[registryState]
	history map[artifactDeclarationIdentity]Publication
}

func New() *Registry {
	registry := &Registry{history: make(map[artifactDeclarationIdentity]Publication)}
	registry.state.Store(emptyState())
	return registry
}

func (r *Registry) Publish(publication Publication) (uint64, error) {
	return r.publish(nil, publication)
}

// PublishIfArtifact is the exact-artifact CAS used by later lifecycle wiring
// for upgrade and rollback publication.
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
	if current := r.load(); current.safeMode && !validCoreArtifactSeal(publication.Artifact) {
		return r.load().revision, ErrSafeMode
	}
	normalized, err := normalizePublication(publication)
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.history == nil {
		r.history = make(map[artifactDeclarationIdentity]Publication)
	}
	current := r.load()
	if current.safeMode && !validCoreArtifactSeal(normalized.Artifact) {
		return current.revision, ErrSafeMode
	}
	active, found := current.publications[normalized.Artifact.ExtensionID]
	if expected != nil {
		if expected.ExtensionID != normalized.Artifact.ExtensionID || !found || active.Artifact != *expected {
			return current.revision, ErrArtifactConflict
		}
	} else if found && active.Artifact != normalized.Artifact {
		return current.revision, ErrArtifactConflict
	}
	if found && active.Artifact == normalized.Artifact && !reflect.DeepEqual(active, normalized) {
		return current.revision, ErrArtifactConflict
	}
	if sealed, exists := r.history[declarationIdentity(normalized.Artifact)]; exists &&
		!equalPublicationDeclarations(sealed, normalized) {
		return current.revision, fmt.Errorf(
			"%w: immutable artifact %s changed its declarations", ErrArtifactConflict, normalized.Artifact.ExtensionID,
		)
	}
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode)
	if err != nil {
		return current.revision, err
	}
	if reflect.DeepEqual(current.publications, next.publications) {
		recordPublicationHistory(r.history, next.publications)
		return current.revision, nil
	}
	r.state.Store(next)
	recordPublicationHistory(r.history, next.publications)
	return next.revision, nil
}

func (r *Registry) ReplaceAll(publications []Publication, safeMode bool) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	return r.ReplaceAllIfRevision(r.load().revision, publications, safeMode)
}

func (r *Registry) ReplaceAllIfRevision(expectedRevision uint64, publications []Publication, safeMode bool) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	next, err := buildState(0, publications, safeMode)
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.history == nil {
		r.history = make(map[artifactDeclarationIdentity]Publication)
	}
	current := r.load()
	if current.revision != expectedRevision {
		return current.revision, ErrRevisionConflict
	}
	if err := validateExactPublicationReplay(current.publications, next.publications); err != nil {
		return current.revision, err
	}
	if err := validatePublicationHistory(r.history, next.publications); err != nil {
		return current.revision, err
	}
	if reflect.DeepEqual(current.publications, next.publications) && current.safeMode == next.safeMode {
		recordPublicationHistory(r.history, next.publications)
		return current.revision, nil
	}
	next.revision = current.revision + 1
	r.state.Store(next)
	recordPublicationHistory(r.history, next.publications)
	return next.revision, nil
}

func validateExactPublicationReplay(current, next map[string]Publication) error {
	for extensionID, previous := range current {
		candidate, found := next[extensionID]
		if found && candidate.Artifact == previous.Artifact && !reflect.DeepEqual(candidate, previous) {
			return fmt.Errorf("%w: exact artifact %s changed its declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

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
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode)
	if err != nil {
		return current.revision, false, err
	}
	r.state.Store(next)
	return next.revision, true, nil
}

func (r *Registry) Snapshot() Snapshot {
	return cloneSnapshot(snapshotFromState(r.load()))
}

func (r *Registry) SnapshotPublication(extensionID string) (Publication, bool) {
	if r == nil {
		return Publication{}, false
	}
	publication, found := r.load().publications[strings.ToLower(strings.TrimSpace(extensionID))]
	if !found {
		return Publication{}, false
	}
	return clonePublication(publication), true
}

func (r *Registry) Inspect(scope string) (ScopeInspection, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if r == nil || (scope != GlobalScope && !idPattern.MatchString(scope)) {
		return ScopeInspection{}, ErrInvalid
	}
	state := r.load()
	return ScopeInspection{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode,
		Scope: scope, Contributions: cloneContributions(contributionsForScope(state, scope)),
		Conflicts: conflictsForScope(state, scope),
	}, nil
}

func (r *Registry) Revision() uint64 { return r.load().revision }

func (r *Registry) CacheState() CacheState {
	state := r.load()
	return CacheState{Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode}
}

func (r *Registry) CacheInvalidated(previous CacheState) bool { return r.CacheState() != previous }

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
	return fmt.Sprintf("SEORegistry(revision=%d,digest=%s,safeMode=%t)", state.revision, state.digest, state.safeMode)
}
