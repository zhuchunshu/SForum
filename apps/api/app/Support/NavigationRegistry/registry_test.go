package navigationregistry

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryCoversNavigationKindsAndThemeRegionsWithExplicitVisibility(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{
		navigation("core.navigation.menu.main", NavigationKindMenu, ActionAdd, "", 20),
		navigation("core.navigation.item.docs", NavigationKindItem, ActionAdd, "core.navigation.menu.main", 10),
		navigation("core.navigation.breadcrumb.main", NavigationKindBreadcrumb, ActionAdd, "", 0),
		navigation("core.navigation.header.main", NavigationKindHeader, ActionAdd, "", 0),
		navigation("core.navigation.footer.main", NavigationKindFooter, ActionAdd, "", 0),
		navigation("core.navigation.sidebar.main", NavigationKindSidebar, ActionAdd, "", 0),
	}
	core.Navigation[1].Permission = "docs.read"
	core.Navigation[1].Href = "/docs"
	theme := publication("theme.signal", false, 'b')
	theme.Regions = []RegionDeclaration{
		region("theme.signal.region.menu", RegionKindMenu, ActionAdd, ""),
		region("theme.signal.region.widget", RegionKindWidget, ActionAdd, ""),
		region("theme.signal.region.header", RegionKindHeader, ActionAdd, ""),
		region("theme.signal.region.footer", RegionKindFooter, ActionAdd, ""),
		region("theme.signal.region.sidebar", RegionKindSidebar, ActionAdd, ""),
		region("theme.signal.region.content", RegionKindContent, ActionAdd, ""),
	}
	theme.Regions[1].Multiple = true

	registry := New()
	if revision, err := registry.ReplaceAll([]Publication{theme, core}); err != nil || revision != 1 {
		t.Fatalf("replace all: revision=%d err=%v", revision, err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Navigation) != 6 || len(snapshot.Regions) != 6 || snapshot.SchemaVersion != SchemaVersion {
		t.Fatalf("snapshot=%#v", snapshot)
	}

	denied, err := registry.ResolveNavigation(NavigationResolveRequest{Kinds: []string{NavigationKindItem}})
	if err != nil || len(denied.Targets) != 0 {
		t.Fatalf("permission denied projection=%#v err=%v", denied, err)
	}
	allowed, err := registry.ResolveNavigation(NavigationResolveRequest{
		Kinds: []string{NavigationKindItem}, Visibility: VisibilityInput{Permissions: []string{"docs.read"}},
	})
	if err != nil || len(allowed.Targets) != 1 || allowed.Targets[0].ParentID != "core.navigation.menu.main" {
		t.Fatalf("permission allowed projection=%#v err=%v", allowed, err)
	}
	hiddenParent, err := registry.ResolveNavigation(NavigationResolveRequest{Visibility: VisibilityInput{
		Permissions: []string{"docs.read"}, HiddenIDs: []string{"core.navigation.menu.main"},
	}})
	if err != nil || hasNavigationTarget(hiddenParent, "core.navigation.item.docs") {
		t.Fatalf("hidden parent retained child=%#v err=%v", hiddenParent, err)
	}
	regions, err := registry.ResolveRegions(RegionResolveRequest{Visibility: VisibilityInput{
		HiddenIDs: []string{"theme.signal.region.footer"},
	}})
	if err != nil || len(regions.Targets) != 5 || hasRegionTarget(regions, "theme.signal.region.footer") {
		t.Fatalf("region visibility=%#v err=%v", regions, err)
	}
	disabledTarget, err := registry.ResolveRegions(RegionResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{providerRef("theme.signal.region.widget", theme)},
	}})
	if err != nil || hasRegionTarget(disabledTarget, "theme.signal.region.widget") || len(disabledTarget.Targets) != 5 {
		t.Fatalf("exact-disabled add target remained visible=%#v err=%v", disabledTarget, err)
	}
}

