import { describe, expect, test } from 'bun:test'

const read = (path: string) => Bun.file(new URL(path, import.meta.url)).text()

const lightBackgroundPresets = [
  'pure_white',
  'porcelain',
  'paper',
  'parchment',
  'mist_gray',
  'cool_frost',
  'cloud_blue',
  'mint_mist',
  'sage',
  'sakura',
  'lilac_mist',
  'morning_apricot'
]

const [navbar, adminLayout, publicBridge, adminBridge, altchaStyles, themeStyles, appRoot, errorRoot, nuxtConfig, zhCN, enUS] = await Promise.all([
  read('../../app/components/SFNavbar.vue'),
  read('../../app/layouts/admin.vue'),
  read('../../app/components/SFExtensionWidget.vue'),
  read('../../app/components/extensions/settings/SFTrustedSettingsComponent.vue'),
  read('../../app/assets/css/sforum-altcha.css'),
  read('../../app/assets/css/sforum-theme.css'),
  read('../../app/app.vue'),
  read('../../app/error.vue'),
  read('../../nuxt.config.ts'),
  Bun.file(new URL('../../i18n/locales/zh-CN.json', import.meta.url)).json(),
  Bun.file(new URL('../../i18n/locales/en-US.json', import.meta.url)).json()
])

describe('color-mode presentation surfaces', () => {
  test('public desktop and mobile utility cycle the color-mode preference', () => {
    expect(navbar).toContain("from '~/composables/appearance/useColorModePreference'")
    expect(navbar).toContain('cyclePreference: cycleColorModePreference')
    expect(navbar).toContain('@click="cycleColorModePreference"')
    expect(navbar).not.toContain('const appearanceMenuItems')
    expect(navbar).toContain('class="navbar__control"')
    expect(navbar).toContain(':aria-label="colorModeTriggerLabel"')
    expect(navbar).toContain('const colorModeTriggerIcon = computed')
    expect(navbar).toContain('i-tabler-language')
    expect(navbar).not.toContain('toggleColorMode')
    expect(navbar).not.toContain('resolvedColorMode')
    expect(navbar).not.toContain('MutationObserver')
  })

  test('admin sidebar and user menu consume the same three-state catalog', () => {
    expect(adminLayout).toContain("from '~/composables/appearance/useColorModePreference'")
    expect(adminLayout).toContain('const appearanceMenuItems')
    expect(adminLayout).toContain("type: 'checkbox'")
    expect(adminLayout).toContain('children: appearanceMenuItems.value')
    expect(adminLayout).toContain('@click="cycleColorModePreference"')
    expect(adminLayout).toContain('const colorModeTriggerIcon = computed')
    expect(adminLayout).toContain('data-ssr-fallback="admin-appearance"')
    expect(adminLayout).toMatch(/class="pointer-events-none[^\n]*"[\s\S]*?:aria-label="t\('nav\.appearance'\)"[\s\S]*?aria-hidden="true"[\s\S]*?tabindex="-1"[\s\S]*?data-ssr-fallback="admin-appearance"/)
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

  test('ALTCHA inherits the active SForum surface and color tokens', () => {
    expect(nuxtConfig).toContain("'~/assets/css/sforum-altcha.css'")
    for (const selector of ['.auth-altcha', '.sf-recovery-altcha']) {
      expect(altchaStyles).toContain(selector)
    }
    expect(altchaStyles).toContain('--altcha-color-base: var(--sf-card);')
    expect(altchaStyles).toContain('--altcha-color-base-content: var(--sf-fg);')
    expect(altchaStyles).toContain('--altcha-input-background-color: var(--sf-card);')
    expect(altchaStyles).toContain('--altcha-input-color: var(--sf-fg);')
    expect(altchaStyles).toContain('--altcha-color-primary: var(--sf-accent);')
  })

  test('daytime background presets cannot override dark-mode surface tokens', () => {
    expect(appRoot).toContain("'data-sforum-light-background': appliedLightBackground.value")
    expect(errorRoot).toContain("'data-sforum-light-background': appliedLightBackground.value")
    for (const preset of lightBackgroundPresets) {
      expect(themeStyles).toContain(`html:not(.dark)[data-sforum-light-background="${preset}"]`)
    }
    expect(themeStyles).not.toContain('.dark[data-sforum-light-background')
    expect(themeStyles).toContain('--sf-public-bg: #111413;')
    expect(themeStyles).toContain('--sf-public-surface: #1b201f;')
    expect(themeStyles).toContain('--sf-public-surface-muted: #242a29;')
  })

  test('admin light surfaces follow the selected daytime palette without a dark selector', () => {
    expect(adminLayout).toContain("class: 'sforum-admin-shell'")
    expect(adminLayout).toContain('class="sforum-admin-shell"')
    expect(themeStyles).toContain('html:not(.dark)[data-sforum-light-background] .sforum-admin-shell')
    expect(themeStyles).toContain('--bg-admin-app: var(--sf-light-background);')
    expect(themeStyles).toContain('--bg-admin-card: var(--sf-light-surface);')
    expect(themeStyles).toContain('--bg-admin-sidebar: var(--sf-light-surface);')
    expect(themeStyles).toContain('--ui-bg: var(--sf-light-surface);')
    expect(themeStyles).not.toContain('.dark .sforum-admin-shell')
    expect(themeStyles).not.toContain('.dark[data-sforum-light-background')
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
