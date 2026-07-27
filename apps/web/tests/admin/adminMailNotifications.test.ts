import { describe, expect, test } from 'bun:test'
import { mailDeliveryCodeKey, recommendedMailPolicy } from '../../app/components/admin/settings/mail/model'

const modules = await Bun.file(new URL('../../app/config/adminModules.ts', import.meta.url)).text()
const overviewTab = await Bun.file(new URL('../../app/components/admin/settings/mail/tabs/SFAdminMailOverviewTab.vue', import.meta.url)).text()
const providerTab = await Bun.file(new URL('../../app/components/admin/settings/mail/tabs/SFAdminMailProviderTab.vue', import.meta.url)).text()
const notificationsTab = await Bun.file(new URL('../../app/components/admin/settings/mail/tabs/SFAdminMailNotificationsTab.vue', import.meta.url)).text()
const deliveriesTab = await Bun.file(new URL('../../app/components/admin/settings/mail/tabs/SFAdminMailDeliveriesTab.vue', import.meta.url)).text()
const zh = JSON.parse(await Bun.file(new URL('../../i18n/locales/zh-CN.json', import.meta.url)).text())
const en = JSON.parse(await Bun.file(new URL('../../i18n/locales/en-US.json', import.meta.url)).text())

describe('mail and notification admin center', () => {
  test('is visible in the System navigation with settings permission', () => {
    expect(modules).toContain("pageId: '/settings/mail'")
    expect(modules).toContain("requiredPermissions: ['settings.mail.manage']")
    expect(zh.admin.nav.mailSettings).toBe('邮箱与通知')
    expect(en.admin.nav.mailSettings).toBe('Mail and notifications')
  })

  test('keeps provider selection generic and preserves the admin email fallback', () => {
    expect(providerTab).toContain("'/admin/mail/provider'")
    expect(providerTab).toContain('`/extensions/${selected}/pages/settings`')
    expect(providerTab).toContain("item.name === 'site.admin_email'")
    expect(providerTab).toContain('recipientOrAdminEmailRequired')
    expect(providerTab).not.toContain('selected === \'sforum.smtp\'')
  })

  test('owns notification policy defaults, save, restore, and self-test behavior', () => {
    expect(recommendedMailPolicy()).toEqual({
      reply: { inAppEnabled: true, emailEnabled: true },
      mention: { inAppEnabled: true, emailEnabled: true },
      moderation: { inAppEnabled: true, emailEnabled: true }
    })
    expect(notificationsTab).toContain("'/admin/mail/policy/restore'")
    expect(notificationsTab).toContain("'/admin/notifications/test'")
    expect(notificationsTab).toContain('duration: 10000')
  })

  test('keeps overview guidance and localizes delivery list codes', () => {
    expect(overviewTab).toContain("t('admin.mailSettings.gettingStarted')")
    expect(overviewTab).toContain("t('admin.mailSettings.stepProvider')")
    expect(deliveriesTab).toContain('mailDeliveryCodeKey')
    expect(mailDeliveryCodeKey('reasons', 'smtp.transport.failed')).toBe('admin.mailSettings.reasons.smtp_transport_failed')
    expect(zh.admin.mailSettings.deliveryStatus.sent).toBe('已发送')
    expect(en.admin.mailSettings.deliveryStatus.sent).toBe('Sent')
    expect(zh.admin.mailSettings.templates.admin_test).toBe('管理后台测试邮件')
    expect(en.admin.mailSettings.templates.identity_password_reset).toBe('Password reset')
    expect(zh.admin.mailSettings.reasons.provider_unavailable).toBe('邮件提供商不可用')
    expect(en.admin.mailSettings.reasons.smtp_transport_failed).toBe('SMTP transport failed')
  })
})
