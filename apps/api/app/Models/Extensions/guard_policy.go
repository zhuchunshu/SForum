package extensions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

const (
	defaultGuardPolicyTTL                 = 30 * time.Second
	RecommendedGuardPolicyRefreshInterval = 5 * time.Second
)

// GuardPolicyEntry 是管理路由 Guard 所需的最小 exact-artifact 视图。
// 该类型不暴露 Manifest，避免请求路径读到可变切片或 map。
type GuardPolicyEntry struct {
	ExtensionID             string
	ExtensionType           string
	Status                  string
	Source                  string
	Version                 string
	PackageDigest           string
	AdminFrontendDigest     string
	HasPrebuiltAdmin        bool
	FrontendArtifactTrusted bool
	HasMailProvider         bool
	HasExecutableBackend    bool
	LifecycleV2             bool
	CurrentTrustRequired    bool
	CurrentArtifactTrusted  bool
	ReviewVersion           string
	ReviewPackageDigest     string
	ReviewTrustRequired     bool
	ReviewArtifactTrusted   bool
	HasStagedArtifact       bool
	StagedVersion           string
	StagedPackageDigest     string
	StagedTrustRequired     bool
	StagedArtifactTrusted   bool
	IsSystem                bool
	IsDeletable             bool
}

type GuardPolicyLookup struct {
	Revision               uint64
	SafeMode               bool
	TrustChallengesEnabled bool
	Entry                  GuardPolicyEntry
	Found                  bool
}

type DeclaredRouteGuardLookup struct {
	Revision         uint64
	ExtensionID      string
	ExtensionVersion string
	PackageDigest    string
	Access           string
	Permission       string
}

type declaredRouteGuardEntry struct {
	path       string
	methods    []string
	access     string
	permission string
}

type guardPolicyExtensionSource interface {
	List(context.Context) ([]Extension, error)
}

type guardPolicyExecutableTrust interface {
	TrustedArtifact(context.Context, Extension) (bool, error)
}

type GuardPolicyConfig struct {
	SafeMode               bool
	TrustChallengesEnabled bool
	TTL                    time.Duration
}

type guardPolicySnapshot struct {
	entries        map[string]GuardPolicyEntry
	declaredRoutes map[string][]declaredRouteGuardEntry
	expiresAt      time.Time
	revision       uint64
}

// GuardPolicyCatalog 在后台冻结扩展类型、制品和信任状态。Lookup 只读内存，
// 不得在 HTTP Guard 热路径触发 PostgreSQL 或文件系统访问。
type GuardPolicyCatalog struct {
	extensions      guardPolicyExtensionSource
	executableTrust guardPolicyExecutableTrust
	frontendTrust   FrontendTrustStore
	config          GuardPolicyConfig

	mu       sync.RWMutex
	snapshot *guardPolicySnapshot
	revision uint64
}

func NewGuardPolicyCatalog(
	extensions guardPolicyExtensionSource,
	executableTrust guardPolicyExecutableTrust,
	frontendTrust FrontendTrustStore,
	config GuardPolicyConfig,
) *GuardPolicyCatalog {
	if config.TTL <= 0 {
		config.TTL = defaultGuardPolicyTTL
	}
	return &GuardPolicyCatalog{
		extensions: extensions, executableTrust: executableTrust,
		frontendTrust: frontendTrust, config: config,
	}
}

func (c *GuardPolicyCatalog) Refresh(ctx context.Context) error {
	if c == nil || c.extensions == nil || ctx == nil {
		return errors.New("extensions: guard policy source is unavailable")
	}
	items, err := c.extensions.List(ctx)
	if err != nil {
		return err
	}
	entries := make(map[string]GuardPolicyEntry, len(items))
	declaredRoutes := make(map[string][]declaredRouteGuardEntry, len(items))
	for _, extension := range items {
		entry, err := c.freezeEntry(ctx, extension)
		if err != nil {
			return err
		}
		if _, duplicate := entries[entry.ExtensionID]; duplicate {
			return fmt.Errorf("extensions: duplicate guard policy extension %q", entry.ExtensionID)
		}
		entries[entry.ExtensionID] = entry
		declaredRoutes[entry.ExtensionID] = freezeDeclaredRouteGuards(extension.Manifest.Routes)
	}

	c.mu.Lock()
	c.revision++
	c.snapshot = &guardPolicySnapshot{
		entries: entries, declaredRoutes: declaredRoutes,
		expiresAt: time.Now().Add(c.config.TTL), revision: c.revision,
	}
	c.mu.Unlock()
	return nil
}