func TestRegistryDeterministicProviderPriorityFallbackAndExactDisable(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{navigation("core.navigation.item.home", NavigationKindItem, ActionAdd, "", 10)}
	alpha := publication("alpha.navigation", false, 'b')
	alpha.Navigation = []NavigationDeclaration{navigation("alpha.navigation.item.home", NavigationKindItem, ActionReplace, "core.navigation.item.home", 0)}
	alpha.Navigation[0].Priority = 10
	beta := publication("beta.navigation", false, 'c')
	beta.Navigation = []NavigationDeclaration{navigation("beta.navigation.item.home", NavigationKindItem, ActionReplace, "core.navigation.item.home", 0)}
	beta.Navigation[0].Priority = 20

	registry := New()
	if _, err := registry.ReplaceAll([]Publication{alpha, core, beta}); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.NavigationConflicts) != 1 || snapshot.NavigationConflicts[0].Winner.ID != beta.Navigation[0].ID {
		t.Fatalf("conflicts=%#v", snapshot.NavigationConflicts)
	}
	resolved := mustResolveNavigation(t, registry, NavigationResolveRequest{})
	if resolved.Targets[0].Provider.ID != beta.Navigation[0].ID || resolved.Targets[0].UsingFallback {
		t.Fatalf("winner=%#v", resolved.Targets[0])
	}
	alphaFallback := mustResolveNavigation(t, registry, NavigationResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{providerRef(beta.Navigation[0].ID, beta)},
	}})
	if alphaFallback.Targets[0].Provider.ID != alpha.Navigation[0].ID {
		t.Fatalf("next provider=%#v", alphaFallback.Targets[0])
	}
	staleQuarantine := providerRef(beta.Navigation[0].ID, beta)
	staleQuarantine.Artifact.PackageDigest = strings.Repeat("f", 64)
	exactWinner := mustResolveNavigation(t, registry, NavigationResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{staleQuarantine},
	}})
	if exactWinner.Targets[0].Provider.ID != beta.Navigation[0].ID {
		t.Fatalf("stale artifact quarantine disabled replacement=%#v", exactWinner.Targets[0])
	}
	coreFallback := mustResolveNavigation(t, registry, NavigationResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{
			providerRef(beta.Navigation[0].ID, beta), providerRef(alpha.Navigation[0].ID, alpha),
		},
	}})
	if coreFallback.Targets[0].Provider.ID != core.Navigation[0].ID || !coreFallback.Targets[0].UsingFallback {
		t.Fatalf("core fallback=%#v", coreFallback.Targets[0])
	}

	before := registry.CacheState()
	stalePackage := beta.Artifact
	stalePackage.PackageDigest = strings.Repeat("f", 64)
	if revision, removed, err := registry.Remove(stalePackage); !errors.Is(err, ErrArtifactConflict) || removed || revision != before.Revision {
		t.Fatalf("stale remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	staleImpact := beta.Artifact
	staleImpact.ImpactDigest = strings.Repeat("e", 64)
	if revision, removed, err := registry.Remove(staleImpact); !errors.Is(err, ErrArtifactConflict) || removed || revision != before.Revision {
		t.Fatalf("stale impact remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if revision, removed, err := registry.Remove(beta.Artifact); err != nil || !removed || revision != before.Revision+1 {
		t.Fatalf("exact remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if !registry.CacheInvalidated(before) {
		t.Fatal("provider removal did not invalidate cache revision")
	}
	after := mustResolveNavigation(t, registry, NavigationResolveRequest{})
	if after.Targets[0].Provider.ID != alpha.Navigation[0].ID {
		t.Fatalf("persistent disable fallback=%#v", after.Targets[0])
	}
}

func TestRegistryBuildsOrderedNavigationAndRegionCompositionPlans(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{navigation("core.navigation.item.home", NavigationKindItem, ActionAdd, "", 10)}
	core.Regions = []RegionDeclaration{region("core.navigation.region.sidebar", RegionKindSidebar, ActionAdd, "")}
	core.Regions[0].Multiple = true

	alpha := publication("alpha.navigation", false, 'b')
	alpha.Navigation = []NavigationDeclaration{
		navigation("alpha.navigation.item.before-low", NavigationKindItem, ActionBefore, core.Navigation[0].ID, 0),
		navigation("alpha.navigation.item.before-high", NavigationKindItem, ActionBefore, core.Navigation[0].ID, 0),
		navigation("alpha.navigation.item.after", NavigationKindItem, ActionAfter, core.Navigation[0].ID, 0),
		navigation("alpha.navigation.item.wrap", NavigationKindItem, ActionWrap, core.Navigation[0].ID, 0),
		navigation("alpha.navigation.item.filter", NavigationKindItem, ActionFilter, core.Navigation[0].ID, 0),
		navigation("alpha.navigation.item.hide", NavigationKindItem, ActionHide, core.Navigation[0].ID, 0),
	}
	alpha.Navigation[0].Priority = 10
	alpha.Navigation[1].Priority = 50
	alpha.Navigation[5].Permission = "home.hide"
	alpha.Regions = []RegionDeclaration{
		region("alpha.navigation.region.replace", RegionKindSidebar, ActionReplace, core.Regions[0].ID),
		region("alpha.navigation.region.hide", RegionKindSidebar, ActionHide, core.Regions[0].ID),
	}
	alpha.Regions[0].Priority = 20

	registry := New()
	if _, err := registry.ReplaceAll([]Publication{alpha, core}); err != nil {
		t.Fatal(err)
	}
	navigationPlan := mustResolveNavigation(t, registry, NavigationResolveRequest{})
	if len(navigationPlan.Targets) != 1 || len(navigationPlan.Targets[0].Before) != 2 ||
		navigationPlan.Targets[0].Before[0].ID != "alpha.navigation.item.before-high" ||
		len(navigationPlan.Targets[0].After) != 1 || len(navigationPlan.Targets[0].Wrap) != 1 ||
		len(navigationPlan.Targets[0].Filters) != 1 {
		t.Fatalf("navigation composition=%#v", navigationPlan)
	}
	hidden := mustResolveNavigation(t, registry, NavigationResolveRequest{Visibility: VisibilityInput{Permissions: []string{"home.hide"}}})
	if len(hidden.Targets) != 0 {
		t.Fatalf("permission-scoped hide remained visible=%#v", hidden)
	}

	regions, err := registry.ResolveRegions(RegionResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{providerRef("alpha.navigation.region.hide", alpha)},
	}})
	// Multiple is a ManifestRegion field on the add target; replace providers do
	// not carry it. Resolution must preserve the target's capacity flag.
	if err != nil || len(regions.Targets) != 1 || regions.Targets[0].Provider.ID != "alpha.navigation.region.replace" ||
		!regions.Targets[0].Target.Multiple {
		t.Fatalf("region provider plan=%#v err=%v", regions, err)
	}
	fallback, err := registry.ResolveRegions(RegionResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{
			providerRef("alpha.navigation.region.hide", alpha), providerRef("alpha.navigation.region.replace", alpha),
		},
	}})
	if err != nil || len(fallback.Targets) != 1 || fallback.Targets[0].Provider.ID != core.Regions[0].ID || !fallback.Targets[0].UsingFallback {
		t.Fatalf("region fallback=%#v err=%v", fallback, err)
	}
}

