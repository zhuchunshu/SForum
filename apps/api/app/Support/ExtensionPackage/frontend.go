package extensionpackage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

var ErrInvalidAdminFrontend = errors.New("extension package: invalid admin frontend")

type HostPeers map[string]string

type FrontendInspectInput struct {
	PackageRoot string
	Root        string
	Components  map[string]string
	Locales     map[string]string
	HostPeers   HostPeers
}

type Dependency struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Integrity string `json:"integrity,omitempty"`
}

type DependencySummary struct {
	Direct     []Dependency `json:"direct"`
	Resolved   []Dependency `json:"resolved"`
	LockDigest string       `json:"lockDigest"`
}

type dependencySets struct {
	dependencies         map[string]string
	devDependencies      map[string]string
	optionalDependencies map[string]string
	peerDependencies     map[string]string
}

var forbiddenPackageFields = []string{
	"workspaces",
	"trustedDependencies",
	"patchedDependencies",
	"nuxt",
	"vite",
	"nitro",
	"routeRules",
	"csp",
	"contentSecurityPolicy",
	"security",
	"server",
	"plugins",
	"modules",
	"middleware",
	"headers",
}

var forbiddenLifecycleScripts = map[string]struct{}{
	"preinstall":     {},
	"install":        {},
	"postinstall":    {},
	"prepare":        {},
	"prepublish":     {},
	"prepublishonly": {},
	"prepack":        {},
	"postpack":       {},
	"prebuild":       {},
	"build":          {},
	"postbuild":      {},
}

var requiredHostPeers = []string{
	"vue",
	"nuxt",
	"@nuxt/ui",
	"vue-router",
	"@sforum/admin-sdk",
}

// InspectAdminFrontend validates static package inputs without invoking Bun,
// importing component modules, or executing package scripts.
func InspectAdminFrontend(input FrontendInspectInput) (DependencySummary, error) {
	if err := validateHostPeerCatalog(input.HostPeers); err != nil {
		return DependencySummary{}, err
	}
	frontendRoot, err := openFrontendRoot(input.PackageRoot, input.Root)
	if err != nil {
		return DependencySummary{}, err
	}
	defer frontendRoot.Close()
	if err := inspectFrontendTree(frontendRoot); err != nil {
		return DependencySummary{}, err
	}
	if err := inspectDeclaredFiles(frontendRoot, input.Components, input.Locales); err != nil {
		return DependencySummary{}, err
	}

	packageBody, err := readFrontendFile(frontendRoot, "package.json")
	if err != nil {
		return DependencySummary{}, err
	}
	sets, direct, err := inspectPackageJSON(packageBody, input.HostPeers)
	if err != nil {
		return DependencySummary{}, err
	}

	lockBody, err := readFrontendFile(frontendRoot, "bun.lock")
	if err != nil {
		return DependencySummary{}, err
	}
	// hujson.Standardize may reuse and mutate its input buffer, so hash the
	// exact installed lock bytes before parsing them.
	digest := sha256.Sum256(lockBody)
	resolved, err := inspectBunLock(lockBody, sets, input.HostPeers)
	if err != nil {
		return DependencySummary{}, err
	}
	return DependencySummary{
		Direct:     direct,
		Resolved:   resolved,
		LockDigest: hex.EncodeToString(digest[:]),
	}, nil
}

