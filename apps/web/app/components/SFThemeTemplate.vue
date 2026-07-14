<script setup lang="ts">
import type { Component } from 'vue'
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'

const props = defineProps<{
  html?: string
  renderOutput?: ThemeRenderOutput | null
  extensionId?: string
  dataSource?: string
  dataRoute?: string
  /** SSR resolve 注入的插件页面数据（唯一数据来源；禁止客户端再请求插件 route） */
  loaderData?: unknown
  loaderError?: string
}>()

const { t } = useI18n()

// 该映射与 ThemeRuntimeSnapshot 的 reviewed Component Registry 对齐；L2 不在此表。
const islandComponents: Record<string, Component> = {
  'forum.component.home_page': resolveComponent('SFHomePage') as Component,
  'navigation.component.navbar': resolveComponent('SFNavbar') as Component,
  'navigation.component.footer': resolveComponent('SFFooter') as Component,
  'navigation.component.home': resolveComponent('SFHomeNavigation') as Component
}
const legacyIslandBindings = {
  'sf-home-page': { componentId: 'forum.component.home_page' },
  'sf-navbar': { componentId: 'navigation.component.navbar' },
  'sf-footer': { componentId: 'navigation.component.footer' },
  'sf-home-navigation': { componentId: 'navigation.component.home' }
} as const
const allowedComponents = new Set(Object.keys(islandComponents))

const renderState = computed(() => {
  try {
    const nodes = props.renderOutput
      ? parseThemeRenderOutput(props.renderOutput, { allowedComponents })
      : parseLegacyThemeHTML(props.html || '', legacyIslandBindings)
    return { nodes, error: false }
  } catch {
    return { nodes: [], error: true }
  }
})

const ThemeRenderTree = defineComponent({
  name: 'SFThemeRenderTree',
  setup() {
    return () => renderThemeRenderNodes(
      renderState.value.nodes,
      componentId => islandComponents[componentId]
    )
  }
})

const pluginDataError = computed(() => Boolean(props.loaderError))

</script>

<template>
  <div
    class="sf-theme-template"
    :data-extension-id="extensionId || ''"
  >
    <SFAlert
      v-if="pluginDataError || renderState.error"
      variant="danger"
      class="mb-4"
      :title="t('errors.page.serviceUnavailable.title')"
    />
    <ThemeRenderTree v-else />
  </div>
</template>