func TestRegistryRequiredOptionalConflictAndCycleSafety(t *testing.T) {
	owner := publication("owner.navigation", false, 'a')
	owner.Navigation = []NavigationDeclaration{navigation("owner.navigation.item.base", NavigationKindItem, ActionAdd, "", 0)}
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{ExtensionID: owner.Artifact.ExtensionID, Version: "^1.0.0", Kind: DependencyRequired}}
	consumer.Navigation = []NavigationDeclaration{navigation("consumer.navigation.item.replace", NavigationKindItem, ActionReplace, owner.Navigation[0].ID, 0)}

	registry := New()
	if _, err := registry.ReplaceAll([]Publication{consumer, owner}); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if revision, removed, err := registry.Remove(owner.Artifact); !errors.Is(err, ErrDependency) || removed || revision != before.Revision {
		t.Fatalf("required owner removal: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed removal changed snapshot: before=%#v after=%#v", before, after)
	}

	undeclared := consumer
	undeclared.Artifact = artifact("undeclared.navigation", false, 'c')
	undeclared.Dependencies = nil
	undeclared.Navigation[0].ID = "undeclared.navigation.item.replace"
	undeclared.Navigation[0].ContractVersion = "undeclared.navigation.item.replace@1"
	if _, err := New().ReplaceAll([]Publication{owner, undeclared}); !errors.Is(err, ErrDependency) {
		t.Fatalf("undeclared cross-plugin target=%v", err)
	}

	optional := publication("optional.navigation", false, 'd')
	optional.Dependencies = []Dependency{{ExtensionID: "missing.navigation", Version: "^1.0.0", Kind: DependencyOptional}}
	optional.Navigation = []NavigationDeclaration{navigation(
		"optional.navigation.item.after", NavigationKindItem, ActionAfter, "missing.navigation.item.base", 0,
	)}
	optional.Navigation = append(optional.Navigation,
		navigation("optional.navigation.menu.child", NavigationKindMenu, ActionAdd, "missing.navigation.menu.base", 0),
		navigation("optional.navigation.item.grandchild", NavigationKindItem, ActionAdd, "optional.navigation.menu.child", 0),
	)
	optionalRegistry := New()
	if _, err := optionalRegistry.ReplaceAll([]Publication{optional}); err != nil {
		t.Fatalf("optional target=%v", err)
	}
	if snapshot := optionalRegistry.Snapshot(); len(snapshot.Navigation) != 0 {
		t.Fatalf("optional missing target published=%#v", snapshot)
	}

	// 短 optional 前缀不能 fail-open 到另一个已存在的插件目标。
	ownerB := publication("plugin.other", false, '7')
	ownerB.Navigation = []NavigationDeclaration{navigation("plugin.other.item.base", NavigationKindItem, ActionAdd, "", 0)}
	shortOptional := publication("short.optional", false, '8')
	shortOptional.Dependencies = []Dependency{{ExtensionID: "plug", Version: "^1.0.0", Kind: DependencyOptional}}
	shortOptional.Navigation = []NavigationDeclaration{navigation(
		"short.optional.item.replace", NavigationKindItem, ActionReplace, ownerB.Navigation[0].ID, 0,
	)}
	if _, err := New().ReplaceAll([]Publication{ownerB, shortOptional}); !errors.Is(err, ErrDependency) {
		t.Fatalf("short optional prefix fail-open=%v", err)
	}

	conflicting := publication("conflict.navigation", false, 'e')
	conflicting.Dependencies = []Dependency{{ExtensionID: owner.Artifact.ExtensionID, Version: "^1.0.0", Kind: DependencyConflict}}
	if _, err := New().ReplaceAll([]Publication{owner, conflicting}); !errors.Is(err, ErrConflict) {
		t.Fatalf("declared conflict=%v", err)
	}
	left := publication("cycle.left", false, 'f')
	right := publication("cycle.right", false, '1')
	left.Dependencies = []Dependency{{ExtensionID: right.Artifact.ExtensionID, Version: "^1.0.0", Kind: DependencyRequired}}
	right.Dependencies = []Dependency{{ExtensionID: left.Artifact.ExtensionID, Version: "^1.0.0", Kind: DependencyRequired}}
	if _, err := New().ReplaceAll([]Publication{left, right}); !errors.Is(err, ErrDependency) {
		t.Fatalf("dependency cycle=%v", err)
	}
}

