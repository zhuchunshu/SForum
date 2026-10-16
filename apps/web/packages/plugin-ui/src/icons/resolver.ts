import type { App, InjectionKey } from 'vue'

/**
 * 宿主图标解析器契约。
 *
 * 与宿主 `AdminPageBridgeV1.icons` / `PublicFrontendBridgeV1.icons` 形状一致，
 * 但 SDK 不依赖宿主包：这是一个纯结构契约。解析不到的名称会从结果中缺失，
 * 调用方必须容忍缺项。
 */
export type PluginIconResolver = Readonly<{
  resolve: (names: readonly string[]) => Promise<Readonly<Record<string, string>>>
}>

export const PLUGIN_ICON_RESOLVER: InjectionKey<PluginIconResolver> = Symbol('sforum.plugin-icon-resolver')

/**
 * 宿主认可的图标名语法。这是对宿主侧 `HOST_ICON_NAME_PATTERN` 的镜像，
 * SDK 作为独立的 npm 包不能从宿主导入，因此保留一份副本；
 * 两侧必须同时修改，否则插件会在运行时静默拿不到图标。
 */
export const PLUGIN_ICON_NAME_PATTERN = /^i-(lucide|tabler)-[a-z0-9]+(?:-[a-z0-9]+)*$/

export function providePluginIcons(app: App, resolver?: PluginIconResolver) {
  if (resolver) {
    app.provide(PLUGIN_ICON_RESOLVER, resolver)
  }
}
