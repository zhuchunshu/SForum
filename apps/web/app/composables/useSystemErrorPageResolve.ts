import { toValue, type MaybeRefOrGetter } from 'vue'
import {
  coreResolveFallback,
  PAGE_RESOLVE_REASON,
  requestPageResolveWithRetry,
  type PageResolvePayload,
  type PageResolveReason
} from '~/utils/pageResolve'

export function useSystemErrorPageResolve(pageId: MaybeRefOrGetter<string>) {
  const route = useRoute()
  const i18n = useNuxtApp().$i18n as { locale?: unknown } | undefined
  const locale = computed(() => localeString(i18n?.locale) || 'zh-CN')
  const { webOption } = useWebOptions()
  const { request } = useApiClient()
  const resolvedPageId = computed(() => toValue(pageId) || 'system.not_found')
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
  const enabled = computed(() => {
    const raw = String(webOption('pages.registry_enabled', 'enabled')).toLowerCase()
    return raw === 'enabled' || raw === 'true' || raw === '1'
  })
  const key = computed(() =>
    `system-error-page-resolve:${resolvedPageId.value}:${String(locale.value || 'zh-CN')}:${route.path}?${requestQuery.value}`
  )
  const stateKey = key.value
  const hydrationData = useState<PageResolvePayload | null>(stateKey, () => null)
  const data = shallowRef<PageResolvePayload | null>(hydrationData.value)
  const failure = shallowRef<unknown>(null)
  const pending = ref(!data.value)
  if (import.meta.client) {
    onMounted(() => clearNuxtState(stateKey))
  }

  async function refresh(options: { deferCommit?: boolean } = {}) {
    if (data.value) {
      pending.value = false
      return data.value
    }

    pending.value = true
    try {
      failure.value = null
      let resolved: PageResolvePayload
      if (!enabled.value) {
        resolved = coreResolveFallback(
          resolvedPageId.value,
          false,
          PAGE_RESOLVE_REASON.authoritativeCore
        )
      } else {
        try {
          const query = new URLSearchParams({ id: resolvedPageId.value, path: route.path })
          if (requestQuery.value) {
            query.set('query', requestQuery.value)
          }
          const resolveRequest = (url: string, options?: { timeout?: number }) => request<PageResolvePayload>(url, {
            timeout: options?.timeout,
            serverInternal: import.meta.server
          })
          resolved = await requestPageResolveWithRetry(
            resolveRequest,
            `/pages/resolve?${query.toString()}`,
            { timeout: import.meta.dev ? 800 : 1000, maxAttempts: 1 }
          )
        } catch (error) {
          failure.value = error
          resolved = coreResolveFallback(
            resolvedPageId.value,
            true,
            PAGE_RESOLVE_REASON.transportUnavailable
          )
        }
      }
      if (!options.deferCommit) {
        commit(resolved)
      }
      return resolved
    } finally {
      pending.value = false
    }
  }

  function commit(resolved: PageResolvePayload) {
    data.value = resolved
    hydrationData.value = resolved
    return resolved
  }

  function useCoreFallback(reason: PageResolveReason = PAGE_RESOLVE_REASON.transportUnavailable) {
    return commit(coreResolveFallback(resolvedPageId.value, true, reason))
  }

  return { data, failure, pending, refresh, commit, useCoreFallback }
}

function localeString(value: unknown) {
  if (typeof value === 'string') {
    return value
  }
  if (value && typeof value === 'object' && 'value' in value) {
    const refValue = (value as { value?: unknown }).value
    return typeof refValue === 'string' ? refValue : ''
  }
  return ''
}
