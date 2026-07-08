import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('auth route rendering', () => {
  test('keeps public auth pages server-rendered so first paint is never an empty Nuxt shell', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    const publicAuthRoutes = [
      '/login',
      '/en/login',
      '/register',
      '/en/register',
      '/forgot-password',
      '/en/forgot-password',
      '/reset-password',
      '/en/reset-password'
    ]

    for (const route of publicAuthRoutes) {
      const escapedRoute = route.replaceAll('/', '\\/')
      expect(config).not.toMatch(new RegExp(`['"]${escapedRoute}['"]\\s*:\\s*\\{[^}]*ssr\\s*:\\s*false`))
    }
  })
})
