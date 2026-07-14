import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import { routeProviderConflictId, routeProviderRisk, useAdminRouteProviders } from '../app/composables/useAdminRouteProviders'

describe('admin route provider selection', () => {
  test('uses every exact route key field for conflict identity', () => {
    expect(routeProviderConflictId({
      targetRouteId: 'core.route.topic.create',
      targetContractVersion: 'core.route.topic.create@1',
      method: 'POST',
      pathSignature: '/api/v1/topics'
    })).toBe('core.route.topic.create\u0000core.route.topic.create@1\u0000POST\u0000/api/v1/topics')
  })

  test('discloses raw request, custom guard, and replacement handler authority', () => {
    const base = {
      routeId: 'demo.route.replace', contractVersion: 'demo.route.replace@1', action: 'replace',
      method: 'POST', path: '/api/v1/topics', pathSignature: '/api/v1/topics', priority: 0,
      providerKind: 'plugin' as const, handler: 'route.replace'
    }
    expect(routeProviderRisk({ ...base, guard: 'core.guard.raw_request' })).toEqual({
      rawRequest: true, customGuard: false, replacementHandler: true
    })
    expect(routeProviderRisk({ ...base, guard: 'demo.guard.owner' })).toEqual({
      rawRequest: false, customGuard: true, replacementHandler: true
    })
  })

  test('keeps selection and reset revision-CAS and active super-admin gated in UI', () => {
    const page = readFileSync(new URL('../app/pages/admin/extensions/route-providers.vue', import.meta.url), 'utf8')
    expect(page).toContain("user.value?.status === 'active'")
    expect(page).toContain("user.value.roleKeys.includes('super_admin')")
    expect(page).toContain('expectedRevision: conflict.selection?.revision ?? 0')
    expect(page).toContain('expectedRevision: conflict.selection.revision')
    expect(page).toContain('...conflict.selection.key')
    expect(page).toContain("reasonCode: 'operator_reset'")
    expect(page).toContain("candidate.providerKind === 'plugin' && candidate.action === 'replace' && Boolean(candidate.artifact)")
    expect(page).toContain('!chosenCandidate(conflict)?.artifact')
    expect(page).toContain('candidate.permission')
    expect(page).toContain('candidate.timeoutMs')
    expect(page).toContain('conflict.selection.selectionAuditEventId')
    expect(page).toContain("duration: 10000")
    expect(page).toContain("color: 'error'")
  })

  test('calls the exact route provider endpoints with the CAS body unchanged', async () => {
    const calls: Array<{ path: string, options?: unknown }> = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string, options?: unknown) => {
        calls.push({ path, options })
        return path.endsWith('/conflicts') ? [] : {}
      }
    })
    const api = useAdminRouteProviders()
    const key = {
      targetRouteId: 'core.route.topic.create', targetContractVersion: 'core.route.topic.create@1',
      method: 'POST', pathSignature: '/api/v1/topics'
    }
    await api.listConflicts()
    await api.selectProvider({
      ...key, providerRouteId: 'demo.route.replace', providerContractVersion: 'demo.route.replace@1',
      providerArtifact: {
        extensionId: 'demo.plugin', extensionVersion: '1.0.0', packageDigest: 'a'.repeat(64), runtimeInstanceId: 'run-1'
      }, expectedRevision: 3
    })
    await api.resetProvider({ ...key, expectedRevision: 4, reasonCode: 'operator_reset' })
    expect(calls.map(call => call.path)).toEqual([
      '/admin/extensions/route-providers/conflicts',
      '/admin/extensions/route-providers/selection',
      '/admin/extensions/route-providers/selection/reset'
    ])
    expect(calls[1]?.options).toMatchObject({ method: 'POST', body: { ...key, expectedRevision: 3 } })
    expect(calls[2]?.options).toEqual({ method: 'POST', body: { ...key, expectedRevision: 4, reasonCode: 'operator_reset' } })
  })

  test('registers the inspector as an extension admin page', () => {
    const modules = readFileSync(new URL('../app/config/adminModules.ts', import.meta.url), 'utf8')
    expect(modules).toContain("id: '/extensions/route-providers'")
    expect(modules).toContain("requiredPermissions: ['extension.view']")
    expect(modules).toContain("pageId: '/extensions/route-providers'")
    const plugins = readFileSync(new URL('../app/pages/admin/extensions/plugins.vue', import.meta.url), 'utf8')
    expect(plugins).toContain("adminRoutes.path('/extensions/route-providers')")
  })
})
