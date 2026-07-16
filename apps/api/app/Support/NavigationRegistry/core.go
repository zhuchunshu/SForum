package navigationregistry

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	CoreNavigationExtensionID = "core.navigation"

	CorePrimaryMenuID       = "core.navigation.menu.primary"
	CoreBreadcrumbsID       = "core.navigation.breadcrumb.primary"
	CoreHeaderNavigationID  = "core.navigation.header.primary"
	CoreFooterNavigationID  = "core.navigation.footer.primary"
	CoreSidebarNavigationID = "core.navigation.sidebar.primary"

	CoreMenuRegionID    = "core.navigation.region.menu.primary"
	CoreHeaderRegionID  = "core.navigation.region.header.primary"
	CoreFooterRegionID  = "core.navigation.region.footer.primary"
	CoreSidebarRegionID = "core.navigation.region.sidebar.primary"
	CoreWidgetRegionID  = "core.navigation.region.widget.primary"
)

// CorePublication freezes structural Host identities only. Product-specific
// links remain SiteChrome data and are attached to CorePrimaryMenuID at
// composition time, so deployments do not inherit hard-coded taxonomy policy.
func CorePublication() Publication {
	digestBytes := sha256.Sum256([]byte(SchemaVersion + "\x00core-navigation-targets@1"))
	digest := hex.EncodeToString(digestBytes[:])
	artifact, err := NewCoreArtifact(CoreNavigationExtensionID, "1.0.0", digest, digest)
	if err != nil {
		panic(err)
	}
	return Publication{
		Artifact: artifact,
		Navigation: []NavigationDeclaration{
			coreNavigation(CorePrimaryMenuID, NavigationKindMenu, "Primary menu", "主导航"),
			coreNavigation(CoreBreadcrumbsID, NavigationKindBreadcrumb, "Breadcrumbs", "面包屑"),
			coreNavigation(CoreHeaderNavigationID, NavigationKindHeader, "Header", "页头"),
			coreNavigation(CoreFooterNavigationID, NavigationKindFooter, "Footer", "页脚"),
			coreNavigation(CoreSidebarNavigationID, NavigationKindSidebar, "Sidebar", "侧边栏"),
		},
		Regions: []RegionDeclaration{
			coreRegion(CoreMenuRegionID, RegionKindMenu, "Menu region", "菜单区域", false),
			coreRegion(CoreHeaderRegionID, RegionKindHeader, "Header region", "页头区域", true),
			coreRegion(CoreFooterRegionID, RegionKindFooter, "Footer region", "页脚区域", true),
			coreRegion(CoreSidebarRegionID, RegionKindSidebar, "Sidebar region", "侧边栏区域", true),
			coreRegion(CoreWidgetRegionID, RegionKindWidget, "Widget region", "小组件区域", true),
		},
	}
}

func coreNavigation(id, kind, english, chinese string) NavigationDeclaration {
	return NavigationDeclaration{
		ID: id, ContractVersion: id + "@1", Kind: kind, Action: ActionAdd,
		Label: english, Labels: map[string]string{"en-US": english, "zh-CN": chinese}, Visibility: VisibilityPublic,
	}
}

func coreRegion(id, kind, english, chinese string, multiple bool) RegionDeclaration {
	return RegionDeclaration{
		ID: id, ContractVersion: id + "@1", Kind: kind, Action: ActionAdd,
		Label: english, Labels: map[string]string{"en-US": english, "zh-CN": chinese},
		Visibility: VisibilityPublic, Multiple: multiple,
	}
}
