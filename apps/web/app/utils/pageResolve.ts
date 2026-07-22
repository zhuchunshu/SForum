import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'
import { apiErrorReason, apiErrorStatusCode } from '~/composables/useApiClient'

export const PAGE_RESOLVE_REASON = {
  authoritativeCore: 'authoritative_core',
  transportUnavailable: 'transport_unavailable',
  viewModelUnavailable: 'view_model_unavailable',
  renderFailed: 'render_failed',
  artifactMismatch: 'artifact_mismatch',
  runtimeUnavailable: 'runtime_unavailable'
} as const

export type PageResolveReason = typeof PAGE_RESOLVE_REASON[keyof typeof PAGE_RESOLVE_REASON]

export type PageResolvePayload = {
  page?: { id: string, contractVersion?: string }
  provider?: string
  extensionId?: string
  contributionId?: string
  action?: string
  fallback?: boolean
  reason?: PageResolveReason | string
  selectedProvider?: string
  selectedExtensionId?: string
  selectedContributionId?: string
  selectedVersion?: string
  selectedPackageDigest?: string
  selectedRuntimeInstanceId?: string
  nodeRevision?: number
  templatePath?: string
  templateHtml?: string
  renderOutput?: ThemeRenderOutput
  dataSource?: string
  dataRoute?: string
  loaderData?: unknown
  loaderError?: string
}

export type PageResolveRequest = (
  url: string,
  options?: { timeout?: number }
) => Promise<PageResolvePayload>

export type PageResolveFailureClass = 'semantic_not_found' | 'retryable' | 'technical'

const SEMANTIC_NOT_FOUND_REASONS = new Set([
  'pages.data_not_found'
])

/**
 * Page Registry 的 404 是资源语义，不是主题运行时不可用。
 * 只允许已审查的短暂 5xx 和无 HTTP 响应的传输错误走一次重试。
 */
export function classifyPageResolveFailure(error: unknown): PageResolveFailureClass {
  const status = apiErrorStatusCode(error)
  const reason = apiErrorReason(error)
  if (status === 404 && SEMANTIC_NOT_FOUND_REASONS.has(reason)) {
    return 'semantic_not_found'
  }
  if (status === 0 || status === 502 || status === 503 || status === 504) {
    return 'retryable'
  }
  return 'technical'
}

export function isPageResolveSemanticNotFound(error: unknown) {
  return classifyPageResolveFailure(error) === 'semantic_not_found'
}

export type MutableRouteRulesContext = {
  _nitro?: {
    routeRules?: {
      cache?: false | Record<string, unknown>
      swr?: false | number
    }
  }
}

export function coreResolveFallback(
  pageId: string,
  fallback = true,
  reason: PageResolveReason = fallback
    ? PAGE_RESOLVE_REASON.transportUnavailable
    : PAGE_RESOLVE_REASON.authoritativeCore
): PageResolvePayload {
  return {
    page: { id: pageId },
    provider: 'core',
    selectedProvider: 'core',
    action: 'core',
    fallback,
    reason
  }
}

export async function requestPageResolveWithRetry(
  request: PageResolveRequest,
  url: string,
  options: {
    timeout: number
    maxAttempts?: number
    retryDelayMs?: number
    sleep?: (ms: number) => Promise<void>
  }
): Promise<PageResolvePayload> {
  const attempts = Math.max(1, Math.floor(options.maxAttempts ?? 2))
  const sleep = options.sleep ?? delay
  let lastError: unknown

  for (let attempt = 1; attempt <= attempts; attempt++) {
    try {
      return await request(url, { timeout: options.timeout })
    } catch (error) {
      lastError = error
      if (attempt >= attempts || classifyPageResolveFailure(error) !== 'retryable') {
        throw error
      }
      await sleep(Math.max(0, options.retryDelayMs ?? 120))
    }
  }

  throw lastError
}

export function shouldDisablePageResolveSharedCache(
  payload: PageResolvePayload | null | undefined
) {
  if (!payload) {
    return false
  }
  return Boolean(payload.fallback || payload.reason === PAGE_RESOLVE_REASON.transportUnavailable)
}

export function disableSharedPageCacheForPageResolve(
  context: MutableRouteRulesContext | null | undefined,
  setCacheControl: (value: string) => void
) {
  setCacheControl('no-store')
  const routeRules = context?._nitro?.routeRules
  if (!routeRules) {
    return
  }
  routeRules.cache = false
  routeRules.swr = false
}

function delay(ms: number) {
  return new Promise<void>(resolve => setTimeout(resolve, ms))
}
