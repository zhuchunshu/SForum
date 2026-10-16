import { useNotFoundPagePresentation } from '~/composables/errors/useNotFoundPagePresentation'
import type { PageResolvePayload } from '~/utils/pageResolve'

/**
 * Page Registry 动态公开页的统一解析入口。
 *
 * 两个 catch-all（`[...sfRegistryPage].vue` 与 `/x/[...path].vue`）此前各自复制了
 * 同一份「解析 + 401/403/404 映射」逻辑。这里收敛成一处，行为保持不变：
 * - 404 → 正常 Nuxt 404（先准备 not-found 呈现，保留原客户端分支）
 * - 401 → 跳登录
 * - 403 → 403 错误页
 *
 * 权威 access/permission 始终由 API resolve-path 决定；此处的映射只是呈现层反应。
 */
export async function useRegistryPageResolve(
  requestPath: Ref<string> | ComputedRef<string>,
  cacheKey: string
) {
  const localePath = useLocalePath()
  const notFoundPresentation = useNotFoundPagePresentation()

  const { data, error, status } = await useAsyncData(
    () => `${cacheKey}:${requestPath.value}`,
    async () => {
      const { request } = useApiClient()
      return await request<PageResolvePayload>(
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

  return { data, error, status }
}