func TestRegistryCapabilityDependencyIsExactAndAmbiguityFailsClosed(t *testing.T) {
	owner := publication("owner.navigation", false, 'a')
	owner.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "1.0.0", Kind: DependencyProvides}}
	owner.Navigation = []NavigationDeclaration{navigation("owner.navigation.item.base", NavigationKindItem, ActionAdd, "", 0)}
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "^1.0.0", Kind: DependencyRequired}}
	consumer.Navigation = []NavigationDeclaration{navigation(
		"consumer.navigation.item.after", NavigationKindItem, ActionAfter, owner.Navigation[0].ID, 0,
	)}
	if _, err := New().ReplaceAll([]Publication{consumer, owner}); err != nil {
		t.Fatalf("single capability provider=%v", err)
	}
	second := publication("second.navigation", false, 'c')
	second.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "1.1.0", Kind: DependencyProvides}}
	if _, err := New().ReplaceAll([]Publication{owner, second, consumer}); !errors.Is(err, ErrDependency) {
		t.Fatalf("ambiguous capability=%v", err)
	}
}

func TestRegistryOptionalCapabilityRemovalOmitsDependentSubtree(t *testing.T) {
	owner := publication("owner.navigation", false, 'a')
	owner.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "1.0.0", Kind: DependencyProvides}}
	owner.Navigation = []NavigationDeclaration{
		navigation("owner.navigation.menu.base", NavigationKindMenu, ActionAdd, "", 0),
	}
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "^1.0.0", Kind: DependencyOptional}}
	consumer.Navigation = []NavigationDeclaration{
		navigation("consumer.navigation.item.after", NavigationKindMenu, ActionAfter, owner.Navigation[0].ID, 0),
		navigation("consumer.navigation.menu.child", NavigationKindMenu, ActionAdd, owner.Navigation[0].ID, 0),
		navigation("consumer.navigation.item.grandchild", NavigationKindItem, ActionAdd, "consumer.navigation.menu.child", 0),
	}
	unrelated := publication("unrelated.navigation", false, 'c')
	unrelated.Navigation = []NavigationDeclaration{
		navigation("unrelated.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}

	registry := New()
	if _, err := registry.ReplaceAll([]Publication{consumer, owner, unrelated}); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if len(before.Navigation) != 5 {
		t.Fatalf("active optional capability graph=%#v", before.Navigation)
	}
	if revision, removed, err := registry.Remove(owner.Artifact); err != nil || !removed || revision != before.Revision+1 {
		t.Fatalf("optional capability owner removal: revision=%d removed=%t err=%v", revision, removed, err)
	}
	after := registry.Snapshot()
	if len(after.Navigation) != 1 || after.Navigation[0].ID != unrelated.Navigation[0].ID {
		t.Fatalf("optional capability subtree remained=%#v", after.Navigation)
	}
	if _, err := registry.Publish(owner); err != nil {
		t.Fatal(err)
	}
	if restored := registry.Snapshot(); len(restored.Navigation) != 5 {
		t.Fatalf("optional capability subtree was not restored=%#v", restored.Navigation)
	}

	missingOwnTarget := consumer
	missingOwnTarget.Navigation = []NavigationDeclaration{
		navigation("consumer.navigation.item.typo", NavigationKindItem, ActionAfter, "consumer.navigation.item.missing", 0),
	}
	if _, err := New().ReplaceAll([]Publication{missingOwnTarget}); !errors.Is(err, ErrConflict) {
		t.Fatalf("unresolved optional capability hid consumer target typo=%v", err)
	}
	missingCoreTarget := consumer
	missingCoreTarget.Navigation = []NavigationDeclaration{
		navigation("consumer.navigation.item.core", NavigationKindItem, ActionAfter, "core.navigation.item.missing", 0),
	}
	if _, err := New().ReplaceAll([]Publication{missingCoreTarget}); !errors.Is(err, ErrConflict) {
		t.Fatalf("unresolved optional capability hid Core target typo=%v", err)
	}
}

func TestRegistryOptionalCapabilityFailsClosedForKnownUnrelatedOwner(t *testing.T) {
	known := publication("known.navigation", false, 'a')
	known.Navigation = []NavigationDeclaration{
		navigation("known.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{Capability: "missing.navigation", Version: "^1.0.0", Kind: DependencyOptional}}
	consumer.Navigation = []NavigationDeclaration{navigation(
		"consumer.navigation.item.typo", NavigationKindItem, ActionAfter, "known.navigation.item.missing", 0,
	)}
	if _, err := New().ReplaceAll([]Publication{known, consumer}); !errors.Is(err, ErrConflict) {
		t.Fatalf("unresolved capability hid known namespace typo=%v", err)
	}
	consumer.Navigation[0].TargetID = known.Navigation[0].ID
	if _, err := New().ReplaceAll([]Publication{known, consumer}); !errors.Is(err, ErrDependency) {
		t.Fatalf("unresolved capability authorized known owner=%v", err)
	}
}

func TestRegistryOptionalCapabilityVersionMismatchOmitsOnlyProvider(t *testing.T) {
	provider := publication("provider.navigation", false, 'a')
	provider.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "2.0.0", Kind: DependencyProvides}}
	provider.Navigation = []NavigationDeclaration{
		navigation("provider.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "^1.0.0", Kind: DependencyOptional}}
	consumer.Navigation = []NavigationDeclaration{navigation(
		"consumer.navigation.item.after", NavigationKindItem, ActionAfter, provider.Navigation[0].ID, 0,
	)}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{provider, consumer}); err != nil {
		t.Fatalf("optional incompatible capability=%v", err)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Navigation) != 1 || snapshot.Navigation[0].ID != provider.Navigation[0].ID {
		t.Fatalf("incompatible optional contribution remained=%#v", snapshot.Navigation)
	}

	unrelated := publication("unrelated.navigation", false, 'c')
	unrelated.Navigation = []NavigationDeclaration{
		navigation("unrelated.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	consumer.Navigation[0].TargetID = unrelated.Navigation[0].ID
	if _, err := New().ReplaceAll([]Publication{provider, unrelated, consumer}); !errors.Is(err, ErrDependency) {
		t.Fatalf("optional capability version mismatch authorized unrelated owner=%v", err)
	}
}

func TestRegistryOptionalExtensionRemovalOmitsDependentSubtree(t *testing.T) {
	owner := publication("owner.navigation", false, 'a')
	owner.Navigation = []NavigationDeclaration{
		navigation("owner.navigation.menu.base", NavigationKindMenu, ActionAdd, "", 0),
	}
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{ExtensionID: owner.Artifact.ExtensionID, Version: "^1.0.0", Kind: DependencyOptional}}
	consumer.Navigation = []NavigationDeclaration{
		navigation("consumer.navigation.menu.child", NavigationKindMenu, ActionAdd, owner.Navigation[0].ID, 0),
		navigation("consumer.navigation.item.grandchild", NavigationKindItem, ActionAdd, "consumer.navigation.menu.child", 0),
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{owner, consumer}); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := registry.Remove(owner.Artifact); err != nil || !removed {
		t.Fatalf("optional extension owner removal: removed=%t err=%v", removed, err)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Navigation) != 0 {
		t.Fatalf("optional extension subtree remained=%#v", snapshot.Navigation)
	}
}

func TestRegistryRequiredCapabilityRemovalFailsAtomically(t *testing.T) {
	owner := publication("owner.navigation", false, 'a')
	owner.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "1.0.0", Kind: DependencyProvides}}
	owner.Navigation = []NavigationDeclaration{navigation("owner.navigation.item.base", NavigationKindItem, ActionAdd, "", 0)}
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "^1.0.0", Kind: DependencyRequired}}
	consumer.Navigation = []NavigationDeclaration{navigation(
		"consumer.navigation.item.after", NavigationKindItem, ActionAfter, owner.Navigation[0].ID, 0,
	)}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{owner, consumer}); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if revision, removed, err := registry.Remove(owner.Artifact); !errors.Is(err, ErrDependency) || removed || revision != before.Revision {
		t.Fatalf("required capability owner removal: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed capability removal changed snapshot: before=%#v after=%#v", before, after)
	}
}

