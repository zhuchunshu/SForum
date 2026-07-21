import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  parseContentCatalog,
  parseEntityCatalog,
  parseEntityImportExportDryRun,
  parseMediaCatalog,
  useAdminRegistryCatalogs
} from '../app/composables/useAdminRegistryCatalogs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

function source(relative: string) {
  return readFileSync(join(root, relative), 'utf8')
}

describe('useAdminRegistryCatalogs parsers', () => {
  test('parses content entity and media catalogs', () => {
    const content = parseContentCatalog({
      schemaVersion: 'sforum.content-catalog@1',
      revision: 2,
      digest: 'a'.repeat(64),
      content: [{ id: 'demo.block', kind: 'block', extensionId: 'demo', packageDigest: 'b'.repeat(64) }]
    })
    expect(content?.entryCount).toBe(1)
    expect(content?.content[0].id).toBe('demo.block')

    const entities = parseEntityCatalog({
      schemaVersion: 'sforum.entity-catalog@1',
      revision: 1,
      digest: 'c'.repeat(64),
      entities: [{
        id: 'demo.article',
        kind: 'entity',
        extensionId: 'demo',
        packageDigest: 'd'.repeat(64),
        importExport: { canImport: true, canExport: true, policy: 'allow' }
      }]
    })
    expect(entities?.entities[0].canExport).toBe(true)

    const media = parseMediaCatalog({
      schemaVersion: 'sforum.media-catalog@1',
      revision: 3,
      digest: 'e'.repeat(64),
      policies: [{ id: 'demo.policy', purpose: 'general', extensionId: 'demo' }],
      processors: [{ id: 'demo.scan', stage: 'scan', extensionId: 'demo' }],
      variants: []
    })
    expect(media?.entryCount).toBe(2)
  })

  test('rejects forged schema versions', () => {
    expect(parseContentCatalog({ schemaVersion: 'forged@1', content: [] })).toBeNull()
    expect(parseEntityCatalog({ schemaVersion: 'forged@1', entities: [] })).toBeNull()
    expect(parseMediaCatalog({ schemaVersion: 'forged@1', policies: [] })).toBeNull()
  })

  test('parses entity dry-run decision envelope', () => {
    const dryRun = parseEntityImportExportDryRun({
      schemaVersion: 'sforum.entity-import-export-dry-run@1',
      executes: false,
      action: 'export',
      plan: { entityId: 'demo.article', canExport: true, policy: 'allow' },
      decision: { allowed: false, permissionKey: 'demo.export', reason: 'denied' }
    })
    expect(dryRun?.executes).toBe(false)
    expect(dryRun?.allowed).toBe(false)
    expect(dryRun?.permissionKey).toBe('demo.export')
  })

  test('loads catalogs and dry-run through api client paths', async () => {
    const calls: string[] = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string) => {
        calls.push(path)
        if (path.includes('content-catalog')) {
          return { schemaVersion: 'sforum.content-catalog@1', revision: 1, digest: 'a'.repeat(64), content: [] }
        }
        if (path.includes('entity-catalog') && path.includes('import-export-dry-run')) {
          return {
            schemaVersion: 'sforum.entity-import-export-dry-run@1',
            executes: false,
            action: 'export',
            plan: { entityId: 'demo.article', canExport: true },
            decision: { allowed: true, permissionKey: 'demo.export' }
          }
        }
        if (path.includes('entity-catalog')) {
          return { schemaVersion: 'sforum.entity-catalog@1', revision: 1, digest: 'b'.repeat(64), entities: [] }
        }
        if (path.includes('media-catalog')) {
          return {
            schemaVersion: 'sforum.media-catalog@1',
            revision: 1,
            digest: 'c'.repeat(64),
            policies: [],
            processors: [],
            variants: []
          }
        }
        throw new Error(`unexpected path ${path}`)
      }
    })

    const api = useAdminRegistryCatalogs()
    expect((await api.loadContentCatalog()).schemaVersion).toBe('sforum.content-catalog@1')
    expect((await api.loadEntityCatalog()).schemaVersion).toBe('sforum.entity-catalog@1')
    expect((await api.loadMediaCatalog()).schemaVersion).toBe('sforum.media-catalog@1')
    expect((await api.dryRunEntityImportExport('demo.article', 'export')).allowed).toBe(true)
    expect(calls).toEqual([
      '/extensions/runtime/content-catalog',
      '/extensions/runtime/entity-catalog',
      '/extensions/runtime/media-catalog',
      '/admin/extensions/entity-catalog/demo.article/import-export-dry-run?action=export'
    ])
  })
})

describe('registry catalogs page contracts', () => {
  const page = source('app/pages/admin/extensions/registry-catalogs.vue')
  const modules = source('app/config/adminModules.ts')
  const plugins = source('app/pages/admin/extensions/plugins.vue')
  const en = source('i18n/locales/en-US.json')
  const zh = source('i18n/locales/zh-CN.json')

  test('registers extension.view registry catalogs page', () => {
    expect(modules).toContain("id: '/extensions/registry-catalogs'")
    expect(modules).toContain("componentName: 'AdminExtensionRegistryCatalogs'")
    expect(modules).toContain("pageId: '/extensions/registry-catalogs'")
    expect(plugins).toContain("adminRoutes.path('/extensions/registry-catalogs')")
    expect(page).toContain("name: 'AdminExtensionRegistryCatalogs'")
    expect(page).toContain("useAdminPage('/extensions/registry-catalogs')")
    expect(page).toContain('registry-catalogs-loading')
    expect(page).toContain('registry-catalogs-error')
    expect(page).toContain('registry-catalogs-dry-run-submit')
  })

  test('ships matching navigation and UI copy without emoji', () => {
    for (const locale of [en, zh]) {
      expect(locale).toContain('"extensionRegistryCatalogs"')
      expect(locale).toContain('"registryCatalogs"')
      expect(locale).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
    }
  })
})
