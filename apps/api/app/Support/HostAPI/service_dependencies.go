package hostapi

import (
	"fmt"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

const (
	ServiceDependencyRequired = "required"
	ServiceDependencyOptional = "optional"
)

// ServiceDependency is the exact runtime copy of one callable manifest edge.
// Conflict/provides declarations are represented separately because they do
// not grant a caller permission to invoke a provider.
type ServiceDependency struct {
	ExtensionID       string
	Capability        string
	VersionConstraint string
	Kind              string
}

type ServiceCapability struct {
	ID      string
	Version string
}

// ServiceRuntimePublication atomically binds discovery registrations to the
// caller/provider identity and dependency graph frozen at process startup.
type ServiceRuntimePublication struct {
	ExtensionID      string
	ExtensionVersion string
	ArtifactDigest   string
	TrustGrantID     string
	RuntimeEpoch     uint64
	InstanceID       string
	Dependencies     []ServiceDependency
	Provides         []ServiceCapability
	Registrations    []ServiceRegistration
}

type preparedServiceDependency struct {
	dependency ServiceDependency
	constraint *semver.Constraints
}

type preparedServiceCapability struct {
	capability ServiceCapability
	version    *semver.Version
}

type preparedServiceRuntime struct {
	publication  ServiceRuntimePublication
	version      *semver.Version
	dependencies []preparedServiceDependency
	provides     []preparedServiceCapability
}

type ServiceDependencyDecision struct {
	Allowed           bool
	Reason            string
	DependencyTarget  string
	VersionConstraint string
	ProviderVersion   string
	Candidates        []string
}

// ReplaceRuntime validates and atomically publishes one runtime's identity,
// dependency contract, capabilities, and complete service set.
func (r *ServiceRegistry) ReplaceRuntime(publication ServiceRuntimePublication) error {
	if r == nil {
		return fmt.Errorf("%w: registry is nil", ErrInvalidServiceRegistration)
	}
	preparedRuntime, preparedServices, err := prepareServiceRuntimePublication(publication)
	if err != nil {
		return err
	}

	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	current := r.loadSnapshot()
	next := make([]preparedServiceRegistration, 0, len(current.registrations)+len(preparedServices))
	for _, item := range current.registrations {
		if item.registration.ExtensionID != preparedRuntime.publication.ExtensionID {
			next = append(next, item)
		}
	}
	next = append(next, preparedServices...)
	sortPreparedServices(next)
	runtimes := clonePreparedServiceRuntimes(current.runtimes)
	runtimes[preparedRuntime.publication.ExtensionID] = preparedRuntime
	r.snapshot.Store(&serviceRegistrySnapshot{
		revision: current.revision + 1, registrations: next, runtimes: runtimes, dependencyEnforced: true,
	})
	return nil
}

func prepareServiceRuntimePublication(publication ServiceRuntimePublication) (preparedServiceRuntime, []preparedServiceRegistration, error) {
	publication.ExtensionID = strings.TrimSpace(publication.ExtensionID)
	publication.ExtensionVersion = strings.TrimSpace(publication.ExtensionVersion)
	publication.ArtifactDigest = strings.TrimSpace(publication.ArtifactDigest)
	publication.TrustGrantID = strings.TrimSpace(publication.TrustGrantID)
	publication.InstanceID = strings.TrimSpace(publication.InstanceID)
	if publication.ExtensionID == "" || publication.ArtifactDigest == "" || publication.TrustGrantID == "" ||
		publication.RuntimeEpoch == 0 || publication.InstanceID == "" {
		return preparedServiceRuntime{}, nil, fmt.Errorf("%w: exact runtime identity is required", ErrInvalidServiceRegistration)
	}
	version, err := semver.StrictNewVersion(publication.ExtensionVersion)
	if err != nil {
		return preparedServiceRuntime{}, nil, fmt.Errorf("%w: extension version %q is not strict SemVer", ErrInvalidServiceRegistration, publication.ExtensionVersion)
	}

	prepared := preparedServiceRuntime{publication: publication, version: version}
	seenDependencies := make(map[string]struct{}, len(publication.Dependencies))
	for index, raw := range publication.Dependencies {
		dependency := ServiceDependency{
			ExtensionID: strings.TrimSpace(raw.ExtensionID), Capability: strings.TrimSpace(raw.Capability),
			VersionConstraint: strings.TrimSpace(raw.VersionConstraint), Kind: strings.ToLower(strings.TrimSpace(raw.Kind)),
		}
		if (dependency.ExtensionID == "") == (dependency.Capability == "") ||
			(dependency.Kind != ServiceDependencyRequired && dependency.Kind != ServiceDependencyOptional) {
			return preparedServiceRuntime{}, nil, fmt.Errorf("%w: dependency %d must declare one target and required/optional kind", ErrInvalidServiceRegistration, index)
		}
		constraint, constraintErr := parseStrictServiceConstraint(dependency.VersionConstraint)
		if constraintErr != nil || constraint == nil {
			return preparedServiceRuntime{}, nil, fmt.Errorf("%w: dependency %d version constraint is invalid", ErrInvalidServiceRegistration, index)
		}
		key := dependency.Kind + "\x00" + dependency.ExtensionID + "\x00" + dependency.Capability
		if _, exists := seenDependencies[key]; exists {
			return preparedServiceRuntime{}, nil, fmt.Errorf("%w: duplicate runtime dependency %q", ErrInvalidServiceRegistration, dependencyTargetName(dependency))
		}
		seenDependencies[key] = struct{}{}
		prepared.dependencies = append(prepared.dependencies, preparedServiceDependency{dependency: dependency, constraint: constraint})
	}

	seenCapabilities := make(map[string]struct{}, len(publication.Provides))
	for index, raw := range publication.Provides {
		capability := ServiceCapability{ID: strings.TrimSpace(raw.ID), Version: strings.TrimSpace(raw.Version)}
		providedVersion, versionErr := semver.StrictNewVersion(capability.Version)
		if capability.ID == "" || versionErr != nil {
			return preparedServiceRuntime{}, nil, fmt.Errorf("%w: provided capability %d is invalid", ErrInvalidServiceRegistration, index)
		}
		if _, exists := seenCapabilities[capability.ID]; exists {
			return preparedServiceRuntime{}, nil, fmt.Errorf("%w: duplicate provided capability %q", ErrInvalidServiceRegistration, capability.ID)
		}
		seenCapabilities[capability.ID] = struct{}{}
		prepared.provides = append(prepared.provides, preparedServiceCapability{capability: capability, version: providedVersion})
	}

	services := make([]preparedServiceRegistration, 0, len(publication.Registrations))
	seenServices := make(map[string]struct{}, len(publication.Registrations))
	for _, registration := range publication.Registrations {
		if registration.InstanceID != "" && strings.TrimSpace(registration.InstanceID) != publication.InstanceID {
			return preparedServiceRuntime{}, nil, fmt.Errorf("%w: service runtime instance does not match publication", ErrInvalidServiceRegistration)
		}
		registration.InstanceID = publication.InstanceID
		item, serviceErr := prepareServiceRegistration(publication.ExtensionID, registration)
		if serviceErr != nil {
			return preparedServiceRuntime{}, nil, serviceErr
		}
		key := item.publishedID + "\x00" + item.registration.Descriptor.GetVersion()
		if _, exists := seenServices[key]; exists {
			return preparedServiceRuntime{}, nil, fmt.Errorf("%w: extension %q declares %s@%s more than once", ErrInvalidServiceRegistration, publication.ExtensionID, item.publishedID, item.registration.Descriptor.GetVersion())
		}
		seenServices[key] = struct{}{}
		services = append(services, item)
	}
	prepared.publication = cloneServiceRuntimePublication(publication)
	return prepared, services, nil
}

func (s ResolvedService) AuthorizeDependency(caller ServiceCaller) ServiceDependencyDecision {
	if s.snapshot == nil || !s.snapshot.dependencyEnforced {
		return ServiceDependencyDecision{Allowed: true}
	}
	if !caller.Attested {
		return ServiceDependencyDecision{Reason: "caller_unattested"}
	}
	callerRuntime, ok := s.snapshot.runtimes[caller.ExtensionID]
	if !ok || !serviceCallerMatchesRuntime(caller, callerRuntime.publication) {
		return ServiceDependencyDecision{Reason: "caller_stale"}
	}
	providerRuntime, ok := s.snapshot.runtimes[s.Winner.ExtensionID]
	if !ok || providerRuntime.publication.InstanceID != s.Winner.InstanceID {
		return ServiceDependencyDecision{Reason: "provider_stale"}
	}
	if caller.ExtensionID == providerRuntime.publication.ExtensionID {
		return ServiceDependencyDecision{Allowed: true}
	}

	var versionMismatch *ServiceDependencyDecision
	for _, item := range callerRuntime.dependencies {
		dependency := item.dependency
		if dependency.ExtensionID != "" {
			if dependency.ExtensionID != providerRuntime.publication.ExtensionID {
				continue
			}
			if item.constraint.Check(providerRuntime.version) {
				return ServiceDependencyDecision{Allowed: true, DependencyTarget: dependency.ExtensionID, VersionConstraint: dependency.VersionConstraint, ProviderVersion: providerRuntime.publication.ExtensionVersion}
			}
			decision := ServiceDependencyDecision{Reason: "version_mismatch", DependencyTarget: dependency.ExtensionID, VersionConstraint: dependency.VersionConstraint, ProviderVersion: providerRuntime.publication.ExtensionVersion}
			versionMismatch = &decision
			continue
		}
		candidates := matchingRuntimeCapabilityProviders(s.snapshot, dependency.Capability, item.constraint)
		if len(candidates) == 0 {
			providers := runtimeCapabilityProviders(s.snapshot, dependency.Capability)
			if len(providers) == 0 {
				return ServiceDependencyDecision{Reason: "missing", DependencyTarget: dependency.Capability, VersionConstraint: dependency.VersionConstraint}
			}
			return ServiceDependencyDecision{Reason: "version_mismatch", DependencyTarget: dependency.Capability, VersionConstraint: dependency.VersionConstraint, Candidates: providers}
		}
		if len(candidates) > 1 {
			return ServiceDependencyDecision{Reason: "ambiguous", DependencyTarget: dependency.Capability, VersionConstraint: dependency.VersionConstraint, Candidates: candidates}
		}
		if len(candidates) == 1 && candidates[0] == providerRuntime.publication.ExtensionID {
			return ServiceDependencyDecision{Allowed: true, DependencyTarget: dependency.Capability, VersionConstraint: dependency.VersionConstraint, ProviderVersion: providedCapabilityVersion(providerRuntime, dependency.Capability)}
		}
	}
	if versionMismatch != nil {
		return *versionMismatch
	}
	return ServiceDependencyDecision{Reason: "undeclared"}
}

func runtimeCapabilityProviders(snapshot *serviceRegistrySnapshot, capability string) []string {
	providers := make([]string, 0)
	for extensionID, runtime := range snapshot.runtimes {
		for _, provided := range runtime.provides {
			if provided.capability.ID == capability {
				providers = append(providers, extensionID)
				break
			}
		}
	}
	sort.Strings(providers)
	return providers
}

func matchingRuntimeCapabilityProviders(snapshot *serviceRegistrySnapshot, capability string, constraint *semver.Constraints) []string {
	providers := make([]string, 0)
	for extensionID, runtime := range snapshot.runtimes {
		for _, provided := range runtime.provides {
			if provided.capability.ID == capability && constraint.Check(provided.version) {
				providers = append(providers, extensionID)
				break
			}
		}
	}
	sort.Strings(providers)
	return providers
}

func providedCapabilityVersion(runtime preparedServiceRuntime, capability string) string {
	for _, provided := range runtime.provides {
		if provided.capability.ID == capability {
			return provided.capability.Version
		}
	}
	return ""
}

func serviceCallerMatchesRuntime(caller ServiceCaller, runtime ServiceRuntimePublication) bool {
	return caller.ExtensionID == runtime.ExtensionID && caller.ExtensionVersion == runtime.ExtensionVersion &&
		caller.ArtifactDigest == runtime.ArtifactDigest && caller.TrustGrantID == runtime.TrustGrantID &&
		caller.RuntimeEpoch == runtime.RuntimeEpoch && caller.InstanceID == runtime.InstanceID
}

func dependencyTargetName(dependency ServiceDependency) string {
	if dependency.ExtensionID != "" {
		return dependency.ExtensionID
	}
	return dependency.Capability
}

func clonePreparedServiceRuntimes(values map[string]preparedServiceRuntime) map[string]preparedServiceRuntime {
	result := make(map[string]preparedServiceRuntime, len(values))
	for key, value := range values {
		value.publication = cloneServiceRuntimePublication(value.publication)
		value.dependencies = append([]preparedServiceDependency(nil), value.dependencies...)
		value.provides = append([]preparedServiceCapability(nil), value.provides...)
		result[key] = value
	}
	return result
}

func cloneServiceRuntimePublication(value ServiceRuntimePublication) ServiceRuntimePublication {
	value.Dependencies = append([]ServiceDependency(nil), value.Dependencies...)
	value.Provides = append([]ServiceCapability(nil), value.Provides...)
	value.Registrations = make([]ServiceRegistration, 0, len(value.Registrations))
	for _, registration := range value.Registrations {
		value.Registrations = append(value.Registrations, cloneServiceRegistration(registration))
	}
	return value
}
