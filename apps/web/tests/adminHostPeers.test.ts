import { describe, expect, test } from 'bun:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  ADMIN_HOST_NPM_PEER_NAMES,
  ADMIN_HOST_PEER_NAMES,
  createAdminHostPeerResolvePlugin,
  listInstalledPackageNames,
  pruneHostPeerNodeModules,
  resolveAdminHostPeerAliases,
  resolveHostPeerDirectory,
  resolveHostPeerId,
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

  test('vite plugin rewrites host peer bare imports', () => {
    const plugin = createAdminHostPeerResolvePlugin(webRoot)
    const resolveId = plugin.resolveId as (source: string) => string | null
    const vue = resolveId('vue')
    expect(vue).toBeTruthy()
    expect(fs.existsSync(vue!)).toBe(true)
    const nuxtApp = resolveId('nuxt/app')
    expect(nuxtApp).toBeTruthy()
    expect(fs.existsSync(nuxtApp!)).toBe(true)
    expect(resolveId('./local')).toBeNull()
    expect(resolveId('@sforum/admin-sdk')).toBeNull()
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
