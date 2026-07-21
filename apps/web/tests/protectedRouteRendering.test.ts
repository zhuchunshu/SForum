import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'

describe('protected route rendering', () => {
  test('avoids the nuxt-i18n Nitro context startup race', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')

    expect(config).toMatch(/experimental\s*:\s*\{[^}]*nitroContextDetection\s*:\s*false/s)
    expect(config).toContain("strategy: 'prefix_except_default'")
    expect(config).toContain('detectBrowserLanguage: {')
  })

  test('does not serve protected user workflows as empty SPA shells', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    const protectedRoutes = [
      '/settings/**',
      '/en/settings/**',
      '/topics/new',
      '/en/topics/new',
      '/t/**',
      '/en/t/**'
    ]

    for (const route of protectedRoutes) {
      const escapedRoute = route.replaceAll('/', '\\/').replaceAll('*', '\\*')
      expect(config).not.toMatch(new RegExp(`['"]${escapedRoute}['"]\\s*:\\s*\\{[^}]*ssr\\s*:\\s*false`))
    }
  })

  test('disables route cache for protected user workflows', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    // /t/** 允许匿名短 SWR；登录/?edit= 由 topic-page-cache 中间件禁缓存（见下测）。
    const protectedRoutes = [
      '/settings/**',
      '/en/settings/**',
      '/topics/new',
      '/en/topics/new'
    ]

    for (const route of protectedRoutes) {
      const escapedRoute = route.replaceAll('/', '\\/').replaceAll('*', '\\*')
      expect(config).toMatch(new RegExp(`['"]${escapedRoute}['"]\\s*:\\s*\\{[^}]*cache\\s*:\\s*false`))
    }
  })

  test('topic detail allows anonymous SWR but gates auth/edit via middleware', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    expect(config).toMatch(/['"]\/t\/\*\*['"]\s*:\s*\{\s*swr\s*:\s*60\s*\}/)
    expect(config).toMatch(/['"]\/en\/t\/\*\*['"]\s*:\s*\{\s*swr\s*:\s*60\s*\}/)

    const middlewarePath = new URL('../server/middleware/topic-page-cache.ts', import.meta.url)
    expect(existsSync(middlewarePath)).toBe(true)
    const source = readFileSync(middlewarePath, 'utf8')
    expect(source).toContain('sforum_session')
    expect(source).toContain('searchParams.has(\'edit\')')
    expect(source).toContain('routeRules.cache = false')
    expect(source).toContain('routeRules.swr = false')
  })

  test('disables shared page caching for every session-bearing SSR request', () => {
    const middlewarePath = new URL('../server/middleware/public-session-cache.ts', import.meta.url)
    expect(existsSync(middlewarePath)).toBe(true)

    const source = readFileSync(middlewarePath, 'utf8')
    expect(source).toContain('sforum_session')
    expect(source).toContain("accept.includes('text/html')")
    expect(source).toContain("path.endsWith('/_payload.json')")
    expect(source).toContain("setHeader(event, 'cache-control', 'no-store')")
    expect(source).toContain('routeRules.cache = false')
    expect(source).toContain('routeRules.swr = false')
  })

  test('has a global middleware that enforces requiresAuth page metadata', () => {
    const middlewarePath = new URL('../app/middleware/auth.global.ts', import.meta.url)

    expect(existsSync(middlewarePath)).toBe(true)

    const source = readFileSync(middlewarePath, 'utf8')
    expect(source).toContain('to.meta.requiresAuth')
    expect(source).toContain("localePath('/login')")
  })

  test('redirects ordinary protected routes to login instead of rendering a 503 shell', () => {
    const middlewarePath = new URL('../app/middleware/auth.global.ts', import.meta.url)
    const source = readFileSync(middlewarePath, 'utf8')

    expect(source).not.toContain("status.value === 'unavailable'")
    expect(source).not.toContain('abortNavigation(createError')
    expect(source).toContain("path: localePath('/login')")
    expect(source).toContain('query: { redirect: to.fullPath }')
  })
})
