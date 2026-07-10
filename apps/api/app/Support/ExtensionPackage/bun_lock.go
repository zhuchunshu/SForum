package extensionpackage

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/tailscale/hujson"
)

type lockedPackage struct {
	Dependency
	sets          dependencySets
	optionalPeers map[string]bool
}

func inspectBunLock(body []byte, packageSets dependencySets, hostPeers HostPeers) ([]Dependency, error) {
	standard, err := hujson.Standardize(body)
	if err != nil {
		return nil, invalidAdminFrontendf("standardize bun.lock: %v", err)
	}
	var lock struct {
		LockfileVersion int                        `json:"lockfileVersion"`
		ConfigVersion   int                        `json:"configVersion"`
		Workspaces      map[string]json.RawMessage `json:"workspaces"`
		Packages        map[string]json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal(standard, &lock); err != nil {
		return nil, invalidAdminFrontendf("decode bun.lock: %v", err)
	}
	if lock.LockfileVersion != 1 {
		return nil, invalidAdminFrontendf("unsupported bun.lock version %d", lock.LockfileVersion)
	}
	if lock.ConfigVersion != 1 {
		return nil, invalidAdminFrontendf("unsupported bun.lock config version %d", lock.ConfigVersion)
	}
	if len(lock.Workspaces) != 1 {
		return nil, invalidAdminFrontendf("bun.lock may contain only the root workspace")
	}
	rootRaw, exists := lock.Workspaces[""]
	if !exists {
		return nil, invalidAdminFrontendf("bun.lock root workspace is missing")
	}
	rootSets, err := decodeLockWorkspace(rootRaw)
	if err != nil {
		return nil, err
	}
	if !sameDependencySets(packageSets, rootSets) {
		return nil, invalidAdminFrontendf("bun.lock root workspace does not match package.json")
	}

	resolvedByIdentity := make(map[string]Dependency)
	resolvedByKey := make(map[string]lockedPackage, len(lock.Packages))
	for key, raw := range lock.Packages {
		locked, err := decodeBunPackageTuple(key, raw)
		if err != nil {
			return nil, err
		}
		dependency := locked.Dependency
		if hostVersionText, hostOwned := hostPeers[dependency.Name]; hostOwned {
			hostVersion, hostErr := semver.StrictNewVersion(hostVersionText)
			resolvedVersion, resolvedErr := semver.StrictNewVersion(dependency.Version)
			if hostErr != nil || resolvedErr != nil || !hostVersion.Equal(resolvedVersion) {
				return nil, invalidAdminFrontendf("bun.lock resolves competing host peer %s@%s", dependency.Name, dependency.Version)
			}
		}
		identity := dependency.Name + "\x00" + dependency.Version
		if current, exists := resolvedByIdentity[identity]; exists && current.Integrity != dependency.Integrity {
			return nil, invalidAdminFrontendf("bun.lock has conflicting integrity for %s@%s", dependency.Name, dependency.Version)
		}
		resolvedByIdentity[identity] = dependency
		resolvedByKey[key] = locked
	}
	for _, dependencies := range []map[string]string{packageSets.dependencies, packageSets.devDependencies, packageSets.optionalDependencies} {
		for name, rangeText := range dependencies {
			resolved, exists := resolvedByKey[name]
			if !exists || resolved.Dependency.Name != name {
				return nil, invalidAdminFrontendf("bun.lock has no pinned resolution for direct dependency %s", name)
			}
			constraint, err := semver.NewConstraint(rangeText)
			if err != nil {
				return nil, invalidAdminFrontendf("direct dependency %s has unsupported version range %q", name, rangeText)
			}
			version, err := semver.StrictNewVersion(resolved.Dependency.Version)
			if err != nil || !constraint.Check(version) {
				return nil, invalidAdminFrontendf("bun.lock resolution %s@%s does not satisfy %q", name, resolved.Dependency.Version, rangeText)
			}
		}
	}
	if err := validateBunDependencyGraph(resolvedByKey, hostPeers); err != nil {
		return nil, err
	}
	resolved := make([]Dependency, 0, len(resolvedByIdentity))
	for _, dependency := range resolvedByIdentity {
		resolved = append(resolved, dependency)
	}
	sortDependencies(resolved)
	return resolved, nil
}

func decodeLockWorkspace(raw json.RawMessage) (dependencySets, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return dependencySets{}, invalidAdminFrontendf("decode bun.lock root workspace: %v", err)
	}
	sets := dependencySets{}
	var err error
	sets.dependencies, err = decodeLockDependencyMap(object, "dependencies")
	if err != nil {
		return dependencySets{}, err
	}
	sets.devDependencies, err = decodeLockDependencyMap(object, "devDependencies")
	if err != nil {
		return dependencySets{}, err
	}
	sets.optionalDependencies, err = decodeLockDependencyMap(object, "optionalDependencies")
	if err != nil {
		return dependencySets{}, err
	}
	sets.peerDependencies, err = decodeLockDependencyMap(object, "peerDependencies")
	if err != nil {
		return dependencySets{}, err
	}
	return sets, nil
}

