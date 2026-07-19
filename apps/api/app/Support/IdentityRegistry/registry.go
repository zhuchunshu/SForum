package identityregistry

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type registryState struct {
	revision     uint64
	digest       string
	safeMode     bool
	publications map[string]Publication
	permissions  map[string]PermissionContribution
	userFields   map[string]UserFieldContribution
	providers    map[string]ProviderContribution
	tombstones   map[string]Tombstone
}

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
	normalized, err := normalizeArtifact(expected)
	if err != nil {
		return r.load().revision, err
	}
	return r.publish(&normalized, publication)
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
	if err := validateExecutableBindings(normalized); err != nil {
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
		if !found || expected.ExtensionID != normalized.Artifact.ExtensionID || active.Artifact != *expected {
			return current.revision, ErrArtifactConflict
		}
	} else if found && active.Artifact != normalized.Artifact {
		return current.revision, ErrArtifactConflict
	}
	if found && active.Artifact == normalized.Artifact && !EqualPublicContract(active, normalized) {
		return current.revision, ErrArtifactConflict
	}
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications), current.tombstoneValues(), current.safeMode, current)
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
	return r.ReplaceAllIfRevision(current.revision, publications, current.tombstoneValues(), safeMode)
}

// ReplaceAllIfRevision is the startup/reconciliation path. Callers must pass
// durable tombstones restored from Host storage; the registry also derives any
// newly retired identities from the current process snapshot.
func (r *Registry) ReplaceAllIfRevision(expectedRevision uint64, publications []Publication, tombstones []Tombstone, safeMode bool) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	prepared, err := buildState(0, publications, tombstones, safeMode, r.load())
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != expectedRevision {
		return current.revision, ErrRevisionConflict
	}
	if err := validateExactReplay(current.publications, prepared.publications); err != nil {
		return current.revision, err
	}
	if equalStates(current, prepared) {
		return current.revision, nil
	}
	prepared.revision = current.revision + 1
	r.state.Store(prepared)
	return prepared.revision, nil
}

func (r *Registry) Remove(artifact Artifact) (uint64, bool, error) {
	if r == nil {
		return 0, false, ErrInvalid
	}
	normalized, err := normalizeArtifact(artifact)
	if err != nil {
		return r.load().revision, false, err
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
	next, err := buildState(current.revision+1, publicationValues(publications), current.tombstoneValues(), current.safeMode, current)
	if err != nil {
		return current.revision, false, err
	}
	r.state.Store(next)
	return next.revision, true, nil
}

func (r *Registry) ResolvePermission(key string) (PermissionContribution, error) {
	value, ok := r.load().permissions[strings.ToLower(strings.TrimSpace(key))]
	if !ok {
		return PermissionContribution{}, ErrNotFound
	}
	return clonePermissionContribution(value), nil
}

func (r *Registry) ResolveUserField(id string) (UserFieldContribution, error) {
	value, ok := r.load().userFields[strings.ToLower(strings.TrimSpace(id))]
	if !ok {
		return UserFieldContribution{}, ErrNotFound
	}
	return cloneUserFieldContribution(value, true), nil
}

func (r *Registry) ResolveProvider(id string) (ProviderContribution, error) {
	value, ok := r.load().providers[strings.ToLower(strings.TrimSpace(id))]
	if !ok {
		return ProviderContribution{}, ErrNotFound
	}
	return cloneProviderContribution(value, true), nil
}

// ResolveProviderSnapshot resolves one provider from a single immutable state
// load. Safe Mode never returns a third-party executable claim.
func (r *Registry) ResolveProviderSnapshot(id string) (ProviderResolution, error) {
	if r == nil {
		return ProviderResolution{}, ErrInvalid
	}
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return ProviderResolution{}, ErrInvalid
	}
	state := r.load()
	resolution := ProviderResolution{
		Revision: state.revision,
		Digest:   state.digest,
		SafeMode: state.safeMode,
	}
	if state.safeMode {
		return resolution, ErrSafeMode
	}
	provider, ok := state.providers[id]
	if !ok {
		return resolution, ErrNotFound
	}
	resolution.Provider = cloneProviderContribution(provider, true)
	return resolution, nil
}

// ValidateProviderResolution verifies that an earlier exact provider claim is
// still executable. Global revision/digest drift alone is not authority drift:
// only Safe Mode or a changed/missing exact provider invalidates the claim.
func (r *Registry) ValidateProviderResolution(resolution ProviderResolution) error {
	if r == nil || resolution.SafeMode || resolution.Revision == 0 ||
		strings.TrimSpace(resolution.Digest) == "" ||
		strings.TrimSpace(resolution.Provider.ID) == "" {
		return ErrInvalid
	}
	state := r.load()
	if state.safeMode {
		return ErrSafeMode
	}
	providerID := strings.ToLower(strings.TrimSpace(resolution.Provider.ID))
	current, ok := state.providers[providerID]
	if !ok {
		return ErrNotFound
	}
	// ProviderResolution intentionally exposes only public contract material.
	// Private compiled validators remain Registry-owned and are resolved later
	// through exact artifact-bound Schema claims.
	if !reflect.DeepEqual(cloneProviderContribution(current, true), resolution.Provider) {
		return ErrArtifactConflict
	}
	return nil
}

