import { describe, expect, test } from 'bun:test'

const renderer = await Bun.file(new URL('../app/components/extensions/settings/SFExtensionSettingsRenderer.vue', import.meta.url)).text()
const group = await Bun.file(new URL('../app/components/extensions/settings/SFExtensionSettingsGroup.vue', import.meta.url)).text()
const field = await Bun.file(new URL('../app/components/extensions/settings/SFExtensionSettingsField.vue', import.meta.url)).text()
const callout = await Bun.file(new URL('../app/components/extensions/settings/SFExtensionSettingsCallout.vue', import.meta.url)).text()
const adminLayout = await Bun.file(new URL('../app/layouts/admin.vue', import.meta.url)).text()
const dynamicPage = await Bun.file(new URL('../app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue', import.meta.url)).text()
const fixture = JSON.parse(await Bun.file(new URL('../../../extensions/fixtures/themes/sforum-schema-theme/sforum.extension.json', import.meta.url)).text())

describe('SFExtensionSettingsRenderer buildless contract', () => {
  test('supports tabs, groups, columns, callouts, and linear fallback', () => {
    expect(renderer).toContain("props.settings?.renderer.layout === 'tabs'")
    expect(renderer).toContain('presentationValid')
    expect(renderer).toContain('legacySections')
    expect(renderer).toContain('SFExtensionSettingsCallout')
    expect(renderer).toContain('SFExtensionSettingsGroup')
    expect(group).toContain("group?.columns === 2")
    expect(callout).toContain('<UAlert')
  })

  test('keeps existing field, secret, save, and reset behavior', () => {
    expect(field).toContain("item.type === 'secret'")
    expect(field).toContain("item.type === 'boolean'")
    expect(field).toContain('item.options?.length')
    expect(renderer).toContain('SFAdminFormFooter')
    expect(renderer).toContain("emit('save')")
    expect(renderer).toContain("emit('reset')")
  })

  test('uploaded runtime theme fixture uses the same schema renderer without frontend code', () => {
    expect(fixture.type).toBe('theme')
    expect(fixture.frontend).toBeUndefined()
    expect(fixture.settings.ui.layout).toBe('tabs')
    expect(fixture.settings.ui.callouts.length).toBe(1)
  })

  test('uses SSR extension navigation metadata for dynamic tab title and icon', () => {
    expect(adminLayout).toContain('navigationItem?.label || extensionRouteFallbackLabel(tabId)')
    expect(adminLayout).toContain("navigationItem?.extensionType === 'theme' ? 'i-lucide-palette'")
    expect(dynamicPage).toContain('dynamicTabHydrated.value = true')
    expect(dynamicPage).toContain('if (dynamicTabHydrated.value)')
  })
})
