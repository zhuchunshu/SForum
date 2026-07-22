<script setup lang="ts">
import type { ThemeRenderNode } from '~/composables/useThemeRenderOutput'

defineOptions({ name: 'SFSystemThemeNode' })

defineProps<{
  node: ThemeRenderNode
}>()
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
    <SFSystemThemeNode
      v-for="(child, index) in node.children"
      :key="index"
      :node="child"
    />
  </component>
</template>
