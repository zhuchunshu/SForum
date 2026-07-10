import { describe, expect, test } from 'bun:test'

import { validateAdminExtensionImport } from '../build/admin-extension-guard'

const root = '/release/extensions/demo.plugin/frontend/admin'
const policy = {
  roots: [{ root, dependencies: ['date-fns'] }],
  hostPeers: ['vue', 'nuxt', '@nuxt/ui', 'vue-router', '@sforum/admin-sdk']
}

describe('admin extension build guard', () => {
  test('allows local modules, declared dependencies, and host peers', () => {
    expect(validateAdminExtensionImport('./shared/value', `${root}/components/Cell.vue`, policy)).toBeUndefined()
    expect(validateAdminExtensionImport('date-fns/format', `${root}/components/Cell.vue`, policy)).toBeUndefined()
    expect(validateAdminExtensionImport('vue', `${root}/components/Cell.vue`, policy)).toBeUndefined()
  })

  test('rejects host aliases, private output, undeclared packages, and local escapes', () => {
    expect(() => validateAdminExtensionImport('~/composables/useAuthSession', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('#build/private', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('@sforum/admin-sdk/internal', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('left-pad', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('../../../../core.ts', `${root}/components/Cell.vue`, policy)).toThrow()
  })

  test('ignores imports whose owner is outside a trusted extension root', () => {
    expect(validateAdminExtensionImport('anything', '/release/apps/web/app.vue', policy)).toBeUndefined()
  })
})
