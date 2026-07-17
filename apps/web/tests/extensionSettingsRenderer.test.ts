import { describe, expect, test } from 'bun:test'
import { compileTemplate, parse } from '@vue/compiler-sfc'

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
    expect(field).toContain("item.type === 'textarea'")
    expect(field).toContain('item.options?.length')
    expect(field).toContain("item.width === 'full'")
    expect(field).toContain('UTextarea')
    expect(field).toContain('max-w-xl')
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
    expect(adminLayout).not.toContain('SFAdminReleaseNotice')
    expect(dynamicPage).toContain('dynamicTabHydrated.value = true')
    expect(dynamicPage).toContain('if (dynamicTabHydrated.value)')
    expect(dynamicPage).toContain("useNuxtData<AdminExtension[]>('admin-extensions')")
    expect(dynamicPage).toContain('cachedExtensions.value?.find(item => item.id === extensionId.value)')
    expect(dynamicPage).toContain('admin-extension-page-bootstrap:${extensionId.value}:${currentPagePath.value}:${locale.value}')
    expect(dynamicPage).toContain('/page-bootstrap?path=${encodeURIComponent(requestedPagePath)}')
    expect(dynamicPage).toContain('await useAsyncData<AdminExtensionPageState>')
    expect(dynamicPage).toContain('requestKey: string')
    expect(dynamicPage).toContain('pageState.value?.requestKey === pageDataKey.value')
    expect(dynamicPage).toContain('current.requestKey !== pageDataKey.value')
    expect(dynamicPage).toContain('if (!pageBootstrap.value.page) return undefined')
    expect(dynamicPage).not.toContain("request<AdminExtension[]>('/admin/extensions')")
    expect(dynamicPage).not.toContain('/admin/extensions?id=')
    expect(dynamicPage.match(/deep: false/g)?.length).toBe(1)
    expect(dynamicPage).toContain('lazy: true')
    expect(dynamicPage).not.toContain('void refresh()')
  })

  test('does not compile the settings renderer branch into a native template element', () => {
    const filename = 'app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue'
    const { descriptor } = parse(dynamicPage, { filename })
    const compiled = compileTemplate({
      source: descriptor.template?.content || '',
      filename,
      id: 'admin-extension-dynamic-page'
    })

    expect(compiled.errors).toEqual([])
    const settingsBranch = descriptor.template?.ast.children.find(child => child.type === 1
      && child.tag === 'div'
      && child.props.some(prop => prop.type === 7
        && prop.name === 'else-if'
        && prop.exp?.type === 4
        && prop.exp.content === 'isSettingsView'))
    expect(settingsBranch?.type).toBe(1)
    if (!settingsBranch || settingsBranch.type !== 1) return

    const rendererBranches = settingsBranch.children.filter(child => child.type === 1
      && (child.tag === 'SFTrustedSettingsComponent' || child.tag === 'SFExtensionSettingsRenderer'))
    expect(rendererBranches.map(child => child.type === 1 ? child.tag : '')).toEqual([
      'SFTrustedSettingsComponent',
      'SFExtensionSettingsRenderer'
    ])
    expect(rendererBranches[0]?.type === 1 && rendererBranches[0].props.some(prop => prop.type === 7 && prop.name === 'if')).toBeTrue()
    expect(rendererBranches[1]?.type === 1 && rendererBranches[1].props.some(prop => prop.type === 7 && prop.name === 'else')).toBeTrue()
  })
})
