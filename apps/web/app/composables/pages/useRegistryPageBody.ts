import { computed, inject, provide, type ComputedRef, type InjectionKey } from 'vue'
import type { ThemeRenderOutput } from '~/composables/themes/useThemeRenderOutput'

/**
 * 插件公开页的正文契约。
 *
 * 外壳与正文分离后，正文不能再靠 slot 传递：主题外壳模板里正文的位置由
 * `sf-plugin-page-body` 岛决定，岛是独立组件、拿不到父组件的 slot 内容。
 * 因此正文以「数据」形式由外壳 provide、由岛 inject。
 */
export type RegistryPageBody = {
  loadFailed: boolean
  templateHtml: string
  renderOutput?: ThemeRenderOutput
  /** 正文提供方式：template | component（缺省等价 template）。 */
  view?: string
  /** view=component 时包内公开 L2 组件 id；正文由正文出口挂载该组件。 */
  component?: string
  /** view=schema 时由 API 返回的有界页面描述符（list|detail|form|notice）。 */
  schemaView?: Record<string, unknown>
  extensionId: string
  dataSource?: string
  dataRoute?: string
  loaderData?: unknown
  loaderError?: string
  fallbackDescription: string
}

export const REGISTRY_PAGE_BODY: InjectionKey<ComputedRef<RegistryPageBody>> = Symbol('sforum.registry-page-body')

/** 外壳侧：把正文以响应式形式提供给岛。传 getter 而非对象，避免父级重新解析后引用失效。 */
export function provideRegistryPageBody(body: () => RegistryPageBody) {
  provide(REGISTRY_PAGE_BODY, computed(body))
}

/** 岛侧：取当前正文；不在外壳内部时返回 null。 */
export function useRegistryPageBody() {
  return inject(REGISTRY_PAGE_BODY, null)
}
