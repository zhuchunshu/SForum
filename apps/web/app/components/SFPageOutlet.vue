<script setup lang="ts">
/**
 * SFPageOutlet — Page Registry 解析钩子。
 * 当前始终渲染默认 slot（core Vue 页）；当 provider 非 core 且拿到模板时渲染 L1 HTML + 宿主岛。
 */
const props = defineProps<{
  page: string
}>()

const { webOption } = useWebOptions()
const registryEnabled = computed(() => {
  const raw = String(webOption('pages.registry_enabled', 'enabled')).toLowerCase()
  return raw === 'enabled' || raw === 'true' || raw === '1'
})

const resolveKey = computed(() => `page-resolve:${props.page}`)

const { data: resolved, error: resolveError } = await useAsyncData(
  resolveKey,
  async () => {
    if (!registryEnabled.value) {
      return {
        page: { id: props.page },
        provider: 'core',
        action: 'core',
        fallback: false
      }
    }
    try {
      const { request } = useApiClient()
      return await request<any>(`/pages/resolve?id=${encodeURIComponent(props.page)}`)
    } catch {
      return {
        page: { id: props.page },
        provider: 'core',
        action: 'core',
        fallback: true
      }
    }
  },
  { watch: [() => props.page, registryEnabled] }
)

const provider = computed(() => resolved.value?.provider || 'core')
const isCore = computed(() => provider.value === 'core' || !resolved.value)
const showFallbackNotice = computed(() => Boolean(resolved.value?.fallback || resolveError.value))
</script>

<template>
  <div
    class="sf-page-outlet"
    :data-page="page"
    :data-provider="provider"
  >
    <!-- 核心路径：主题 Vue / host 页面内容 -->
    <slot v-if="isCore" name="default" />
    <!-- 非 core：仍先走 default（宿主 Vue 岛），L1 HTML 增强由 SFThemeTemplate 可选包裹 -->
    <slot v-else name="default" />
    <p
      v-if="showFallbackNotice"
      class="sr-only"
    >
      page registry fallback to core
    </p>
  </div>
</template>
