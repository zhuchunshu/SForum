import { describe, expect, it } from 'bun:test'
import {
  coreResolveFallback,
  disableSharedPageCacheForPageResolve,
  PAGE_RESOLVE_REASON,
  requestPageResolveWithRetry,
  shouldDisablePageResolveSharedCache,
  type MutableRouteRulesContext,
  type PageResolvePayload
} from '../app/utils/pageResolve'

describe('page resolve resilience', () => {
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
})
