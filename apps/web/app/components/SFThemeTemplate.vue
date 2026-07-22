<script setup lang="ts">
import type { Component } from 'vue'
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'
import {
  collectPublicL2ComponentRefsFromRenderNodes
} from '~/runtime/public-extensions/pagePolicy'

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
// 系统 not_found 等仍可 HostPageIsland；公开 body 岛均为自包含 Host 组件。
const islandComponents: Record<string, Component> = {
  // 首页 body 岛由 SFHomePage 自包含数据+UI；主题 L1 拥有壳层结构。
  // 不再经 HostPageIsland 嵌回 pages/index slot（slot 仅 fail-closed 紧急回退）。
  'forum.component.home_page': resolveComponent('LazySFHomePage') as Component,
  'forum.component.category_index': resolveComponent('LazySFCategoryIndexPage') as Component,
  'forum.component.category_show': resolveComponent('LazySFCategoryShowPage') as Component,
  'forum.component.tag_index': resolveComponent('LazySFTagIndexPage') as Component,
  'forum.component.tag_show': resolveComponent('LazySFTagShowPage') as Component,
  // 帖子详情是高频核心阅读路径，避免主题岛自身再制造一层异步导航边界。
  'forum.component.topic_show': resolveComponent('SFTopicShowPage') as Component,
  'forum.component.profile_show': resolveComponent('LazySFProfileShowPage') as Component,
  'forum.component.notifications': resolveComponent('LazySFNotificationsPage') as Component,
  'site.component.terms': resolveComponent('LazySFTermsPage') as Component,
  'site.component.privacy': resolveComponent('LazySFPrivacyPage') as Component,
  'site.component.guidelines': resolveComponent('LazySFGuidelinesPage') as Component,
  'system.component.not_found': HostPageIsland,
  'navigation.component.navbar': resolveComponent('SFNavbar') as Component,
  'navigation.component.footer': resolveComponent('SFFooter') as Component,
  'navigation.component.home': resolveComponent('LazySFHomeNavigation') as Component,
  'forum.component.topic_composer': resolveComponent('LazySFTopicComposerPage') as Component,
  'forum.component.topic_reply': resolveComponent('LazySFTopicReplyPage') as Component,
  'profile.component.settings_form': resolveComponent('LazySFProfileSettingsPage') as Component,
  'identity.component.security_settings': resolveComponent('LazySFSecuritySettingsPage') as Component,
  'identity.component.login_form': resolveComponent('LazySFLoginFormPage') as Component,
  'identity.component.register_form': resolveComponent('LazySFRegisterFormPage') as Component,
  'identity.component.recovery_request_form': resolveComponent('LazySFRecoveryRequestPage') as Component,
  'identity.component.recovery_confirm_form': resolveComponent('LazySFRecoveryConfirmPage') as Component,
  'core.component.shared.sfextension_widget': resolveComponent('LazySFExtensionWidget') as Component
}
const legacyIslandBindings = {
  'sf-home-page': { componentId: 'forum.component.home_page' },
  'sf-category-index-page': { componentId: 'forum.component.category_index' },
  'sf-category-show-page': { componentId: 'forum.component.category_show' },
  'sf-tag-index-page': { componentId: 'forum.component.tag_index' },
  'sf-tag-show-page': { componentId: 'forum.component.tag_show' },
  'sf-topic-show-page': { componentId: 'forum.component.topic_show' },
  'sf-profile-page': { componentId: 'forum.component.profile_show' },
  'sf-notifications-page': { componentId: 'forum.component.notifications' },
  'sf-terms-page': { componentId: 'site.component.terms' },
  'sf-privacy-page': { componentId: 'site.component.privacy' },
  'sf-guidelines-page': { componentId: 'site.component.guidelines' },
  'sf-not-found-page': { componentId: 'system.component.not_found' },
  'sf-navbar': { componentId: 'navigation.component.navbar' },
  'sf-footer': { componentId: 'navigation.component.footer' },
  'sf-home-navigation': { componentId: 'navigation.component.home' },
  'sf-topic-composer': { componentId: 'forum.component.topic_composer' },
  'sf-topic-reply': { componentId: 'forum.component.topic_reply' },
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

// SSR：按页面实际 L2 岛聚合 Host document CSP。
// 策略不可用（public L2 默认关 / 信任撤销）时不写 header；widget 自身回退 L1。
const publicL2Refs = computed(() => {
  if (renderState.value.error) return []
  try {
    return collectPublicL2ComponentRefsFromRenderNodes(renderState.value.nodes)
  } catch {
    return []
  }
})
// 仅在页面声明了 L2 岛时聚合 CSP，避免每个公开页多一次 404。
const publicPagePolicy = import.meta.server && publicL2Refs.value.length > 0
  ? await applyPublicPageDocumentPolicy(publicL2Refs.value)
  : null

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
    :data-document-policy-digest="publicPagePolicy?.documentPolicy.digest || undefined"
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
