import { describe, expect, it } from 'bun:test'
import {
  coreResolveFallback,
  classifyPageResolveFailure,
  disableSharedPageCacheForPageResolve,
  exactThemeIdentityForPageResolve,
  PAGE_RESOLVE_REASON,
  requestPageResolveWithRetry,
  shouldDisablePageResolveSharedCache,
  type MutableRouteRulesContext,
  type PageResolvePayload
} from '../../app/utils/pageResolve'

describe('page resolve resilience', () => {
  const exactThemePayload = (): PageResolvePayload => ({
    page: { id: 'system.not_found' },
    provider: 'sforum.default-theme',
    extensionId: 'sforum.default-theme',
    selectedProvider: 'sforum.default-theme',
    selectedExtensionId: 'sforum.default-theme',
    selectedVersion: '1.0.0',
    selectedPackageDigest: 'a'.repeat(64),
    nodeRevision: 17,
    fallback: false,
    renderOutput: {
      htmlSegments: ['<main>not found</main>'],
      source: 'active_theme',
      fallback: false,
      nodeRevision: 17,
      attempts: [{
        source: 'active_theme',
        extensionId: 'sforum.default-theme',
        packageDigest: 'a'.repeat(64),
        outcome: 'rendered'
      }]
    }
  })

  it('treats the normal empty async-data error state as retryable transport absence', () => {
    expect(classifyPageResolveFailure(undefined)).toBe('retryable')
    expect(classifyPageResolveFailure(null)).toBe('retryable')
  })

  it('captures SSR request headers before any delayed retry', () => {
    const source = Bun.file(new URL('../../app/composables/useApiClient.ts', import.meta.url)).text()
    return source.then((text) => {
      expect(text).toContain("const requestCookie = import.meta.server")
      expect(text.match(/useRequestHeaders\(\['cookie'\]\)/g)).toHaveLength(1)
    })
  })

  it('does not retry a semantic resource 404 and preserves its identity', async () => {
    const error = {
      response: { status: 404, _data: { code: 404, message: 'Not found', data: { reason: 'pages.data_not_found' } } }
    }
    let calls = 0

    expect(classifyPageResolveFailure(error)).toBe('semantic_not_found')
    await expect(requestPageResolveWithRetry(
      async () => {
        calls++
        throw error
      },
      '/pages/resolve?id=forum.topic.show&path=/t/missing',
      { timeout: 5000, maxAttempts: 2, retryDelayMs: 0, sleep: async () => {} }
    )).rejects.toBe(error)
    expect(calls).toBe(1)
  })

  it('does not retry non-retryable API failures', async () => {
    let calls = 0
    const error = { response: { status: 500, _data: { code: 500, message: 'Internal error', data: {} } } }

    expect(classifyPageResolveFailure(error)).toBe('technical')
    await expect(requestPageResolveWithRetry(
      async () => {
        calls++
        throw error
      },
      '/pages/resolve?id=forum.topic.show&path=/t/failure',
      { timeout: 5000, maxAttempts: 2, retryDelayMs: 0, sleep: async () => {} }
    )).rejects.toBe(error)
    expect(calls).toBe(1)
  })

  it('retries a transient resolve failure and keeps the theme response', async () => {
    let calls = 0
    const payload = await requestPageResolveWithRetry(
      async () => {
        calls++
        if (calls === 1) {
          throw new Error('connect ECONNREFUSED')
        }
        return {
          page: { id: 'forum.home' },
          provider: 'sforum.default-theme',
          selectedProvider: 'sforum.default-theme',
          extensionId: 'sforum.default-theme',
          selectedExtensionId: 'sforum.default-theme',
          selectedPackageDigest: 'digest-a',
          fallback: false,
          templateHtml: '<main><sf-home-page></sf-home-page></main>'
        } satisfies PageResolvePayload
      },
      '/pages/resolve?id=forum.home&path=/',
      { timeout: 5000, maxAttempts: 2, retryDelayMs: 0, sleep: async () => {} }
    )

    expect(calls).toBe(2)
    expect(payload.provider).toBe('sforum.default-theme')
    expect(payload.fallback).toBe(false)
    expect(payload.templateHtml).toContain('sf-home-page')
  })

  it('preserves the last transient error when retry attempts are exhausted', async () => {
    const first = new Error('first transport failure')
    const last = Object.assign(new Error('gateway still unavailable'), { statusCode: 503 })
    let calls = 0

    await expect(requestPageResolveWithRetry(
      async () => {
        calls++
        throw calls === 1 ? first : last
      },
      '/pages/resolve?id=forum.home&path=/',
      { timeout: 5000, maxAttempts: 2, retryDelayMs: 0, sleep: async () => {} }
    )).rejects.toBe(last)
    expect(calls).toBe(2)
  })

  it('marks transient core fallback as no-store and disables Nitro SWR', () => {
    const fallback = coreResolveFallback(
      'forum.topic.show',
      true,
      PAGE_RESOLVE_REASON.transportUnavailable
    )
    const context: MutableRouteRulesContext = {
      _nitro: { routeRules: { swr: 60, cache: { maxAge: 60 } } }
    }
    let cacheControl = ''

    expect(shouldDisablePageResolveSharedCache(fallback)).toBe(true)
    disableSharedPageCacheForPageResolve(context, value => {
      cacheControl = value
    })

    expect(cacheControl).toBe('no-store')
    expect(context._nitro?.routeRules?.cache).toBe(false)
    expect(context._nitro?.routeRules?.swr).toBe(false)
  })

  it('marks HTTP 200 fail-closed resolve payloads as no-store too', () => {
    const fallback = {
      page: { id: 'forum.topic.show' },
      provider: 'core',
      selectedProvider: 'sforum.default-theme',
      selectedExtensionId: 'sforum.default-theme',
      selectedPackageDigest: 'digest-a',
      fallback: true,
      reason: PAGE_RESOLVE_REASON.renderFailed
    } satisfies PageResolvePayload
    const context: MutableRouteRulesContext = {
      _nitro: { routeRules: { swr: 60, cache: { maxAge: 60 } } }
    }
    let cacheControl = ''

    expect(shouldDisablePageResolveSharedCache(fallback)).toBe(true)
    disableSharedPageCacheForPageResolve(context, value => {
      cacheControl = value
    })

    expect(cacheControl).toBe('no-store')
    expect(context._nitro?.routeRules?.cache).toBe(false)
    expect(context._nitro?.routeRules?.swr).toBe(false)
  })

  it('keeps authoritative core cacheable by the normal route policy', () => {
    const authoritative = coreResolveFallback(
      'forum.home',
      false,
      PAGE_RESOLVE_REASON.authoritativeCore
    )

    expect(authoritative.provider).toBe('core')
    expect(authoritative.fallback).toBe(false)
    expect(authoritative.reason).toBe(PAGE_RESOLVE_REASON.authoritativeCore)
    expect(shouldDisablePageResolveSharedCache(authoritative)).toBe(false)
  })

  it('extracts exact active-theme L1 identity for matching L0 validation', () => {
    expect(exactThemeIdentityForPageResolve(exactThemePayload())).toEqual({
      extensionId: 'sforum.default-theme',
      version: '1.0.0',
      packageDigest: 'a'.repeat(64),
      nodeRevision: 17
    })
  })

  it('rejects default-theme chain fallback and publication revision mismatch', () => {
    const fallback = exactThemePayload()
    fallback.renderOutput = {
      ...fallback.renderOutput!,
      source: 'default_theme',
      fallback: true,
      attempts: [{
        source: 'default_theme',
        extensionId: 'sforum.default-theme',
        packageDigest: 'a'.repeat(64),
        outcome: 'rendered'
      }]
    }
    expect(exactThemeIdentityForPageResolve(fallback)).toBeNull()

    const stale = exactThemePayload()
    stale.renderOutput = { ...stale.renderOutput!, nodeRevision: 18 }
    expect(exactThemeIdentityForPageResolve(stale)).toBeNull()
  })
})
