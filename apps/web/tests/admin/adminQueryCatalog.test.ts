import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  isQueryCatalogHostInternal,
  parseQueryCatalogSnapshot,
  queryCatalogDigestShort,
  type QueryCatalogEntry
} from '../../app/composables/admin/useAdminQueryCatalog'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

function catalogEntry(overrides: Record<string, unknown> = {}): QueryCatalogEntry {
  return {
    queryId: 'core.query.site_stats.current',
    hostQueryId: 'sforum.core.site_stats.current',
    hostPlanVersion: '1',
    entity: 'core.query.site_stats',
    fields: ['topic_count', 'comment_count', 'user_count', 'public_attachment_count'],
    filters: [],
    sort: [],
    pagination: 'none',
    permissionPolicy: 'public',
    resultSchema: 'sforum.core.site_stats@1',
    resultSchemaDigest: 'b'.repeat(64),
    ...overrides
  }
}

describe('admin query catalog parsing', () => {
  test('parses a sealed catalog snapshot and keeps totals honest', () => {
    const snapshot = parseQueryCatalogSnapshot({
      extensionId: 'core.query',
      extensionVersion: '1.0.0',
      artifactDigest: 'a'.repeat(64),
      costMaximum: 500,
      total: 2,
      queries: [
        catalogEntry(),
        catalogEntry({
          queryId: 'core.query.shortcode.public_users.batch',
          hostQueryId: 'sforum.core.shortcode.public_users.batch',
          pagination: 'offset',
          filters: ['ids'],
          sort: ['id']
        })
      ]
    })
    expect(snapshot).toBeDefined()
    expect(snapshot?.total).toBe(2)
    expect(snapshot?.costMaximum).toBe(500)
    expect(isQueryCatalogHostInternal(snapshot!.queries[0])).toBe(false)
    expect(isQueryCatalogHostInternal(snapshot!.queries[1])).toBe(true)
  })

  test('rejects incomplete or lying payloads', () => {
    expect(parseQueryCatalogSnapshot(null)).toBeUndefined()
    expect(parseQueryCatalogSnapshot({})).toBeUndefined()
    const entry = catalogEntry()
    expect(
      parseQueryCatalogSnapshot({
        extensionId: 'core.query', extensionVersion: '1.0.0', artifactDigest: 'a'.repeat(64),
        costMaximum: 500, total: 2, queries: [entry]
      })
    ).toBeUndefined()
    expect(
      parseQueryCatalogSnapshot({
        extensionId: 'core.query', extensionVersion: '1.0.0', artifactDigest: 'a'.repeat(64),
        costMaximum: 0, total: 1, queries: [entry]
      })
    ).toBeUndefined()
    expect(
      parseQueryCatalogSnapshot({
        extensionId: 'core.query', extensionVersion: '1.0.0', artifactDigest: 'a'.repeat(64),
        costMaximum: 500, total: 1,
        queries: [{ ...entry, fields: 'not-a-list' }]
      })
    ).toBeUndefined()
  })

  test('digest shortening stays bounded', () => {
    expect(queryCatalogDigestShort('b'.repeat(64))).toHaveLength(8)
    expect(queryCatalogDigestShort(undefined)).toBe('')
  })

  test('page shell keeps the route-catalog geometry contract', () => {
    const page = source('../../app/pages/admin/extensions/query-catalog.vue')
    expect(page).toContain('data-testid="admin-query-catalog"')
    expect(page).toContain("useAdminPage('/extensions/query-catalog')")
    expect(page).toContain('UDashboardToolbar')
    expect(page).toContain('queryCatalog.metaTitle')
    const modules = source('../../app/config/adminModules.ts')
    expect(modules).toContain("id: '/extensions/query-catalog'")
    expect(modules).toContain('AdminExtensionQueryCatalog')
    expect(modules).toContain("{ type: 'page', pageId: '/extensions/query-catalog' }")
  })
})
