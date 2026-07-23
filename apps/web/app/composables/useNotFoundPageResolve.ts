import {
  coreResolveFallback,
  PAGE_RESOLVE_REASON,
  requestPageResolveWithRetry,
  type PageResolvePayload
} from '~/utils/pageResolve'

/** 错误根组件预取 system.not_found；生产错误渲染器要求 setup 保持同步。 */
export function useNotFoundPageResolve() {
  const route = useRoute()
  const { locale } = useI18n()
  const { webOption } = useWebOptions()
  const { request } = useApiClient()
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
            'system.not_found',
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

  function useCoreFallback(reason = PAGE_RESOLVE_REASON.transportUnavailable) {
    return commit(coreResolveFallback('system.not_found', true, reason))
  }

  return { data, failure, pending, refresh, commit, useCoreFallback }
}