func decodeLockDependencyMap(object map[string]json.RawMessage, field string) (map[string]string, error) {
	result := map[string]string{}
	raw, exists := object[field]
	if !exists {
		return result, nil
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, invalidAdminFrontendf("bun.lock root %s is invalid", field)
	}
	return result, nil
}

func sameDependencySets(left dependencySets, right dependencySets) bool {
	return reflect.DeepEqual(left.dependencies, right.dependencies) &&
		reflect.DeepEqual(left.devDependencies, right.devDependencies) &&
		reflect.DeepEqual(left.optionalDependencies, right.optionalDependencies) &&
		reflect.DeepEqual(left.peerDependencies, right.peerDependencies)
}

func decodeBunPackageTuple(key string, raw json.RawMessage) (lockedPackage, error) {
	var tuple []json.RawMessage
	if err := json.Unmarshal(raw, &tuple); err != nil || len(tuple) < 3 {
		return lockedPackage{}, invalidAdminFrontendf("bun.lock package %s is not a supported tuple", key)
	}
	var resolution string
	var source string
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(tuple[0], &resolution); err != nil || strings.TrimSpace(resolution) == "" {
		return lockedPackage{}, invalidAdminFrontendf("bun.lock package %s has invalid resolution", key)
	}
	if err := json.Unmarshal(tuple[1], &source); err != nil {
		return lockedPackage{}, invalidAdminFrontendf("bun.lock package %s has invalid source", key)
	}
	if strings.TrimSpace(source) != "" {
		return lockedPackage{}, invalidAdminFrontendf("bun.lock package %s uses a non-registry source", key)
	}
	if err := json.Unmarshal(tuple[2], &metadata); err != nil || metadata == nil {
		return lockedPackage{}, invalidAdminFrontendf("bun.lock package %s has invalid metadata", key)
	}
	sets, optionalPeers, err := decodeBunPackageMetadata(key, metadata)
	if err != nil {
		return lockedPackage{}, err
	}
	name, version, ok := parsePinnedResolution(resolution)
	if !ok {
		return lockedPackage{}, invalidAdminFrontendf("bun.lock package %s is not pinned: %s", key, resolution)
	}
	integrity := ""
	if len(tuple) >= 4 {
		if err := json.Unmarshal(tuple[3], &integrity); err != nil {
			return lockedPackage{}, invalidAdminFrontendf("bun.lock package %s has invalid integrity", key)
		}
	}
	if !validSHA512Integrity(integrity) {
		return lockedPackage{}, invalidAdminFrontendf("bun.lock registry package %s has invalid sha512 integrity", key)
	}
	return lockedPackage{
		Dependency:    Dependency{Name: name, Version: version, Integrity: strings.TrimSpace(integrity)},
		sets:          sets,
		optionalPeers: optionalPeers,
	}, nil
}

func validSHA512Integrity(integrity string) bool {
	const prefix = "sha512-"
	integrity = strings.TrimSpace(integrity)
	if !strings.HasPrefix(integrity, prefix) {
		return false
	}
	digest, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(integrity, prefix))
	return err == nil && len(digest) == sha512.Size
}

func forbiddenLocalSource(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	if forbiddenDependencyProtocol(source) || strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") {
		return true
	}
	return len(source) >= 2 && source[1] == ':'
}

func decodeBunPackageMetadata(key string, metadata map[string]json.RawMessage) (dependencySets, map[string]bool, error) {
	sets := dependencySets{
		dependencies:         map[string]string{},
		devDependencies:      map[string]string{},
		optionalDependencies: map[string]string{},
		peerDependencies:     map[string]string{},
	}
	fields := []struct {
		name   string
		target *map[string]string
	}{
		{name: "dependencies", target: &sets.dependencies},
		{name: "devDependencies", target: &sets.devDependencies},
		{name: "optionalDependencies", target: &sets.optionalDependencies},
		{name: "peerDependencies", target: &sets.peerDependencies},
	}
	for _, field := range fields {
		raw, exists := metadata[field.name]
		if !exists {
			continue
		}
		var dependencies map[string]string
		if err := json.Unmarshal(raw, &dependencies); err != nil {
			return dependencySets{}, nil, invalidAdminFrontendf("bun.lock package %s has invalid %s metadata", key, field.name)
		}
		for name, version := range dependencies {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
				return dependencySets{}, nil, invalidAdminFrontendf("bun.lock package %s has an empty %s dependency", key, field.name)
			}
			if forbiddenDependencyProtocol(version) || forbiddenLocalSource(version) {
				return dependencySets{}, nil, invalidAdminFrontendf("bun.lock package %s dependency %s uses a local protocol", key, name)
			}
		}
		*field.target = dependencies
	}
	optionalPeers := map[string]bool{}
	if raw, exists := metadata["optionalPeers"]; exists {
		var names []string
		if err := json.Unmarshal(raw, &names); err != nil {
			return dependencySets{}, nil, invalidAdminFrontendf("bun.lock package %s has invalid optionalPeers metadata", key)
		}
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				return dependencySets{}, nil, invalidAdminFrontendf("bun.lock package %s has an empty optional peer", key)
			}
			optionalPeers[name] = true
		}
	}
	return sets, optionalPeers, nil
}

