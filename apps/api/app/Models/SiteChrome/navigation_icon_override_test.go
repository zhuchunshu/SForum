package sitechrome

import "testing"

func TestResolvedNavigationItemCanSuppressDefinitionIcon(t *testing.T) {
	definition := NavigationDefinition{
		SourceKey:  "core.home",
		SourceKind: NavigationSourceCore,
		LinkKind:   NavigationLinkCoreRoute,
		LabelZhCN:  "首页",
		Href:       "/",
		Icon:       "i-lucide-layout-list",
	}
	placement := NavigationPlacement{
		SourceKey:  definition.SourceKey,
		Location:   NavigationLocationTopbar,
		Enabled:    true,
		Visibility: NavigationVisibilityPublic,
		IconHidden: true,
	}

	if item := resolvedNavigationItem(definition, placement, "zh-CN"); item.Icon != "" || !item.IconHidden {
		t.Fatalf("suppressed placement icon = %q hidden=%t, want empty and hidden", item.Icon, item.IconHidden)
	}
}
