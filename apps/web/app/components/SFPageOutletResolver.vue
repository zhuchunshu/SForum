<script setup lang="ts">
import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'
import {
  coreResolveFallback,
  disableSharedPageCacheForPageResolve,
  PAGE_RESOLVE_REASON,
  isPageResolveSemanticNotFound,
  requestPageResolveWithRetry,
  shouldDisablePageResolveSharedCache,
  type MutableRouteRulesContext,
  type PageResolvePayload
} from '~/utils/pageResolve'

const props = defineProps<{
  page: string
}>()

const route = useRoute()
const { locale } = useI18n()
const { user } = useAuthSession()
const { webOption } = useWebOptions()
const notFoundPresentation = useNotFoundPagePresentation()
const responseCacheControl = import.meta.server
  ? useResponseHeader('cache-control')
  : undefined
const requestEvent = import.meta.server ? useRequestEvent() : undefined
const registryEnabled = computed(() => {
  const raw = String(webOption('pages.registry_enabled', 'enabled')).toLowerCase()
  return raw === 'enabled' || raw === 'true' || raw === '1'
})

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

function disableSharedDocumentCache() {
  if (!import.meta.server) {
    return
  }
  disableSharedPageCacheForPageResolve(
    requestEvent?.context as MutableRouteRulesContext | undefined,
    value => {
      if (responseCacheControl) {
        responseCacheControl.value = value
      }
    }
  )
}

function applyFallbackCachePolicy(payload: ResolvePayload) {
  if (shouldDisablePageResolveSharedCache(payload)) {
    disableSharedDocumentCache()
  }
}

const { data: resolved, error: resolveError, pending } = await useAsyncData(
  resolveKey,
  async () => {
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
    } catch (error) {
      if (isPageResolveSemanticNotFound(error)) {
        disableSharedDocumentCache()
        throw error
      }
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

if (isPageResolveSemanticNotFound(resolveError.value)) {
  if (import.meta.client) {
    await notFoundPresentation.prepare()
  }
  throw createError({ statusCode: 404, statusMessage: 'Page not found' })
}

if (import.meta.server) {
  watchEffect(() => {
    if (resolved.value) {
      applyFallbackCachePolicy(resolved.value as ResolvePayload)
    }
  })
}
</script>

<template>
  <div v-if="pending && !resolved" class="min-h-screen" aria-busy="true" />
  <SFPageOutletRender
    v-else
    :page="page"
    :resolved="resolved"
    :resolve-error="resolveError"
  >
    <slot />
  </SFPageOutletRender>
</template>
