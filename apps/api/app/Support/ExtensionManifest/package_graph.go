package extensionmanifest

import (
	"errors"
	"fmt"
	"sort"

	semver "github.com/Masterminds/semver/v3"
)

var (
	ErrDependencyMissing   = errors.New("extensions: required dependency missing")
	ErrDependencyConflict  = errors.New("extensions: dependency conflict")
	ErrDependencyAmbiguous = errors.New("extensions: dependency provider ambiguous")
	ErrDependencyCycle     = errors.New("extensions: dependency cycle")
	ErrDependencyVersion   = errors.New("extensions: dependency version mismatch")
	ErrManifestSetConflict = errors.New("extensions: manifest set registry conflict")
	ErrDuplicateExtension  = errors.New("extensions: duplicate extension id")
)

type PackageGraphNode struct {
	ID        string   `json:"id"`
	Version   string   `json:"version"`
	DependsOn []string `json:"dependsOn"`
	Provides  []string `json:"provides"`
}

type PackageGraph struct {
	Order     []string            `json:"order"`
	Nodes     []PackageGraphNode  `json:"nodes"`
	Providers map[string][]string `json:"providers"`
}

type capabilityProvider struct {
	extensionID string
	version     *semver.Version
}

// ResolvePackageGraph validates a complete desired package set and returns a
// stable provider-before-consumer activation order.
func ResolvePackageGraph(input []Manifest) (PackageGraph, error) {
	manifests := make(map[string]Manifest, len(input))
	versions := make(map[string]*semver.Version, len(input))
	providers := map[string][]capabilityProvider{}
	for _, raw := range input {
		manifest := Normalize(raw)
		if err := Validate(manifest); err != nil {
			return PackageGraph{}, err
		}
		if _, duplicate := manifests[manifest.ID]; duplicate {
			return PackageGraph{}, fmt.Errorf("%w: %s", ErrDuplicateExtension, manifest.ID)
		}
		version, err := semver.StrictNewVersion(manifest.Version)
		if err != nil && len(manifest.Dependencies) > 0 {
			return PackageGraph{}, fmt.Errorf("%w: %s@%s", ErrDependencyVersion, manifest.ID, manifest.Version)
		}
		manifests[manifest.ID] = manifest
		versions[manifest.ID] = version
	}
	manifestIDs := sortedManifestIDs(manifests)
	for _, id := range manifestIDs {
		manifest := manifests[id]
		for _, dependency := range manifest.Dependencies {
			if dependency.Kind != "provides" {
				continue
			}
			providedVersion, _ := semver.StrictNewVersion(dependency.Version)
			providers[dependency.Capability] = append(providers[dependency.Capability], capabilityProvider{
				extensionID: manifest.ID,
				version:     providedVersion,
			})
		}
	}
	if err := validateManifestSetConflicts(manifests); err != nil {
		return PackageGraph{}, err
	}

	edges := make(map[string]map[string]bool, len(manifests))
	dependencies := make(map[string]map[string]bool, len(manifests))
	for _, id := range manifestIDs {
		edges[id] = map[string]bool{}
		dependencies[id] = map[string]bool{}
	}
	for _, id := range manifestIDs {
		manifest := manifests[id]
		for _, dependency := range manifest.Dependencies {
			switch dependency.Kind {
			case "required", "optional":
				providerID, found, err := resolveDependencyProvider(dependency, manifests, versions, providers)
				if err != nil {
					return PackageGraph{}, fmt.Errorf("%s: %w", manifest.ID, err)
				}
				if !found {
					if dependency.Kind == "required" {
						return PackageGraph{}, fmt.Errorf("%w: %s", ErrDependencyMissing, dependencyTarget(dependency))
					}
					continue
				}
				if providerID != manifest.ID {
					edges[providerID][manifest.ID] = true
					dependencies[manifest.ID][providerID] = true
				}
			case "conflict":
				_, found, err := resolveConflict(dependency, manifests, versions, providers)
				if err != nil {
					return PackageGraph{}, err
				}
				if found {
					return PackageGraph{}, fmt.Errorf("%w: %s conflicts with %s", ErrDependencyConflict, manifest.ID, dependencyTarget(dependency))
				}
			}
		}
	}

	order, err := stableTopologicalOrder(edges, dependencies)
	if err != nil {
		return PackageGraph{}, err
	}
	providerIndex := make(map[string][]string, len(providers))
	for capability, candidates := range providers {
		for _, candidate := range candidates {
			providerIndex[capability] = append(providerIndex[capability], candidate.extensionID)
		}
		sort.Strings(providerIndex[capability])
	}
	nodes := make([]PackageGraphNode, 0, len(order))
	for _, id := range order {
		dependsOn := sortedKeys(dependencies[id])
		provided := make([]string, 0)
		for capability, candidates := range providers {
			for _, candidate := range candidates {
				if candidate.extensionID == id {
					provided = append(provided, capability)
				}
			}
		}
		sort.Strings(provided)
		nodes = append(nodes, PackageGraphNode{ID: id, Version: manifests[id].Version, DependsOn: dependsOn, Provides: provided})
	}
	return PackageGraph{Order: order, Nodes: nodes, Providers: providerIndex}, nil
}

