import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('admin list surface consumers', () => {
  test('binds exact placed query descriptors to same-extension commands', () => {
    const composable = source('../../app/composables/admin/useAdminListSurfaces.ts')
    expect(composable).toContain('surface.placementId === target.id')
    expect(composable).toContain('surface.placementContractVersion === target.contractVersion')
    expect(composable).toContain("surface.operation === 'query'")
    expect(composable).toContain("candidate.operation === 'command'")
    expect(composable).toContain('candidate.extensionId === item.surface.extensionId')
    expect(composable).toContain('candidate.placementContractVersion === item.surface.placementContractVersion')
  })

  test('fails closed for selected filters and keeps command idempotency at the client boundary', () => {
    const composable = source('../../app/composables/admin/useAdminListSurfaces.ts')
    const client = source('../../app/composables/admin/useAdminSurfaces.ts')
    expect(composable).toContain('if (!item.view.visibleResourceIdsDeclared) return false')
    expect(composable).toContain('resources.slice(0, 1000)')
    expect(composable).toContain('resourceIds,')
    expect(composable).toContain('filters: currentFilters()')
    expect(client).toContain("headers['Idempotency-Key'] = adminSurfaceIdempotencyKey(surface.id)")
  })

  test('connects columns, filters, row and bulk actions, and selected-user regions', () => {
    const page = source('../../app/pages/admin/users.vue')
    expect(page).toContain('userListSurfaces.columns.value.map')
    expect(page).toContain('userListSurfaces.filters.value')
    expect(page).toContain('userListSurfaces.bulkActions.value')
    expect(page).toContain('userListSurfaces.rowActionsFor')
    expect(page).toContain(":kinds=\"['detail_region', 'editor_panel']\"")
  })
})
