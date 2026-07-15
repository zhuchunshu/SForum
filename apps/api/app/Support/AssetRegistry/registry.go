package assetregistry

import (
	"errors"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	SchemaVersion = "sforum.asset-registry@1"
)

var (
	ErrInvalid          = errors.New("asset registry declaration is invalid")
	ErrConflict         = errors.New("asset registry handle conflicts with the active snapshot")
	ErrDependency       = errors.New("asset registry dependency is unavailable or cyclic")
	ErrNotFound         = errors.New("asset registry handle is not found")
	ErrArtifactConflict = errors.New("asset registry artifact does not own the active publication")
)

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,120}$`)
	digestPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	contractPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
)

type Artifact struct {
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	ImpactDigest     string `json:"impactDigest"`
	Core             bool   `json:"core,omitempty"`
}

type Declaration struct {
	Handle          string   `json:"handle"`
	ContractVersion string   `json:"contractVersion"`
	Type            string   `json:"type"`
	Path            string   `json:"path"`
	Digest          string   `json:"digest"`
	Dependencies    []string `json:"dependencies,omitempty"`
	Scope           []string `json:"scope,omitempty"`
	Module          bool     `json:"module,omitempty"`
	Loading         string   `json:"loading,omitempty"`
	Integrity       string   `json:"integrity"`
	CSP             []string `json:"csp,omitempty"`
}

type Asset struct {
	Declaration
	Artifact Artifact `json:"artifact"`
}

type Publication struct {
	Artifact Artifact      `json:"artifact"`
	Assets   []Declaration `json:"assets"`
}

type PlanRequest struct {
	Handles       []string
	Scopes        []string
	IncludeGlobal bool
}

type Snapshot struct {
	SchemaVersion string        `json:"schemaVersion"`
	Revision      uint64        `json:"revision"`
	Digest        string        `json:"digest"`
	Publications  []Publication `json:"publications"`
	Assets        []Asset       `json:"assets"`
}

type registryState struct {
	revision     uint64
	digest       string
	publications map[string]Publication
	assets       map[string]Asset
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
		return 0, err
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

// Remove revokes one exact artifact publication in one snapshot swap. A stale
// shutdown cannot remove a newer package, version, or trust-impact publication.
// Required consumers keep the removal fail-closed until they are removed too.
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

func (r *Registry) Resolve(handle string) (Asset, bool) {
	if r == nil {
		return Asset{}, false
	}
	asset, ok := r.load().assets[strings.ToLower(strings.TrimSpace(handle))]
	if !ok {
		return Asset{}, false
	}
	return cloneAsset(asset), true
}

// Plan returns a deterministic dependency-first, de-duplicated loading plan.
func (r *Registry) Plan(request PlanRequest) ([]Asset, error) {
	if r == nil {
		return nil, ErrInvalid
	}
	handles, err := strictPlanIDs(request.Handles, maxPlanHandles)
	if err != nil {
		return nil, ErrInvalid
	}
	scopeValues, err := strictPlanIDs(request.Scopes, maxPlanScopes)
	if err != nil {
		return nil, ErrInvalid
	}
	scopes := stringSet(scopeValues)
	state := r.load()
	roots := map[string]struct{}{}
	for _, handle := range handles {
		asset, ok := state.assets[handle]
		if !ok {
			return nil, ErrNotFound
		}
		// 显式 handle 可以单独解析；一旦调用方同时声明页面/组件 scope，
		// scoped asset 就不能借显式 handle 绕过其声明的适用范围。
		if len(scopes) > 0 && len(asset.Scope) > 0 && !intersects(asset.Scope, scopes) {
			return nil, ErrInvalid
		}
		roots[handle] = struct{}{}
	}
	for handle, asset := range state.assets {
		if len(asset.Scope) == 0 {
			if request.IncludeGlobal {
				roots[handle] = struct{}{}
			}
			continue
		}
		if intersects(asset.Scope, scopes) {
			roots[handle] = struct{}{}
		}
	}

	orderedRoots := sortedKeys(roots)
	visiting := map[string]bool{}
	visited := map[string]bool{}
	result := make([]Asset, 0, len(roots))
	var visit func(string) error
	visit = func(handle string) error {
		if visited[handle] {
			return nil
		}
		if visiting[handle] {
			return ErrDependency
		}
		asset, ok := state.assets[handle]
		if !ok {
			if strings.HasPrefix(handle, "core.asset.") {
				return nil
			}
			return ErrDependency
		}
		visiting[handle] = true
		for _, dependency := range asset.Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		delete(visiting, handle)
		visited[handle] = true
		result = append(result, cloneAsset(asset))
		return nil
	}
	for _, handle := range orderedRoots {
		if err := visit(handle); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *Registry) Snapshot() Snapshot {
	state := r.load()
	publications := publicationValues(state.publications)
	sort.Slice(publications, func(i, j int) bool {
		return publications[i].Artifact.ExtensionID < publications[j].Artifact.ExtensionID
	})
	handles := make([]string, 0, len(state.assets))
	for handle := range state.assets {
		handles = append(handles, handle)
	}
	sort.Strings(handles)
	assets := make([]Asset, 0, len(handles))
	for _, handle := range handles {
		assets = append(assets, cloneAsset(state.assets[handle]))
	}
	return Snapshot{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest,
		Publications: publications, Assets: assets,
	}
}

func (r *Registry) load() *registryState {
	if r != nil {
		if state := r.state.Load(); state != nil {
			return state
		}
	}
	return emptyState()
}

func emptyState() *registryState {
	return &registryState{
		digest: computeGraphDigest(nil), publications: map[string]Publication{}, assets: map[string]Asset{},
	}
}

func stringSet(values []string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func intersects(values []string, set map[string]struct{}) bool {
	for _, value := range values {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func cloneAsset(asset Asset) Asset {
	asset.Declaration = cloneDeclaration(asset.Declaration)
	return asset
}

func cloneDeclaration(declaration Declaration) Declaration {
	declaration.Dependencies = slices.Clone(declaration.Dependencies)
	declaration.Scope = slices.Clone(declaration.Scope)
	declaration.CSP = slices.Clone(declaration.CSP)
	return declaration
}

func clonePublication(publication Publication) Publication {
	publication.Assets = slices.Clone(publication.Assets)
	for index := range publication.Assets {
		publication.Assets[index] = cloneDeclaration(publication.Assets[index])
	}
	return publication
}

func clonePublicationMap(publications map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(publications))
	for extensionID, publication := range publications {
		result[extensionID] = clonePublication(publication)
	}
	return result
}

func publicationValues(publications map[string]Publication) []Publication {
	result := make([]Publication, 0, len(publications))
	for _, publication := range publications {
		result = append(result, clonePublication(publication))
	}
	return result
}
