package sitechrome

// Navigation v1 constants are frozen before persistence lands. Keeping the
// public vocabulary in the domain package prevents the controller, migration,
// UI, and extension bridge from inventing separate variants.
const (
	NavigationBackupSchemaID = "sforum.site-navigation-backup@1"

	NavigationLocationTopbar  = "public.topbar.primary"
	NavigationLocationSidebar = "public.sidebar.primary"
	NavigationLocationMobile  = "public.mobile.primary"
	NavigationLocationFooter  = "public.footer.primary"

	NavigationSourceCore      = "core"
	NavigationSourceOperator  = "operator"
	NavigationSourceExtension = "extension"
	NavigationSourceDynamic   = "dynamic"

	NavigationVisibilityPublic        = "public"
	NavigationVisibilityAnonymous     = "anonymous"
	NavigationVisibilityAuthenticated = "authenticated"
	NavigationVisibilityPermission    = "permission"

	NavigationLinkCoreRoute      = "coreRoute"
	NavigationLinkInternal       = "internalLink"
	NavigationLinkExternal       = "externalLink"
	NavigationLinkExtensionHost  = "extensionHostLink"
	NavigationLinkExtensionRoute = "extensionRoute"
	NavigationLinkDynamicBlock   = "dynamicBlock"

	NavigationMaxDefinitions  = 200
	NavigationMaxPlacements   = 800
	NavigationMaxSnapshots    = 20
	NavigationMaxLabelRunes   = 80
	NavigationMaxIconRunes    = 120
	NavigationMaxReasonRunes  = 240
	NavigationMaxBackupBytes  = 512 * 1024
	NavigationMaxDynamicItems = 100

	NavigationPreviewChangeLocation    = "location"
	NavigationPreviewChangeDefinitions = "definitions"

	NavigationPreviewWarningExtensionReferenceInert = "extension_reference_inert"
)

// NavigationLocations is deliberately ordered for a consistent editor and
// default catalog. It is not a theme capability list: unsupported locations
// remain configured and are reported to the operator.
func NavigationLocations() []string {
	return []string{
		NavigationLocationTopbar,
		NavigationLocationSidebar,
		NavigationLocationMobile,
		NavigationLocationFooter,
	}
}

// NavigationRecommendedPlacement is the code-owned v1 recommendation. It is
// intentionally independent of migration seed rows so restore-defaults keeps
// working for both fresh and long-lived installations.
type NavigationRecommendedPlacement struct {
	SourceKey string
	Location  string
	Order     int
}

func NavigationRecommendedPlacements() []NavigationRecommendedPlacement {
	return []NavigationRecommendedPlacement{
		{SourceKey: "core.home", Location: NavigationLocationTopbar, Order: 10},
		{SourceKey: "core.categories", Location: NavigationLocationTopbar, Order: 20},
		{SourceKey: "core.tags", Location: NavigationLocationTopbar, Order: 30},
		{SourceKey: "core.home", Location: NavigationLocationSidebar, Order: 10},
		{SourceKey: "core.categories", Location: NavigationLocationSidebar, Order: 20},
		{SourceKey: "core.tags", Location: NavigationLocationSidebar, Order: 30},
		{SourceKey: "core.dynamic.categories", Location: NavigationLocationSidebar, Order: 40},
		{SourceKey: "core.home", Location: NavigationLocationMobile, Order: 10},
		{SourceKey: "core.categories", Location: NavigationLocationMobile, Order: 20},
		{SourceKey: "core.tags", Location: NavigationLocationMobile, Order: 30},
	}
}
