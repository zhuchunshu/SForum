import type { PublicFrontendComponentRef, PublicFrontendPolicy } from '~/runtime/public-extensions/pagePolicy'
import {
  PUBLIC_PAGE_CSP_HEADER,
  PUBLIC_PAGE_POLICY_TIMEOUT_MS,
  parsePublicFrontendPolicy,
  publicPagePolicyPath
} from '~/runtime/public-extensions/pagePolicy'

/**
 * 取页面 L2 聚合策略。服务端用于写文档 CSP 头；客户端用于同帧预注入
 * `<link rel="stylesheet">`（消除样式/h1 首帧闪变）。策略端点是 no-store 的
 * 权威读取；客户端换页与 widget 描述符读取同级成本，可接受。
 */
export async function fetchPublicPagePolicy(
  refs: readonly PublicFrontendComponentRef[] = []
): Promise<PublicFrontendPolicy | null> {
  const { request } = useApiClient()
  try {
    const raw = await request<unknown>(publicPagePolicyPath(refs), {
      timeout: PUBLIC_PAGE_POLICY_TIMEOUT_MS
    })
    return parsePublicFrontendPolicy(raw)
  } catch {
    // 制品/信任不可用或 Safe Mode 生效时，调用方应对 L2 失败关闭。
    return null
  }
}

/**
 * 在 Nuxt SSR 上应用 Host 聚合的 Content-Security-Policy。
 * 仅在服务端写入响应头；客户端换页复用 fetchPublicPagePolicy 做样式预注入，
 * 但文档 CSP 只能由 SSR 响应头下发（浏览器无法在换页时改文档头）。
 * 策略不可用时返回 null，调用方应对 L2 失败关闭（保留 L1/SSR 主内容）。
 */
export async function applyPublicPageDocumentPolicy(
  refs: readonly PublicFrontendComponentRef[] = []
): Promise<PublicFrontendPolicy | null> {
  if (!import.meta.server) {
    return null
  }
  const event = useRequestEvent()
  if (!event) {
    return null
  }
  const policy = await fetchPublicPagePolicy(refs)
  if (!policy) {
    return null
  }
  if (import.meta.dev) {
    // dev 下宿主自身样式由 Vite 以内联 <style> 注入，L2 基线 style-src 'self' 会拦截它；
    // CSP 头只在生产（外链样式）下生效，dev 仅保留 digest 样式预注入。
    return policy
  }
  const headerValue = policy.documentPolicy.headerValue.trim()
  if (!headerValue) {
    return null
  }
  // Nuxt 4 未导出 setResponseHeader 组合式，直接写 nitro 响应对象。
  event.node.res.setHeader(PUBLIC_PAGE_CSP_HEADER, headerValue)
  event.node.res.setHeader('X-SForum-Document-Policy-Digest', policy.documentPolicy.digest)
  return policy
}
