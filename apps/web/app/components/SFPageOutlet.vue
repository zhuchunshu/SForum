<script setup lang="ts">
/**
 * SFPageOutlet — Page Registry 解析钩子。
 * core：渲染默认 slot（host Vue 页）。
 * replace：渲染 L1 模板 HTML + 宿主岛；失败回退 slot。
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

type ResolvePayload = {
  page?: { id: string }
  provider?: string
  extensionId?: string
  contributionId?: string
  action?: string
  fallback?: boolean
  templatePath?: string
  templateHtml?: string
  dataSource?: string
  dataRoute?: string
}

const { data: resolved, error: resolveError } = await useAsyncData(
  resolveKey,
  async () => {
    if (!registryEnabled.value) {
      return {
        page: { id: props.page },
        provider: 'core',
        action: 'core',
        fallback: false
      } satisfies ResolvePayload
    }
    try {
      const { request } = useApiClient()
      return await request<ResolvePayload>(`/pages/resolve?id=${encodeURIComponent(props.page)}`)
    } catch {
      return {
        page: { id: props.page },
        provider: 'core',
        action: 'core',
        fallback: true
      } satisfies ResolvePayload
    }
  },
  { watch: [() => props.page, registryEnabled] }
)

const provider = computed(() => resolved.value?.provider || 'core')
const templateHtml = computed(() => (resolved.value?.templateHtml || '').trim())
const useTemplate = computed(() =>
  provider.value !== 'core'
  && Boolean(templateHtml.value)
  && !resolved.value?.fallback
)
const showFallbackNotice = computed(() => Boolean(resolved.value?.fallback || resolveError.value))
</script>

<template>
  <div
    class="sf-page-outlet"
    :data-page="page"
    :data-provider="provider"
    :data-template="useTemplate ? '1' : '0'"
  >
    <SFThemeTemplate
      v-if="useTemplate"
      :html="templateHtml"
      :extension-id="resolved?.extensionId || provider"
      :data-source="resolved?.dataSource"
      :data-route="resolved?.dataRoute"
    />
    <slot v-else />
    <p
      v-if="showFallbackNotice"
      class="sr-only"
    >
      page registry fallback to core
    </p>
  </div>
</template>
