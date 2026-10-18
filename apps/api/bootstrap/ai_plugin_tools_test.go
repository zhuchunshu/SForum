package bootstrap

import (
	"context"
	"encoding/json"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
	toolplugin "github.com/zhuchunshu/sforum/apps/api/app/Support/AI/toolplugin"
)

type bootstrapPluginToolInvoker struct {
	calls []toolplugin.PluginToolCall
}

func (i *bootstrapPluginToolInvoker) InvokeVersionedAITool(
	_ context.Context,
	_ extensions.Extension,
	call toolplugin.PluginToolCall,
) (toolplugin.PluginToolResult, error) {
	i.calls = append(i.calls, call)
	return toolplugin.PluginToolResult{Content: "ok"}, nil
}

func bootstrapAIToolPlugin(id, name string) extensions.Extension {
	extension := extensions.Extension{
		ID: id, Name: id, Version: "1.0.0", Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, PackageDigest: "digest",
	}
	extension.Manifest.AITools = []extensions.ManifestAITool{{
		ID: id + ".lookup", ContractVersion: id + ".lookup@1", Name: name,
		Description: "Look something up.",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		Handler:     id + ".lookup", TimeoutMS: 1000,
	}}
	return extension
}

func TestRegisterPluginAIToolsRegistersOnlyValidPluginDeclarations(t *testing.T) {
	registry := supportai.NewToolRegistry()
	invoker := &bootstrapPluginToolInvoker{}

	valid := bootstrapAIToolPlugin("plugin.ai-tool.valid", "valid-order-lookup")
	invalid := bootstrapAIToolPlugin("plugin.ai-tool.invalid", "invalid-order-lookup")
	invalid.Manifest.AITools[0].Name = "invalid.order.lookup" // 点号：供应商会直接 400
	theme := bootstrapAIToolPlugin("theme.ai-tool", "theme-order-lookup")
	theme.Type = extensions.TypeTheme
	undeclared := extensions.Extension{ID: "plugin.plain", Type: extensions.TypePlugin}
	disabled := bootstrapAIToolPlugin("plugin.ai-tool.disabled", "disabled-order-lookup")
	disabled.Status = extensions.StatusDisabled

	failures := registerPluginAITools(registry, invoker, []extensions.Extension{valid, invalid, theme, undeclared, disabled})
	if len(failures) != 1 {
		t.Fatalf("invalid declaration should be reported once, got %#v", failures)
	}
	if _, ok := registry.Lookup("valid-order-lookup"); !ok {
		t.Fatal("valid plugin tool must be registered in the host tool registry")
	}
	for _, absent := range []string{"invalid-order-lookup", "theme-order-lookup", "disabled-order-lookup"} {
		if _, ok := registry.Lookup(absent); ok {
			t.Fatalf("%s must not be registered", absent)
		}
	}
	// 登记后必须能被模型侧白名单枚举到：Descriptors 只回白名单内的工具。
	descriptors := registry.Descriptors([]string{"valid-order-lookup", "missing-tool"})
	if len(descriptors) != 1 || descriptors[0].Name != "valid-order-lookup" ||
		descriptors[0].Description != "Look something up." {
		t.Fatalf("descriptors = %#v", descriptors)
	}
}

func TestRegisterPluginAIToolsIsInertWithoutRegistryOrInvoker(t *testing.T) {
	plugin := bootstrapAIToolPlugin("plugin.ai-tool.valid", "valid-order-lookup")
	if failures := registerPluginAITools(nil, &bootstrapPluginToolInvoker{}, []extensions.Extension{plugin}); failures != nil {
		t.Fatalf("nil registry must be inert, got %#v", failures)
	}
	registry := supportai.NewToolRegistry()
	if failures := registerPluginAITools(registry, nil, []extensions.Extension{plugin}); failures != nil {
		t.Fatalf("nil invoker must be inert, got %#v", failures)
	}
	if _, ok := registry.Lookup("valid-order-lookup"); ok {
		t.Fatal("nothing may be registered without an invoker")
	}
}
