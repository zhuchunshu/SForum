package cacheregistry

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry publishes complete immutable cache declaration graphs. It contains
// no CacheService, Redis client, lifecycle code, or provider selector.
type Registry struct {
	mu          sync.Mutex
	state       atomic.Pointer[registryState]
	history     map[artifactDeclarationIdentity]Publication
	admissionMu sync.RWMutex
	admission   func(Artifact) bool
}

func New() *Registry {
	registry := &Registry{history: make(map[artifactDeclarationIdentity]Publication)}
	registry.state.Store(emptyState())
	return registry
}

// WithPluginAdmission installs the Host's exact-runtime availability gate.
// Missing admission fails all third-party Resolve and Plan calls closed.
func (r *Registry) WithPluginAdmission(admission func(Artifact) bool) *Registry {
	if r == nil {
		return r
	}
	r.admissionMu.Lock()
	r.admission = admission
	r.admissionMu.Unlock()
	return r
}

func (r *Registry) artifactAdmitted(artifact Artifact) bool {
	if validCoreArtifactSeal(artifact) {
		return true
	}
	if r == nil {
		return false
	}
	r.admissionMu.RLock()
	admission := r.admission
	r.admissionMu.RUnlock()
	return admission != nil && admission(artifact)
}

func (r *Registry) requireArtifactAdmitted(artifact Artifact) error {
	if !r.artifactAdmitted(artifact) {
		return ErrArtifactUnavailable
	}
	return nil
}

// requireStableAdmission checks both the exact runtime and immutable snapshot
// twice. The Host callback may inspect external state or trigger reconciliation.
func (r *Registry) requireStableAdmission(state *registryState, artifact Artifact) error {
	if err := r.requireArtifactAdmitted(artifact); err != nil {
		return err
	}
	if r.load() != state {
		return ErrArtifactConflict
	}
	if err := r.requireArtifactAdmitted(artifact); err != nil {
		return err
	}
	if r.load() != state {
		return ErrArtifactConflict
	}
	return nil
}

func (r *Registry) Publish(publication Publication) (uint64, error) {
	return r.publish(nil, publication)
}

// PublishIfArtifact replaces one extension only while expected is the exact
// active artifact. Upgrades and rollbacks use the same explicit CAS.
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
		return current.revision, ErrSafeMode
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
	if found && active.Artifact == normalized.Artifact && !equalPublications(active, normalized) {
		return current.revision, ErrArtifactConflict
	}
	if sealed, exists := r.history[declarationIdentity(normalized.Artifact)]; exists && !equalPublicationDeclarations(sealed, normalized) {
		return current.revision, fmt.Errorf("%w: immutable artifact %s changed declarations", ErrArtifactConflict, normalized.Artifact.ExtensionID)
	}
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode)
	if err != nil {
		return current.revision, err
	}
	if equalPublicationMaps(current.publications, next.publications) {
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
	// 完整图在锁外构建；锁内只做 revision CAS、exact replay 检查和发布。
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
	if equalPublicationMaps(current.publications, next.publications) && current.safeMode == next.safeMode {
		recordPublicationHistory(r.history, next.publications)
		return current.revision, nil
	}
	next.revision = current.revision + 1
	r.state.Store(next)
	recordPublicationHistory(r.history, next.publications)
	return next.revision, nil
}

// Remove removes only the exact active artifact. A stale runtime cannot remove
// a newer version or a replacement runtime instance.
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

// Resolve returns a declaration only while its exact runtime is admitted.
// Snapshot remains the admission-independent inspection surface.
func (r *Registry) Resolve(cacheID string) (Contribution, error) {
	if r == nil {
		return Contribution{}, ErrInvalid
	}
	cacheID = strings.ToLower(strings.TrimSpace(cacheID))
	if !idPattern.MatchString(cacheID) {
		return Contribution{}, ErrInvalid
	}
	state := r.load()
	contribution, found := state.caches[cacheID]
	if !found {
		return Contribution{}, ErrNotFound
	}
	if err := r.requireStableAdmission(state, contribution.Artifact); err != nil {
		return Contribution{}, err
	}
	return cloneContribution(contribution), nil
}

func (r *Registry) Revision() uint64 {
	return r.load().revision
}

func (r *Registry) CacheState() CacheState {
	state := r.load()
	return CacheState{Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode}
}

func (r *Registry) CacheInvalidated(previous CacheState) bool {
	current := r.CacheState()
	return current != previous
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
	return fmt.Sprintf("CacheRegistry(revision=%d,digest=%s,safeMode=%t)", state.revision, state.digest, state.safeMode)
}
