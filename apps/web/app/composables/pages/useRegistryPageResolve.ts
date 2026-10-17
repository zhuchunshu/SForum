import { useNotFoundPagePresentation } from '~/composables/errors/useNotFoundPagePresentation'
import type { PageResolvePayload } from '~/utils/pageResolve'

/**
 * Page Registry 动态公开页的统一解析入口。
 *
 * 两个 catch-all（`[...sfRegistryPage].vue` 与 `/x/[...path].vue`）共用此处：
 * - 404 → 正常 Nuxt 404（先准备 not-found 呈现，保留原客户端分支）
 * - 401 → 跳登录
 * - 403 → 403 错误页
 *
 * 权威 access/permission 始终由 API resolve-path 决定；此处的映射只是呈现层反应。
 *
 * 等待策略：SSR 仍整体 await（完整加载保持静态渲染、零水合替换）；
 * 客户端换页改为非阻塞——调用页在 `pending && !data` 期间渲染骨架外壳，
 * 消除此前 async setup suspense 造成的整页空白帧。
 */
export async function useRegistryPageResolve(
  requestPath: Ref<string> | ComputedRef<string>,
  cacheKey: string
) {
  const nuxtApp = useNuxtApp()
  const localePath = useLocalePath()
  const notFoundPresentation = useNotFoundPagePresentation()

  const resolve = useAsyncData(
    () => `${cacheKey}:${requestPath.value}`,
    async () => {
      const { request } = useApiClient()
      return await request<PageResolvePayload>(
        `/pages/resolve-path?path=${encodeURIComponent(requestPath.value)}`
      )
    },
    { watch: [requestPath] }
  )

  const resolveErrorCode = (raw: unknown): 'not-found' | 'unauthorized' | 'forbidden' | null => {
    const err = raw as { statusCode?: number, status?: number, data?: { statusCode?: number } } | undefined
    const code = err?.statusCode || err?.status || err?.data?.statusCode
    if (code === 404) return 'not-found'
    if (code === 401) return 'unauthorized'
    if (code === 403) return 'forbidden'
    return null
  }

  const mapResolveError = async (raw: unknown) => {
    const kind = resolveErrorCode(raw)
    if (kind === null) return
    if (kind === 'not-found') {
      if (import.meta.client) {
        await notFoundPresentation.prepare()
      }
      throw createError({ statusCode: 404, statusMessage: 'Page not found' })
    }
    if (kind === 'unauthorized') {
      // await 之后已脱离 setup 上下文，navigateTo 需显式挂回 Nuxt 实例。
      await nuxtApp.runWithContext(() => navigateTo(localePath('/login')))
      return
    }
    throw createError({ statusCode: 403, statusMessage: 'Forbidden' })
  }

  if (import.meta.server) {
    // AsyncData 对象可等待：阻塞 SSR 渲染直到解析完成，错误原地映射（setup 上下文）。
    await resolve
    if (resolve.error.value) {
      await mapResolveError(resolve.error.value)
    }
  } else {
    // 客户端：等待阶段由调用页骨架承接；错误改为响应式映射。
    watch(resolve.error, (raw) => {
      if (raw) void mapResolveError(raw)
    })
  }

  return { data: resolve.data, error: resolve.error, status: resolve.status, pending: resolve.pending }
}
