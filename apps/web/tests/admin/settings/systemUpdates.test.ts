import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const component = readFileSync(
  new URL('../../../app/components/admin/settings/site/tabs/SFAdminSiteUpdatesTab.vue', import.meta.url),
  'utf8'
)
const composable = readFileSync(
  new URL('../../../app/composables/admin/useAdminSystemUpdates.ts', import.meta.url),
  'utf8'
)

describe('admin system update surface', () => {
  test('keeps cached status and forced checks on their permission-specific endpoints', () => {
    expect(composable).toContain("'/admin/system-updates/check'")
    expect(composable).toContain("'/admin/system-updates'")
    expect(composable).toContain("method: options.force ? 'POST' : undefined")
  })

  test('supports mirror save, official-source restore, and persistent error feedback', () => {
    expect(component).toContain("system.updates.github_mirror_url")
    expect(component).toContain("form.mirrorUrl = ''")
    expect(component).toContain('updates.refresh({ force: true })')
    expect(component).toContain("toast.add({ color: 'error'")
    expect(component).not.toContain("color: 'error', duration:")
  })
})
