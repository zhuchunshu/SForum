import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('auth route rendering', () => {
  test('guards default auth pages before setup and returns users after authentication', () => {
    // guest middleware 属于 Nuxt 路由壳；凭证表单逻辑在 Host body 岛上。
    const authRouteShells = [
      readFileSync(new URL('../app/pages/login.vue', import.meta.url), 'utf8'),
      readFileSync(new URL('../app/pages/register.vue', import.meta.url), 'utf8')
    ]
    for (const source of authRouteShells) {
      expect(source).toMatch(/definePageMeta\s*\(\s*\{[^}]*middleware\s*:\s*['"]guest['"][^}]*\}\s*\)/)
      expect(source).toContain('SFPageOutlet')
    }

    const authFormIslands = [
      readFileSync(new URL('../app/components/SFLoginFormPage.vue', import.meta.url), 'utf8'),
      readFileSync(new URL('../app/components/SFRegisterFormPage.vue', import.meta.url), 'utf8')
    ]
    for (const source of authFormIslands) {
      expect(source).toContain('useAuthReturnNavigation()')
      expect(source).toMatch(/const\s*\{\s*setUser\s*\}\s*=\s*useAuthSession\s*\(\s*\)/)
      expect(source).toMatch(/setUser\s*\(\s*currentUser\s*\)/)
      expect(source).not.toMatch(/if\s*\(\s*user\.value\s*\)/)
      expect(source).not.toContain('adminRoutes')
      expect(source).not.toContain("can('admin.access')")

      const setUserCall = /setUser\s*\(\s*currentUser\s*\)/.exec(source)
      const returnCalls = [...source.matchAll(/await\s+returnFromAuth\s*\(\s*\)/g)]

      expect(setUserCall).not.toBeNull()
      expect(returnCalls).toHaveLength(1)
      expect(returnCalls.some(call => call.index > setUserCall!.index)).toBe(true)
    }
  })

  test('preserves default theme authentication success toasts', () => {
    const loginPage = readFileSync(new URL('../../../apps/web/app/components/SFLoginFormPage.vue', import.meta.url), 'utf8')
    const registerPage = readFileSync(new URL('../../../apps/web/app/components/SFRegisterFormPage.vue', import.meta.url), 'utf8')

    for (const source of [loginPage, registerPage]) {
      expect(source).toContain('toast.add({')
      expect(source).toContain("color: 'success'")
      expect(source).toContain('duration: 10000')
    }
  })

  test('offers a login-risk verification recovery flow after account-level failures', () => {
    const loginPage = readFileSync(new URL('../../../apps/web/app/components/SFLoginFormPage.vue', import.meta.url), 'utf8')

    expect(loginPage).toContain("purpose=login_risk")
    expect(loginPage).toContain("reason === 'human_verification.required'")
    expect(loginPage).toContain('body.humanVerification = {')
    expect(loginPage).toContain('<altcha-widget')
  })

  test('preserves explicit return targets when switching between auth forms', () => {
    const loginPage = readFileSync(new URL('../../../apps/web/app/components/SFLoginFormPage.vue', import.meta.url), 'utf8')
    const registerPage = readFileSync(new URL('../../../apps/web/app/components/SFRegisterFormPage.vue', import.meta.url), 'utf8')

    expect(loginPage).toContain('const { returnFromAuth, authPageLink } = useAuthReturnNavigation()')
    expect(loginPage.match(/:to="authPageLink\('\/register'\)"/g)).toHaveLength(2)
    expect(registerPage).toContain('const { returnFromAuth, authPageLink } = useAuthReturnNavigation()')
    // 注册页：顶栏 tab、验证提示、表单旁链接、页脚登录入口（共 4 处）。
    expect(registerPage.match(/:to="authPageLink\('\/login'\)"/g)).toHaveLength(4)
    expect(registerPage).not.toContain(":to=\"localePath('/login')\"")
  })

  test('keeps public auth pages server-rendered so first paint is never an empty Nuxt shell', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    const publicAuthRoutes = [
      '/login',
      '/register',
      '/forgot-password',
      '/reset-password'
    ]

    for (const route of publicAuthRoutes) {
      const escapedRoute = route.replaceAll('/', '\\/')
      expect(config).not.toMatch(new RegExp(`['"]${escapedRoute}['"]\\s*:\\s*\\{[^}]*ssr\\s*:\\s*false`))
    }
  })

  test('wires password policy feedback into registration and reset forms', () => {
    const registerPage = readFileSync(new URL('../../../apps/web/app/components/SFRegisterFormPage.vue', import.meta.url), 'utf8')
    const resetPage = readFileSync(new URL('../../../apps/web/app/components/SFRecoveryConfirmPage.vue', import.meta.url), 'utf8')
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
