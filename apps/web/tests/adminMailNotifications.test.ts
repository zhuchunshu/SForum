import { describe, expect, test } from 'bun:test'

const page = await Bun.file(new URL('../app/pages/admin/settings/mail.vue', import.meta.url)).text()
const modules = await Bun.file(new URL('../app/config/adminModules.ts', import.meta.url)).text()
const zh = JSON.parse(await Bun.file(new URL('../i18n/locales/zh-CN.json', import.meta.url)).text())
const en = JSON.parse(await Bun.file(new URL('../i18n/locales/en-US.json', import.meta.url)).text())

describe('mail and notification admin center', () => {
  test('is visible in the System navigation with settings permission', () => {
    expect(modules).toContain("pageId: '/settings/mail'")
    expect(modules).toContain("requiredPermissions: ['settings.manage']")
    expect(zh.admin.nav.mailSettings).toBe('邮箱与通知')
    expect(en.admin.nav.mailSettings).toBe('Mail and notifications')
  })

  test('exposes four views and provider-neutral test actions', () => {
    expect(page).toContain("value: 'overview'")
    expect(page).toContain("value: 'mail'")
    expect(page).toContain("value: 'notifications'")
    expect(page).toContain("value: 'deliveries'")
    expect(page).toContain("'/admin/mail/policy/restore'")
    expect(page).toContain("'/admin/notifications/test'")
    expect(page).toContain('loadAdminEmailDefault')
    expect(page).toContain("item.name === 'site.admin_email'")
    expect(page).toContain('recipientOrAdminEmailRequired')
  })

  test('follows the established admin settings visual contract', () => {
    expect(page).toContain('rounded-lg border border-slate-200 bg-white')
    expect(page).toContain("t('admin.mailSettings.recommendedTitle')")
    expect(page).toContain('<UCard')
    expect(page).toContain("t('admin.mailSettings.gettingStarted')")
    expect(page).toContain("t('admin.mailSettings.providerHelp')")
  })
})
