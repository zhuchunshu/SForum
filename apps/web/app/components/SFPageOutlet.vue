<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'

/**
 * SFPageOutlet — Page Registry 解析钩子。
 * core：渲染默认 slot（host Vue 页）。
 * replace：渲染 L1 模板 HTML + 宿主岛；失败回退 slot。
 *
 * constrained 页面（auth / settings / topic create 等）即使有 replace 绑定，
 * 仍强制渲染核心 slot，确保 mutation 始终由宿主安全组件执行。
 * 外观差异由 L0 skin CSS 承担；完整 chrome 模板替换需宿主表单岛（后续）。
 */
const props = defineProps<{
  page: string
}>()

const route = useRoute()

/** 必须由核心 Vue 页执行 mutation 的页面 id */
const CONSTRAINED_PAGES = new Set([
  'auth.login',
  'auth.register',
  'auth.forgot_password',
  'auth.reset_password',
  'forum.settings.profile',
  'forum.settings.security',
  'forum.topic.create',
  'forum.my.home',
  'forum.my.content_review',
  'forum.notifications',
  'moderation.review'
])

const { webOption } = useWebOptions()
const registryEnabled = computed(() => {
  const raw = String(webOption('pages.registry_enabled', 'enabled')).toLowerCase()
  return raw === 'enabled' || raw === 'true' || raw === '1'
})

const isConstrained = computed(() => CONSTRAINED_PAGES.has(props.page))

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
    if (!registryEnabled.value || isConstrained.value) {
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
  { watch: [() => props.page, () => route.path, requestQuery, registryEnabled, isConstrained] }
)

const provider = computed(() => {
  if (isConstrained.value) {
    return 'core'
  }
  return resolved.value?.provider || 'core'
})
const templateHtml = computed(() => (resolved.value?.templateHtml || '').trim())
const renderOutput = computed(() => resolved.value?.renderOutput)
const useTemplate = computed(() =>
  !isConstrained.value
  && provider.value !== 'core'
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
    :data-constrained="isConstrained ? '1' : '0'"
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
