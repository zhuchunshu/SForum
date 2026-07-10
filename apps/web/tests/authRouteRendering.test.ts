import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('auth route rendering', () => {
  test('returns authenticated users from login and registration pages in both themes', () => {
    const authPages = [
      readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/login.vue', import.meta.url), 'utf8'),
      readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/register.vue', import.meta.url), 'utf8'),
      readFileSync(new URL('../../../extensions/dev/themes/sforum-signal-garden/layer/app/pages/login.vue', import.meta.url), 'utf8'),
      readFileSync(new URL('../../../extensions/dev/themes/sforum-signal-garden/layer/app/pages/register.vue', import.meta.url), 'utf8')
    ]

    for (const source of authPages) {
      expect(source).toContain('useAuthReturnNavigation()')
      expect(source).toContain('if (user.value)')
      expect(source).toContain('await returnFromAuth()')
      expect(source).not.toContain('adminRoutes')
      expect(source).not.toContain("can('admin.access')")
    }
  })

  test('preserves default theme authentication success toasts', () => {
    const loginPage = readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/login.vue', import.meta.url), 'utf8')
    const registerPage = readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/register.vue', import.meta.url), 'utf8')

    for (const source of [loginPage, registerPage]) {
      expect(source).toContain('toast.add({')
      expect(source).toContain("color: 'success'")
      expect(source).toContain('duration: 10000')
    }
  })

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