func TestRegistryReplaceAllConvergesAndCacheKeyIncludesVisibility(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{
		navigation("core.navigation.item.public", NavigationKindItem, ActionAdd, "", 20),
		navigation("core.navigation.item.private", NavigationKindItem, ActionAdd, "", 10),
	}
	core.Navigation[1].Permission = "forum.private"
	theme := publication("theme.signal", false, 'b')
	theme.Regions = []RegionDeclaration{region("theme.signal.region.widget", RegionKindWidget, ActionAdd, "")}

	first, second := New(), New()
	if _, err := first.ReplaceAll([]Publication{core, theme}); err != nil {
		t.Fatal(err)
	}
	if _, err := second.ReplaceAll([]Publication{theme, core}); err != nil {
		t.Fatal(err)
	}
	if left, right := first.Snapshot(), second.Snapshot(); !reflect.DeepEqual(left, right) {
		t.Fatalf("input order changed snapshot: left=%#v right=%#v", left, right)
	}
	before := second.CacheState()
	if revision, err := second.ReplaceAll([]Publication{core, theme}); err != nil || revision != before.Revision {
		t.Fatalf("idempotent replace: revision=%d err=%v", revision, err)
	}
	if second.CacheInvalidated(before) {
		t.Fatal("idempotent replace invalidated cache")
	}

	one := mustResolveNavigation(t, first, NavigationResolveRequest{Visibility: VisibilityInput{
		Permissions: []string{"forum.private", "forum.read"},
	}})
	two := mustResolveNavigation(t, first, NavigationResolveRequest{Visibility: VisibilityInput{
		Permissions: []string{"forum.read", "forum.private", "forum.read"},
	}})
	if one.CacheKey != two.CacheKey {
		t.Fatalf("equivalent visibility changed key: %s != %s", one.CacheKey, two.CacheKey)
	}
	hidden := mustResolveNavigation(t, first, NavigationResolveRequest{Visibility: VisibilityInput{
		Permissions: []string{"forum.private", "forum.read"}, HiddenIDs: []string{core.Navigation[0].ID},
	}})
	if hidden.CacheKey == one.CacheKey {
		t.Fatal("visibility change reused cache key")
	}
}

