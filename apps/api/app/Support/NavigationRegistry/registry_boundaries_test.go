package navigationregistry

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryRemoveRequiresExactArtifactTuple(t *testing.T) {
	active := publication("exact.navigation", false, 'a')
	active.Navigation = []NavigationDeclaration{
		navigation("exact.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	registry := New()
	if _, err := registry.Publish(active); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	tests := []struct {
		name string
		edit func(*Artifact)
		want error
	}{
		{name: "version", edit: func(value *Artifact) { value.ExtensionVersion = "1.0.1" }, want: ErrArtifactConflict},
		{name: "package", edit: func(value *Artifact) { value.PackageDigest = strings.Repeat("b", 64) }, want: ErrArtifactConflict},
		{name: "impact", edit: func(value *Artifact) { value.ImpactDigest = strings.Repeat("c", 64) }, want: ErrArtifactConflict},
		{name: "core", edit: func(value *Artifact) { value.Core = true }, want: ErrInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stale := active.Artifact
			test.edit(&stale)
			revision, removed, err := registry.Remove(stale)
			if !errors.Is(err, test.want) || removed || revision != before.Revision {
				t.Fatalf("remove: revision=%d removed=%t err=%v", revision, removed, err)
			}
			if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
				t.Fatalf("rejected remove changed snapshot: before=%#v after=%#v", before, after)
			}
		})
	}
	if revision, removed, err := registry.Remove(active.Artifact); err != nil || !removed || revision != before.Revision+1 {
		t.Fatalf("exact remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
}

func TestRegistryExactDisabledAddTargetHidesWholeSubtree(t *testing.T) {
	plugin := publication("tree.navigation", false, 'a')
	plugin.Navigation = []NavigationDeclaration{
		navigation("tree.navigation.menu.parent", NavigationKindMenu, ActionAdd, "", 0),
		navigation("tree.navigation.item.child", NavigationKindItem, ActionAdd, "tree.navigation.menu.parent", 0),
	}
	plugin.Regions = []RegionDeclaration{
		region("tree.navigation.region.parent", RegionKindSidebar, ActionAdd, ""),
		region("tree.navigation.region.child", RegionKindWidget, ActionAdd, "tree.navigation.region.parent"),
	}
	registry := New()
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}

	stale := providerRef(plugin.Navigation[0].ID, plugin)
	stale.Artifact.ImpactDigest = strings.Repeat("b", 64)
	staleResult := mustResolveNavigation(t, registry, NavigationResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{stale},
	}})
	if len(staleResult.Targets) != 2 {
		t.Fatalf("stale exact ref hid subtree=%#v", staleResult.Targets)
	}

	navigationResult := mustResolveNavigation(t, registry, NavigationResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{providerRef(plugin.Navigation[0].ID, plugin)},
	}})
	if len(navigationResult.Targets) != 0 {
		t.Fatalf("disabled navigation parent retained subtree=%#v", navigationResult.Targets)
	}
	regionResult, err := registry.ResolveRegions(RegionResolveRequest{Visibility: VisibilityInput{
		DisabledProviders: []ProviderRef{providerRef(plugin.Regions[0].ID, plugin)},
	}})
	if err != nil || len(regionResult.Targets) != 0 {
		t.Fatalf("disabled region parent retained subtree=%#v err=%v", regionResult.Targets, err)
	}
}

