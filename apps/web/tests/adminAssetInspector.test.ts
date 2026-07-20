import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  parseAssetInspectorSnapshot,
  useAdminAssetInspector
} from '../app/composables/useAdminAssetInspector'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

function assetSnapshot(overrides: Record<string, unknown> = {}) {
  return {
    schemaVersion: 'sforum.asset-inspector@1',
    revision: 2,
    digest: 'a'.repeat(64),
    publicationCount: 1,
    assetCount: 2,
    publications: [
      {
        extensionId: 'demo.assets',
        extensionVersion: '1.0.0',
        packageDigest: 'b'.repeat(64),
        ownerKind: 'plugin',
        assets: [
          {
            handle: 'demo.assets.entry',
            type: 'script',
            path: 'public/entry.mjs',
            module: true,
            loading: 'lazy',
            integrity: 'sha256-demo',
            csp: ["script-src 'self'"],
            scope: ['forum.component.topic']
          },
          {
            handle: 'demo.assets.style',
            type: 'style',
            path: 'public/style.css',
            module: false,
            loading: '',
            integrity: 'sha256-style',
            csp: [],
            scope: []
          }
        ]
      }
    ],
    ...overrides
  }
}

describe('asset inspector snapshot parsing', () => {
  test('accepts valid redacted snapshots', () => {
    const snapshot = parseAssetInspectorSnapshot(assetSnapshot())
    expect(snapshot?.revision).toBe(2)
    expect(snapshot?.publicationCount).toBe(1)
    expect(snapshot?.assetCount).toBe(2)
    expect(snapshot?.publications).toHaveLength(1)
    expect(snapshot?.publications[0]?.assets[0]?.handle).toBe('demo.assets.entry')
    expect(snapshot?.publications[0]?.assets[0]?.module).toBe(true)
  })

  test('rejects forged or incomplete shapes', () => {
    expect(parseAssetInspectorSnapshot(null)).toBeNull()
    expect(parseAssetInspectorSnapshot(assetSnapshot({ schemaVersion: 'sforum.asset-inspector@2' }))).toBeNull()
    expect(parseAssetInspectorSnapshot(assetSnapshot({ revision: '2' }))).toBeNull()
    expect(parseAssetInspectorSnapshot(assetSnapshot({ digest: 1 }))).toBeNull()
    expect(parseAssetInspectorSnapshot(assetSnapshot({ publications: null }))).toBeNull()
    expect(parseAssetInspectorSnapshot(assetSnapshot({
      publications: [{ extensionId: 'x' }]
    }))).toBeNull()
  })
})

describe('asset inspector API boundary', () => {
  test('calls exact read-only endpoint and rejects invalid limits', async () => {
    const calls: string[] = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string) => {
        calls.push(path)
        return assetSnapshot()
      }
    })

    const api = useAdminAssetInspector()
    expect((await api.inspect(50)).revision).toBe(2)
    expect(calls).toEqual(['/admin/extensions/asset-inspector?limit=50'])

    await expect(api.inspect(0)).rejects.toThrow(/limit/)
    await expect(api.inspect(201)).rejects.toThrow(/limit/)
    expect(calls).toHaveLength(1)

    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async () => ({ schemaVersion: 'forged' })
    })
    await expect(useAdminAssetInspector().inspect()).rejects.toThrow(/invalid|failed/i)
  })
})

describe('asset inspector page and navigation contracts', () => {
  const page = source('../app/pages/admin/extensions/asset-inspector.vue')
  const modules = source('../app/config/adminModules.ts')
  const plugins = source('../app/pages/admin/extensions/plugins.vue')
  const en = source('../i18n/locales/en-US.json')
  const zh = source('../i18n/locales/zh-CN.json')

  test('registers extension.view read-only inspector page', () => {
    expect(modules).toContain("id: '/extensions/asset-inspector'")
    expect(modules).toContain("componentName: 'AdminExtensionAssetInspector'")
    expect(modules).toContain("pageId: '/extensions/asset-inspector'")
    expect(plugins).toContain("adminRoutes.path('/extensions/asset-inspector')")

    expect(page).toContain("name: 'AdminExtensionAssetInspector'")
    expect(page).toContain("useAdminPage('/extensions/asset-inspector')")
    expect(page).toContain('asset-inspector-loading')
    expect(page).toContain('asset-inspector-error')
    expect(page).toContain('asset-inspector-empty-publications')
  })

  test('ships matching navigation and UI copy without emoji', () => {
    for (const locale of [en, zh]) {
      expect(locale).toContain('"extensionAssetInspector"')
      expect(locale).toContain('"assetInspector"')
      expect(locale).toContain('"redactionHint"')
    }
    expect(page).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
    expect(page).toContain('break-all')
  })
})
