<script setup lang="ts">
import { useSystemErrorPageContext } from '~/composables/errors/useSystemErrorPageContext'
import type { NuxtError } from '#app'
import type { Component } from 'vue'
import type { ThemeRenderOutput } from '~/composables/themes/useThemeRenderOutput'
import {
  parseLegacyThemeHTML,
  parseThemeRenderOutput,
  renderThemeRenderNodes
} from '~/composables/themes/useThemeRenderOutput'
import SFFooter from './SFFooter.vue'
import SFNavbar from './SFNavbar.vue'
import SFNotFoundPageContent from './errors/SFNotFoundPageContent.vue'
import SFSystemErrorActions from './errors/SFSystemErrorActions.vue'
import SFSystemErrorDetails from './errors/SFSystemErrorDetails.vue'
import SFSystemErrorEmergencyPage from './errors/SFSystemErrorEmergencyPage.vue'
import SFSystemErrorRail from './errors/SFSystemErrorRail.vue'
import SFSystemErrorRecovery from './errors/SFSystemErrorRecovery.vue'
import SFSystemErrorSidebar from './errors/SFSystemErrorSidebar.vue'

const props = defineProps<{
  html?: string
  renderOutput?: ThemeRenderOutput | null
  extensionId?: string
}>()

const context = useSystemErrorPageContext()
const fallbackError = computed(() => context?.error.value || ({ statusCode: 500 } as NuxtError))

// 错误模板是 L0/L1 封闭面：不加载公开 L2，也不走一般页面的异步 CSP 聚合路径。
const allowedComponents = new Set([
  'system.component.not_found',
  'system.component.error_details',
  'system.component.error_actions',
  'system.component.error_recovery',
  'system.component.error_sidebar',
  'system.component.error_rail',
  'navigation.component.navbar',
  'navigation.component.footer'
])
const legacyBindings = {
  'sf-not-found-page': { componentId: 'system.component.not_found' },
  'sf-error-details': { componentId: 'system.component.error_details' },
  'sf-error-actions': { componentId: 'system.component.error_actions' },
  'sf-error-recovery': { componentId: 'system.component.error_recovery' },
  'sf-error-sidebar': { componentId: 'system.component.error_sidebar' },
  'sf-error-rail': { componentId: 'system.component.error_rail' },
  'sf-navbar': { componentId: 'navigation.component.navbar' },
  'sf-footer': { componentId: 'navigation.component.footer' }
} as const
const rendered = computed(() => {
  try {
    return {
      nodes: props.renderOutput
        ? parseThemeRenderOutput(props.renderOutput, { allowedComponents })
        : parseLegacyThemeHTML(props.html || '', legacyBindings),
      error: false
    }
  } catch {
    return { nodes: [], error: true }
  }
})
const islandComponents: Record<string, Component> = {
  'system.component.not_found': SFNotFoundPageContent,
  'system.component.error_details': SFSystemErrorDetails,
  'system.component.error_actions': SFSystemErrorActions,
  'system.component.error_recovery': SFSystemErrorRecovery,
  'system.component.error_sidebar': SFSystemErrorSidebar,
  'system.component.error_rail': SFSystemErrorRail,
  'navigation.component.navbar': SFNavbar,
  'navigation.component.footer': SFFooter
}
const ThemeNodes = () => renderThemeRenderNodes(
  rendered.value.nodes,
  componentId => islandComponents[componentId]
)
</script>

<template>
  <div class="sf-system-theme-template" :data-extension-id="extensionId || ''">
    <template v-if="!rendered.error">
      <ThemeNodes />
    </template>
    <SFSystemErrorEmergencyPage v-else :error="fallbackError" />
  </div>
</template>
