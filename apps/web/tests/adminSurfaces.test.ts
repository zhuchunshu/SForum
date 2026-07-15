import { describe, expect, test } from 'bun:test'
import {
  adminSurfaceIdempotencyKey,
  adminSurfacePlacementPageId,
  normalizeAdminSurfaceOutput,
  resolveAdminSurfacePlacement
} from '../app/utils/adminSurfaces'

describe('admin surface consumers', () => {
  test('resolves static and dynamic exact placements', () => {
    expect(resolveAdminSurfacePlacement('/users')).toEqual({
      id: 'core.component.page.admin.users',
      contractVersion: 'sforum.component.page.admin.users@1',
      route: '/admin/users'
    })
    expect(resolveAdminSurfacePlacement('/extensions/demo.plugin/pages/audit/log')).toEqual({
      id: 'core.component.page.admin.extensions.extension.id.pages.page.path',
      contractVersion: 'sforum.component.page.admin.extensions.extension.id.pages.page.path@1',
      route: '/admin/extensions/:extensionId/pages/:pagePath*'
    })
    expect(resolveAdminSurfacePlacement('/missing')).toBeUndefined()
    expect(adminSurfacePlacementPageId('core.component.page.admin.users')).toBe('/users')
    expect(adminSurfacePlacementPageId('core.component.page.admin.extensions.extension.id.pages.page.path')).toBeUndefined()
  })

  test('keeps only bounded Host-rendered result descriptors', () => {
    expect(normalizeAdminSurfaceOutput({
      title: 'Risk review',
      tone: 'warning',
      icon: 'i-lucide-shield-alert',
      pageId: '/users',
      commandSurfaceId: 'fixture.admin.surface.submit',
      items: [{ label: 'Score', value: 7 }],
      cells: { '42': 'trusted', '<script>': 'blocked' },
      options: [{ label: 'All', value: 'all' }],
      fields: [{ key: 'note', label: 'Note', type: 'textarea', required: true }],
      download: { url: '/api/v1/exports/42', filename: 'users.csv' },
      html: '<img src=x onerror=alert(1)>'
    })).toEqual({
      title: 'Risk review',
      description: undefined,
      message: undefined,
      value: undefined,
      tone: 'warning',
      icon: 'i-lucide-shield-alert',
      pageId: '/users',
      commandSurfaceId: 'fixture.admin.surface.submit',
      items: [{ label: 'Score', value: 7 }],
      cells: { '42': 'trusted' },
      options: [{ label: 'All', value: 'all' }],
      visibleResourceIds: [],
      fields: [{ key: 'note', label: 'Note', type: 'textarea', required: true, placeholder: undefined, options: [] }],
      values: {},
      refresh: false,
      download: { url: '/api/v1/exports/42', filename: 'users.csv' }
    })
    expect(normalizeAdminSurfaceOutput({ html: '<script>alert(1)</script>', pageId: '//evil.test', icon: '<svg>' })).toBeNull()
  })

  test('builds bounded printable idempotency keys', () => {
    const key = adminSurfaceIdempotencyKey('fixture.admin.surface.command')
    expect(key.length).toBeGreaterThan(20)
    expect(key.length).toBeLessThanOrEqual(128)
    expect(key).toMatch(/^[\x21-\x7e]+$/)
  })
})
