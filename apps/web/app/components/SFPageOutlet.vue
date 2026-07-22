<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'
import {
  coreResolveFallback,
  disableSharedPageCacheForPageResolve,
  PAGE_RESOLVE_REASON,
  requestPageResolveWithRetry,
  shouldDisablePageResolveSharedCache,
  type MutableRouteRulesContext,
  type PageResolvePayload
} from '~/utils/pageResolve'

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
  forceDefaultTheme?: boolean
}>()

const route = useRoute()
const { locale } = useI18n()
const { user } = useAuthSession()

const { webOption } = useWebOptions()
const registryEnabled = computed(() => {
  const raw = String(webOption('pages.registry_enabled', 'enabled')).toLowerCase()
  return raw === 'enabled' || raw === 'true' || raw === '1'
})

// 开发热重载时 API 偶发变慢；给 resolve 明确上限和一次重试，避免无限挂起。
const PAGE_RESOLVE_TIMEOUT_MS = import.meta.dev ? 5000 : 8000
const PAGE_RESOLVE_RETRY_DELAY_MS = import.meta.dev ? 180 : 120

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

const resolveLocale = computed(() => String(locale.value || 'zh-CN'))
const resolveActorKey = computed(() => user.value?.id ? `user:${user.value.id}` : 'guest')
const resolveKey = computed(() =>
  `page-resolve:${resolveLocale.value}:${resolveActorKey.value}:${props.page}:${route.path}?${requestQuery.value}`
)

type ResolvePayload = PageResolvePayload & {
  page?: { id: string, contractVersion?: string }
  renderOutput?: ThemeRenderOutput
}

function applyFallbackCachePolicy(payload: ResolvePayload) {
  if (!import.meta.server || !shouldDisablePageResolveSharedCache(payload)) {
    return
  }
  const cacheControl = useResponseHeader('cache-control')
  const event = useRequestEvent()
  disableSharedPageCacheForPageResolve(
    event?.context as MutableRouteRulesContext | undefined,
    value => {
      cacheControl.value = value
    }
  )
}

const { data: resolved, error: resolveError } = await useAsyncData(
  resolveKey,
  async () => {
    if (props.forceDefaultTheme) {
      return coreResolveFallback(
        props.page,
        true,
        PAGE_RESOLVE_REASON.authoritativeCore
      ) as ResolvePayload
    }

    if (!registryEnabled.value) {
      return coreResolveFallback(
        props.page,
        false,
        PAGE_RESOLVE_REASON.authoritativeCore
      ) as ResolvePayload
    }

    try {
      const { request } = useApiClient()
      const query = new URLSearchParams({ id: props.page, path: route.path })
      if (requestQuery.value) {
        query.set('query', requestQuery.value)
      }
      const payload = await requestPageResolveWithRetry(
        request,
        `/pages/resolve?${query.toString()}`,
        {
          timeout: PAGE_RESOLVE_TIMEOUT_MS,
          maxAttempts: 2,
          retryDelayMs: PAGE_RESOLVE_RETRY_DELAY_MS
        }
      ) as ResolvePayload
      applyFallbackCachePolicy(payload)
      return payload
    } catch {
      const payload = coreResolveFallback(
        props.page,
        true,
        PAGE_RESOLVE_REASON.transportUnavailable
      ) as ResolvePayload
      applyFallbackCachePolicy(payload)
      return payload
    }
  },
  {
    watch: [
      () => props.page,
      () => route.path,
      requestQuery,
      registryEnabled,
      resolveLocale,
      resolveActorKey
    ]
  }
)

if (import.meta.server) {
  watchEffect(() => {
    if (resolved.value) {
      applyFallbackCachePolicy(resolved.value as ResolvePayload)
    }
  })
}

const provider = computed(() => resolved.value?.provider || 'core')
const templateHtml = computed(() => (resolved.value?.templateHtml || '').trim())
const renderOutput = computed(() => resolved.value?.renderOutput)
const useTemplate = computed(() =>
  provider.value !== 'core'
  && Boolean(renderOutput.value || templateHtml.value)
  && !resolved.value?.fallback
)
const showFallbackNotice = computed(() => Boolean(resolved.value?.fallback || resolveError.value))

// auth / system.not_found 自带 chrome 或使用 auth layout；其余 fail-closed 走宿主公开 chrome。
const useHostPublicChrome = computed(() => {
  if (useTemplate.value) {
    return false
  }
  const id = props.page
  return !id.startsWith('auth.') && id !== 'system.not_found' && id !== 'dev.components'
})

const forceDefaultTheme = computed(() => !!props.forceDefaultTheme)
</script>

<template>
  <div
    class="sf-page-outlet"
    :data-page="page"
    :data-provider="provider"
    :data-template="useTemplate ? '1' : '0'"
    :data-host-chrome="useHostPublicChrome ? '1' : '0'"
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
    <SFHostPublicChrome v-else-if="useHostPublicChrome">
      <slot />
    </SFHostPublicChrome>
    <slot v-else />
    <p
      v-if="showFallbackNotice"
      class="sr-only"
    >
      page registry fallback to core
    </p>
  </div>
</template>
