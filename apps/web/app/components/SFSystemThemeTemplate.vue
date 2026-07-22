<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'
import {
  parseLegacyThemeHTML,
  parseThemeRenderOutput
} from '~/composables/useThemeRenderOutput'
import SFSystemThemeNode from './SFSystemThemeNode.vue'

const props = defineProps<{
  html?: string
  renderOutput?: ThemeRenderOutput | null
  extensionId?: string
}>()

// 错误模板是 L0/L1 封闭面：不加载公开 L2，也不走一般页面的异步 CSP 聚合路径。
const allowedComponents = new Set([
  'system.component.not_found',
  'navigation.component.navbar',
  'navigation.component.footer'
])
const legacyBindings = {
  'sf-not-found-page': { componentId: 'system.component.not_found' },
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
</script>

<template>
  <div class="sf-system-theme-template" :data-extension-id="extensionId || ''">
    <template v-if="!rendered.error">
      <SFSystemThemeNode
        v-for="(node, index) in rendered.nodes"
        :key="index"
        :node="node"
      />
    </template>
    <SFAlert v-else variant="danger" class="mb-4" />
  </div>
</template>
