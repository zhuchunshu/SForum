import type { ThemeRenderOutput } from '~/composables/themes/useThemeRenderOutput'
import { apiErrorReason, apiErrorStatusCode } from '~/composables/useApiClient'
import {
  normalizeActiveThemeIdentity,
  sameActiveThemeIdentity,
  type ActiveThemeIdentity
} from '~/utils/themes/activeThemeClientCache'

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
  options?: { timeout?: number, serverInternal?: boolean }
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

/**
 * 只有活动主题自身的已编译 L1 能与 L0 组成 selected-theme 首屏。
 * 默认主题链式回退虽然可单独渲染，但不能冒充当前主题的 exact artifact。
 */
export function exactThemeIdentityForPageResolve(
  payload: PageResolvePayload | null | undefined
): ActiveThemeIdentity | null {
  const output = payload?.renderOutput
  if (
    !payload
    || payload.provider === 'core'
    || payload.fallback
    || !output
    || output.source !== 'active_theme'
    || output.fallback
    || !Number.isInteger(payload.nodeRevision)
    || payload.nodeRevision !== output.nodeRevision
  ) {
    return null
  }

  const identity = normalizeActiveThemeIdentity({
    extensionId: payload.selectedExtensionId,
    version: payload.selectedVersion,
    packageDigest: payload.selectedPackageDigest,
    nodeRevision: payload.nodeRevision
  }, { requireRevision: true })
  if (
    !identity?.version
    || payload.provider !== identity.extensionId
    || payload.selectedProvider !== identity.extensionId
    || (payload.extensionId && payload.extensionId !== identity.extensionId)
  ) {
    return null
  }

  const renderedAttempts = (output.attempts || []).filter(attempt => attempt.outcome === 'rendered')
  if (renderedAttempts.length) {
    const rendered = renderedAttempts.at(-1)!
    const renderedIdentity = normalizeActiveThemeIdentity({
      extensionId: rendered.extensionId,
      packageDigest: rendered.packageDigest,
      nodeRevision: output.nodeRevision
    }, { requireRevision: true })
    if (
      rendered.source !== 'active_theme'
      || !sameActiveThemeIdentity(renderedIdentity, identity, { requireRevision: true })
    ) {
      return null
    }
  }

  return identity
}

export async function requestPageResolveWithRetry(
  request: PageResolveRequest,
  url: string,
  options: {
    timeout: number
    maxAttempts?: number
    retryDelayMs?: number
    serverInternal?: boolean
    sleep?: (ms: number) => Promise<void>
  }
): Promise<PageResolvePayload> {
  const attempts = Math.max(1, Math.floor(options.maxAttempts ?? 2))
  const sleep = options.sleep ?? delay
  let lastError: unknown

  for (let attempt = 1; attempt <= attempts; attempt++) {
    try {
      return await request(url, {
        timeout: options.timeout,
        serverInternal: options.serverInternal
      })
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
