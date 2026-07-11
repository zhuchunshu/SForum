import { describe, expect, test } from 'bun:test'

const page = await Bun.file(new URL('../app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue', import.meta.url)).text()
const smtpManifest = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/sforum.extension.json', import.meta.url)).text())
const sdk = await Bun.file(new URL('../packages/admin-sdk/src/index.ts', import.meta.url)).text()
const zh = JSON.parse(await Bun.file(new URL('../i18n/locales/zh-CN.json', import.meta.url)).text())
const smtpZh = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/frontend/admin/locales/zh-CN.json', import.meta.url)).text())

describe('extension settings plugin ownership', () => {
  test('host dynamic page supports custom settings slots without plugin-specific copy', () => {
    expect(page).toContain("admin.extension.settings.page")
    expect(page).toContain("admin.extension.settings.header")
    expect(page).toContain("admin.extension.settings.footer")
    expect(page).toContain('SFAdminFormFooter')
    expect(page).toContain('settingsSlotContext')
    // Core must not hardcode SMTP product copy.
    expect(page).not.toContain('STARTTLS + 端口 587')
    expect(page).not.toContain('sforum.smtp')
    expect(zh.admin.extensions.dynamic.recommendedTitle).toBeTruthy()
    expect(zh.admin.extensions.dynamic.mailProviderRecommendedTitle).toBeUndefined()
  })

  test('admin SDK declares extension settings slot contexts', () => {
    expect(sdk).toContain("'admin.extension.settings.page'")
    expect(sdk).toContain('AdminExtensionSettingsContext')
  })

  test('smtp plugin owns multilingual settings and custom page contribution', () => {
    expect(smtpManifest.frontend.admin.components['smtp-settings-page']).toBe('components/SmtpSettingsPage.vue')
    expect(smtpManifest.contributions[0].point).toBe('admin.extension.settings.page')
    expect(typeof smtpManifest.settings[0].label).toBe('object')
    expect(smtpManifest.settings[0].label['zh-CN']).toBeTruthy()
    expect(smtpManifest.settings[0].label['en-US']).toBeTruthy()
    expect(smtpZh.title).toContain('SMTP')
    expect(smtpZh.fields.host.label).toBeTruthy()
  })
})