func (r *Registry) Providers(kind string) []ProviderContribution {
	kind = strings.ToLower(strings.TrimSpace(kind))
	result := make([]ProviderContribution, 0)
	for _, provider := range r.load().providers {
		if kind == "" || provider.Kind == kind {
			result = append(result, cloneProviderContribution(provider, true))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].Artifact.ExtensionID < result[j].Artifact.ExtensionID
	})
	return result
}

func (r *Registry) Snapshot() Snapshot {
	return snapshotFromState(r.load())
}

func (r *Registry) SnapshotPublication(extensionID string) (Publication, bool) {
	publication, ok := r.load().publications[strings.ToLower(strings.TrimSpace(extensionID))]
	if !ok {
		return Publication{}, false
	}
	return clonePublication(publication), true
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

func emptyState() *registryState {
	state := &registryState{
		publications: map[string]Publication{}, permissions: map[string]PermissionContribution{},
		userFields: map[string]UserFieldContribution{}, providers: map[string]ProviderContribution{},
		tombstones: map[string]Tombstone{},
	}
	state.digest = publicationDigest(snapshotFromState(state))
	return state
}

func buildState(revision uint64, publications []Publication, tombstones []Tombstone, safeMode bool, previous *registryState) (*registryState, error) {
	if len(publications) > maxPublications {
		return nil, ErrInvalid
	}
	state := emptyState()
	state.revision = revision
	state.safeMode = safeMode
	owners := map[string]string{}
	// Process-local ownership history is append-only. The caller-supplied list
	// restores durable history after boot, but can never erase history already
	// observed by this process through a partial reconciliation payload.
	if previous != nil {
		for _, raw := range previous.tombstoneValues() {
			if err := addTombstone(state, owners, raw); err != nil {
				return nil, err
			}
		}
	}
	for _, raw := range tombstones {
		if err := addTombstone(state, owners, raw); err != nil {
			return nil, err
		}
	}
	if previous != nil {
		for _, retired := range retiredTombstones(previous.publications, publications, safeMode) {
			if err := addTombstone(state, owners, retired); err != nil {
				return nil, err
			}
		}
	}
	permissionCount, userFieldCount, providerCount := 0, 0, 0
	for _, raw := range publications {
		if safeMode && !validCoreArtifactSeal(raw.Artifact) {
			continue
		}
		publication, err := normalizePublication(raw)
		if err != nil {
			return nil, err
		}
		if err := validateExecutableBindings(publication); err != nil {
			return nil, err
		}
		permissionCount += len(publication.Permissions)
		if publication.Identity != nil {
			userFieldCount += len(publication.Identity.UserFields)
			providerCount += len(publication.Identity.Providers)
		}
		if permissionCount > maxPermissionsTotal || userFieldCount > maxUserFieldsTotal || providerCount > maxProvidersTotal {
			return nil, ErrInvalid
		}
		if _, duplicate := state.publications[publication.Artifact.ExtensionID]; duplicate {
			return nil, ErrConflict
		}
		state.publications[publication.Artifact.ExtensionID] = publication
		for _, permission := range publication.Permissions {
			if err := claimOwner(owners, TombstoneKindPermission, permission.Key, publication.Artifact.ExtensionID); err != nil {
				return nil, err
			}
			if _, duplicate := state.permissions[permission.Key]; duplicate {
				return nil, ErrConflict
			}
			state.permissions[permission.Key] = PermissionContribution{PermissionDefinition: permission, Artifact: publication.Artifact}
		}
		if publication.Identity == nil {
			continue
		}
		for _, field := range publication.Identity.UserFields {
			if err := claimOwner(owners, TombstoneKindUserField, field.ID, publication.Artifact.ExtensionID); err != nil {
				return nil, err
			}
			if _, duplicate := state.userFields[field.ID]; duplicate {
				return nil, ErrConflict
			}
			state.userFields[field.ID] = UserFieldContribution{UserField: cloneUserField(field), Artifact: publication.Artifact}
		}
		for _, provider := range publication.Identity.Providers {
			if err := claimOwner(owners, TombstoneKindProvider, provider.ID, publication.Artifact.ExtensionID); err != nil {
				return nil, err
			}
			if _, duplicate := state.providers[provider.ID]; duplicate {
				return nil, ErrConflict
			}
			state.providers[provider.ID] = ProviderContribution{Provider: cloneProvider(provider), Artifact: publication.Artifact}
		}
	}
	state.digest = publicationDigest(snapshotFromState(state))
	return state, nil
}

func addTombstone(state *registryState, owners map[string]string, raw Tombstone) error {
	tombstone, err := normalizeTombstone(raw)
	if err != nil {
		return err
	}
	if owner := owners[ownershipKey(tombstone.Kind, tombstone.ID)]; owner != "" && owner != tombstone.OwnerExtensionID {
		return ErrConflict
	}
	owners[ownershipKey(tombstone.Kind, tombstone.ID)] = tombstone.OwnerExtensionID
	state.tombstones[tombstoneKey(tombstone)] = tombstone
	return nil
}

func claimOwner(owners map[string]string, kind, id, owner string) error {
	key := ownershipKey(kind, id)
	if existing := owners[key]; existing != "" && existing != owner {
		return fmt.Errorf("%w: %s %s remains owned by %s", ErrConflict, kind, id, existing)
	}
	owners[key] = owner
	return nil
}

func retiredTombstones(previous map[string]Publication, next []Publication, safeMode bool) []Tombstone {
	nextByExtension := map[string]Publication{}
	for _, publication := range next {
		if normalized, err := normalizePublication(publication); err == nil {
			if safeMode && !normalized.Artifact.Core {
				continue
			}
			nextByExtension[normalized.Artifact.ExtensionID] = normalized
		}
	}
	result := make([]Tombstone, 0)
	for extensionID, publication := range previous {
		candidate, found := nextByExtension[extensionID]
		active := activeDeclarationVersions(candidate, found)
		for _, declaration := range publicationTombstones(publication) {
			if active[tombstoneKey(declaration)] {
				continue
			}
			result = append(result, declaration)
		}
	}
	return result
}

func activeDeclarationVersions(publication Publication, found bool) map[string]bool {
	result := map[string]bool{}
	if !found {
		return result
	}
	for _, value := range publicationTombstones(publication) {
		result[tombstoneKey(value)] = true
	}
	return result
}

func publicationTombstones(publication Publication) []Tombstone {
	result := make([]Tombstone, 0, len(publication.Permissions))
	for _, permission := range publication.Permissions {
		result = append(result, Tombstone{Kind: TombstoneKindPermission, ID: permission.Key, ContractVersion: permission.ContractVersion, OwnerExtensionID: publication.Artifact.ExtensionID})
	}
	if publication.Identity != nil {
		for _, field := range publication.Identity.UserFields {
			result = append(result, Tombstone{Kind: TombstoneKindUserField, ID: field.ID, ContractVersion: field.ContractVersion, OwnerExtensionID: publication.Artifact.ExtensionID})
		}
		for _, provider := range publication.Identity.Providers {
			result = append(result, Tombstone{Kind: TombstoneKindProvider, ID: provider.ID, ContractVersion: provider.ContractVersion, OwnerExtensionID: publication.Artifact.ExtensionID})
		}
	}
	return result
}

func validateExactReplay(current, next map[string]Publication) error {
	for extensionID, active := range current {
		candidate, found := next[extensionID]
		if found && active.Artifact == candidate.Artifact && !EqualPublicContract(active, candidate) {
			return ErrArtifactConflict
		}
	}
	return nil
}

func equalStates(left, right *registryState) bool {
	return left.safeMode == right.safeMode && equalPublicationMaps(left.publications, right.publications) &&
		reflect.DeepEqual(left.tombstones, right.tombstones)
}

func equalPublicationMaps(left, right map[string]Publication) bool {
	if len(left) != len(right) {
		return false
	}
	for extensionID, publication := range left {
		candidate, found := right[extensionID]
		if !found || !EqualPublicContract(publication, candidate) {
			return false
		}
	}
	return true
}

// EqualPublicContract compares stable inspectable metadata and Host-derived
// digests without comparing raw Schema bytes or validator pointers.
func EqualPublicContract(left, right Publication) bool {
	return reflect.DeepEqual(publicationContract(left), publicationContract(right))
}

func publicationContract(value Publication) Publication {
	value = clonePublication(value)
	if value.Identity == nil {
		return value
	}
	for index := range value.Identity.UserFields {
		value.Identity.UserFields[index].boundSchema = nil
	}
	for providerIndex := range value.Identity.Providers {
		for operationIndex := range value.Identity.Providers[providerIndex].Operations {
			operation := &value.Identity.Providers[providerIndex].Operations[operationIndex]
			operation.boundInputSchema = nil
			operation.boundOutputSchema = nil
		}
	}
	return value
}

func (s *registryState) tombstoneValues() []Tombstone {
	result := make([]Tombstone, 0, len(s.tombstones))
	for _, value := range s.tombstones {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].ContractVersion < result[j].ContractVersion
	})
	return result
}

