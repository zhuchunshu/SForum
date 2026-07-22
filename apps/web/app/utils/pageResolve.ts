import type { ThemeRenderOutput } from '~/composables/useThemeRenderOutput'

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
      if (attempt < attempts) {
        await sleep(Math.max(0, options.retryDelayMs ?? 120))
      }
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
