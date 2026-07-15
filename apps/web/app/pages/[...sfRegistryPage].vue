<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'

/**
 * 动态公开路由宿主：匹配 Page Registry 的 action=add 贡献。
 * 仅在无其它 Nuxt 文件路由命中时生效（catch-all）。
 * 权威 access/permission 由 API resolve-path 返回 401/403/404。
 */
definePageMeta({
  public: true
})

const route = useRoute()
const localePath = useLocalePath()

const requestPath = computed(() => {
  const raw = route.params.sfRegistryPage
  const parts = Array.isArray(raw) ? raw.map(String) : (raw ? [String(raw)] : [])
  // 去掉可选 locale 前缀由 API strip；此处保留完整 path
  return '/' + parts.join('/')
})

type ResolvePayload = {
  page?: { id: string, pathPattern?: string, access?: string }
  provider?: string
  extensionId?: string
  contributionId?: string
  action?: string
  fallback?: boolean
  templateHtml?: string
  renderOutput?: ThemeRenderOutput
  dataSource?: string
  dataRoute?: string
  loaderData?: unknown
  loaderError?: string
}

const { data, error, status } = await useAsyncData(
  () => `page-resolve-path:${requestPath.value}`,
  async () => {
    const { request } = useApiClient()
    return await request<ResolvePayload>(
      `/pages/resolve-path?path=${encodeURIComponent(requestPath.value)}`
    )
  },
  { watch: [requestPath] }
)

// API 404 → 正常 Nuxt 404
if (error.value) {
  const err: any = error.value
  const code = err?.statusCode || err?.status || err?.data?.statusCode
  if (code === 404) {
    throw createError({ statusCode: 404, statusMessage: 'Page not found' })
  }
  if (code === 401) {
    await navigateTo(localePath('/login'))
  }
  if (code === 403) {
    throw createError({ statusCode: 403, statusMessage: 'Forbidden' })
  }
}

const templateHtml = computed(() => (data.value?.templateHtml || '').trim())
const useTemplate = computed(() => Boolean(data.value?.renderOutput || templateHtml.value))

useSForumSeo({
  title: () => data.value?.page?.id || requestPath.value,
  type: 'website',
  noindex: true
})
</script>

<template>
  <main class="sf-public-page sf-registry-add mx-auto w-full max-w-4xl px-4 py-8">
    <SFAlert
      v-if="error && status === 'error'"
      variant="danger"
      title="Failed to load extension page"
    />
    <SFThemeTemplate
      v-else-if="useTemplate"
      :html="templateHtml"
      :render-output="data?.renderOutput"
      :extension-id="data?.extensionId || data?.provider || ''"
      :data-source="data?.dataSource"
      :data-route="data?.dataRoute"
      :loader-data="data?.loaderData"
      :loader-error="data?.loaderError"
    />
    <SFEmptyState
      v-else
      title="Extension page unavailable"
      :description="data?.loaderError || requestPath"
    />
  </main>
</template>
