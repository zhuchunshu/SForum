<script setup lang="ts">
import type { ThemeRenderNode } from '~/composables/themes/useThemeRenderOutput'
import SFFooter from './SFFooter.vue'
import SFNavbar from './SFNavbar.vue'
import SFNotFoundPageContent from './errors/SFNotFoundPageContent.vue'

defineOptions({ name: 'SFSystemThemeNode' })

defineProps<{
  node: ThemeRenderNode
}>()

// Nuxt 会把模板里的同名递归标签误转成模块自导入；直接复用当前组件类型避免 SSR 初始化环。
const self = getCurrentInstance()!.type
</script>

<template>
  <template v-if="node.kind === 'text'">{{ node.value }}</template>
  <template v-else-if="node.kind === 'comment'" />
  <SFNavbar
    v-else-if="node.kind === 'island' && node.descriptor.componentId === 'navigation.component.navbar'"
    v-bind="node.props"
  />
  <SFNotFoundPageContent
    v-else-if="node.kind === 'island' && node.descriptor.componentId === 'system.component.not_found'"
    v-bind="node.props"
  />
  <SFFooter
    v-else-if="node.kind === 'island' && node.descriptor.componentId === 'navigation.component.footer'"
    v-bind="node.props"
  />
  <component
    :is="node.tag"
    v-else-if="node.kind === 'element'"
    v-bind="node.attrs"
  >
    <component
      v-for="(child, index) in node.children"
      :is="self"
      :key="index"
      :node="child"
    />
  </component>
</template>
