package mediaregistry

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry owns only an immutable declaration graph. Storage, attachment
// records, lifecycle transitions, provider processes, and River remain outside.
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

func (r *Registry) PublishIfArtifact(expected Artifact, publication Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	normalized, err := normalizeArtifact(expected)
	if err != nil || normalized.Core && !validCoreArtifactSeal(expected) {
		return r.load().revision, ErrInvalid
	}
	return r.publish(&normalized, publication)
}

func (r *Registry) publish(expected *Artifact, publication Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	current := r.load()
	if current.safeMode && !validCoreArtifactSeal(publication.Artifact) {
		return current.revision, ErrSafeMode
	}
	normalized, err := normalizePublication(publication)
	if err != nil {
		return current.revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.history == nil {
		r.history = make(map[artifactDeclarationIdentity]Publication)
	}
	current = r.load()
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
	if sealed, exists := r.history[declarationIdentity(normalized.Artifact)]; exists &&
		!equalPublicationDeclarations(sealed, normalized) {
		return current.revision, fmt.Errorf("%w: immutable artifact %s changed declarations", ErrArtifactConflict, normalized.Artifact.ExtensionID)
	}
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode, sortedSelections(current.selections))
	if err != nil {
		return current.revision, err
	}
	if equalPublicationMaps(current.publications, next.publications) && current.digest == next.digest {
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
	current := r.load()
	next, err := buildState(0, publications, safeMode, sortedSelections(current.selections))
	if err != nil {
		return current.revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.history == nil {
		r.history = make(map[artifactDeclarationIdentity]Publication)
	}
	current = r.load()
	if current.revision != expectedRevision {
		return current.revision, ErrRevisionConflict
	}
	if err := validateExactReplay(current.publications, next.publications); err != nil {
		return current.revision, err
	}
	if err := validatePublicationHistory(r.history, next.publications); err != nil {
		return current.revision, err
	}
	if equalPublicationMaps(current.publications, next.publications) && current.digest == next.digest && current.safeMode == next.safeMode {
		recordPublicationHistory(r.history, next.publications)
		return current.revision, nil
	}
	next.revision = current.revision + 1
	r.state.Store(next)
	recordPublicationHistory(r.history, next.publications)
	return next.revision, nil
}

// Remove removes only an exact active artifact. Source media is not stored in
// this graph, so disable/uninstall cannot delete an original through Registry.
func (r *Registry) Remove(artifact Artifact) (uint64, bool, error) {
	if r == nil {
		return 0, false, ErrInvalid
	}
	normalized, err := normalizeArtifact(artifact)
	if err != nil || normalized.Core && !validCoreArtifactSeal(artifact) {
		return r.load().revision, false, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	active, found := current.publications[normalized.ExtensionID]
	if !found {
		return current.revision, false, nil
	}
	if active.Artifact != normalized {
		return current.revision, false, ErrArtifactConflict
	}
	publications := clonePublicationMap(current.publications)
	delete(publications, normalized.ExtensionID)
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode, sortedSelections(current.selections))
	if err != nil {
		return current.revision, false, err
	}
	r.state.Store(next)
	return next.revision, true, nil
}

func (r *Registry) SelectProvider(expectedRevision uint64, selection ProviderSelection) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	selection.Family = strings.ToLower(strings.TrimSpace(selection.Family))
	selection.Key = strings.ToLower(strings.TrimSpace(selection.Key))
	selection.Provider.ContributionID = strings.ToLower(strings.TrimSpace(selection.Provider.ContributionID))
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != expectedRevision {
		return current.revision, ErrRevisionConflict
	}
	canonical, valid := canonicalSelectionForState(current, selection)
	if !valid {
		return current.revision, ErrConflict
	}
	selection = canonical
	selections := sortedSelections(current.selections)
	replaced := false
	for index := range selections {
		if selections[index].Family == selection.Family && selections[index].Key == selection.Key {
			selections[index] = selection
			replaced = true
			break
		}
	}
	if !replaced {
		selections = append(selections, selection)
	}
	next, err := buildState(current.revision+1, publicationValues(current.publications), current.safeMode, selections)
	if err != nil {
		return current.revision, err
	}
	if current.digest == next.digest {
		return current.revision, nil
	}
	r.state.Store(next)
	return next.revision, nil
}

func (r *Registry) ResetProvider(expectedRevision uint64, family, key string) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	family = strings.ToLower(strings.TrimSpace(family))
	key = strings.ToLower(strings.TrimSpace(key))
	if !validConflictFamily(family) || key == "" {
		return r.load().revision, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != expectedRevision {
		return current.revision, ErrRevisionConflict
	}
	mapKey := selectionMapKey(family, key)
	if _, found := current.selections[mapKey]; !found {
		return current.revision, nil
	}
	selections := sortedSelections(current.selections)
	filtered := selections[:0]
	for _, selection := range selections {
		if selectionMapKey(selection.Family, selection.Key) != mapKey {
			filtered = append(filtered, selection)
		}
	}
	next, err := buildState(current.revision+1, publicationValues(current.publications), current.safeMode, filtered)
	if err != nil {
		return current.revision, err
	}
	r.state.Store(next)
	return next.revision, nil
}

func (r *Registry) Snapshot() Snapshot { return cloneSnapshot(snapshotFromState(r.load())) }

func (r *Registry) SnapshotPublication(extensionID string) (Publication, bool) {
	if r == nil {
		return Publication{}, false
	}
	state := r.load()
	value, found := state.publications[strings.ToLower(strings.TrimSpace(extensionID))]
	if !found {
		return Publication{}, false
	}
	return activePublication(state, value), true
}

func (r *Registry) Revision() uint64 { return r.load().revision }

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
	return fmt.Sprintf("MediaRegistry(revision=%d,digest=%s,safeMode=%t)", state.revision, state.digest, state.safeMode)
}