func TestRegistrySnapshotsAndResolutionsAreDetached(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{navigation("core.navigation.item.home", NavigationKindItem, ActionAdd, "", 0)}
	alpha := publication("alpha.navigation", false, 'b')
	alpha.Navigation = []NavigationDeclaration{navigation("alpha.navigation.item.home", NavigationKindItem, ActionReplace, core.Navigation[0].ID, 0)}
	beta := publication("beta.navigation", false, 'c')
	beta.Navigation = []NavigationDeclaration{navigation("beta.navigation.item.home", NavigationKindItem, ActionReplace, core.Navigation[0].ID, 0)}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, alpha, beta}); err != nil {
		t.Fatal(err)
	}

	snapshot := registry.Snapshot()
	snapshot.Navigation[0].Label = "forged"
	snapshot.NavigationConflicts[0].Candidates[0].Label = "forged"
	resolved := mustResolveNavigation(t, registry, NavigationResolveRequest{})
	resolved.Targets[0].ReplaceCandidates[0].Label = "forged"
	againSnapshot := registry.Snapshot()
	againResolved := mustResolveNavigation(t, registry, NavigationResolveRequest{})
	if againSnapshot.Navigation[0].Label == "forged" || againSnapshot.NavigationConflicts[0].Candidates[0].Label == "forged" ||
		againResolved.Targets[0].ReplaceCandidates[0].Label == "forged" {
		t.Fatalf("caller mutation escaped: snapshot=%#v resolution=%#v", againSnapshot, againResolved)
	}
}

