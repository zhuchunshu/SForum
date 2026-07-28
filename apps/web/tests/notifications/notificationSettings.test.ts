import { describe, expect, test } from 'bun:test'
import {
  canOverrideNotificationPreference,
  preferenceUpdateItems,
  type NotificationPreferenceItem
} from '../../app/composables/notifications/useNotificationPreferences'
import {
  canEditNotificationPolicy,
  notificationPolicyUpdates,
  type NotificationPolicyItem
} from '../../app/components/admin/settings/notifications/model'
import {
  decodeApplicationServerKey,
  serializePushSubscription,
  WEB_PUSH_SCOPE,
  WEB_PUSH_WORKER_PATH
} from '../../app/composables/notifications/useWebPush'

const source = (path: string) => Bun.file(new URL(path, import.meta.url)).text()
const userPage = await source('../../app/pages/settings/notifications.vue')
const defaultThemeSettings = await source('../../../../extensions/builtin/themes/sforum-default/templates/settings-notifications.html')
const nocturneThemeSettings = await source('../../../../extensions/builtin/themes/sforum-nocturne/templates/settings-notifications.html')
const themeRenderer = await source('../../app/components/SFThemeTemplate.vue')
const userSettings = await source('../../app/components/settings/SFNotificationSettingsPage.vue')
const settingsStyles = await source('../../app/assets/css/sforum-settings.css')
const preferencesApi = await source('../../app/composables/notifications/useNotificationPreferences.ts')
const webPushApi = await source('../../app/composables/notifications/useWebPush.ts')
const webPushSettings = await source('../../app/components/settings/SFWebPushSettingsSection.vue')
const accountNav = await source('../../app/components/settings/SFSettingsAccountNav.vue')
const adminPage = await source('../../app/pages/admin/settings/notifications.vue')
const adminSettings = await source('../../app/components/admin/settings/notifications/SFAdminNotificationPolicyPage.vue')
const adminChannelsApi = await source('../../app/composables/notifications/useAdminNotificationChannels.ts')
const adminChannels = await source('../../app/components/admin/settings/notifications/SFAdminNotificationChannels.vue')
const mailPage = await source('../../app/pages/admin/settings/mail.vue')
const adminModules = await source('../../app/config/adminModules.ts')
const roleTemplates = await source('../../app/config/roleTemplates.ts')
const zh = JSON.parse(await source('../../i18n/locales/zh-CN.json'))
const en = JSON.parse(await source('../../i18n/locales/en-US.json'))

function preference(overrides: Partial<NotificationPreferenceItem> = {}): NotificationPreferenceItem {
  return {
    type: 'reply', category: 'conversation', channel: 'in_app', active: true,
    enabled: true, recommendedEnabled: true, userConfigurable: true,
    required: false, state: 'inherit', effective: true, ...overrides
  }
}

function policy(overrides: Partial<NotificationPolicyItem> = {}): NotificationPolicyItem {
  return {
    type: 'reply', category: 'conversation', channel: 'in_app', active: true,
    enabled: true, recommendedEnabled: true, userConfigurable: true,
    required: false, ...overrides
  }
}

