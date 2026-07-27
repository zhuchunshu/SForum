import { describe, expect, it } from 'bun:test'
import {
  ACTIVE_THEME_CACHE_TTL_MS,
  ACTIVE_THEME_SKIN_STORAGE_KEY,
  canUseActiveThemeSettingsRecord,
  canUseActiveThemeSkinRecord,
  makeActiveThemeSettingsRecord,
  normalizeActiveThemeSkinPayload,
  parseStoredSkinRecord,
  readStoredSkinRecord,
  writeStoredSkinRecord,
  type ActiveThemeIdentity,
  type ActiveThemeSkinCacheRecord
} from '../../app/utils/themes/activeThemeClientCache'

class MemoryStorage {
  items = new Map<string, string>()

  getItem(key: string) {
    return this.items.get(key) ?? null
  }

  setItem(key: string, value: string) {
    this.items.set(key, value)
  }

  removeItem(key: string) {
    this.items.delete(key)
  }
}

const now = 1_000_000
const themeA: ActiveThemeIdentity = {
  extensionId: 'theme.a',
  packageDigest: 'digest-a',
  nodeRevision: 7
}
const themeB: ActiveThemeIdentity = {
  extensionId: 'theme.b',
  packageDigest: 'digest-b',
  nodeRevision: 8
}

function skinRecord(identity: ActiveThemeIdentity, href = `/_sforum/assets/themes/${identity.extensionId}/${identity.packageDigest}/assets/theme.css`): ActiveThemeSkinCacheRecord {
  return {
    schema: 'sforum.active-theme-skin@1',
    createdAt: now,
    identity,
    links: [href]
  }
}

describe('active theme client cache', () => {
  it('does not restore Theme A skin after current identity switches to Theme B', () => {
    const record = skinRecord(themeA)

    expect(canUseActiveThemeSkinRecord(record, themeB, now)).toBe(false)
    expect(parseStoredSkinRecord(JSON.stringify(record), themeB, now)).toBeNull()
  })

  it('rejects stale package digest, node revision, expired records, and arbitrary URLs', () => {
    const storage = new MemoryStorage()
    writeStoredSkinRecord(storage, skinRecord(themeA))

    expect(readStoredSkinRecord(storage, themeA, now)).not.toBeNull()
    expect(readStoredSkinRecord(storage, { ...themeA, packageDigest: 'digest-new' }, now)).toBeNull()
    expect(readStoredSkinRecord(storage, { ...themeA, nodeRevision: 9 }, now)).toBeNull()
    expect(readStoredSkinRecord(storage, themeA, now + ACTIVE_THEME_CACHE_TTL_MS + 1)).toBeNull()

    storage.setItem(ACTIVE_THEME_SKIN_STORAGE_KEY, JSON.stringify(
      skinRecord(themeA, 'https://example.com/theme.css')
    ))
    expect(readStoredSkinRecord(storage, themeA, now)).toBeNull()
  })

  it('clears old skin cache on an authoritative empty skin response', () => {
    const normalized = normalizeActiveThemeSkinPayload({
      extensionId: 'theme.empty',
      version: '1.0.0',
      packageDigest: 'digest-empty',
      nodeRevision: 10,
      css: [],
      tokens: ''
    }, now)

    expect(normalized.identity?.extensionId).toBe('theme.empty')
    expect(normalized.links).toEqual([])
    expect(normalized.record).toBeNull()
  })

  it('does not restore Theme A settings after current identity switches to Theme B', () => {
    const record = makeActiveThemeSettingsRecord({
      themeId: themeA.extensionId,
      packageDigest: themeA.packageDigest,
      nodeRevision: themeA.nodeRevision,
      settings: { 'layout.show_footer': 'false' }
    }, now)

    expect(record).not.toBeNull()
    expect(canUseActiveThemeSettingsRecord(record, themeA, now)).toBe(true)
    expect(canUseActiveThemeSettingsRecord(record, themeB, now)).toBe(false)
    expect(canUseActiveThemeSettingsRecord(record, { ...themeA, nodeRevision: 9 }, now)).toBe(false)
    expect(canUseActiveThemeSettingsRecord(record, { ...themeA, packageDigest: 'digest-new' }, now)).toBe(false)
    expect(canUseActiveThemeSettingsRecord(record, themeA, now + ACTIVE_THEME_CACHE_TTL_MS + 1)).toBe(false)
  })

  it('uses settings cache without revision only when the current identity also lacks revision', () => {
    const identityWithoutRevision = {
      extensionId: themeA.extensionId,
      packageDigest: themeA.packageDigest
    }
    const record = makeActiveThemeSettingsRecord({
      themeId: themeA.extensionId,
      packageDigest: themeA.packageDigest,
      settings: { 'home.notice.zh-CN': '短暂保留' }
    }, now)

    expect(record).not.toBeNull()
    expect(canUseActiveThemeSettingsRecord(record, identityWithoutRevision, now)).toBe(true)
    expect(canUseActiveThemeSettingsRecord(record, themeA, now)).toBe(false)
  })
})
