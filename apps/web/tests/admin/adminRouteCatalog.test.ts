import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  groupRouteCatalog,
  parseRouteCatalogSnapshot,
  routeCatalogDigestShort,
  routeCatalogStats,
  useAdminRouteCatalog,
  type RouteCatalogRoute
} from '../../app/composables/admin/useAdminRouteCatalog'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

function catalogRoute(overrides: Record<string, unknown> = {}): RouteCatalogRoute {
  return {
    routeId: 'core.route.topics',
    contractVersion: 'core.topics@1',
    action: 'add',
    method: 'GET',
    path: '/api/v1/topics',
    pathSignature: '/api/v1/topics',
    priority: 0,
    providerKind: 'core',
    guard: 'core.guard.public',
    mode: 'http',
    fallback: 'closed',
    timeoutMs: 3000,
    guardRisk: 'host_owned',
    admitted: true,
    conflictKinds: [],
    ...overrides
  }
}

function pluginRoute(overrides: Record<string, unknown> = {}): RouteCatalogRoute {
  return catalogRoute({
    routeId: 'pay.settle',
    path: '/api/v1/payments/:id/settle',
    providerKind: 'plugin',
    artifact: {
      extensionId: 'payment-alipay',
      extensionVersion: '2.0.1',
      packageDigest: 'a'.repeat(64),
      runtimeInstanceId: 'rt-pay-01'
    },
    ...overrides
  })
}

describe('groupRouteCatalog', () => {
  test('puts core first, sorts plugins by extension id, and counts flags', () => {
    const groups = groupRouteCatalog([
      pluginRoute({ artifact: { extensionId: 'payment-alipay', extensionVersion: '2.0.1', packageDigest: 'a'.repeat(64), runtimeInstanceId: 'rt' } }),
      catalogRoute(),
      catalogRoute({ routeId: 'core.route.session', path: '/api/v1/session' }),
      pluginRoute({ routeId: 'mail.test', path: '/api/v1/mail/test', artifact: { extensionId: 'mail-smtp', extensionVersion: '1.1.0', packageDigest: 'b'.repeat(64), runtimeInstanceId: 'rt-mail' } })
    ])
    expect(groups.map(group => group.key)).toEqual(['core', 'mail-smtp', 'payment-alipay'])
    expect(groups[0]?.kind).toBe('core')
    expect(groups[0]?.routes).toHaveLength(2)
    expect(groups[1]?.version).toBe('1.1.0')
    expect(groups[2]?.unadmittedCount).toBe(0)
    expect(groups[2]?.replaceCount).toBe(0)
  })

  test('counts unadmitted and replace routes per group', () => {
    const groups = groupRouteCatalog([
      pluginRoute({ action: 'replace', conflictKinds: ['provider_selection'] }),
      pluginRoute({ routeId: 'pay.redirect', admitted: false }),
      pluginRoute({ routeId: 'pay.webhook' })
    ])
    expect(groups).toHaveLength(1)
    const group = groups[0]
    expect(group?.unadmittedCount).toBe(1)
    expect(group?.replaceCount).toBe(1)
  })

  test('keeps an unknown extension bucket when the artifact is missing', () => {
    const groups = groupRouteCatalog([
      pluginRoute({ artifact: undefined })
    ])
    expect(groups.map(group => group.key)).toEqual(['unknown'])
  })

  test('returns no core group when the snapshot has no core routes', () => {
    const groups = groupRouteCatalog([pluginRoute()])
    expect(groups.map(group => group.kind)).toEqual(['plugin'])
  })
})

describe('routeCatalogStats', () => {
  test('aggregates providers, admission, and conflicts', () => {
    const snapshot = parseRouteCatalogSnapshot({
      revision: 9,
      safeMode: false,
      total: 4,
      routes: [
        catalogRoute(),
        pluginRoute(),
        pluginRoute({ routeId: 'pay.redirect', admitted: false, artifact: { extensionId: 'oauth-login', extensionVersion: '0.9.0', packageDigest: 'c'.repeat(64), runtimeInstanceId: '' } }),
        pluginRoute({ routeId: 'pay.settle', action: 'replace', conflictKinds: ['provider_selection'] })
      ]
    })
    expect(snapshot).toBeDefined()
    const stats = routeCatalogStats(snapshot!)
    expect(stats.total).toBe(4)
    expect(stats.pluginProviders).toBe(2)
    expect(stats.pluginRoutes).toBe(3)
    expect(stats.unadmitted).toBe(1)
    expect(stats.conflicts).toBe(1)
  })
})

describe('routeCatalogDigestShort', () => {
  test('shortens long digests and keeps short values', () => {
    expect(routeCatalogDigestShort('a'.repeat(64))).toBe('aaaaaa…aaaa')
    expect(routeCatalogDigestShort('abc')).toBe('abc')
    expect(routeCatalogDigestShort(undefined)).toBe('')
  })
})

describe('parseRouteCatalogSnapshot', () => {
  test('accepts a valid snapshot', () => {
    const snapshot = parseRouteCatalogSnapshot({
      revision: 2,
      safeMode: true,
      total: 1,
      routes: [catalogRoute()]
    })
    expect(snapshot?.revision).toBe(2)
    expect(snapshot?.safeMode).toBe(true)
    expect(snapshot?.routes[0]?.admitted).toBe(true)
  })

  test('fails closed on malformed payloads', () => {
    const base = { revision: 2, safeMode: false, total: 1, routes: [catalogRoute()] }
    expect(parseRouteCatalogSnapshot(null)).toBeUndefined()
    expect(parseRouteCatalogSnapshot({ ...base, total: 2 })).toBeUndefined()
    expect(parseRouteCatalogSnapshot({ ...base, revision: -1 })).toBeUndefined()
    expect(parseRouteCatalogSnapshot({ ...base, routes: [catalogRoute({ providerKind: 'plugin', artifact: undefined })] })).toBeUndefined()
    expect(parseRouteCatalogSnapshot({ ...base, routes: [catalogRoute({ guardRisk: 'unknown' })] })).toBeUndefined()
    expect(parseRouteCatalogSnapshot({ ...base, routes: [catalogRoute({ conflictKinds: ['nope'] })] })).toBeUndefined()
    expect(parseRouteCatalogSnapshot({ ...base, routes: [catalogRoute({ admitted: 'yes' })] })).toBeUndefined()
  })
})

describe('useAdminRouteCatalog', () => {
  test('exposes a loader bound to the read-only endpoint', () => {
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async () => null
    })
    const { load } = useAdminRouteCatalog()
    expect(typeof load).toBe('function')
  })
})

describe('route catalog page and group component contract', () => {
  const page = source('../../app/pages/admin/extensions/route-catalog.vue')
  const group = source('../../app/components/admin/SFAdminRouteCatalogGroup.vue')

  test('page stays a read-only shell with admin middleware', () => {
    expect(page).toContain("middleware: 'admin'")
    expect(page).toContain("name: 'AdminExtensionRouteCatalog'")
    expect(page).toContain('SFAdminRouteCatalogGroup')
    expect(page).toContain('data-testid="admin-route-catalog"')
    expect(page).not.toContain('api.Post')
  })

  test('group component renders admission badges and source notes', () => {
    expect(group).toContain('routeCatalog.unadmittedBadge')
    expect(group).toContain('routeCatalog.replaceBadge')
    expect(group).toContain('routeCatalog.notAdmittedHint')
    expect(group).toContain(':aria-expanded="!collapsed"')
    expect(group).toContain("group.kind === 'core'")
  })
})
