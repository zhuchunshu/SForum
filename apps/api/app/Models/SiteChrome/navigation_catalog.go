package sitechrome

// CoreNavigationDefinitions is code-owned so recommended defaults and recovery
// do not depend on a historical migration seed. Operator records only provide
// bounded presentation overrides for these stable keys.
func CoreNavigationDefinitions() []NavigationDefinition {
	return []NavigationDefinition{
		{SourceKey: "core.home", SourceKind: NavigationSourceCore, LinkKind: NavigationLinkCoreRoute, LabelZhCN: "首页", LabelEnUS: "Home", Href: "/", Icon: "i-lucide-layout-list"},
		{SourceKey: "core.categories", SourceKind: NavigationSourceCore, LinkKind: NavigationLinkCoreRoute, LabelZhCN: "分类", LabelEnUS: "Categories", Href: "/categories", Icon: "i-lucide-layout-grid"},
		{SourceKey: "core.tags", SourceKind: NavigationSourceCore, LinkKind: NavigationLinkCoreRoute, LabelZhCN: "标签", LabelEnUS: "Tags", Href: "/tags", Icon: "i-lucide-tags"},
		{SourceKey: "core.dynamic.categories", SourceKind: NavigationSourceDynamic, LinkKind: NavigationLinkDynamicBlock, LabelZhCN: "分类", LabelEnUS: "Categories", Icon: "i-lucide-folders"},
		{SourceKey: "core.terms", SourceKind: NavigationSourceCore, LinkKind: NavigationLinkCoreRoute, LabelZhCN: "服务条款", LabelEnUS: "Terms", Href: "/terms", Icon: "i-lucide-file-text"},
		{SourceKey: "core.privacy", SourceKind: NavigationSourceCore, LinkKind: NavigationLinkCoreRoute, LabelZhCN: "隐私政策", LabelEnUS: "Privacy", Href: "/privacy", Icon: "i-lucide-shield-check"},
		{SourceKey: "core.guidelines", SourceKind: NavigationSourceCore, LinkKind: NavigationLinkCoreRoute, LabelZhCN: "社区指南", LabelEnUS: "Guidelines", Href: "/guidelines", Icon: "i-lucide-book-open"},
	}
}