func validateBunDependencyGraph(packages map[string]lockedPackage, hostPeers HostPeers) error {
	for key, locked := range packages {
		for _, dependencies := range []map[string]string{locked.sets.dependencies, locked.sets.optionalDependencies} {
			for name, rangeText := range dependencies {
				if err := validateBunDependencyEdge(packages, key, name, rangeText); err != nil {
					return err
				}
			}
		}
		for name, rangeText := range locked.sets.peerDependencies {
			if hostVersion, hostOwned := hostPeers[name]; hostOwned {
				if err := validateVersionRange(name, rangeText, hostVersion, "host peer"); err != nil {
					return err
				}
				continue
			}
			_, exists, err := resolveBunDependency(packages, key, name)
			if err != nil {
				return err
			}
			if !exists && locked.optionalPeers[name] {
				continue
			}
			if err := validateBunDependencyEdge(packages, key, name, rangeText); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBunDependencyEdge(packages map[string]lockedPackage, parentKey string, name string, rangeText string) error {
	resolved, exists, err := resolveBunDependency(packages, parentKey, name)
	if err != nil {
		return err
	}
	if !exists {
		return invalidAdminFrontendf("bun.lock package %s has no resolution for dependency %s", parentKey, name)
	}
	return validateVersionRange(name, rangeText, resolved.Dependency.Version, "locked dependency")
}

func resolveBunDependency(packages map[string]lockedPackage, parentKey string, name string) (lockedPackage, bool, error) {
	chain, ok := bunPackageKeyChain(parentKey)
	if !ok {
		return lockedPackage{}, false, invalidAdminFrontendf("bun.lock has invalid package key %q", parentKey)
	}
	for length := len(chain); length >= 0; length-- {
		candidate := name
		if length > 0 {
			candidate = strings.Join(chain[:length], "/") + "/" + name
		}
		resolved, exists := packages[candidate]
		if !exists {
			continue
		}
		if resolved.Dependency.Name != name {
			return lockedPackage{}, false, invalidAdminFrontendf("bun.lock key %s resolves unexpected package %s", candidate, resolved.Dependency.Name)
		}
		return resolved, true, nil
	}
	return lockedPackage{}, false, nil
}

func bunPackageKeyChain(key string) ([]string, bool) {
	segments := strings.Split(strings.TrimSpace(key), "/")
	chain := make([]string, 0, len(segments))
	for index := 0; index < len(segments); index++ {
		segment := segments[index]
		if segment == "" {
			return nil, false
		}
		if strings.HasPrefix(segment, "@") {
			if index+1 >= len(segments) || segments[index+1] == "" {
				return nil, false
			}
			segment += "/" + segments[index+1]
			index++
		}
		chain = append(chain, segment)
	}
	return chain, len(chain) > 0
}

func validateVersionRange(name string, rangeText string, versionText string, source string) error {
	constraint, err := semver.NewConstraint(rangeText)
	if err != nil {
		return invalidAdminFrontendf("dependency %s has unsupported version range %q", name, rangeText)
	}
	version, err := semver.StrictNewVersion(versionText)
	if err != nil || !constraint.Check(version) {
		return invalidAdminFrontendf("%s %s@%s does not satisfy %q", source, name, versionText, rangeText)
	}
	return nil
}

func parsePinnedResolution(resolution string) (string, string, bool) {
	resolution = strings.TrimSpace(resolution)
	separator := strings.LastIndex(resolution, "@")
	if separator <= 0 || separator == len(resolution)-1 {
		return "", "", false
	}
	name := resolution[:separator]
	version := resolution[separator+1:]
	if strings.TrimSpace(name) != name || strings.TrimSpace(version) != version {
		return "", "", false
	}
	if _, err := semver.StrictNewVersion(version); err != nil {
		return "", "", false
	}
	if strings.HasPrefix(name, "@") && !strings.Contains(name, "/") {
		return "", "", false
	}
	return name, version, true
}
