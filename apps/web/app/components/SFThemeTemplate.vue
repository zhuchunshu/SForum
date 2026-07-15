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
const slots = useSlots()

// 受保护表单岛只决定核心页面在主题 HTML 中的位置，不接收凭证或会话 props。
const HostPageIsland = defineComponent({
  name: 'SFHostPageIsland',
  setup() {
    return () => slots.default?.() ?? []
  }
})

// 该映射与 ThemeRuntimeSnapshot 的 reviewed Component Registry 对齐。
const islandComponents: Record<string, Component> = {
  'forum.component.home_page': resolveComponent('SFHomePage') as Component,
  'navigation.component.navbar': resolveComponent('SFNavbar') as Component,
  'navigation.component.footer': resolveComponent('SFFooter') as Component,
  'navigation.component.home': resolveComponent('SFHomeNavigation') as Component,
  'forum.component.topic_composer': HostPageIsland,
  'profile.component.settings_form': HostPageIsland,
  'identity.component.security_settings': HostPageIsland,
  'identity.component.login_form': HostPageIsland,
  'identity.component.register_form': HostPageIsland,
  'identity.component.recovery_request_form': HostPageIsland,
  'identity.component.recovery_confirm_form': HostPageIsland,
  'core.component.shared.sfextension_widget': resolveComponent('SFExtensionWidget') as Component
}
const legacyIslandBindings = {
  'sf-home-page': { componentId: 'forum.component.home_page' },
  'sf-navbar': { componentId: 'navigation.component.navbar' },
  'sf-footer': { componentId: 'navigation.component.footer' },
  'sf-home-navigation': { componentId: 'navigation.component.home' },
  'sf-topic-composer': { componentId: 'forum.component.topic_composer' },
  'sf-profile-settings': { componentId: 'profile.component.settings_form' },
  'sf-security-settings': { componentId: 'identity.component.security_settings' },
  'sf-login-form': { componentId: 'identity.component.login_form' },
  'sf-register-form': { componentId: 'identity.component.register_form' },
  'sf-recovery-request': { componentId: 'identity.component.recovery_request_form' },
  'sf-recovery-confirm': { componentId: 'identity.component.recovery_confirm_form' },
  'sf-extension-widget': { componentId: 'core.component.shared.sfextension_widget' }
} as const
const allowedComponents = new Set(Object.keys(islandComponents))
const fallbackComponents = new Set(['core.component.shared.sfextension_widget'])

const renderState = computed(() => {
  try {
    const nodes = props.renderOutput
      ? parseThemeRenderOutput(props.renderOutput, { allowedComponents, fallbackComponents })
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
