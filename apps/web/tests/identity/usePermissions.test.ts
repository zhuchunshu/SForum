import { describe, expect, test } from 'bun:test'
import type { CurrentUser } from '../../app/composables/identity/useAuthSession'
import { hasPermission } from '../../app/composables/identity/usePermissions'

function user(overrides: Partial<CurrentUser> = {}): CurrentUser {
  return {
    id: 1,
    username: 'admin',
    displayName: 'Admin',
    locale: 'zh-CN',
    status: 'active',
    isInitialSuperAdmin: false,
    avatar: { kind: 'initials', url: '', alt: 'Admin' },
    roleKeys: [],
    permissions: [],
    ...overrides
  }
}

describe('forum permission helpers', () => {
  test('treats super_admin as full permission authority', () => {
    expect(hasPermission(user({ roleKeys: ['super_admin'] }), 'post.create')).toBe(true)
  })

  test('still checks explicit permissions for ordinary users', () => {
    expect(hasPermission(user({ permissions: ['post.create'] }), 'post.create')).toBe(true)
    expect(hasPermission(user(), 'post.create')).toBe(false)
    expect(hasPermission(null, 'post.create')).toBe(false)
  })
})
