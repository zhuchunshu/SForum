package toolplugin_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/AI/toolplugin"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

func pluginAIToolFixture() (extensions.Extension, extensions.ManifestAITool) {
	extension := extensions.Extension{
		ID: "fixture.ai-tool", Name: "AI Tool", Version: "1.0.0",
		Type: extensions.TypePlugin, Status: "enabled", PackageDigest: "digest-1",
	}
	declaration := extensions.ManifestAITool{
		ID:              "fixture.ai-tool.order_lookup",
		ContractVersion: "fixture.ai-tool.order_lookup@1",
		Name:            "fixture-order-lookup",
		Description:     "Look up an order by its public number.",
		Parameters:      json.RawMessage(`{"type":"object","properties":{"orderNumber":{"type":"string"}}}`),
		Handler:         "fixture.ai-tool.order_lookup",
		TimeoutMS:       1500,
	}
	return extension, declaration
}

type recordingPluginToolInvoker struct {
	calls  []toolplugin.PluginToolCall
	result toolplugin.PluginToolResult
	err    error
}

func (i *recordingPluginToolInvoker) InvokeVersionedAITool(
	_ context.Context,
	_ extensions.Extension,
	call toolplugin.PluginToolCall,
) (toolplugin.PluginToolResult, error) {
	i.calls = append(i.calls, call)
	return i.result, i.err
}

func TestPluginToolRegistryRejectsAmbiguousAndInvalidNames(t *testing.T) {
	extension, declaration := pluginAIToolFixture()
	invoker := &recordingPluginToolInvoker{}
	tool, err := toolplugin.NewPluginTool(extension, declaration, invoker)
	if err != nil {
		t.Fatalf("valid declaration must construct: %v", err)
	}
	registry := toolplugin.NewPluginToolRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("first registration must succeed: %v", err)
	}
	if err := registry.Register(tool); !errors.Is(err, toolplugin.ErrAIToolDeclarationInvalid) {
		t.Fatalf("duplicate tool name must be rejected, got %v", err)
	}

	invalid := declaration
	invalid.ID = "fixture.ai-tool.other"
	invalid.ContractVersion = invalid.ID + "@1"
	invalid.Handler = invalid.ID
	invalid.Name = "fixture.order.lookup" // 点号不在供应商接受的字符集里
	if _, err := toolplugin.NewPluginTool(extension, invalid, invoker); !errors.Is(err, toolplugin.ErrAIToolDeclarationInvalid) {
		t.Fatalf("dotted tool name must be rejected, got %v", err)
	}
}

// 插件工具不得覆盖 Core 工具：模型看到的工具名必须唯一，否则一次调用该派给谁
// 是不确定的，而运营者也看不出 forum-search 被换掉了。
func TestPluginToolRegistryNeverShadowsCoreTools(t *testing.T) {
	extension, declaration := pluginAIToolFixture()
	declaration.Name = "forum-search"
	invoker := &recordingPluginToolInvoker{}
	tool, err := toolplugin.NewPluginTool(extension, declaration, invoker)
	if err != nil {
		t.Fatalf("declaration should construct: %v", err)
	}
	coreTool := &stubTool{definition: supportai.ToolDefinition{Name: "forum-search", Description: "Core search."}}
	registry := supportai.NewToolRegistry()
	if err := registry.Register(coreTool); err != nil {
		t.Fatalf("register core tool: %v", err)
	}
	pluginRegistry := toolplugin.NewPluginToolRegistry()
	if err := pluginRegistry.Register(tool); err != nil {
		t.Fatalf("plugin registry registration: %v", err)
	}
	if err := pluginRegistry.Apply(registry); err != nil {
		t.Fatalf("apply must skip the conflict instead of failing: %v", err)
	}
	resolved, ok := registry.Lookup("forum-search")
	if !ok || resolved != coreTool {
		t.Fatalf("core tool was replaced by a plugin tool: %#v", resolved)
	}
}

func TestPluginToolInvokeKeepsDeclarationBindingAndFailureShape(t *testing.T) {
	extension, declaration := pluginAIToolFixture()
	invoker := &recordingPluginToolInvoker{result: toolplugin.PluginToolResult{Content: "order 42 shipped", IsError: false}}
	tool, err := toolplugin.NewPluginTool(extension, declaration, invoker)
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.Invoke(context.Background(), supportai.ToolCall{
		ID: "call-1", Name: declaration.Name, Arguments: `{"orderNumber":"42"}`,
	}, supportai.ToolContext{ViewerUserID: 7})
	if err != nil || result.IsError {
		t.Fatalf("invoke result=%#v err=%v", result, err)
	}
	if len(invoker.calls) != 1 {
		t.Fatalf("invoker calls = %d", len(invoker.calls))
	}
	call := invoker.calls[0]
	if call.DeclarationID != declaration.ID || call.ContractVersion != declaration.ContractVersion ||
		call.Handler != declaration.Handler || call.Name != declaration.Name ||
		call.Arguments != `{"orderNumber":"42"}` ||
		call.Timeout != time.Duration(declaration.TimeoutMS)*time.Millisecond {
		t.Fatalf("dispatch drifted from the declaration: %#v", call)
	}

	// 模型捏造工具名：给模型一句可解释的失败，不向插件派发。
	mismatched, err := tool.Invoke(context.Background(), supportai.ToolCall{Name: "other-tool"}, supportai.ToolContext{})
	if err != nil || !mismatched.IsError {
		t.Fatalf("name mismatch must be a tool-level error: result=%#v err=%v", mismatched, err)
	}
	if len(invoker.calls) != 1 {
		t.Fatalf("mismatched name must not reach the plugin, calls=%d", len(invoker.calls))
	}

	// 插件内部故障：返回 error（编排层给模型通用说明），不把插件错误串当工具结果。
	invoker.err = errors.New("plugin exploded with secret internal detail")
	if _, err := tool.Invoke(context.Background(), supportai.ToolCall{Name: declaration.Name}, supportai.ToolContext{}); err == nil {
		t.Fatal("plugin failure must surface as an error, not as model-visible content")
	}
}

