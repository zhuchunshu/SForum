import { describe, expect, test } from 'bun:test'

import { packageNameFromNodeModulesPath, validateAdminExtensionImport } from '../build/admin-extension-guard'

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

  // Vite/Vue SFC 会把同一文件以绝对路径 + query 作为 resolveId source。
  test('allows absolute paths inside the frontend root including vue query suffixes', () => {
    const importer = `${root}/components/SmtpSettingsPage.vue`
    const vueScript = `${root}/components/SmtpSettingsPage.vue?vue&type=script&setup=true&lang.ts`
    expect(validateAdminExtensionImport(vueScript, importer, policy)).toBeUndefined()
    expect(validateAdminExtensionImport(`${root}/components/shared.ts`, `${importer}?vue&type=script&setup=true&lang.ts`, policy)).toBeUndefined()
  })

  // 插件模板里的 UButton 会解析到宿主 node_modules/@nuxt/ui 的绝对路径。
  test('allows absolute host peer and dependency package paths outside the extension root', () => {
    const importer = `${root}/components/SmtpSettingsPage.vue`
    expect(validateAdminExtensionImport(
      '/Users/inkedus/Code/SForum/apps/web/node_modules/@nuxt/ui/dist/runtime/components/Button.vue',
      importer,
      policy
    )).toBeUndefined()
    expect(validateAdminExtensionImport(
      '/release/workspace/node_modules/date-fns/format.js',
      importer,
      policy
    )).toBeUndefined()
    expect(validateAdminExtensionImport(
      '/release/workspace/packages/admin-sdk/src/index.ts',
      importer,
      policy
    )).toBeUndefined()
  })

  test('rejects host aliases, private output, undeclared packages, and local escapes', () => {
    expect(() => validateAdminExtensionImport('~/composables/useAuthSession', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('#build/private', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('@sforum/admin-sdk/internal', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('left-pad', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('../../../../core.ts', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport('/tmp/outside.ts', `${root}/components/Cell.vue`, policy)).toThrow()
    expect(() => validateAdminExtensionImport(
      '/Users/inkedus/Code/SForum/apps/web/node_modules/left-pad/index.js',
      `${root}/components/Cell.vue`,
      policy
    )).toThrow()
  })

  test('ignores imports whose owner is outside a trusted extension root', () => {
    expect(validateAdminExtensionImport('anything', '/release/apps/web/app.vue', policy)).toBeUndefined()
  })

  test('packageNameFromNodeModulesPath reads the owning package after the last node_modules', () => {
    expect(packageNameFromNodeModulesPath('/a/node_modules/@nuxt/ui/dist/Button.vue')).toBe('@nuxt/ui')
    expect(packageNameFromNodeModulesPath('/a/node_modules/vue/dist/vue.mjs')).toBe('vue')
    expect(packageNameFromNodeModulesPath('/a/node_modules/@nuxt/ui/node_modules/foo/index.js')).toBe('foo')
  })
})
