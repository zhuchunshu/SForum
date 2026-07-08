import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'

describe('protected route rendering', () => {
  test('does not serve protected user workflows as empty SPA shells', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    const protectedRoutes = [
      '/settings/**',
      '/en/settings/**',
      '/topics/new',
      '/en/topics/new',
      '/t/:topicID/:topicSlug/edit',
      '/en/t/:topicID/:topicSlug/edit'
    ]

    for (const route of protectedRoutes) {
      const escapedRoute = route.replaceAll('/', '\\/').replaceAll('*', '\\*')
      expect(config).not.toMatch(new RegExp(`['"]${escapedRoute}['"]\\s*:\\s*\\{[^}]*ssr\\s*:\\s*false`))
    }
  })

  test('disables route cache for protected user workflows', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    const protectedRoutes = [
      '/settings/**',
      '/en/settings/**',
      '/topics/new',
      '/en/topics/new',
      '/t/:topicID/:topicSlug/edit',
      '/en/t/:topicID/:topicSlug/edit'
    ]

    for (const route of protectedRoutes) {
      const escapedRoute = route.replaceAll('/', '\\/').replaceAll('*', '\\*')
      expect(config).toMatch(new RegExp(`['"]${escapedRoute}['"]\\s*:\\s*\\{[^}]*cache\\s*:\\s*false`))
    }
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
    expect(source).toContain("navigateTo(localePath('/login'))")
  })
})
