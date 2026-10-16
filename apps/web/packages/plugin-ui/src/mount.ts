import { createApp, type Component } from 'vue'

import { providePluginIcons, type PluginIconResolver } from './icons/resolver'

/**
 * 插件页面挂载桥接器的最小形状。宿主的管理端页面桥接与公共页面桥接都满足它；
 * SDK 不依赖任何一个宿主包，因此只声明自己真正需要的那一部分。
 */
export type PluginPageMountBridge = Readonly<{
  icons?: PluginIconResolver
  [key: string]: unknown
}>

export type PluginPageMountCleanup = () => void

/**
 * 把 Vue 组件挂载到宿主的页面 body 容器。
 *
 * 相比手写 `createApp(...)`，这里额外完成图标解析器的依赖注入，因此作者不需要
 * 在每次生成 `admin.ts` 时自己接线，也不会因为忘记注入而静默丢掉图标。
 */
export function mountPluginPage(
  component: Component,
  target: HTMLElement,
  bridge: PluginPageMountBridge
): PluginPageMountCleanup {
  const app = createApp(component, { bridge })
  providePluginIcons(app, bridge.icons)
  app.mount(target)
  return () => app.unmount()
}

export type PluginPageModule = Readonly<{
  apiVersion: number
  mount: (target: HTMLElement, bridge: PluginPageMountBridge) => PluginPageMountCleanup
}>

/**
 * 生成宿主 Micro-frontend 契约要求的模块形状：
 *
 * ```ts
 * export const { apiVersion, mount } = definePluginPage(AdminDashboard, ADMIN_MICRO_FRONTEND_API_VERSION)
 * ```
 *
 * 宿主在 `import()` 之后按名称读取 `apiVersion` 与 `mount`，因此不能用默认导出。
 */
export function definePluginPage(component: Component, apiVersion: number): PluginPageModule {
  return {
    apiVersion,
    mount: (target, bridge) => mountPluginPage(component, target, bridge)
  }
}
