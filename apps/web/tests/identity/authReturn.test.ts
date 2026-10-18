import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  normalizeAuthReturnPath,
  requiresDocumentNavigation,
  resolveAuthReturnPath
} from '../../app/utils/identity/authReturn'
import * as authReturn from '../../app/utils/identity/authReturn'

describe('normalizeAuthReturnPath', () => {
  test('preserves safe local paths with query strings and hashes', () => {
    expect(normalizeAuthReturnPath('/t/42?page=2#reply')).toBe('/t/42?page=2#reply')
    expect(normalizeAuthReturnPath('/en/settings/security')).toBe('/en/settings/security')
  })

  test('rejects external, protocol-relative, and malformed paths', () => {
    expect(normalizeAuthReturnPath('https://evil.example')).toBeNull()
    expect(normalizeAuthReturnPath('//evil.example')).toBeNull()
    expect(normalizeAuthReturnPath('/%E0%A4%A')).toBeNull()
  })

  test('rejects malformed percent encoding in query strings', () => {
    expect(normalizeAuthReturnPath('/safe?next=%E0%A4%A')).toBeNull()
  })

  test('rejects malformed percent encoding in hashes', () => {
    expect(normalizeAuthReturnPath('/safe#%E0%A4%A')).toBeNull()
  })

  test('rejects absent and non-string candidates', () => {
    expect(normalizeAuthReturnPath(undefined)).toBeNull()
    expect(normalizeAuthReturnPath(['/settings'])).toBeNull()
  })

  test('rejects authentication pages at any locale depth', () => {
    expect(normalizeAuthReturnPath('/login')).toBeNull()
    expect(normalizeAuthReturnPath('/en/register?redirect=/settings')).toBeNull()
  })
})

describe('resolveAuthReturnPath', () => {
  test('prefers an explicit safe target over the referrer', () => {
    expect(resolveAuthReturnPath('/settings', '/t/42', '/en')).toBe('/settings')
  })

  test('falls back to a safe referrer', () => {
    expect(resolveAuthReturnPath('https://evil.example', '/t/42?page=2#reply', '/en'))
      .toBe('/t/42?page=2#reply')
  })

  test('falls back to localized home and ultimately the root path', () => {
    expect(resolveAuthReturnPath(undefined, '/login', '/en')).toBe('/en')
    expect(resolveAuthReturnPath(undefined, '/login', 'https://evil.example')).toBe('/')
  })
})

describe('buildAuthPageLink', () => {
  test('preserves only a safe explicit redirect across auth forms', () => {
    const buildAuthPageLink = (authReturn as typeof authReturn & {
      buildAuthPageLink?: (path: string, redirect: unknown) => unknown
    }).buildAuthPageLink

    expect(typeof buildAuthPageLink).toBe('function')
    if (!buildAuthPageLink) {
      return
    }

    expect(buildAuthPageLink('/register', '/settings/security')).toEqual({
      path: '/register',
      query: { redirect: '/settings/security' }
    })
    expect(buildAuthPageLink('/register', 'https://evil.example')).toBe('/register')
    expect(buildAuthPageLink('/login', '/en/register?redirect=/settings')).toBe('/login')
    expect(buildAuthPageLink('/en/login', undefined)).toBe('/en/login')
  })
})

describe('requiresDocumentNavigation', () => {
  test('treats API and plugin endpoints as document navigations', () => {
    expect(requiresDocumentNavigation('/api/oauth/authorize?client_id=sfo_client_x&state=s')).toBe(true)
    expect(requiresDocumentNavigation('/api/v1/auth/session')).toBe(true)
    expect(requiresDocumentNavigation('/plugin/echo')).toBe(true)
  })

  test('keeps Nuxt pages on client-side navigation', () => {
    expect(requiresDocumentNavigation('/settings/security')).toBe(false)
    expect(requiresDocumentNavigation('/t/42?page=2')).toBe(false)
    expect(requiresDocumentNavigation('/en/settings/oauth-apps')).toBe(false)
  })

  test('rejects non-path values', () => {
    expect(requiresDocumentNavigation('https://evil.example/api/x')).toBe(false)
    expect(requiresDocumentNavigation('api/oauth/authorize')).toBe(false)
    expect(requiresDocumentNavigation('')).toBe(false)
  })
})

describe('useAuthReturnNavigation source contract', () => {
  test('resolves a safe return destination and replaces the auth history entry', () => {
    const source = readFileSync(
      new URL('../../app/composables/identity/useAuthReturnNavigation.ts', import.meta.url),
      'utf8'
    )

    expect(source).toContain('const route = options ? null : useRoute()')
    expect(source).toContain('const localePath = useLocalePath()')
    expect(source).toContain('const referrerPath = ref<string>()')
    expect(source).toContain('if (import.meta.client && document.referrer)')
    const referrerHandling = source.match(
      /try\s*\{\s*const referrer = new URL\(document\.referrer\)\s*if \(referrer\.origin === window\.location\.origin\) \{\s*referrerPath\.value = `\$\{referrer\.pathname\}\$\{referrer\.search\}\$\{referrer\.hash\}`\s*\}\s*\}\s*catch\s*\{([\s\S]*?)\}/
    )
    expect(referrerHandling).not.toBeNull()
    expect(referrerHandling?.[1]).not.toMatch(/\b(?:throw|navigateTo)\b/)
    expect(source).toContain('const redirect = computed(() => options ? options.explicitRedirect : route?.query.redirect)')
    expect(source).toContain('referrerPath.value')
    expect(source).toContain("localePath('/')")
    expect(source).toContain('buildAuthPageLink(localePath(path), redirect.value)')
    expect(source).toContain('return { destination, returnFromAuth, authPageLink }')
  })

  test('navigates with a full document load for API and plugin endpoints', () => {
    const source = readFileSync(
      new URL('../../app/composables/identity/useAuthReturnNavigation.ts', import.meta.url),
      'utf8'
    )

    // 客户端路由会把 API/插件端点交给 catch-all 页面，回跳必须整页导航。
    expect(source).toContain('if (requiresDocumentNavigation(target))')
    expect(source).toContain('window.location.assign(target)')
    expect(source).toContain("navigateTo(target, { external: true })")
    // 普通页面保持 SPA 回跳。
    expect(source).toContain('navigateTo(target, { replace: true })')
  })
})
