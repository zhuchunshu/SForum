import { describe, expect, test } from 'bun:test'

import {
  ASSET_RELOAD_COOLDOWN_MS,
  extractErrorChainMessage,
  isDynamicImportFailureMessage,
  isNuxtAssetUrl,
  isWithinAssetReloadCooldown,
  readAssetReloadTimestamp,
  writeAssetReloadTimestamp
} from '../../app/utils/errors/assetReload'

describe('asset reload failure classification', () => {
  test('recognizes nuxt chunk urls only', () => {
    expect(isNuxtAssetUrl('/_nuxt/entry.Ck3m2a.js')).toBe(true)
    expect(isNuxtAssetUrl('https://cdn.example.com/_nuxt/vendor.Bq9x1b.css')).toBe(true)
    expect(isNuxtAssetUrl('/api/v1/health')).toBe(false)
    expect(isNuxtAssetUrl('/_nuxtent/lookalike.js')).toBe(false)
    expect(isNuxtAssetUrl('')).toBe(false)
    expect(isNuxtAssetUrl(null)).toBe(false)
  })

  test('matches dynamic import failure messages across engines', () => {
    expect(isDynamicImportFailureMessage(
      'Failed to fetch dynamically imported module: https://site.test/_nuxt/pages/index.Ck3m2a.js'
    )).toBe(true)
    expect(isDynamicImportFailureMessage('Importing a module script failed.')).toBe(true)
    expect(isDynamicImportFailureMessage('error loading dynamically imported module: /_nuxt/x.js')).toBe(true)
    expect(isDynamicImportFailureMessage('TypeError: network error (ERR_NETWORK_CHANGED)')).toBe(true)
    expect(isDynamicImportFailureMessage('账号或密码不正确。')).toBe(false)
    expect(isDynamicImportFailureMessage(undefined)).toBe(false)
  })

  test('collects messages along the cause chain', () => {
    const wrapped = new Error('router navigation failed', {
      cause: new Error('Failed to fetch dynamically imported module: /_nuxt/x.js')
    })

    expect(extractErrorChainMessage(wrapped)).toBe(
      'router navigation failed Failed to fetch dynamically imported module: /_nuxt/x.js'
    )
  })
})

describe('asset reload cooldown bookkeeping', () => {
  const createStorageStub = () => {
    const entries = new Map<string, string>()
    return {
      getItem: (key: string) => entries.get(key) ?? null,
      setItem: (key: string, value: string) => {
        entries.set(key, value)
      }
    }
  }

  test('round-trips the last reload timestamp', () => {
    const storage = createStorageStub()
    expect(readAssetReloadTimestamp(storage)).toBeNull()

    writeAssetReloadTimestamp(storage, 1729600000000)
    expect(readAssetReloadTimestamp(storage)).toBe(1729600000000)
  })

  test('treats reloads inside the cooldown window as blocked and expired ones as free', () => {
    const lastAt = 1000000000000
    expect(isWithinAssetReloadCooldown(lastAt, lastAt + ASSET_RELOAD_COOLDOWN_MS - 1)).toBe(true)
    expect(isWithinAssetReloadCooldown(lastAt, lastAt + ASSET_RELOAD_COOLDOWN_MS)).toBe(false)
    expect(isWithinAssetReloadCooldown(null, lastAt)).toBe(false)
  })
})
