import type { SiteNavigationBackup } from '~/composables/admin/useSiteChromeApi'

export const navigationBackupMaxBytes = 512 * 1024
export const navigationBackupSchema = 'sforum.site-navigation-backup@1' as const

export type NavigationBackupParseError = 'empty' | 'oversized' | 'malformed' | 'schema' | 'shape'

export class NavigationBackupError extends Error {
  constructor(public readonly reason: NavigationBackupParseError) {
    super(reason)
  }
}

export function parseNavigationBackup(raw: string, rawBytes = new TextEncoder().encode(raw).byteLength): SiteNavigationBackup {
  if (!raw.trim()) throw new NavigationBackupError('empty')
  if (rawBytes > navigationBackupMaxBytes) throw new NavigationBackupError('oversized')

  let value: unknown
  try {
    value = JSON.parse(raw)
  } catch {
    throw new NavigationBackupError('malformed')
  }
  if (!value || typeof value !== 'object') throw new NavigationBackupError('shape')

  const backup = value as Partial<SiteNavigationBackup>
  if (backup.schema !== navigationBackupSchema) throw new NavigationBackupError('schema')
  if (!Array.isArray(backup.definitions) || !Array.isArray(backup.placements)) throw new NavigationBackupError('shape')
  return backup as SiteNavigationBackup
}

export function serializeNavigationBackup(backup: SiteNavigationBackup) {
  return `${JSON.stringify(backup, null, 2)}\n`
}

export function navigationBackupFilename(exportedAt?: string) {
  const stamp = exportedAt && !Number.isNaN(Date.parse(exportedAt))
    ? new Date(exportedAt).toISOString().slice(0, 19).replaceAll(':', '-')
    : new Date().toISOString().slice(0, 10)
  return `sforum-navigation-${stamp}.json`
}
