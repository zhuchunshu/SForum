package entityregistry

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry publishes an immutable Entity/Taxonomy/Field graph. Writers validate
// off to the side; readers observe the previous or next snapshot only.
//
// Registry owns declaration storage, inspection, and Host-derived permission/
// index/import-export/deletion plans. Durable entity rows, term storage, and
// field values remain outside this package.
type Registry struct {
	mu    sync.Mutex
	state atomic.Pointer[registryState]
}

func New() *Registry {
	registry := &Registry{}
	registry.state.Store(emptyState())
	return registry
}

func (r *Registry) Publish(publication Publication) (uint64, error) {
	return r.publish(nil, publication)
}

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

	current := r.load()
	if current.safeMode && !normalized.Artifact.Core {
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
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode)
	if err != nil {
		return current.revision, err
	}
	if equalStates(current, next) {
		return current.revision, nil
	}
	r.state.Store(next)
	return next.revision, nil
}

func (r *Registry) ReplaceAll(publications []Publication, safeMode bool) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	current := r.load()
	if current.revision != 0 {
		return current.revision, ErrRevisionConflict
	}
	return r.ReplaceAllIfRevision(current.revision, publications, safeMode)
}

func (r *Registry) ReplaceAllIfRevision(
	expectedRevision uint64,
	publications []Publication,
	safeMode bool,
) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	next, err := buildState(0, publications, safeMode)
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
	if equalStates(current, next) {
		return current.revision, nil
	}
	next.revision = current.revision + 1
	r.state.Store(next)
	return next.revision, nil
}

// Remove disables one exact artifact publication. Source entity rows are not
// stored here; disable cannot rewrite or delete user data.
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

func (r *Registry) Revision() uint64 {
	return r.load().revision
}

func (r *Registry) Snapshot() Snapshot {
	return cloneSnapshot(snapshotFromState(r.load()))
}

func (r *Registry) SnapshotPublication(extensionID string) (Publication, bool) {
	if r == nil {
		return Publication{}, false
	}
	publication, ok := r.load().publications[strings.ToLower(strings.TrimSpace(extensionID))]
	if !ok {
		return Publication{}, false
	}
	return clonePublication(publication), true
}

func (r *Registry) Resolve(entityID string) (Contribution, error) {
	if r == nil {
		return Contribution{}, ErrInvalid
	}
	contribution, ok := r.load().entities[strings.ToLower(strings.TrimSpace(entityID))]
	if !ok {
		return Contribution{}, ErrNotFound
	}
	return cloneContribution(contribution), nil
}

// List returns contributions for one frozen kind (or all when kind empty).
func (r *Registry) List(kind string) []Contribution {
	if r == nil {
		return nil
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind != "" && !validKind(kind) {
		return nil
	}
	values := r.load().entities
	result := make([]Contribution, 0, len(values))
	for _, contribution := range values {
		if kind == "" || contribution.Kind == kind {
			result = append(result, cloneContribution(contribution))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return contributionBefore(result[i], result[j])
	})
	return result
}

// ListFieldsForEntity returns field declarations bound to one entity type id.
func (r *Registry) ListFieldsForEntity(entityID string) []Contribution {
	if r == nil {
		return nil
	}
	entityID = strings.ToLower(strings.TrimSpace(entityID))
	if entityID == "" {
		return nil
	}
	result := make([]Contribution, 0)
	for _, contribution := range r.List(KindField) {
		if contribution.EntityID == entityID {
			result = append(result, contribution)
		}
	}
	return result
}

// ListTaxonomiesForEntity returns taxonomies that declare the entity id.
func (r *Registry) ListTaxonomiesForEntity(entityID string) []Contribution {
	if r == nil {
		return nil
	}
	entityID = strings.ToLower(strings.TrimSpace(entityID))
	if entityID == "" {
		return nil
	}
	result := make([]Contribution, 0)
	for _, contribution := range r.List(KindTaxonomy) {
		for _, bound := range contribution.EntityIDs {
			if bound == entityID {
				result = append(result, contribution)
				break
			}
		}
	}
	return result
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
	return fmt.Sprintf("EntityRegistry(revision=%d,digest=%s,safeMode=%t)", state.revision, state.digest, state.safeMode)
}
