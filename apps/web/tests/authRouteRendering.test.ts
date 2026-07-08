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

  test('wires password policy feedback into registration and reset forms', () => {
    const registerPage = readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/register.vue', import.meta.url), 'utf8')
    const resetPage = readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/reset-password.vue', import.meta.url), 'utf8')
    const zhCN = readFileSync(new URL('../i18n/locales/zh-CN.json', import.meta.url), 'utf8')
    const enUS = readFileSync(new URL('../i18n/locales/en-US.json', import.meta.url), 'utf8')

    for (const source of [registerPage, resetPage]) {
      expect(source).toContain('passwordPolicyRequirements')
      expect(source).toContain('passwordPolicyProgress')
      expect(source).toContain('auth.passwordStrength')
    }
    expect(zhCN).toContain('passwordRequirementSymbol')
    expect(enUS).toContain('passwordRequirementSymbol')
  })
})
