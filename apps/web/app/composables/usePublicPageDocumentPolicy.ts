import type { PublicFrontendComponentRef, PublicFrontendPolicy } from '~/runtime/public-extensions/pagePolicy'
import {
  PUBLIC_PAGE_CSP_HEADER,
  PUBLIC_PAGE_POLICY_TIMEOUT_MS,
  parsePublicFrontendPolicy,
  publicPagePolicyPath
} from '~/runtime/public-extensions/pagePolicy'

/**
 * 在 Nuxt SSR 上应用 Host 聚合的 Content-Security-Policy。
 * 仅在服务端写入响应头；客户端 hydration 复用已下发的文档策略。
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
  try {
    const { request } = useApiClient()
    const raw = await request<unknown>(publicPagePolicyPath(refs), {
      timeout: PUBLIC_PAGE_POLICY_TIMEOUT_MS
    })
    const policy = parsePublicFrontendPolicy(raw)
    const headerValue = policy.documentPolicy.headerValue.trim()
    if (!headerValue) {
      return null
    }
    setHeader(event, PUBLIC_PAGE_CSP_HEADER, headerValue)
    setHeader(event, 'X-SForum-Document-Policy-Digest', policy.documentPolicy.digest)
    return policy
  } catch {
    // public L2 默认关闭或制品/信任不可用时，不写 CSP、不挂载 L2。
    return null
  }
}