describe('Notification Platform V2 settings', () => {
  test('only submits user controls that are active, available, configurable, and non-required', () => {
    expect(canOverrideNotificationPreference(preference())).toBe(true)
    expect(canOverrideNotificationPreference(preference({ required: true }))).toBe(false)
    expect(canOverrideNotificationPreference(preference({ active: false }))).toBe(false)
    expect(canOverrideNotificationPreference(preference({ enabled: false }))).toBe(false)
    expect(canOverrideNotificationPreference(preference({ channel: 'email', enabled: false, state: 'enabled' }))).toBe(false)
    expect(canOverrideNotificationPreference(preference({ channelAvailable: false }))).toBe(false)
    expect(canOverrideNotificationPreference(preference({ userConfigurable: false }))).toBe(false)
    expect(preferenceUpdateItems([
      preference(),
      preference({ type: 'mention', required: true }),
      preference({ type: 'moderation_approved', channel: 'email', channelAvailable: false })
    ], { 'reply:in_app': 'disabled' })).toEqual([
      { type: 'reply', channel: 'in_app', state: 'disabled' }
    ])
  })

  test('keeps required and inactive policy rows outside administrator mutations', () => {
    expect(canEditNotificationPolicy(policy())).toBe(true)
    expect(canEditNotificationPolicy(policy({ required: true }))).toBe(false)
    expect(canEditNotificationPolicy(policy({ active: false }))).toBe(false)
    expect(notificationPolicyUpdates([
      policy({ enabled: false, recommendedEnabled: false }),
      policy({ type: 'admin_test', required: true }),
      policy({ type: 'plugin.notice', active: false })
    ])).toEqual([
      { type: 'reply', channel: 'in_app', enabled: false, recommendedEnabled: false, userConfigurable: true }
    ])
  })

  test('uses the shared account shell and only the current-user preference endpoints', () => {
    expect(userPage).toContain('page="forum.settings.notifications"')
    expect(userSettings).toContain('<SFSettingsShell')
    expect(preferencesApi).toContain("'/notification-preferences'")
    expect(preferencesApi).toContain("'/notification-preferences/restore'")
    expect(userSettings).toContain('applyCategory')
    expect(userSettings).toContain('canOverrideNotificationPreference')
    expect(userSettings).toContain('v-if="canOverrideNotificationPreference(item)"')
    expect(settingsStyles).toContain('.sforum-settings__mobile-nav .sf-home-navigation__select-control')
    expect(accountNav).toContain("localePath('/settings/notifications')")
  })

  test('freezes the replaceable Page Registry theme and Host island contract', () => {
    for (const template of [defaultThemeSettings, nocturneThemeSettings]) {
      expect(template).toContain('data-theme-owned="presentation"')
      expect(template).toContain('data-page="forum.settings.notifications"')
      expect(template).toContain('data-layout="fullwidth-3col"')
      expect(template).toContain('<sf-notification-settings>')
      expect(template).toContain('<sf-navbar>')
      expect(template).toContain('<sf-footer>')
    }
    expect(themeRenderer).toContain("'notifications.component.settings': defineAsyncComponent(() => import('./settings/SFNotificationSettingsPage.vue'))")
    expect(themeRenderer).toContain("'sf-notification-settings': { componentId: 'notifications.component.settings' }")
  })

  test('uses permission-aware tabs in the unified Mail and Notifications admin surface', () => {
    expect(adminPage).toContain("adminRoutes.path('/settings/mail')")
    expect(mailPage).toContain('SFAdminFixedTabNav')
    expect(mailPage).toContain('SFAdminNotificationPolicyPage')
    expect(mailPage).toContain('SFAdminNotificationChannels')
    expect(adminSettings).toContain("'/admin/notifications/policy'")
    expect(adminSettings).toContain("'/admin/notifications/policy/restore'")
    expect(adminSettings).toContain("can('settings.notifications.manage')")
    expect(adminSettings).toContain("'/admin/notifications/test'")
    expect(mailPage).not.toContain('SFAdminMailNotificationsTab')
    expect(adminModules).toContain("id: '/settings/notifications'")
    expect(adminModules).toContain("requiredPermissions: ['settings.mail.manage', 'settings.notifications.manage']")
    expect(adminModules).toContain("requiredPermissions: ['settings.notifications.manage']")
    expect(roleTemplates).toContain("'settings.notifications.manage'")
  })

  test('keeps Web Push inside the Host-owned worker scope and sends browser keys only to the create API', () => {
    expect(WEB_PUSH_WORKER_PATH).toBe('/_sforum/notifications/sw.js')
    expect(WEB_PUSH_SCOPE).toBe('/_sforum/notifications/')
    expect([...decodeApplicationServerKey('AQID')]).toEqual([1, 2, 3])
    expect(serializePushSubscription({
      toJSON: () => ({ endpoint: 'https://push.example/subscription-secret', keys: { p256dh: 'public-client-key', auth: 'auth-secret' } })
    } as PushSubscription)).toEqual({
      endpoint: 'https://push.example/subscription-secret',
      keys: { p256dh: 'public-client-key', auth: 'auth-secret' }
    })
    expect(webPushApi).toContain("'/web-push/config'")
    expect(webPushApi).toContain("'/web-push/subscriptions'")
    expect(webPushSettings).toContain('Notification.requestPermission()')
    expect(webPushSettings).toContain('navigator.serviceWorker.register(WEB_PUSH_WORKER_PATH, { scope: WEB_PUSH_SCOPE })')
    expect(webPushSettings).toContain('item.endpointOrigin')
    expect(webPushSettings).not.toContain('item.endpoint }}')
    expect(webPushSettings).not.toContain('item.p256dh')
    expect(webPushSettings).not.toContain('item.auth')
  })

  test('manages generic channel selection, reset, self-test, and redacted delivery health', () => {
    expect(adminSettings).not.toContain('<SFAdminNotificationChannels')
    expect(mailPage).toContain('channels: SFAdminNotificationChannels')
    expect(adminChannelsApi).toContain("`${base}/channels`")
    expect(adminChannelsApi).toContain("`${base}/channels/${channel}`")
    expect(adminChannelsApi).toContain("`${base}/channels/${channel}/reset`")
    expect(adminChannelsApi).toContain("`${base}/channels/${channel}/test`")
    expect(adminChannelsApi).toContain("`${base}/deliveries?limit=")
    expect(adminChannels).toContain('item.selection.providerArtifact.packageDigest')
    expect(adminChannels).toContain('delivery.providerArtifactDigest')
    expect(adminChannels).not.toContain('delivery.recipientUserId')
    expect(adminChannels).not.toContain('delivery.payload')
    expect(adminChannels).not.toContain('delivery.idempotencyKey')
  })

  test('ships both operator and member copy', () => {
    expect(en.notificationSettings.states.inherit).toBe('Inherit site choice')
    expect(zh.notificationSettings.states.inherit).toBe('继承站点选择')
    expect(en.admin.notificationSettings.channelEnabled).toBe('Channel available')
    expect(zh.admin.notificationSettings.channelEnabled).toBe('渠道可用')
    expect(en.admin.notificationSettings.emailEnabled).toBe('Send email notification')
    expect(zh.admin.notificationSettings.emailEnabled).toBe('发送邮件通知')
    expect(en.admin.nav.notificationSettings).toBe('Notification settings')
    expect(zh.admin.nav.notificationSettings).toBe('通知设置')
    expect(en.notificationSettings.webPush.enableBrowser).toBe('Enable on this browser')
    expect(zh.notificationSettings.webPush.enableBrowser).toBe('在此浏览器启用')
    expect(en.admin.notificationSettings.channelsAdmin.deliveryTitle).toBe('Delivery health')
    expect(zh.admin.notificationSettings.channelsAdmin.deliveryTitle).toBe('投递健康')
  })
})
