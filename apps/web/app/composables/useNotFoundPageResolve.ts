import {
  coreResolveFallback,
  PAGE_RESOLVE_REASON,
  requestPageResolveWithRetry,
  type PageResolvePayload
} from '~/utils/pageResolve'
import type { ApiEnvelope } from '~/composables/useApiClient'

/** 错误根组件预取 system.not_found；生产错误渲染器要求 setup 保持同步。 */
export function useNotFoundPageResolve() {
  const route = useRoute()
  const { locale } = useI18n()
  const { webOption } = useWebOptions()
  const { request, apiHeaders } = useApiClient()
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
  // system.not_found 的 L1 选择是站点级公开状态；用户态由 navbar 岛独立恢复。
  // key 若包含 actor，SSR 的 guest -> hydration 的 user 切换会制造两棵不同的错误树。
  const key = computed(() =>
    `not-found-page-resolve:${String(locale.value || 'zh-CN')}:${route.path}?${requestQuery.value}`
  )
  const stateKey = key.value
  // Nuxt error.vue 的 useAsyncData 不会稳定写入 hydration payload；用一次性 state
  // 传递 SSR 精确主题结果，挂载后清理，避免后续客户端错误复用陈旧 artifact。
  const hydrationData = useState<PageResolvePayload | null>(stateKey, () => null)
  const data = shallowRef<PageResolvePayload | null>(hydrationData.value)
  const pending = ref(!data.value)
  if (import.meta.client) {
    onMounted(() => clearNuxtState(stateKey))
  }

  async function refresh() {
    if (data.value) {
      pending.value = false
      return data.value
    }

    pending.value = true
    try {
      if (!enabled.value) {
        data.value = coreResolveFallback(
          'system.not_found',
          false,
          PAGE_RESOLVE_REASON.authoritativeCore
        )
      } else {
        try {
          const query = new URLSearchParams({ id: 'system.not_found', path: route.path })
          if (requestQuery.value) {
            query.set('query', requestQuery.value)
          }
          const resolveRequest = import.meta.server
            ? async (url: string, options?: { timeout?: number }) => {
                const apiBaseUrl = (process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1')
                  .replace(/\/+$/, '')
                const envelope = await $fetch<ApiEnvelope<PageResolvePayload>>(`${apiBaseUrl}${url}`, {
                  headers: apiHeaders(),
                  timeout: options?.timeout
                })
                return envelope.data
              }
            : request
          data.value = await requestPageResolveWithRetry(
            resolveRequest,
            `/pages/resolve?${query.toString()}`,
            { timeout: import.meta.dev ? 800 : 1000, maxAttempts: 1 }
          )
        } catch {
          data.value = coreResolveFallback(
            'system.not_found',
            true,
            PAGE_RESOLVE_REASON.transportUnavailable
          )
        }
      }
      hydrationData.value = data.value
      return data.value
    } finally {
      pending.value = false
    }
  }

  return { data, pending, refresh }
}
