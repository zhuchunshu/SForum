package navigationregistry

import (
	"fmt"
	"sort"

	semver "github.com/Masterminds/semver/v3"
)

type dependencyResolution struct {
	authorized                   map[string]map[string]bool
	optionalByConsumer           map[string][]Dependency
	optionalCapabilityOwners     map[string]map[string]bool
	unresolvedOptionalCapability map[string]bool
	activeExtensionIDs           map[string]bool
}

type capabilityProvider struct {
	extensionID string
	version     *semver.Version
}

func resolveDependencies(publications []Publication) (dependencyResolution, error) {
	result := dependencyResolution{
		authorized:                   map[string]map[string]bool{},
		optionalByConsumer:           map[string][]Dependency{},
		optionalCapabilityOwners:     map[string]map[string]bool{},
		unresolvedOptionalCapability: map[string]bool{},
		activeExtensionIDs:           map[string]bool{},
	}
	byID := map[string]Publication{}
	versions := map[string]*semver.Version{}
	providers := map[string][]capabilityProvider{}
	for _, publication := range publications {
		id := publication.Artifact.ExtensionID
		byID[id] = publication
		versions[id], _ = semver.StrictNewVersion(publication.Artifact.ExtensionVersion)
		result.authorized[id] = map[string]bool{}
		result.optionalCapabilityOwners[id] = map[string]bool{}
		result.activeExtensionIDs[id] = true
		for _, dependency := range publication.Dependencies {
			if dependency.Kind == DependencyProvides {
				version, _ := semver.StrictNewVersion(dependency.Version)
				providers[dependency.Capability] = append(providers[dependency.Capability], capabilityProvider{id, version})
			}
		}
	}
	edges := map[string]map[string]bool{}
	for id := range byID {
		edges[id] = map[string]bool{}
	}
	for _, publication := range publications {
		consumer := publication.Artifact.ExtensionID
		for _, dependency := range publication.Dependencies {
			if dependency.Kind == DependencyProvides {
				continue
			}
			providerID, found, err := resolveDependencyProvider(dependency, byID, versions, providers)
			if err != nil {
				return dependencyResolution{}, fmt.Errorf("%w: %s: %v", ErrDependency, consumer, err)
			}
			switch dependency.Kind {
			case DependencyConflict:
				if found {
					return dependencyResolution{}, fmt.Errorf("%w: %s conflicts with %s", ErrConflict, consumer, providerID)
				}
			case DependencyRequired:
				if !found {
					return dependencyResolution{}, fmt.Errorf("%w: %s requires %s", ErrDependency, consumer, dependencyTarget(dependency))
				}
				result.authorized[consumer][providerID] = true
				edges[providerID][consumer] = true
			case DependencyOptional:
				result.optionalByConsumer[consumer] = append(result.optionalByConsumer[consumer], dependency)
				for _, provider := range providers[dependency.Capability] {
					result.optionalCapabilityOwners[consumer][provider.extensionID] = true
				}
				if found {
					result.authorized[consumer][providerID] = true
					edges[providerID][consumer] = true
				} else if dependency.Capability != "" {
					result.unresolvedOptionalCapability[consumer] = true
				}
			}
		}
	}
	if dependencyCycle(edges) {
		return dependencyResolution{}, ErrDependency
	}
	return result, nil
}

func resolveDependencyProvider(
	dependency Dependency,
	byID map[string]Publication,
	versions map[string]*semver.Version,
	providers map[string][]capabilityProvider,
) (string, bool, error) {
	constraint, _ := semver.NewConstraint(dependency.Version)
	if dependency.ExtensionID != "" {
		_, found := byID[dependency.ExtensionID]
		if !found || !constraint.Check(versions[dependency.ExtensionID]) {
			return "", false, nil
		}
		return dependency.ExtensionID, true, nil
	}
	matches := make([]string, 0)
	for _, provider := range providers[dependency.Capability] {
		if constraint.Check(provider.version) {
			matches = append(matches, provider.extensionID)
		}
	}
	sort.Strings(matches)
	if len(matches) > 1 && dependency.Kind != DependencyConflict {
		return "", false, fmt.Errorf("capability %s has ambiguous providers %v", dependency.Capability, matches)
	}
	if len(matches) == 0 {
		return "", false, nil
	}
	return matches[0], true, nil
}

func dependencyCycle(edges map[string]map[string]bool) bool {
	indegree := map[string]int{}
	for id := range edges {
		indegree[id] = 0
	}
	for _, consumers := range edges {
		for consumer := range consumers {
			indegree[consumer]++
		}
	}
	ready := make([]string, 0)
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	visited := 0
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		visited++
		for consumer := range edges[id] {
			indegree[consumer]--
			if indegree[consumer] == 0 {
				ready = append(ready, consumer)
				sort.Strings(ready)
			}
		}
	}
	return visited != len(indegree)
}

func dependencyTarget(dependency Dependency) string {
	if dependency.ExtensionID != "" {
		return dependency.ExtensionID
	}
	return dependency.Capability
}
