<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'

/**
 * SFPageOutlet — Page Registry 解析钩子。
 * core：渲染默认 slot（host Vue 页）。
 * replace：渲染 L1 模板 HTML + 宿主岛；失败回退 slot。
 *
 * auth / settings / topic create 等受保护页面只能通过精确 Host island
 * 放置默认 slot，主题可以控制页面外观，但 mutation 仍由宿主组件执行。
 */
const props = defineProps<{
  page: string
}>()

const route = useRoute()

const { webOption } = useWebOptions()
const registryEnabled = computed(() => {
  const raw = String(webOption('pages.registry_enabled', 'enabled')).toLowerCase()
  return raw === 'enabled' || raw === 'true' || raw === '1'
})

const requestQuery = computed(() => {
  const query = new URLSearchParams()
  for (const key of Object.keys(route.query).sort()) {
    const raw = route.query[key]
    const values = Array.isArray(raw) ? raw : [raw]
    for (const value of values) {
      if (value !== null && value !== undefined) {
        query.append(key, String(value))
      }
    }
  }
  return query.toString()
})

const resolveKey = computed(() => `page-resolve:${props.page}:${route.path}?${requestQuery.value}`)

type ResolvePayload = {
  page?: { id: string, contractVersion?: string }
  provider?: string
  extensionId?: string
  contributionId?: string
  action?: string
  fallback?: boolean
  templatePath?: string
  templateHtml?: string
  renderOutput?: ThemeRenderOutput
  dataSource?: string
  dataRoute?: string
  loaderData?: unknown
  loaderError?: string
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
      const query = new URLSearchParams({ id: props.page, path: route.path })
      if (requestQuery.value) {
        query.set('query', requestQuery.value)
      }
      return await request<ResolvePayload>(`/pages/resolve?${query.toString()}`)
    } catch {
      return {
        page: { id: props.page },
        provider: 'core',
        action: 'core',
        fallback: true
      } satisfies ResolvePayload
    }
  },
  { watch: [() => props.page, () => route.path, requestQuery, registryEnabled] }
)

const provider = computed(() => resolved.value?.provider || 'core')
const templateHtml = computed(() => (resolved.value?.templateHtml || '').trim())
const renderOutput = computed(() => resolved.value?.renderOutput)
const useTemplate = computed(() =>
  provider.value !== 'core'
  && Boolean(renderOutput.value || templateHtml.value)
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
      :render-output="renderOutput"
      :extension-id="resolved?.extensionId || provider"
      :data-source="resolved?.dataSource"
      :data-route="resolved?.dataRoute"
      :loader-data="resolved?.loaderData"
      :loader-error="resolved?.loaderError"
    >
      <slot />
    </SFThemeTemplate>
    <slot v-else />
    <p
      v-if="showFallbackNotice"
      class="sr-only"
    >
      page registry fallback to core
    </p>
  </div>
</template>