func TestRegistryRejectsInvalidTargetsWithoutPartialPublication(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{navigation("core.navigation.item.home", NavigationKindItem, ActionAdd, "", 0)}
	registry := New()
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	invalid := publication("invalid.navigation", false, 'b')
	invalid.Navigation = []NavigationDeclaration{navigation(
		"invalid.navigation.menu.replace", NavigationKindMenu, ActionReplace, core.Navigation[0].ID, 0,
	)}
	if _, err := registry.Publish(invalid); !errors.Is(err, ErrConflict) {
		t.Fatalf("kind mismatch=%v", err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed publication changed snapshot: before=%#v after=%#v", before, after)
	}
	badLink := publication("bad.navigation", false, 'c')
	badLink.Navigation = []NavigationDeclaration{navigation("bad.navigation.item.external", NavigationKindItem, ActionAdd, "", 0)}
	badLink.Navigation[0].Href = "https://example.com"
	if _, err := registry.Publish(badLink); !errors.Is(err, ErrInvalid) {
		t.Fatalf("external href=%v", err)
	}
	spoofedCore := publication("spoofed.navigation", true, 'e')
	if _, err := registry.Publish(spoofedCore); !errors.Is(err, ErrInvalid) {
		t.Fatalf("spoofed core artifact=%v", err)
	}
	spoofedPlugin := publication("core.spoofed-navigation", false, 'e')
	if _, err := registry.Publish(spoofedPlugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("plugin in reserved Core namespace=%v", err)
	}

	cyclic := publication("cycle.navigation", false, 'd')
	cyclic.Regions = []RegionDeclaration{
		region("cycle.navigation.region.one", RegionKindWidget, ActionAdd, "cycle.navigation.region.two"),
		region("cycle.navigation.region.two", RegionKindWidget, ActionAdd, "cycle.navigation.region.one"),
	}
	if _, err := registry.Publish(cyclic); !errors.Is(err, ErrDependency) {
		t.Fatalf("target cycle=%v", err)
	}
}

func publication(id string, core bool, digest rune) Publication {
	return Publication{Artifact: artifact(id, core, digest)}
}

func artifact(id string, core bool, digest rune) Artifact {
	value := strings.Repeat(string(digest), 64)
	return Artifact{
		ExtensionID: id, ExtensionVersion: "1.0.0", PackageDigest: value, ImpactDigest: value, Core: core,
	}
}

func providerRef(contributionID string, publication Publication) ProviderRef {
	return ProviderRef{ContributionID: contributionID, Artifact: publication.Artifact}
}

func navigation(id, kind, action, targetID string, order int) NavigationDeclaration {
	return NavigationDeclaration{
		ID: id, ContractVersion: id + "@1", Kind: kind, Action: action, TargetID: targetID,
		Label: id, Href: "/" + strings.ReplaceAll(id, ".", "/"), Order: order,
	}
}

func region(id, kind, action, targetID string) RegionDeclaration {
	return RegionDeclaration{
		ID: id, ContractVersion: id + "@1", Kind: kind, Action: action, TargetID: targetID, Label: id,
	}
}

func mustResolveNavigation(t *testing.T, registry *Registry, request NavigationResolveRequest) NavigationResolution {
	t.Helper()
	result, err := registry.ResolveNavigation(request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func hasNavigationTarget(result NavigationResolution, id string) bool {
	for _, target := range result.Targets {
		if target.Target.ID == id {
			return true
		}
	}
	return false
}

func hasRegionTarget(result RegionResolution, id string) bool {
	for _, target := range result.Targets {
		if target.Target.ID == id {
			return true
		}
	}
	return false
}
