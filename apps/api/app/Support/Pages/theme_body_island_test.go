package pages

import "testing"

// 宿主岛必须同时出现在模板校验白名单与生产绑定表里：
// 只在一侧出现会让主题模板要么校验失败、要么渲染时找不到组件。
// 插件正文岛是主题外壳的正文出口，同样受这条配对约束。
func TestEveryAllowlistedHostIslandHasAProductionBinding(t *testing.T) {
	bindings := productionThemeIslandBindings()
	for tag := range allowedHostIslands {
		binding, ok := bindings[tag]
		if !ok {
			t.Fatalf("allowlisted host island %q has no production binding", tag)
		}
		if binding.ComponentID == "" {
			t.Fatalf("host island %q has an empty component id", tag)
		}
	}
}

func TestPluginPageBodyIslandIsAllowlistedAndBound(t *testing.T) {
	if _, ok := allowedHostIslands["sf-plugin-page-body"]; !ok {
		t.Fatal("sf-plugin-page-body must be allowlisted for theme templates")
	}
	binding, ok := productionThemeIslandBindings()["sf-plugin-page-body"]
	if !ok {
		t.Fatal("sf-plugin-page-body must have a production binding")
	}
	if binding.ComponentID != "plugin.component.page_body" {
		t.Fatalf("plugin body island component id = %q", binding.ComponentID)
	}
}