// LookupDeclaredRoute 在同一不可变目录内解析 legacy proxy route。V3 custom/raw/
// inherited guards 不进入该视图，必须由独立可信 Route Registry 执行。
func (c *GuardPolicyCatalog) LookupDeclaredRoute(extensionID, method, routePath string) (DeclaredRouteGuardLookup, bool) {
	if c == nil {
		return DeclaredRouteGuardLookup{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.snapshot == nil || c.snapshot.revision == 0 || !time.Now().Before(c.snapshot.expiresAt) || c.config.SafeMode {
		return DeclaredRouteGuardLookup{}, false
	}
	id := normalizeID(extensionID)
	entry, found := c.snapshot.entries[id]
	if !found || entry.ExtensionID != extensionID || entry.ExtensionType != TypePlugin || entry.Status != StatusEnabled ||
		entry.Version == "" || entry.PackageDigest == "" ||
		(entry.CurrentTrustRequired && !entry.CurrentArtifactTrusted) {
		return DeclaredRouteGuardLookup{}, false
	}
	wantedPath := normalizeRoutePath(routePath)
	wantedMethod := strings.ToUpper(strings.TrimSpace(method))
	var matched *declaredRouteGuardEntry
	for index := range c.snapshot.declaredRoutes[id] {
		route := &c.snapshot.declaredRoutes[id][index]
		if route.path != wantedPath || !slices.Contains(route.methods, wantedMethod) {
			continue
		}
		if matched != nil {
			return DeclaredRouteGuardLookup{}, false
		}
		matched = route
	}
	if matched == nil {
		return DeclaredRouteGuardLookup{}, false
	}
	return DeclaredRouteGuardLookup{
		Revision: c.snapshot.revision, ExtensionID: entry.ExtensionID,
		ExtensionVersion: entry.Version, PackageDigest: entry.PackageDigest,
		Access: matched.access, Permission: matched.permission,
	}, true
}

func (c *GuardPolicyCatalog) Lookup(extensionID string) (GuardPolicyLookup, bool) {
	if c == nil {
		return GuardPolicyLookup{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.snapshot == nil || c.snapshot.revision == 0 || !time.Now().Before(c.snapshot.expiresAt) {
		return GuardPolicyLookup{}, false
	}
	id := normalizeID(extensionID)
	entry, found := c.snapshot.entries[id]
	return GuardPolicyLookup{
		Revision: c.snapshot.revision, SafeMode: c.config.SafeMode,
		TrustChallengesEnabled: c.config.TrustChallengesEnabled,
		Entry:                  entry, Found: found,
	}, true
}

func (c *GuardPolicyCatalog) RunRefresh(ctx context.Context, interval time.Duration) {
	if c == nil || ctx == nil {
		return
	}
	if interval <= 0 {
		interval = RecommendedGuardPolicyRefreshInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.Refresh(ctx)
		}
	}
}

func (c *GuardPolicyCatalog) freezeEntry(ctx context.Context, extension Extension) (GuardPolicyEntry, error) {
	id := normalizeID(extension.ID)
	if id == "" || id != extension.ID || (extension.Type != TypePlugin && extension.Type != TypeTheme) ||
		(extension.Status != StatusInstalled && extension.Status != StatusEnabled && extension.Status != StatusDisabled) ||
		strings.TrimSpace(extension.Version) == "" || strings.TrimSpace(extension.PackageDigest) == "" ||
		extension.Manifest.ID != extension.ID || extension.Manifest.Version != extension.Version || extension.Manifest.Type != extension.Type {
		return GuardPolicyEntry{}, fmt.Errorf("extensions: invalid guard policy artifact %q", extension.ID)
	}

	currentTrusted, err := c.artifactTrusted(ctx, extension)
	if err != nil {
		return GuardPolicyEntry{}, fmt.Errorf("extensions: resolve current guard trust for %s: %w", extension.ID, err)
	}
	review := trustReviewArtifact(extension)
	reviewTrusted := currentTrusted
	if review.Version != extension.Version || review.PackageDigest != extension.PackageDigest {
		reviewTrusted, err = c.artifactTrusted(ctx, review)
		if err != nil {
			return GuardPolicyEntry{}, fmt.Errorf("extensions: resolve review guard trust for %s: %w", extension.ID, err)
		}
	}
	frontendTrusted := currentTrusted
	if !c.config.TrustChallengesEnabled {
		frontendTrusted, err = c.legacyFrontendTrusted(ctx, extension)
		if err != nil {
			return GuardPolicyEntry{}, fmt.Errorf("extensions: resolve frontend guard trust for %s: %w", extension.ID, err)
		}
	}
	entry := GuardPolicyEntry{
		ExtensionID: extension.ID, ExtensionType: extension.Type, Status: extension.Status,
		Source: extension.Source, Version: extension.Version, PackageDigest: extension.PackageDigest,
		AdminFrontendDigest:     extension.AdminFrontendDigest,
		HasPrebuiltAdmin:        prebuiltSettingsComponent(extension) != nil,
		FrontendArtifactTrusted: frontendTrusted,
		HasMailProvider:         manifestHasProvider(extension.Manifest, "mail.provider"),
		HasExecutableBackend:    hasExecutableBackend(extension.Manifest), LifecycleV2: usesLifecycleV2(extension),
		CurrentTrustRequired: RequiresExecutableTrust(extension), CurrentArtifactTrusted: currentTrusted,
		ReviewVersion: review.Version, ReviewPackageDigest: review.PackageDigest,
		ReviewTrustRequired: RequiresExecutableTrust(review), ReviewArtifactTrusted: reviewTrusted,
		IsSystem: extension.IsSystem, IsDeletable: extension.IsDeletable,
	}
	if staged, ok := extension.StagedArtifact(); ok {
		stagedTrusted := reviewTrusted
		if staged.Version != review.Version || staged.PackageDigest != review.PackageDigest {
			stagedTrusted, err = c.artifactTrusted(ctx, staged)
			if err != nil {
				return GuardPolicyEntry{}, fmt.Errorf("extensions: resolve staged guard trust for %s: %w", extension.ID, err)
			}
		}
		entry.HasStagedArtifact = true
		entry.StagedVersion = staged.Version
		entry.StagedPackageDigest = staged.PackageDigest
		entry.StagedTrustRequired = RequiresExecutableTrust(staged)
		entry.StagedArtifactTrusted = stagedTrusted
	}
	return entry, nil
}

func (c *GuardPolicyCatalog) artifactTrusted(ctx context.Context, extension Extension) (bool, error) {
	if !RequiresExecutableTrust(extension) {
		return true, nil
	}
	if c.executableTrust == nil {
		return false, errors.New("executable trust source is unavailable")
	}
	return c.executableTrust.TrustedArtifact(ctx, extension)
}

func manifestHasProvider(manifest Manifest, slot string) bool {
	for _, provider := range manifest.Providers {
		if provider.Slot == slot {
			return true
		}
	}
	return false
}

func freezeDeclaredRouteGuards(routes []ManifestRoute) []declaredRouteGuardEntry {
	result := make([]declaredRouteGuardEntry, 0, len(routes))
	for _, route := range routes {
		access, permission, ok := declaredRouteGuardAccess(route)
		if !ok || route.Path == "" || !strings.HasPrefix(route.Path, "/") || strings.Contains(route.Path, "..") {
			continue
		}
		methods := make([]string, 0, len(route.Methods))
		for _, method := range route.Methods {
			method = strings.ToUpper(strings.TrimSpace(method))
			if !validDeclaredRouteGuardMethod(method) ||
				(access == RouteAccessPublic && method != "GET" && method != "HEAD" && method != "OPTIONS") {
				methods = nil
				break
			}
			if !slices.Contains(methods, method) {
				methods = append(methods, method)
			}
		}
		if len(methods) == 0 {
			continue
		}
		result = append(result, declaredRouteGuardEntry{
			path: normalizeRoutePath(route.Path), methods: methods,
			access: access, permission: permission,
		})
	}
	return result
}

func validDeclaredRouteGuardMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func declaredRouteGuardAccess(route ManifestRoute) (access string, permission string, ok bool) {
	access = strings.TrimSpace(route.Access)
	guard := strings.TrimSpace(route.Guard)
	if guard == "" {
		if access == "" {
			access = RouteAccessLogin
		}
	} else {
		switch guard {
		case extensionmanifest.GuardCorePublic:
			if access != RouteAccessPublic {
				return "", "", false
			}
		case extensionmanifest.GuardCoreLogin:
			if access == "" {
				access = RouteAccessLogin
			}
			if access != RouteAccessLogin {
				return "", "", false
			}
		case extensionmanifest.GuardCorePermission:
			if access != RouteAccessPermission {
				return "", "", false
			}
		default:
			return "", "", false
		}
	}
	switch access {
	case RouteAccessPublic, RouteAccessLogin:
		return access, "", true
	case RouteAccessPermission:
		permission = strings.TrimSpace(route.Permission)
		return access, permission, permission != ""
	default:
		return "", "", false
	}
}

func (c *GuardPolicyCatalog) legacyFrontendTrusted(ctx context.Context, extension Extension) (bool, error) {
	component := prebuiltSettingsComponent(extension)
	if component == nil {
		return false, nil
	}
	if extension.Source == SourceBuiltin && extension.IsSystem && !extension.IsDeletable {
		return true, nil
	}
	if c.frontendTrust == nil {
		return false, errors.New("frontend trust source is unavailable")
	}
	grant, err := c.frontendTrust.FrontendGrant(ctx, extension.ID, extension.Version, extension.AdminFrontendDigest)
	if err != nil {
		if errors.Is(err, ErrFrontendGrantNotFound) {
			return false, nil
		}
		return false, err
	}
	return grant.APIVersion == component.APIVersion && slices.Contains(grant.ComponentIDs, component.ID), nil
}