// 清单侧与运行期共用同一份边界：这些常量一旦漂移，就会出现「能声明、不能注册」
// 或反过来的规则分裂。两侧都是字面量，因此这里必须把它们钉死。
func TestAIToolContractLimitsStayAlignedAcrossPackages(t *testing.T) {
	aligned := []struct {
		name string
		host int
		sdk  int
	}{
		{"declarations per extension", toolplugin.AIToolsMaximumPerExtension, pluginv2.HostAIToolsMaximumPerExtension},
		{"tool name length", toolplugin.AIToolNameMaximumLength, pluginv2.HostAIToolNameMaximumLength},
		{"description bytes", supportai.MaxToolDescriptionBytes, pluginv2.HostAIToolDescriptionMaximum},
		{"parameters bytes", supportai.MaxToolSchemaBytes, pluginv2.HostAIToolSchemaMaximumBytes},
		{"timeout milliseconds", extensionmanifest.AIToolMaximumTimeoutMS, pluginv2.HostAIToolMaximumTimeoutMS},
	}
	for _, item := range aligned {
		if item.host != item.sdk {
			t.Fatalf("%s drifted: host=%d sdk=%d", item.name, item.host, item.sdk)
		}
	}
	if toolplugin.AIToolsMaximumPerExtension != extensionmanifest.AIToolsMaximum ||
		toolplugin.AIToolNameMaximumLength != extensionmanifest.AIToolNameMaximumLength ||
		supportai.MaxToolSchemaBytes != extensionmanifest.AIToolSchemaMaximumBytes ||
		supportai.MaxToolDescriptionBytes != extensionmanifest.AIToolDescriptionMaximumBytes {
		t.Fatal("manifest declaration limits drifted from the runtime registration limits")
	}
}

// slot / operation / schema 是 Host 与插件之间的线上契约：任何一侧单独改名都会
// 让工具调用变成「宿主发出去了，插件认不出来」。
func TestAIToolWireContractIsPinned(t *testing.T) {
	if toolplugin.AIToolSlot != "sforum.ai.tool" || toolplugin.AIToolOperation != "invoke" ||
		toolplugin.AIToolContract != "sforum.ai.tool@1" ||
		toolplugin.AIToolRequestSchema != "sforum.ai.tool.invoke.request@1" ||
		toolplugin.AIToolResponseSchema != "sforum.ai.tool.invoke.response@1" {
		t.Fatalf("ai tool wire contract drifted: slot=%q op=%q contract=%q req=%q res=%q",
			toolplugin.AIToolSlot, toolplugin.AIToolOperation, toolplugin.AIToolContract,
			toolplugin.AIToolRequestSchema, toolplugin.AIToolResponseSchema)
	}
	if pluginv2.HostAIToolSlot != toolplugin.AIToolSlot || pluginv2.HostAIToolOperation != toolplugin.AIToolOperation ||
		pluginv2.HostAIToolRequestSchemaRef != toolplugin.AIToolRequestSchema ||
		pluginv2.HostAIToolResponseSchemaRef != toolplugin.AIToolResponseSchema {
		t.Fatal("sdk AI tool slot or schema refs drifted from the host contract")
	}
}

// 声明校验必须与清单校验接受同一集合：能通过清单校验的声明，登记表也应当接受。
func TestValidateAIToolDeclarationAcceptsWhatTheManifestAccepts(t *testing.T) {
	extension, declaration := pluginAIToolFixture()
	if err := toolplugin.ValidateAIToolDeclaration(extension.ID, declaration); err != nil {
		t.Fatalf("declaration rejected by the registry: %v", err)
	}
	// 声明挂在别的插件命名空间下时必须被拒：一个插件不得注册另一个插件的 handler。
	if err := toolplugin.ValidateAIToolDeclaration("other.plugin", declaration); err == nil {
		t.Fatal("foreign namespace must be rejected")
	}
	declaration.TimeoutMS = 0
	if err := toolplugin.ValidateAIToolDeclaration(extension.ID, declaration); err == nil {
		t.Fatal("zero timeout must be rejected")
	}
	if !strings.Contains(toolplugin.AIToolContract, "sforum.ai.tool") {
		t.Fatal("contract identity drifted")
	}
}

type stubTool struct {
	definition supportai.ToolDefinition
}

func (t *stubTool) Definition() supportai.ToolDefinition { return t.definition }

func (t *stubTool) Invoke(context.Context, supportai.ToolCall, supportai.ToolContext) (supportai.ToolResult, error) {
	return supportai.ToolResult{}, nil
}
