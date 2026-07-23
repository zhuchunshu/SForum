<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'

/**
 * 兼容路径 /x/*：转发到统一 Registry 解析（与根 catch-all 相同逻辑）。
 * 正式 add 路径应使用 manifest 声明的真实 path（由 [...sfRegistryPage] 承接）。
 */
definePageMeta({ public: true })

const route = useRoute()
const localePath = useLocalePath()
const notFoundPresentation = useNotFoundPagePresentation()

const requestPath = computed(() => {
  const raw = route.params.path
  const parts = Array.isArray(raw) ? raw.map(String) : (raw ? [String(raw)] : [])
  return '/x/' + parts.join('/')
})

type ResolvePayload = {
  page?: { id: string }
  provider?: string
  extensionId?: string
  fallback?: boolean
  templateHtml?: string
  renderOutput?: ThemeRenderOutput
  dataSource?: string
  dataRoute?: string
  loaderError?: string
}

const { data, error } = await useAsyncData(
  () => `page-resolve-path-x:${requestPath.value}`,
  async () => {
    const { request } = useApiClient()
    return await request<ResolvePayload>(
      `/pages/resolve-path?path=${encodeURIComponent(requestPath.value)}`
    )
  },
  { watch: [requestPath] }
)

if (error.value) {
  const err: any = error.value
  const code = err?.statusCode || err?.status || err?.data?.statusCode
  if (code === 404) {
    if (import.meta.client) {
      await notFoundPresentation.prepare()
    }
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
    <SFThemeTemplate
      v-if="useTemplate"
      :html="templateHtml"
      :render-output="data?.renderOutput"
      :extension-id="data?.extensionId || data?.provider || ''"
      :data-source="data?.dataSource"
      :data-route="data?.dataRoute"
    />
    <SFEmptyState
      v-else
      title="Extension page unavailable"
      :description="data?.loaderError || requestPath"
    />
  </main>
</template>
