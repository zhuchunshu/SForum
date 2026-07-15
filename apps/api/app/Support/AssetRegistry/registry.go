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
	SchemaVersion     = "sforum.asset-registry@1"
	maxRegistryAssets = 4096
	maxRegistryOwners = 512
)

var (
	ErrInvalid    = errors.New("asset registry declaration is invalid")
	ErrConflict   = errors.New("asset registry handle conflicts with the active snapshot")
	ErrDependency = errors.New("asset registry dependency is unavailable or cyclic")
	ErrNotFound   = errors.New("asset registry handle is not found")
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
	Artifact Artifact
	Assets   []Declaration
}

type PlanRequest struct {
	Handles       []string
	Scopes        []string
	IncludeGlobal bool
}

type Snapshot struct {
	SchemaVersion string  `json:"schemaVersion"`
	Revision      uint64  `json:"revision"`
	Assets        []Asset `json:"assets"`
}

type registryState struct {
	revision uint64
	assets   map[string]Asset
}

type Registry struct {
	mu    sync.Mutex
	state atomic.Pointer[registryState]
}

func New() *Registry {
	registry := &Registry{}
	registry.state.Store(&registryState{assets: map[string]Asset{}})
	return registry
}

// Publish atomically replaces one extension's complete asset publication.
func (r *Registry) Publish(publication Publication) (uint64, error) {
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
	next := make(map[string]Asset, len(current.assets)+len(normalized.Assets))
	for handle, asset := range current.assets {
		if asset.Artifact.ExtensionID != normalized.Artifact.ExtensionID {
			next[handle] = asset
		}
	}
	for _, declaration := range normalized.Assets {
		if _, exists := next[declaration.Handle]; exists {
			return current.revision, ErrConflict
		}
		next[declaration.Handle] = Asset{Declaration: declaration, Artifact: normalized.Artifact}
	}
	if len(next) > maxRegistryAssets {
		return current.revision, ErrInvalid
	}
	if err := validateGraph(next); err != nil {
		return current.revision, err
	}
	if equalAssetMaps(current.assets, next) {
		return current.revision, nil
	}
	revision := current.revision + 1
	r.state.Store(&registryState{revision: revision, assets: next})
	return revision, nil
}

// Remove revokes all handles owned by an extension in one snapshot swap. It
// refuses to publish a graph that would strand another extension's assets.
func (r *Registry) Remove(extensionID string) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	if !idPattern.MatchString(extensionID) {
		return r.load().revision, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	next := make(map[string]Asset, len(current.assets))
	removed := false
	for handle, asset := range current.assets {
		if asset.Artifact.ExtensionID == extensionID {
			removed = true
			continue
		}
		next[handle] = asset
	}
	if !removed {
		return current.revision, nil
	}
	if err := validateGraph(next); err != nil {
		return current.revision, err
	}
	revision := current.revision + 1
	r.state.Store(&registryState{revision: revision, assets: next})
	return revision, nil
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
	if len(request.Handles) > maxRegistryAssets || len(request.Scopes) > maxRegistryAssets {
		return nil, ErrInvalid
	}
	state := r.load()
	roots := map[string]struct{}{}
	for _, handle := range request.Handles {
		handle = strings.ToLower(strings.TrimSpace(handle))
		if _, ok := state.assets[handle]; !ok {
			return nil, ErrNotFound
		}
		roots[handle] = struct{}{}
	}
	scopes := stringSet(request.Scopes)
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
	handles := make([]string, 0, len(state.assets))
	for handle := range state.assets {
		handles = append(handles, handle)
	}
	sort.Strings(handles)
	assets := make([]Asset, 0, len(handles))
	for _, handle := range handles {
		assets = append(assets, cloneAsset(state.assets[handle]))
	}
	return Snapshot{SchemaVersion: SchemaVersion, Revision: state.revision, Assets: assets}
}

func (r *Registry) load() *registryState {
	if state := r.state.Load(); state != nil {
		return state
	}
	return &registryState{assets: map[string]Asset{}}
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
	asset.Dependencies = slices.Clone(asset.Dependencies)
	asset.Scope = slices.Clone(asset.Scope)
	asset.CSP = slices.Clone(asset.CSP)
	return asset
}
