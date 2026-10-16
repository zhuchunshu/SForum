<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/themes/useThemeRenderOutput'
import SFPublicSchemaView from './SFPublicSchemaView.vue'

/**
 * Page Registry 插件页正文出口（唯一）。
 *
 * 外壳与正文分离：外壳（裸 main 或宿主公共 chrome）由路由壳按贡献的 layout 选择，
 * 本组件只负责正文本身。这样 `layout: site|bare` 不会复制一份渲染分支，
 * 「恰好一个 navbar/footer」由「只有一个外壳」保证，而不是靠两个分支各自小心。
 */
defineProps<{
  /** 解析失败（SSR 请求本身出错） */
  loadFailed: boolean
  templateHtml: string
  renderOutput?: ThemeRenderOutput
  /** 正文提供方式：template | component（缺省等价 template）。 */
  view?: string
  /** view=component 时包内公开 L2 组件 id。 */
  component?: string
  /** view=schema 时的有界页面描述符；渲染复用 AdminSurface 形状，但不带管理端能力。 */
  schemaView?: Record<string, unknown>
  extensionId: string
  dataSource?: string
  dataRoute?: string
  loaderData?: unknown
  loaderError?: string
  /** 无模板可渲染时的说明文本 */
  fallbackDescription: string
}>()
</script>

<template>
  <SFAlert
    v-if="loadFailed"
    variant="danger"
    title="Failed to load extension page"
  />
  <SFExtensionWidget
    v-else-if="component"
    :extension-id="extensionId"
    :component-id="component"
  />
  <SFPublicSchemaView
    v-else-if="schemaView"
    :descriptor="schemaView"
    :extension-id="extensionId"
  />
  <SFThemeTemplate
    v-else-if="renderOutput || templateHtml"
    :html="templateHtml"
    :render-output="renderOutput"
    :extension-id="extensionId"
    :data-source="dataSource"
    :data-route="dataRoute"
    :loader-data="loaderData"
    :loader-error="loaderError"
  />
  <SFEmptyState
    v-else
    title="Extension page unavailable"
    :description="loaderError || fallbackDescription"
  />
</template>
