import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dir, '../..')
const read = (path: string) => readFileSync(join(root, path), 'utf8')

describe('personal appearance settings', () => {
  test('global appearance precedence includes preview, saved preference, and site default', () => {
    const app = read('app/app.vue')
    const errorPage = read('app/error.vue')
    const appliedAppearance = read('app/composables/appearance/useAppliedAppearance.ts')

    expect(appliedAppearance).toContain('appearanceOverride.value || userAppearancePreview.value || savedUserAppearance.value')
    expect(appliedAppearance).toContain(': siteAppearanceTheme.value')
    expect(appliedAppearance).toContain('effectiveAppearance.value?.lightBackground || siteLightBackground.value')
    expect(app).toContain('useAppliedAppearance(activeAdminAppearancePreview)')
    expect(errorPage).toContain('useAppliedAppearance()')
    expect(errorPage).toContain("'data-sforum-theme': appliedAppearanceTheme.value.dataTheme")
    expect(errorPage).toContain("'data-sforum-light-background': appliedLightBackground.value")
  })

  test('settings surface previews drafts and persists only on save', () => {
    const page = read('app/components/settings/SFAppearanceSettingsPage.vue')
    expect(page).toContain('appearance.showPreview(effectiveDraft.value)')
    expect(page).toContain("appearance.save(mode.value === 'site' ? null : effectiveDraft.value)")
    expect(page).toContain('appearance.clearPreview()')
    expect(page).toContain(':disabled="!hasChanges"')
  })

  test('account navigation exposes the dedicated appearance route', () => {
    const nav = read('app/components/settings/SFSettingsAccountNav.vue')
    expect(nav).toContain("localePath('/settings/appearance')")
    expect(nav).toContain("active === 'appearance'")
  })
})
