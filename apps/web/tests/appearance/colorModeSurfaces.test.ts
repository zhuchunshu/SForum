import { describe, expect, test } from 'bun:test'

const read = (path: string) => Bun.file(new URL(path, import.meta.url)).text()

const [navbar, adminLayout, publicBridge, adminBridge, zhCN, enUS] = await Promise.all([
  read('../../app/components/SFNavbar.vue'),
  read('../../app/layouts/admin.vue'),
  read('../../app/components/SFExtensionWidget.vue'),
  read('../../app/components/extensions/settings/SFTrustedSettingsComponent.vue'),
  Bun.file(new URL('../../i18n/locales/zh-CN.json', import.meta.url)).json(),
  Bun.file(new URL('../../i18n/locales/en-US.json', import.meta.url)).json()
])

describe('color-mode presentation surfaces', () => {
  test('public desktop and mobile utility share one explicit preference menu', () => {
    expect(navbar).toContain("from '~/composables/appearance/useColorModePreference'")
    expect(navbar).toContain('const appearanceMenuItems')
    expect(navbar).toContain("type: 'checkbox'")
    expect(navbar).toContain('checked: isCurrent')
    expect(navbar).toContain('setColorModePreference(option.value)')
    expect(navbar).toContain('description: option.descriptionKey')
    expect(navbar).toContain('checked-icon="i-lucide-check"')
    expect(navbar).toContain('class="navbar__control"')
    expect(navbar).toContain(':aria-label="colorModeTriggerLabel"')
    expect(navbar).not.toContain('toggleColorMode')
    expect(navbar).not.toContain('resolvedColorMode')
    expect(navbar).not.toContain('MutationObserver')
  })

  test('admin sidebar and user menu consume the same three-state catalog', () => {
    expect(adminLayout).toContain("from '~/composables/appearance/useColorModePreference'")
    expect(adminLayout).toContain('const appearanceMenuItems')
    expect(adminLayout).toContain("type: 'checkbox'")
    expect(adminLayout).toContain('children: appearanceMenuItems.value')
    expect(adminLayout).toContain(':items="appearanceMenuItems"')
    expect(adminLayout).toContain('checked-icon="i-lucide-check"')
    expect(adminLayout).not.toContain('toggleColorMode')
    expect(adminLayout).not.toContain('resolvedColorMode')
    expect(adminLayout).not.toContain('MutationObserver')
  })

  test('extension bridges receive resolved appearance without preference authority', () => {
    for (const bridge of [publicBridge, adminBridge]) {
      expect(bridge).toContain('const { resolvedMode } = useColorModePreference()')
      expect(bridge).toContain('colorMode: resolvedMode.value')
      expect(bridge).not.toContain('setColorModePreference')
      expect(bridge).not.toContain('colorMode.preference')
    }
  })

  test('ships the accepted Chinese and English preference copy', () => {
    expect(zhCN.appearance.colorMode).toEqual({
      system: '自动',
      systemDescription: '跟随系统（推荐）',
      light: '浅色',
      dark: '深色',
      currentPreference: '外观：{preference}'
    })
    expect(enUS.appearance.colorMode).toEqual({
      system: 'Automatic',
      systemDescription: 'Follows your system setting (recommended)',
      light: 'Light',
      dark: 'Dark',
      currentPreference: 'Appearance: {preference}'
    })
  })
})
