<script setup lang="ts">
import type { PageResolvePayload } from '~/utils/pageResolve'

/**
 * Page Outlet 的同步入口。
 * Nuxt 错误渲染器不会等待嵌套异步组件，因此 system.not_found 由 error.vue
 * 预先解析后从 resolvedPayload 注入；普通页面仍交给异步 resolver。
 */
defineProps<{
  page: string
  resolvedPayload?: PageResolvePayload | null
}>()
</script>

<template>
  <SFPageOutletRender
    v-if="resolvedPayload"
    :page="page"
    :resolved="resolvedPayload"
  >
    <slot />
  </SFPageOutletRender>
  <SFPageOutletResolver v-else :page="page">
    <slot />
  </SFPageOutletResolver>
</template>
