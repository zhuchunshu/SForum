import { describe, expect, test } from 'bun:test'
import { adminSurfacePlacements } from '../../app/config/adminSurfaceCatalog.gen'

describe('admin surface placement catalog', () => {
  test('publishes unique exact admin page identities', () => {
    const ids = new Set<string>()
    const contracts = new Set<string>()
    const routes = new Set<string>()

    for (const placement of adminSurfacePlacements) {
      expect(placement.id).toMatch(/^core\.component\.page\.admin(?:\.|$)/)
      expect(placement.contractVersion).toMatch(/^sforum\.component\.page\.admin(?:\.|@)/)
      expect(placement.route).toStartWith('/admin')
      expect(ids.has(placement.id)).toBe(false)
      expect(contracts.has(placement.contractVersion)).toBe(false)
      expect(routes.has(placement.route)).toBe(false)
      ids.add(placement.id)
      contracts.add(placement.contractVersion)
      routes.add(placement.route)
    }

    // 与 apps/web/app/pages/admin 下页面及 v3-catalog 生成结果对齐；新增后台页必须先进入 generator。
    expect(adminSurfacePlacements.length).toBe(47)
    expect(adminSurfacePlacements).toContainEqual({
      id: 'core.component.page.admin.users',
      contractVersion: 'sforum.component.page.admin.users@1',
      route: '/admin/users'
    })
    expect(adminSurfacePlacements).toContainEqual({
      id: 'core.component.page.admin.extensions.extension.id.pages.page.path',
      contractVersion: 'sforum.component.page.admin.extensions.extension.id.pages.page.path@1',
      route: '/admin/extensions/:extensionId/pages/:pagePath*'
    })
  })
})