func snapshotFromState(state *registryState) Snapshot {
	snapshot := Snapshot{SchemaVersion: SchemaVersion, Revision: state.revision, SafeMode: state.safeMode}
	snapshot.Publications = publicationValues(state.publications)
	for _, value := range state.permissions {
		snapshot.Permissions = append(snapshot.Permissions, clonePermissionContribution(value))
	}
	for _, value := range state.userFields {
		snapshot.UserFields = append(snapshot.UserFields, cloneUserFieldContribution(value, true))
	}
	for _, value := range state.providers {
		snapshot.Providers = append(snapshot.Providers, cloneProviderContribution(value, true))
	}
	snapshot.Tombstones = state.tombstoneValues()
	sort.Slice(snapshot.Permissions, func(i, j int) bool { return snapshot.Permissions[i].Key < snapshot.Permissions[j].Key })
	sort.Slice(snapshot.UserFields, func(i, j int) bool { return snapshot.UserFields[i].ID < snapshot.UserFields[j].ID })
	sort.Slice(snapshot.Providers, func(i, j int) bool {
		if snapshot.Providers[i].Kind != snapshot.Providers[j].Kind {
			return snapshot.Providers[i].Kind < snapshot.Providers[j].Kind
		}
		if snapshot.Providers[i].Priority != snapshot.Providers[j].Priority {
			return snapshot.Providers[i].Priority > snapshot.Providers[j].Priority
		}
		return snapshot.Providers[i].ID < snapshot.Providers[j].ID
	})
	snapshot.Digest = state.digest
	return snapshot
}

