import { describe, expect, test } from 'bun:test'

const read = (path: string) => Bun.file(new URL(path, import.meta.url)).text()

const [appearanceTab, appRoot, previewComposable, zhCN, enUS] = await Promise.all([
  read('../../../app/components/admin/settings/personalization/tabs/SFAdminPersonalizationAppearanceTab.vue'),
  read('../../../app/app.vue'),
  read('../../../app/composables/admin/settings/useAdminAppearancePreview.ts'),
  Bun.file(new URL('../../../i18n/locales/zh-CN.json', import.meta.url)).json(),
  Bun.file(new URL('../../../i18n/locales/en-US.json', import.meta.url)).json()
])

describe('personalization appearance settings', () => {
  test('persists a separate light-only background preset with recommended reset', () => {
    expect(appearanceTab).toContain("map['appearance.light_background']?.value")
    expect(appearanceTab).toContain("{ name: 'appearance.light_background', value: value.lightBackground }")
    expect(appearanceTab).toContain('lightBackground: recommendedLightBackground')
    expect(appearanceTab).toContain('@click="selectLightBackground(choice.value)"')
  })

  test('uses focus-stable radio buttons instead of hidden native inputs', () => {
    expect(appearanceTab.match(/role="radiogroup"/g)).toHaveLength(2)
    expect(appearanceTab).toContain('role="radio"')
    expect(appearanceTab).toContain(':aria-checked="state.form.lightBackground === choice.value"')
    expect(appearanceTab).not.toContain('type="radio"')
    expect(appearanceTab).not.toContain('class="sr-only"')
  })

  test('previews unsaved appearance choices only on admin routes', () => {
    expect(appearanceTab).toContain('useAdminAppearancePreview()')
    expect(appearanceTab).toContain('watch([previewTheme, () => state.form.lightBackground], syncAppearancePreview')
    expect(appearanceTab).toContain('onDeactivated(() =>')
    expect(appearanceTab).toContain('appearancePreview.clear()')
    expect(previewComposable).toContain("useState<AdminAppearancePreview | null>('admin-appearance-preview'")
    expect(appRoot).toContain('adminAppearancePreview.value && isAdminRoute.value')
    expect(appRoot).toContain("'data-sforum-admin-appearance-preview':")
  })

  test('offers the same twelve localized presets and explains the dark-mode boundary', () => {
    const presets = [
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
    expect(Object.keys(zhCN.admin.personalization.lightBackground.presets)).toEqual(presets)
    expect(Object.keys(enUS.admin.personalization.lightBackground.presets)).toEqual(presets)
    expect(zhCN.admin.personalization.lightBackground.description).toContain('前台与后台')
    expect(enUS.admin.personalization.lightBackground.description).toContain('public site and admin')
    expect(zhCN.admin.personalization.lightBackground.description).toContain('夜间模式始终保持')
    expect(enUS.admin.personalization.lightBackground.description).toContain('Dark mode always keeps')
  })
})