func TestRegistryEnforcesPublicationDependencyAndContributionLimits(t *testing.T) {
	publications := make([]Publication, 0, maxPublications+1)
	for index := 0; index < maxPublications+1; index++ {
		publications = append(publications, publication(fmt.Sprintf("limit.plugin.%03d", index), false, 'a'))
	}
	if _, err := New().ReplaceAll(publications[:maxPublications]); err != nil {
		t.Fatalf("maximum publications rejected=%v", err)
	}
	if _, err := New().ReplaceAll(publications); !errors.Is(err, ErrInvalid) {
		t.Fatalf("publication overflow=%v", err)
	}

	dependencyBound := publication("limit.dependencies", false, 'b')
	for index := 0; index < maxDependenciesPerPublication+1; index++ {
		dependencyBound.Dependencies = append(dependencyBound.Dependencies, Dependency{
			Capability: fmt.Sprintf("limit.capability.%03d", index), Version: "1.0.0", Kind: DependencyProvides,
		})
	}
	if _, err := New().Publish(Publication{
		Artifact: dependencyBound.Artifact, Dependencies: dependencyBound.Dependencies[:maxDependenciesPerPublication],
	}); err != nil {
		t.Fatalf("maximum dependencies rejected=%v", err)
	}
	if _, err := New().Publish(dependencyBound); !errors.Is(err, ErrInvalid) {
		t.Fatalf("dependency overflow=%v", err)
	}

	contributionBound := publication("limit.contributions", false, 'c')
	for index := 0; index < maxContributions+1; index++ {
		id := fmt.Sprintf("limit.contributions.item.%04d", index)
		contributionBound.Navigation = append(contributionBound.Navigation, navigation(id, NavigationKindItem, ActionAdd, "", index))
	}
	if _, err := New().Publish(Publication{
		Artifact: contributionBound.Artifact, Navigation: contributionBound.Navigation[:maxContributions],
	}); err != nil {
		t.Fatalf("maximum contributions rejected=%v", err)
	}
	if _, err := New().Publish(contributionBound); !errors.Is(err, ErrInvalid) {
		t.Fatalf("contribution overflow=%v", err)
	}
}

func TestRegistryDetachesPublishedInput(t *testing.T) {
	input := publication("detached.navigation", false, 'a')
	input.Dependencies = []Dependency{{Capability: "detached.capability", Version: "1.0.0", Kind: DependencyProvides}}
	input.Navigation = []NavigationDeclaration{
		navigation("detached.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	input.Regions = []RegionDeclaration{
		region("detached.navigation.region.base", RegionKindWidget, ActionAdd, ""),
	}
	registry := New()
	if _, err := registry.Publish(input); err != nil {
		t.Fatal(err)
	}

	input.Artifact.PackageDigest = strings.Repeat("f", 64)
	input.Dependencies[0].Capability = "forged.capability"
	input.Navigation[0].Label = "forged navigation"
	input.Regions[0].Label = "forged region"
	snapshot := registry.Snapshot()
	if snapshot.Navigation[0].Label == "forged navigation" || snapshot.Regions[0].Label == "forged region" {
		t.Fatalf("caller input mutation escaped=%#v", snapshot)
	}
	if revision, err := registry.Publish(Publication{
		Artifact:     snapshot.Navigation[0].Artifact,
		Dependencies: []Dependency{{Capability: "detached.capability", Version: "1.0.0", Kind: DependencyProvides}},
		Navigation:   []NavigationDeclaration{snapshot.Navigation[0].NavigationDeclaration},
		Regions:      []RegionDeclaration{snapshot.Regions[0].RegionDeclaration},
	}); err != nil || revision != snapshot.Revision {
		t.Fatalf("detached idempotent publish: revision=%d err=%v", revision, err)
	}
}

func TestRegistryBindsContractsAndBoundsHostPolicyInputs(t *testing.T) {
	plugin := publication("bounded.navigation", false, 'a')
	item := navigation("bounded.navigation.item.base", NavigationKindItem, ActionAdd, "", 0)
	item.ContractVersion = "bounded.navigation.item.other@1"
	plugin.Navigation = []NavigationDeclaration{item}
	if _, err := New().Publish(plugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("mismatched navigation contract=%v", err)
	}

	regionPlugin := publication("bounded.region", false, 'b')
	itemRegion := region("bounded.region.region.base", RegionKindWidget, ActionAdd, "")
	itemRegion.ContractVersion = "bounded.region.region.other@1"
	regionPlugin.Regions = []RegionDeclaration{itemRegion}
	if _, err := New().Publish(regionPlugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("mismatched region contract=%v", err)
	}

	item.ContractVersion = item.ID + "@1"
	item.Priority = maxPriority + 1
	plugin.Navigation = []NavigationDeclaration{item}
	if _, err := New().Publish(plugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("navigation priority overflow=%v", err)
	}
	itemRegion.ContractVersion = itemRegion.ID + "@1"
	itemRegion.Priority = -maxPriority - 1
	regionPlugin.Regions = []RegionDeclaration{itemRegion}
	if _, err := New().Publish(regionPlugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("region priority overflow=%v", err)
	}

	kinds := make([]string, maxKindFilters+1)
	for index := range kinds {
		kinds[index] = NavigationKindItem
	}
	if _, err := New().ResolveNavigation(NavigationResolveRequest{Kinds: kinds}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("kind filter overflow=%v", err)
	}
}