func resolveDependencyProvider(dependency ManifestDependency, manifests map[string]Manifest, versions map[string]*semver.Version, providers map[string][]capabilityProvider) (string, bool, error) {
	constraint, _ := semver.NewConstraint(dependency.Version)
	if dependency.ID != "" {
		_, found := manifests[dependency.ID]
		if !found {
			return "", false, nil
		}
		version := versions[dependency.ID]
		if version == nil || !constraint.Check(version) {
			if dependency.Kind == "optional" {
				return "", false, nil
			}
			return "", false, fmt.Errorf("%w: %s requires %s", ErrDependencyVersion, dependency.ID, dependency.Version)
		}
		return dependency.ID, true, nil
	}
	matches := matchingCapabilityProviders(providers[dependency.Capability], constraint)
	if len(matches) == 0 {
		return "", false, nil
	}
	if len(matches) > 1 {
		return "", false, fmt.Errorf("%w: %s", ErrDependencyAmbiguous, dependency.Capability)
	}
	return matches[0], true, nil
}

func resolveConflict(dependency ManifestDependency, manifests map[string]Manifest, versions map[string]*semver.Version, providers map[string][]capabilityProvider) (string, bool, error) {
	constraint, _ := semver.NewConstraint(dependency.Version)
	if dependency.ID != "" {
		_, found := manifests[dependency.ID]
		if !found || versions[dependency.ID] == nil || !constraint.Check(versions[dependency.ID]) {
			return "", false, nil
		}
		return dependency.ID, true, nil
	}
	matches := matchingCapabilityProviders(providers[dependency.Capability], constraint)
	if len(matches) == 0 {
		return "", false, nil
	}
	return matches[0], true, nil
}

func matchingCapabilityProviders(candidates []capabilityProvider, constraint *semver.Constraints) []string {
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.version != nil && constraint.Check(candidate.version) {
			result = append(result, candidate.extensionID)
		}
	}
	sort.Strings(result)
	return result
}

func stableTopologicalOrder(edges map[string]map[string]bool, dependencies map[string]map[string]bool) ([]string, error) {
	indegree := make(map[string]int, len(dependencies))
	ready := make([]string, 0, len(dependencies))
	for id, items := range dependencies {
		indegree[id] = len(items)
		if len(items) == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	order := make([]string, 0, len(dependencies))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		for _, consumer := range sortedKeys(edges[id]) {
			indegree[consumer]--
			if indegree[consumer] == 0 {
				ready = append(ready, consumer)
				sort.Strings(ready)
			}
		}
	}
	if len(order) != len(dependencies) {
		remaining := make([]string, 0)
		for id, degree := range indegree {
			if degree > 0 {
				remaining = append(remaining, id)
			}
		}
		sort.Strings(remaining)
		return nil, fmt.Errorf("%w: %v", ErrDependencyCycle, remaining)
	}
	return order, nil
}

func validateManifestSetConflicts(manifests map[string]Manifest) error {
	claims := map[string]string{}
	claim := func(key string, extensionID string) error {
		if previous, exists := claims[key]; exists && previous != extensionID {
			return fmt.Errorf("%w: %s and %s claim %s", ErrManifestSetConflict, previous, extensionID, key)
		}
		claims[key] = extensionID
		return nil
	}
	for _, extensionID := range sortedManifestIDs(manifests) {
		manifest := manifests[extensionID]
		for _, route := range manifest.Routes {
			if route.Action != RouteActionReplace {
				continue
			}
			for _, method := range route.Methods {
				if err := claim("route\x00"+method+"\x00"+route.TargetID, extensionID); err != nil {
					return err
				}
			}
		}
		for _, component := range manifest.Components {
			if component.Action == ComponentActionReplace {
				if err := claim("component\x00"+component.TargetID, extensionID); err != nil {
					return err
				}
			}
		}
		for _, service := range manifest.Services {
			if service.Action == "replace" {
				if err := claim("service\x00"+service.TargetID, extensionID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func sortedManifestIDs(manifests map[string]Manifest) []string {
	result := make([]string, 0, len(manifests))
	for id := range manifests {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func dependencyTarget(dependency ManifestDependency) string {
	if dependency.ID != "" {
		return dependency.ID
	}
	return dependency.Capability
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