func openFrontendRoot(packageRoot string, declaredRoot string) (*os.Root, error) {
	packageRoot = strings.TrimSpace(packageRoot)
	if packageRoot == "" {
		return nil, invalidAdminFrontendf("package root is empty")
	}
	absolutePackageRoot, err := filepath.Abs(packageRoot)
	if err != nil {
		return nil, invalidAdminFrontendf("resolve package root: %v", err)
	}
	info, err := os.Lstat(absolutePackageRoot)
	if err != nil {
		return nil, invalidAdminFrontendf("read package root: %v", err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return nil, invalidAdminFrontendSymlinkf("package root %s", absolutePackageRoot)
	}
	if !info.IsDir() {
		return nil, invalidAdminFrontendf("package root is not a directory")
	}
	packageHandle, err := os.OpenRoot(absolutePackageRoot)
	if err != nil {
		return nil, invalidAdminFrontendf("open package root: %v", err)
	}
	defer packageHandle.Close()
	openedPackageInfo, err := packageHandle.Stat(".")
	if err != nil {
		return nil, invalidAdminFrontendf("stat opened package root: %v", err)
	}
	if !os.SameFile(info, openedPackageInfo) {
		return nil, invalidAdminFrontendf("package root changed while opening")
	}

	root, ok := safeFrontendRelativePath(declaredRoot)
	if !ok {
		return nil, invalidAdminFrontendf("unsafe frontend root %q", declaredRoot)
	}
	if err := inspectPathSegments(packageHandle, root); err != nil {
		return nil, err
	}
	info, err = packageHandle.Lstat(root)
	if err != nil {
		return nil, invalidAdminFrontendf("read frontend root: %v", err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return nil, invalidAdminFrontendSymlinkf("frontend root %s", root)
	}
	if !info.IsDir() {
		return nil, invalidAdminFrontendf("frontend root is not a directory")
	}
	frontendRoot, err := packageHandle.OpenRoot(root)
	if err != nil {
		return nil, invalidAdminFrontendf("open frontend root: %v", err)
	}
	openedFrontendInfo, err := frontendRoot.Stat(".")
	if err != nil {
		_ = frontendRoot.Close()
		return nil, invalidAdminFrontendf("stat opened frontend root: %v", err)
	}
	if !os.SameFile(info, openedFrontendInfo) {
		_ = frontendRoot.Close()
		return nil, invalidAdminFrontendf("frontend root changed while opening")
	}
	return frontendRoot, nil
}

func inspectPathSegments(packageRoot *os.Root, relative string) error {
	current := ""
	for _, segment := range strings.Split(relative, "/") {
		current = path.Join(current, segment)
		info, err := packageRoot.Lstat(current)
		if err != nil {
			return invalidAdminFrontendf("read frontend path %s: %v", current, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return invalidAdminFrontendSymlinkf("frontend path %s", current)
		}
	}
	return nil
}

func inspectFrontendTree(root *os.Root) error {
	return fs.WalkDir(root.FS(), ".", func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return invalidAdminFrontendf("walk frontend tree: %v", walkErr)
		}
		info, err := root.Lstat(current)
		if err != nil {
			return invalidAdminFrontendf("inspect frontend path %s: %v", current, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return invalidAdminFrontendSymlinkf("frontend path %s", current)
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return invalidAdminFrontendf("non-regular frontend path %s", current)
		}
		if current == "." {
			return nil
		}
		relative := strings.ToLower(current)
		if forbiddenFrontendPath(relative, info.IsDir()) {
			return invalidAdminFrontendf("forbidden host build input %s", relative)
		}
		return nil
	})
}

func forbiddenFrontendPath(relative string, directory bool) bool {
	segments := strings.Split(relative, "/")
	if directory && len(segments) == 1 {
		switch segments[0] {
		case "server", "plugins", "modules", "middleware":
			return true
		}
	}
	base := path.Base(relative)
	if base == "bunfig.toml" || base == ".npmrc" {
		return true
	}
	for _, prefix := range []string{"nuxt.config.", "vite.config.", "nitro.config.", "app.config."} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return base == "nuxt.config" || base == "vite.config" || base == "nitro.config" || base == "app.config"
}

func validateHostPeerCatalog(hostPeers HostPeers) error {
	for _, name := range requiredHostPeers {
		version := strings.TrimSpace(hostPeers[name])
		if version == "" {
			return invalidAdminFrontendf("required host peer %s is missing", name)
		}
		if _, err := semver.StrictNewVersion(version); err != nil {
			return invalidAdminFrontendf("required host peer %s has invalid exact version %q", name, version)
		}
	}
	return nil
}

func inspectDeclaredFiles(root *os.Root, components map[string]string, locales map[string]string) error {
	for id, componentPath := range components {
		if strings.TrimSpace(id) == "" {
			return invalidAdminFrontendf("component id is empty")
		}
		extension := strings.ToLower(path.Ext(strings.ReplaceAll(componentPath, "\\", "/")))
		switch extension {
		case ".vue", ".ts", ".tsx", ".js", ".jsx":
		default:
			return invalidAdminFrontendf("component %s has unsupported module extension", id)
		}
		if _, err := readFrontendFile(root, componentPath); err != nil {
			return err
		}
	}

	for _, locale := range []string{"zh-CN", "en-US"} {
		if strings.TrimSpace(locales[locale]) == "" {
			return invalidAdminFrontendf("required locale %s is missing", locale)
		}
	}
	for locale, localePath := range locales {
		if !strings.EqualFold(path.Ext(strings.ReplaceAll(localePath, "\\", "/")), ".json") {
			return invalidAdminFrontendf("locale %s must be a JSON file", locale)
		}
		body, err := readFrontendFile(root, localePath)
		if err != nil {
			return err
		}
		if err := validateLocaleJSON(body); err != nil {
			return invalidAdminFrontendf("locale %s: %v", locale, err)
		}
	}
	return nil
}

func readFrontendFile(root *os.Root, relative string) ([]byte, error) {
	normalized, ok := safeFrontendRelativePath(relative)
	if !ok {
		return nil, invalidAdminFrontendf("unsafe frontend path %q", relative)
	}
	body, _, err := readRootRegularFile(root, normalized)
	if err != nil {
		if errors.Is(err, ErrSymlink) {
			return nil, invalidAdminFrontendSymlinkf("frontend file %s", normalized)
		}
		return nil, invalidAdminFrontendf("read frontend file %s: %v", normalized, err)
	}
	return body, nil
}

func safeFrontendRelativePath(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.ContainsRune(value, '\x00') || strings.HasPrefix(value, "/") {
		return "", false
	}
	if len(value) >= 2 && value[1] == ':' {
		return "", false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", false
		}
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func validateLocaleJSON(body []byte) error {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return errors.New("locale root must be an object")
	}
	if !localeObjectHasStringLeaves(root) {
		return errors.New("locale values must be nested objects with string leaves")
	}
	return nil
}

func localeObjectHasStringLeaves(object map[string]any) bool {
	for _, value := range object {
		switch typed := value.(type) {
		case string:
		case map[string]any:
			if !localeObjectHasStringLeaves(typed) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func inspectPackageJSON(body []byte, hostPeers HostPeers) (dependencySets, []Dependency, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return dependencySets{}, nil, invalidAdminFrontendf("decode package.json: %v", err)
	}
	if object == nil {
		return dependencySets{}, nil, invalidAdminFrontendf("package.json must be an object")
	}
	for _, field := range forbiddenPackageFields {
		if _, exists := object[field]; exists {
			return dependencySets{}, nil, invalidAdminFrontendf("package.json field %s is not allowed", field)
		}
	}
	if err := inspectPackageScripts(object["scripts"]); err != nil {
		return dependencySets{}, nil, err
	}

	sets := dependencySets{}
	var err error
	sets.dependencies, err = decodeDependencyMap(object, "dependencies")
	if err != nil {
		return dependencySets{}, nil, err
	}
	sets.devDependencies, err = decodeDependencyMap(object, "devDependencies")
	if err != nil {
		return dependencySets{}, nil, err
	}
	sets.optionalDependencies, err = decodeDependencyMap(object, "optionalDependencies")
	if err != nil {
		return dependencySets{}, nil, err
	}
	sets.peerDependencies, err = decodeDependencyMap(object, "peerDependencies")
	if err != nil {
		return dependencySets{}, nil, err
	}
	if err := inspectDependencyDeclarations(sets, hostPeers); err != nil {
		return dependencySets{}, nil, err
	}
	return sets, directDependencySummary(sets), nil
}

func inspectPackageScripts(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var scripts map[string]string
	if err := json.Unmarshal(raw, &scripts); err != nil {
		return invalidAdminFrontendf("package.json scripts must be string commands")
	}
	for name := range scripts {
		if _, forbidden := forbiddenLifecycleScripts[strings.ToLower(strings.TrimSpace(name))]; forbidden {
			return invalidAdminFrontendf("package lifecycle/build script %s is not allowed", name)
		}
	}
	return nil
}

func decodeDependencyMap(object map[string]json.RawMessage, field string) (map[string]string, error) {
	result := map[string]string{}
	raw, exists := object[field]
	if !exists {
		return result, nil
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, invalidAdminFrontendf("package.json %s must map names to versions", field)
	}
	for name, version := range result {
		trimmedName := strings.TrimSpace(name)
		trimmedVersion := strings.TrimSpace(version)
		if trimmedName == "" || trimmedVersion == "" {
			return nil, invalidAdminFrontendf("package.json %s contains an empty dependency", field)
		}
		if name != trimmedName || version != trimmedVersion {
			return nil, invalidAdminFrontendf("package.json %s contains unnormalized dependency text", field)
		}
	}
	return result, nil
}

func inspectDependencyDeclarations(sets dependencySets, hostPeers HostPeers) error {
	all := []map[string]string{sets.dependencies, sets.devDependencies, sets.optionalDependencies, sets.peerDependencies}
	for _, dependencies := range all {
		for name, version := range dependencies {
			if forbiddenDependencyProtocol(version) {
				return invalidAdminFrontendf("dependency %s uses forbidden local protocol", name)
			}
		}
	}
	for _, dependencies := range []map[string]string{sets.dependencies, sets.devDependencies, sets.optionalDependencies} {
		for name := range dependencies {
			if _, hostOwned := hostPeers[name]; hostOwned {
				return invalidAdminFrontendf("host peer %s cannot be a private dependency", name)
			}
		}
	}
	for name, constraintText := range sets.peerDependencies {
		hostVersionText, hostOwned := hostPeers[name]
		if !hostOwned {
			return invalidAdminFrontendf("peer dependency %s is not provided by the host", name)
		}
		hostVersion, err := semver.StrictNewVersion(hostVersionText)
		if err != nil {
			return invalidAdminFrontendf("host peer %s has invalid exact version %q", name, hostVersionText)
		}
		constraint, err := semver.NewConstraint(constraintText)
		if err != nil || !constraint.Check(hostVersion) {
			return invalidAdminFrontendf("peer dependency %s range %q does not include host version %s", name, constraintText, hostVersionText)
		}
	}
	return nil
}

func forbiddenDependencyProtocol(version string) bool {
	version = strings.ToLower(strings.TrimSpace(version))
	return strings.HasPrefix(version, "workspace:") || strings.HasPrefix(version, "file:") || strings.HasPrefix(version, "link:")
}

func directDependencySummary(sets dependencySets) []Dependency {
	direct := make(map[string]string)
	for _, dependencies := range []map[string]string{sets.dependencies, sets.devDependencies, sets.optionalDependencies, sets.peerDependencies} {
		for name, version := range dependencies {
			if current, exists := direct[name]; !exists || current == version {
				direct[name] = version
			}
		}
	}
	result := make([]Dependency, 0, len(direct))
	for name, version := range direct {
		result = append(result, Dependency{Name: name, Version: version})
	}
	sortDependencies(result)
	return result
}

func sortDependencies(dependencies []Dependency) {
	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].Name != dependencies[j].Name {
			return dependencies[i].Name < dependencies[j].Name
		}
		if dependencies[i].Version != dependencies[j].Version {
			return dependencies[i].Version < dependencies[j].Version
		}
		return dependencies[i].Integrity < dependencies[j].Integrity
	})
}

func invalidAdminFrontendf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidAdminFrontend, fmt.Sprintf(format, args...))
}

func invalidAdminFrontendSymlinkf(format string, args ...any) error {
	return fmt.Errorf("%w: %w: %s", ErrInvalidAdminFrontend, ErrSymlink, fmt.Sprintf(format, args...))
}
