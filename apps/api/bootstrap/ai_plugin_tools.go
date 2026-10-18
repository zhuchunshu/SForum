package bootstrap

import (
	"fmt"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
	toolplugin "github.com/zhuchunshu/sforum/apps/api/app/Support/AI/toolplugin"
)

// 具体传输适配器（toolplugin.PluginToolInvoker → 扩展运行时的 versioned
// ProviderCall）放在 api_assembly_domains.go：那里是已获准 import 扩展运行时的
// 装配文件，本文件保持只依赖契约与登记表，不新增对 legacy 运行时的 import。

// registerPluginAITools 把已启用插件声明的 AI 工具登记进宿主工具登记表。
//
// 它刻意在装配期一次性完成，而不是每次补全时去问运行时「现在有哪些工具」：
// 工具集是清单里的静态事实，模型看到的工具声明必须在一次补全内保持一致，否则
// 同一轮对话里工具列表会变，而模型可能已经基于旧列表发起了调用。代价是插件启用/
// 停用要等 API 重启才反映到工具表——与运行时全量发布同步属于后续工作项。
//
// 登记不等于暴露：模型在请求里看到的工具由用途自己的白名单决定
// （RunInput.ToolNames），Core 用途的白名单不含插件工具，因此这里登记的结果
// 不会改变任何现有对话的行为。
//
// 单条声明非法不阻断整体装配：清单在安装时已经校验过，这里出现非法声明意味着
// 数据库里的历史产物与当前校验规则不一致，跳过它并继续比让整个 API 起不来更有
// 价值——但会返回错误集合，由调用方决定是否记日志。
func registerPluginAITools(
	registry *supportai.ToolRegistry,
	invoker toolplugin.PluginToolInvoker,
	items []extensions.Extension,
) []error {
	if registry == nil || invoker == nil {
		return nil
	}
	pluginRegistry := toolplugin.NewPluginToolRegistry()
	var failures []error
	for _, item := range items {
		if item.Type != extensions.TypePlugin || len(item.Manifest.AITools) == 0 {
			continue
		}
		// 停用的插件没有运行时可派发：登记它只会造出「看得见、调不动」的工具，
		// 并在控制台里把运营者引到一个必然失败的白名单条目上。
		if item.Status != extensions.StatusEnabled {
			continue
		}
		extension := item
		for _, declaration := range extension.Manifest.AITools {
			tool, err := toolplugin.NewPluginTool(extension, declaration, invoker)
			if err != nil {
				failures = append(failures, fmt.Errorf("%s/%s: %w", extension.ID, declaration.ID, err))
				continue
			}
			if err := pluginRegistry.Register(tool); err != nil {
				failures = append(failures, fmt.Errorf("%s/%s: %w", extension.ID, declaration.ID, err))
			}
		}
	}
	if err := pluginRegistry.Apply(registry); err != nil {
		failures = append(failures, err)
	}
	return failures
}
