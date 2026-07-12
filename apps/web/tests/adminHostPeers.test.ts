import { describe, expect, test } from 'bun:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  ADMIN_HOST_NPM_PEER_NAMES,
  ADMIN_HOST_PEER_NAMES,
  createAdminHostPeerResolvePlugin,
  listInstalledPackageNames,
  pickExportTarget,
  pruneHostPeerNodeModules,
  resolveAdminHostPeerAliases,
  resolveHostPeerDirectory,
  resolveHostPeerId,
  shouldForceHostPeerResolve,
} from '../build/admin-host-peers.mjs'

describe('admin host peers', () => {
  const webRoot = path.resolve(import.meta.dir, '..')

  test('resolves every required host peer under apps/web', () => {
    for (const name of ADMIN_HOST_PEER_NAMES) {
      const dir = resolveHostPeerDirectory(webRoot, name)
      expect(dir, name).toBeTruthy()
      expect(fs.existsSync(dir!)).toBe(true)
    }
  })

  test('builds file aliases only for admin-sdk (safe for Nuxt top-level alias)', () => {
    const aliases = resolveAdminHostPeerAliases(webRoot)
    expect(aliases['@sforum/admin-sdk']).toContain(`${path.sep}packages${path.sep}admin-sdk${path.sep}src${path.sep}index.ts`)
    expect(aliases['@sforum/admin-sdk/internal']).toContain('internal.ts')
    // 目录 alias 会破坏 @nuxt/ui 模块加载与 nuxt/app 等 exports
    expect(aliases['@nuxt/ui']).toBeUndefined()
    expect(aliases.nuxt).toBeUndefined()
    expect(aliases.vue).toBeUndefined()
    expect(aliases['vue-router']).toBeUndefined()
  })

  test('resolves npm peer ids and subpaths via package exports', () => {
    for (const name of ADMIN_HOST_NPM_PEER_NAMES) {
      const resolved = resolveHostPeerId(webRoot, name)
      // @nuxt/ui / nuxt 的 "." 在 CJS 下可能无 main，但 ESM resolve 应成功
      expect(resolved, name).toBeTruthy()
      expect(fs.existsSync(resolved!)).toBe(true)
    }
    const nuxtApp = resolveHostPeerId(webRoot, 'nuxt/app')
    expect(nuxtApp).toBeTruthy()
    expect(nuxtApp!.includes(`${path.sep}nuxt${path.sep}`)).toBe(true)
    expect(fs.existsSync(nuxtApp!)).toBe(true)
  })

  // 浏览器侧需要 vue.runtime.esm-bundler；Node 条件的 index.mjs 没有 effectScope 等具名导出
  test('resolves vue to bundler runtime, not Node index.mjs', () => {
    const vue = resolveHostPeerId(webRoot, 'vue')
    expect(vue).toBeTruthy()
    expect(vue!.replace(/\\/g, '/')).toMatch(/vue\.runtime\.esm-bundler\.js$/)
    expect(vue!.endsWith('index.mjs')).toBe(false)
  })

  test('pickExportTarget skips node condition in favor of bundler default', () => {
    const picked = pickExportTarget({
      import: {
        types: './dist/vue.d.mts',
        node: './index.mjs',
        default: './dist/vue.runtime.esm-bundler.js',
      },
      require: {
        node: './index.js',
        default: './index.js',
      },
    })
    expect(picked).toBe('./dist/vue.runtime.esm-bundler.js')
  })

  test('only forces host peer resolve for importers outside apps/web', () => {
    expect(shouldForceHostPeerResolve(webRoot, path.join(webRoot, 'app/app.vue'))).toBe(false)
    expect(shouldForceHostPeerResolve(webRoot, path.join(webRoot, 'node_modules/nuxt/dist/app/nuxt.js'))).toBe(false)
    expect(shouldForceHostPeerResolve(webRoot, undefined)).toBe(false)
    expect(
      shouldForceHostPeerResolve(
        webRoot,
        path.resolve(webRoot, '../../../extensions/dev/sample-plugin/admin/pages/x.vue'),
      ),
    ).toBe(true)
  })

  test('vite plugin rewrites host peer bare imports only for external importers', () => {
    const plugin = createAdminHostPeerResolvePlugin(webRoot)
    const resolveId = plugin.resolveId as (
      source: string,
      importer?: string,
    ) => string | null

    // 宿主内 import：不拦截，交给 Vite 默认条件解析
    expect(resolveId('vue', path.join(webRoot, 'app/app.vue'))).toBeNull()
    expect(resolveId('vue')).toBeNull()

    // 扩展树外 importer：强制到宿主 bundler 构建
    const externalImporter = path.resolve(webRoot, '../../../extensions/dev/sample-plugin/admin/x.vue')
    const vue = resolveId('vue', externalImporter)
    expect(vue).toBeTruthy()
    expect(fs.existsSync(vue!)).toBe(true)
    expect(vue!.replace(/\\/g, '/')).toMatch(/vue\.runtime\.esm-bundler\.js$/)

    const nuxtApp = resolveId('nuxt/app', externalImporter)
    expect(nuxtApp).toBeTruthy()
    expect(fs.existsSync(nuxtApp!)).toBe(true)
    expect(resolveId('./local', externalImporter)).toBeNull()
    expect(resolveId('@sforum/admin-sdk', externalImporter)).toBeNull()
  })

  test('prunes host-peer-only node_modules and refuses unknown packages', () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'sforum-peer-prune-'))
    try {
      const nm = path.join(root, 'node_modules')
      fs.mkdirSync(path.join(nm, 'vue'), { recursive: true })
      fs.writeFileSync(path.join(nm, 'vue', 'package.json'), JSON.stringify({ name: 'vue' }))
      expect(listInstalledPackageNames(nm)).toEqual(['vue'])
      expect(pruneHostPeerNodeModules(root).pruned).toBe(true)
      expect(fs.existsSync(nm)).toBe(false)

      fs.mkdirSync(path.join(nm, 'lodash'), { recursive: true })
      expect(() => pruneHostPeerNodeModules(root)).toThrow(/unexpected package lodash/)
    } finally {
      fs.rmSync(root, { recursive: true, force: true })
    }
  })
})
