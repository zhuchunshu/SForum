import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const component = readFileSync(
  new URL('../../../app/components/admin/settings/site/tabs/SFAdminSiteClientTab.vue', import.meta.url),
  'utf8'
)
const shell = readFileSync(
  new URL('../../../app/pages/admin/settings/index.vue', import.meta.url),
  'utf8'
)
const zh = JSON.parse(readFileSync(new URL('../../../i18n/locales/zh-CN.json', import.meta.url), 'utf8'))
const en = JSON.parse(readFileSync(new URL('../../../i18n/locales/en-US.json', import.meta.url), 'utf8'))

describe('admin client version policy surface', () => {
  test('writes the public client.* options through the shared option tab composable', () => {
    expect(component).toContain("'client.minimum_version'")
    expect(component).toContain("'client.recommended_version'")
    expect(component).toContain("'client.update_notice'")
    expect(component).toContain('useAdminOptionTab')
    expect(component).toContain('saveOptions([')
  })

  test('offers one-click restore to the unrestricted recommended defaults', () => {
    expect(component).toContain('function restoreRecommended()')
    expect(component).toContain("form.minimumVersion = ''")
    expect(component).toContain("form.recommendedVersion = ''")
    expect(component).toContain("form.updateNotice = ''")
    expect(component).toContain("t('admin.settings.client.restoredRecommended')")
  })

  test('keeps unsaved changes visible and non-error toasts auto-dismissible', () => {
    expect(component).toContain(':show-unsaved-alert="hasChanges"')
    expect(component).toContain("color: 'neutral'")
    expect(component).toContain('duration: 10000')
  })

  test('registers the client tab inside the fixed site settings shell', () => {
    expect(shell).toContain("import SFAdminSiteClientTab from '~/components/admin/settings/site/tabs/SFAdminSiteClientTab.vue'")
    expect(shell).toContain("'client'")
    expect(shell).toContain("{ id: 'client', label: t('admin.settings.tabs.client'), icon: 'i-lucide-smartphone' }")
    expect(shell).toContain('client: SFAdminSiteClientTab')
  })

  test('ships the tab label and field copy in both locales', () => {
    for (const locale of [zh, en]) {
      expect(locale.admin.settings.tabs.client).toBeTruthy()
      expect(locale.admin.settings.client.title).toBeTruthy()
      for (const key of [
        'minimumVersion',
        'minimumVersionHint',
        'recommendedVersion',
        'recommendedVersionHint',
        'updateNotice',
        'updateNoticeHint',
        'formatHint',
        'restoreRecommended',
        'restoredRecommended',
        'resetChanges'
      ]) {
        expect(locale.admin.settings.client[key]).toBeTruthy()
      }
    }
  })
})