func publicationValues(input map[string]Publication) []Publication {
	result := make([]Publication, 0, len(input))
	for _, publication := range input {
		result = append(result, clonePublication(publication))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Artifact.ExtensionID < result[j].Artifact.ExtensionID })
	return result
}

func clonePublicationMap(input map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(input))
	for id, publication := range input {
		result[id] = clonePublication(publication)
	}
	return result
}

func clonePublication(input Publication) Publication {
	result := input
	result.Permissions = append([]PermissionDefinition(nil), input.Permissions...)
	for index := range result.Permissions {
		result.Permissions[index].RecommendedRoles = append([]string(nil), input.Permissions[index].RecommendedRoles...)
	}
	if input.Identity != nil {
		identity := *input.Identity
		identity.UserFields = append([]UserField(nil), input.Identity.UserFields...)
		for index := range input.Identity.UserFields {
			identity.UserFields[index] = cloneUserField(input.Identity.UserFields[index])
		}
		identity.Providers = append([]Provider(nil), input.Identity.Providers...)
		for index := range input.Identity.Providers {
			identity.Providers[index] = cloneProvider(input.Identity.Providers[index])
		}
		identity.RiskHooks = append([]string(nil), input.Identity.RiskHooks...)
		result.Identity = &identity
	}
	return result
}

func cloneUserField(input UserField) UserField {
	result := input
	if input.boundSchema != nil {
		compiled := cloneCompiledIdentitySchema(*input.boundSchema)
		result.boundSchema = &compiled
	}
	return result
}

func publicUserField(input UserField) UserField {
	result := cloneUserField(input)
	result.boundSchema = nil
	return result
}

func cloneProvider(input Provider) Provider {
	result := input
	result.Operations = append([]ProviderOperation(nil), input.Operations...)
	for index := range input.Operations {
		result.Operations[index] = cloneProviderOperation(input.Operations[index])
	}
	return result
}

func publicProvider(input Provider) Provider {
	result := input
	result.Operations = append([]ProviderOperation(nil), input.Operations...)
	for index := range result.Operations {
		result.Operations[index].boundInputSchema = nil
		result.Operations[index].boundOutputSchema = nil
	}
	return result
}

func cloneProviderOperation(input ProviderOperation) ProviderOperation {
	result := input
	if input.boundInputSchema != nil {
		compiled := cloneCompiledIdentitySchema(*input.boundInputSchema)
		result.boundInputSchema = &compiled
	}
	if input.boundOutputSchema != nil {
		compiled := cloneCompiledIdentitySchema(*input.boundOutputSchema)
		result.boundOutputSchema = &compiled
	}
	return result
}

func cloneUserFieldContribution(input UserFieldContribution, public bool) UserFieldContribution {
	result := input
	result.UserField = cloneUserField(input.UserField)
	if public {
		result.UserField = publicUserField(result.UserField)
	}
	return result
}

func cloneProviderContribution(input ProviderContribution, public bool) ProviderContribution {
	result := input
	if public {
		result.Provider = publicProvider(input.Provider)
	} else {
		result.Provider = cloneProvider(input.Provider)
	}
	return result
}

func clonePermissionContribution(input PermissionContribution) PermissionContribution {
	result := input
	result.RecommendedRoles = append([]string(nil), input.RecommendedRoles...)
	return result
}
