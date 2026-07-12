import { describe, expect, test } from 'bun:test'

const page = await Bun.file(new URL('../app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue', import.meta.url)).text()
// SMTP 使用 multi-file manifest：入口 + includes partials。
const smtpRoot = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/sforum.extension.json', import.meta.url)).text())
const smtpFrontend = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/manifest/frontend.json', import.meta.url)).text())
const smtpContributions = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/manifest/contributions.json', import.meta.url)).text())
const smtpSettings = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/manifest/settings.json', import.meta.url)).text())
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
    expect(smtpRoot.includes.settings).toBe('manifest/settings.json')
    expect(smtpRoot.includes.langs).toBe('manifest/langs')
    expect(smtpFrontend.admin.components['smtp-settings-page']).toBe('components/SmtpSettingsPage.vue')
    expect(smtpContributions[0].point).toBe('admin.extension.settings.page')
    expect(typeof smtpSettings[0].label).toBe('object')
    expect(smtpSettings[0].label['zh-CN']).toBeTruthy()
    expect(smtpSettings[0].label['en-US']).toBeTruthy()
    expect(smtpZh.title).toContain('SMTP')
    expect(smtpZh.fields.host.label).toBeTruthy()
  })

  test('default theme owns tabbed custom settings page contribution', async () => {
    const themeRoot = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/sforum.extension.json', import.meta.url)).text())
    const themeFrontend = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/manifest/frontend.json', import.meta.url)).text())
    const themeContributions = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/manifest/contributions.json', import.meta.url)).text())
    const themeSettings = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/manifest/settings.json', import.meta.url)).text())
    const themePage = await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/frontend/admin/components/ThemeSettingsPage.vue', import.meta.url)).text()
    const themeZh = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/frontend/admin/locales/zh-CN.json', import.meta.url)).text())

    expect(themeRoot.type).toBe('theme')
    expect(themeRoot.includes.frontend).toBe('manifest/frontend.json')
    expect(themeRoot.includes.contributions).toBe('manifest/contributions.json')
    expect(themeFrontend.admin.components['theme-settings-page']).toBe('components/ThemeSettingsPage.vue')
    expect(themeContributions[0].point).toBe('admin.extension.settings.page')
    expect(themeSettings.length).toBeGreaterThanOrEqual(15)
    expect(themePage).toContain("activeTab")
    expect(themePage).toContain('tabs.home')
    expect(themePage).toContain('tabs.rail')
    expect(themePage).toContain('tabs.nav')
    expect(themePage).toContain('tabs.layout')
    expect(themeZh.tabs.home).toBeTruthy()
    expect(themeZh.fields['home.notice.zh-CN'].label).toBeTruthy()
  })
})
