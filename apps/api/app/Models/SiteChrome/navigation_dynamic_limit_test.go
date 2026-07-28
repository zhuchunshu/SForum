package sitechrome

import "testing"

func TestResolvedDynamicNavigationItemCarriesPlacementLimit(t *testing.T) {
	definition := NavigationDefinition{
		SourceKey: "core.dynamic.categories", SourceKind: NavigationSourceDynamic, LinkKind: NavigationLinkDynamicBlock,
		LabelZhCN: "分类",
	}
	placement := NavigationPlacement{
		SourceKey: definition.SourceKey, Location: NavigationLocationSidebar, Enabled: true,
		Visibility: NavigationVisibilityPublic, MaxItems: 8,
	}

	if item := resolvedNavigationItem(definition, placement, "zh-CN"); item.MaxItems != 8 {
		t.Fatalf("resolved dynamic max items = %d, want 8", item.MaxItems)
	}
}
