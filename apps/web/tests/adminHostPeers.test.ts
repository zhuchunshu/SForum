import { describe, expect, test } from 'bun:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  ADMIN_HOST_PEER_NAMES,
  listInstalledPackageNames,
  pruneHostPeerNodeModules,
  resolveAdminHostPeerAliases,
  resolveHostPeerDirectory,
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

  test('builds absolute aliases for Nuxt/Vite', () => {
    const aliases = resolveAdminHostPeerAliases(webRoot)
    expect(aliases.vue).toContain(`${path.sep}vue`)
    expect(aliases['@sforum/admin-sdk']).toContain(`${path.sep}packages${path.sep}admin-sdk${path.sep}src${path.sep}index.ts`)
    expect(aliases['@sforum/admin-sdk/internal']).toContain('internal.ts')
    expect(aliases['@nuxt/ui']).toBeTruthy()
    expect(aliases.nuxt).toBeTruthy()
    expect(aliases['vue-router']).toBeTruthy()
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
