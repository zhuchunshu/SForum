import { describe, expect, test } from 'bun:test'
import type { SiteNavigationBackup } from '../../app/composables/admin/useSiteChromeApi'
import {
  NavigationBackupError,
  navigationBackupFilename,
  navigationBackupMaxBytes,
  parseNavigationBackup,
  serializeNavigationBackup
} from '../../app/utils/admin/navigationBackup'

const backup: SiteNavigationBackup = {
  schema: 'sforum.site-navigation-backup@1',
  exportedAt: '2026-07-28T08:09:10Z',
  definitions: [{ sourceKey: 'operator.docs', sourceKind: 'operator', linkKind: 'internalLink', labelEnUS: 'Docs', href: '/docs' }],
  placements: [{ sourceKey: 'operator.docs', location: 'public.footer.primary', order: 10, enabled: true, visibility: 'public' }]
}

describe('navigation backup helpers', () => {
  test('round trips the backend-provided portable document', () => {
    expect(parseNavigationBackup(serializeNavigationBackup(backup))).toEqual(backup)
    expect(navigationBackupFilename(backup.exportedAt)).toBe('sforum-navigation-2026-07-28T08-09-10.json')
  })

  test('rejects malformed, incompatible, and structurally invalid documents', () => {
    expectReason('{', 'malformed')
    expectReason(JSON.stringify({ ...backup, schema: 'sforum.site-navigation-backup@2' }), 'schema')
    expectReason(JSON.stringify({ schema: backup.schema, definitions: [] }), 'shape')
  })

  test('rejects raw files beyond the bounded import size before parsing', () => {
    expectReason(JSON.stringify(backup), 'oversized', navigationBackupMaxBytes + 1)
  })
})

function expectReason(raw: string, reason: NavigationBackupError['reason'], bytes?: number) {
  try {
    parseNavigationBackup(raw, bytes)
    throw new Error('backup was accepted')
  } catch (error) {
    expect(error).toBeInstanceOf(NavigationBackupError)
    expect((error as NavigationBackupError).reason).toBe(reason)
  }
}
