import { describe, expect, test } from 'bun:test'

const page = await Bun.file(new URL('../app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue', import.meta.url)).text()
const renderer = await Bun.file(new URL('../app/components/extensions/settings/SFExtensionSettingsRenderer.vue', import.meta.url)).text()
// SMTP 使用 multi-file manifest：入口 + includes partials。
const smtpRoot = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/sforum.extension.json', import.meta.url)).text())
const smtpSettings = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/plugins/sforum-smtp/manifest/settings.json', import.meta.url)).text())
const sdk = await Bun.file(new URL('../packages/admin-sdk/src/index.ts', import.meta.url)).text()
const zh = JSON.parse(await Bun.file(new URL('../i18n/locales/zh-CN.json', import.meta.url)).text())

describe('extension settings plugin ownership', () => {
  test('host dynamic page owns schema and prebuilt settings rendering without plugin-specific copy', () => {
    expect(page).toContain('SFExtensionSettingsRenderer')
    expect(page).toContain('SFTrustedSettingsComponent')
    expect(renderer).toContain('SFAdminFormFooter')
    expect(page).toContain('settingsComponentContext')
    // Core must not hardcode SMTP product copy.
    expect(page).not.toContain('STARTTLS + 端口 587')
    expect(page).not.toContain('sforum.smtp')
    expect(zh.admin.extensions.dynamic.recommendedTitle).toBeTruthy()
    expect(zh.admin.extensions.dynamic.mailProviderRecommendedTitle).toBeUndefined()
  })

  test('admin SDK exposes only the prebuilt settings component bridge', () => {
    expect(sdk).toContain('AdminMicroFrontendBridgeV1')
    expect(sdk).not.toContain('AdminSlotContextMap')
    expect(sdk).not.toContain('useSForumAdminHost')
  })

  test('smtp plugin owns multilingual schema settings and probe action', () => {
    expect(smtpRoot.includes.settings).toBe('manifest/settings.json')
    expect(smtpRoot.includes.langs).toBe('manifest/langs')
    expect(smtpRoot.includes.frontend).toBeUndefined()
    expect(smtpRoot.includes.contributions).toBeUndefined()
    expect(smtpSettings.ui.layout).toBe('tabs')
    expect(smtpSettings.actions[0].kind).toBe('provider_probe')
    expect(typeof smtpSettings.fields[0].label).toBe('object')
    expect(smtpSettings.fields[0].label['zh-CN']).toBeTruthy()
    expect(smtpSettings.fields[0].label['en-US']).toBeTruthy()
  })

  test('default theme uses host schema settings without frontend.admin', async () => {
    // 普通主题设置走宿主 schema-driven 页，不需要任何前端构建。
    const themeRoot = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/sforum.extension.json', import.meta.url)).text())
    const themeSettings = JSON.parse(await Bun.file(new URL('../../../extensions/builtin/themes/sforum-default/manifest/settings.json', import.meta.url)).text())

    expect(themeRoot.type).toBe('theme')
    expect(themeRoot.includes.settings).toBe('manifest/settings.json')
    expect(themeRoot.includes.frontend).toBeUndefined()
    expect(themeRoot.includes.contributions).toBeUndefined()
    expect(themeSettings.schemaVersion).toBe(1)
    expect(themeSettings.ui.layout).toBe('tabs')
    expect(themeSettings.ui.tabs.length).toBeGreaterThanOrEqual(3)
    expect(themeSettings.fields.length).toBeGreaterThanOrEqual(15)
    expect(themeSettings.fields.some((item: { group?: Record<string, string> }) => item.group?.['zh-CN'] === '首页文案')).toBe(true)
    expect(themeSettings.fields.some((item: { key?: string }) => item.key === 'home.notice.zh-CN')).toBe(true)
  })
})
