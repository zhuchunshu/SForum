import { describe, expect, test } from 'bun:test'

const page = await Bun.file(new URL('../app/pages/admin/settings/mail.vue', import.meta.url)).text()
const modules = await Bun.file(new URL('../app/config/adminModules.ts', import.meta.url)).text()
const zh = JSON.parse(await Bun.file(new URL('../i18n/locales/zh-CN.json', import.meta.url)).text())
const en = JSON.parse(await Bun.file(new URL('../i18n/locales/en-US.json', import.meta.url)).text())

describe('mail and notification admin center', () => {
  test('is visible in the System navigation with settings permission', () => {
    expect(modules).toContain("pageId: '/settings/mail'")
    expect(modules).toContain("requiredPermissions: ['settings.mail.manage']")
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

  test('uses readable tab size and localizes delivery list codes', () => {
    expect(page).toContain('size="md"')
    expect(page).toContain('deliveryStatusLabel')
    expect(page).toContain('deliveryTemplateLabel')
    expect(page).toContain('deliveryReasonLabel')
    expect(page).not.toContain('<SFBadge>{{ item.status }}</SFBadge>')
    expect(zh.admin.mailSettings.deliveryStatus.sent).toBe('已发送')
    expect(en.admin.mailSettings.deliveryStatus.sent).toBe('Sent')
    expect(zh.admin.mailSettings.templates.admin_test).toBe('管理后台测试邮件')
    expect(en.admin.mailSettings.templates.identity_password_reset).toBe('Password reset')
    expect(zh.admin.mailSettings.reasons.provider_unavailable).toBe('邮件提供商不可用')
    expect(en.admin.mailSettings.reasons.smtp_transport_failed).toBe('SMTP transport failed')
  })
})
