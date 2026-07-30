import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dir, '../..')
const read = (path: string) => readFileSync(join(root, path), 'utf8')

describe('personal appearance settings', () => {
  test('global appearance precedence includes preview, saved preference, and site default', () => {
    const app = read('app/app.vue')
    expect(app).toContain('userAppearancePreview.value || savedUserAppearance.value')
    expect(app).toContain(': resolvedAppearanceTheme.value')
    expect(app).toContain('effectiveUserAppearance.value?.lightBackground || lightBackground.value')
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
