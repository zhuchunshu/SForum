import { describe, expect, test } from 'bun:test'

const route = await Bun.file(new URL('../../app/pages/email-verification.vue', import.meta.url)).text()
const component = await Bun.file(new URL('../../app/components/identity/SFEmailVerificationPage.vue', import.meta.url)).text()
const register = await Bun.file(new URL('../../app/components/identity/SFRegisterFormPage.vue', import.meta.url)).text()
const zhCN = JSON.parse(await Bun.file(new URL('../../i18n/locales/zh-CN.json', import.meta.url)).text())
const enUS = JSON.parse(await Bun.file(new URL('../../i18n/locales/en-US.json', import.meta.url)).text())

describe('email verification waiting flow', () => {
  test('uses an authenticated auth-layout route', () => {
    expect(route).toContain("definePageMeta({ layout: 'auth', requiresAuth: true })")
    expect(route).toContain('<SFEmailVerificationPage />')
  })

  test('supports link verification without manual code entry', () => {
    expect(component).toContain("'/auth/email-verification/request'")
    expect(component).toContain('await refresh()')
    expect(component).toContain('data-testid="email-verification-check"')
    expect(component).toContain('data-testid="email-verification-resend"')
    expect(component).not.toContain('<input')
    expect(component).not.toContain('one-time-code')
  })

  test('redirects required unverified registrations before the normal auth return', () => {
    expect(register).toContain("path: localePath('/email-verification')")
    expect(register.indexOf("path: localePath('/email-verification')"))
      .toBeLessThan(register.indexOf('await returnFromAuth()', register.indexOf('async function finishRegistration')))
  })

  test('describes the implemented link flow instead of deferred mail work', () => {
    expect(zhCN.admin.settings.registration.requireEmailVerificationHint).toContain('验证链接')
    expect(zhCN.admin.settings.registration.requireEmailVerificationHint).not.toContain('将在邮件能力中完善')
    expect(enUS.admin.settings.registration.requireEmailVerificationHint).toContain('verification link')
    expect(zhCN.auth.emailVerificationDescription).toContain('点击邮件中的链接')
    expect(enUS.auth.emailVerificationDescription).toContain('Click the link')
  })
})
