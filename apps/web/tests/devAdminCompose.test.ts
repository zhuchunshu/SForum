import { afterEach, describe, expect, test } from 'bun:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  composeDevAdmin,
  DEV_COMPOSE_RELEASE_ID,
  DEV_HOST_PEERS,
  discoverBuiltinAdminPackages,
  shouldIgnoreWatchPath,
} from '../scripts/dev-admin-compose.mjs'

const tempRoots: string[] = []

function tempRoot() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'sforum-dev-compose-'))
  tempRoots.push(root)
  return root
}

afterEach(() => {
  for (const root of tempRoots.splice(0)) {
    fs.rmSync(root, { recursive: true, force: true })
  }
})

describe('dev admin compose (P1)', () => {
  test('discovers builtin theme and smtp admin packages from the real tree', () => {
    const repoRoot = path.resolve(import.meta.dir, '../../..')
    const packages = discoverBuiltinAdminPackages(path.join(repoRoot, 'extensions/builtin'))
    const ids = packages.map(item => item.id).sort()
    expect(ids).toContain('sforum.default-theme')
    expect(ids).toContain('sforum.smtp')
    // content-policy 无 frontend.admin，不应进入 compose
    expect(ids).not.toContain('sforum.content-policy')
  })

  test('composes registry metadata with dotted locale keys and symlink admin roots', () => {
    const repoRoot = path.resolve(import.meta.dir, '../../..')
    const outDir = path.join(tempRoot(), 'dev-compose')
    const result = composeDevAdmin({
      repoRoot,
      outDir,
      webRoot: path.join(repoRoot, 'apps/web'),
    })

    expect(result.releaseId).toBe(DEV_COMPOSE_RELEASE_ID)
    expect(result.extensions).toContain('sforum.default-theme')
    expect(result.extensions).toContain('sforum.smtp')
    expect(fs.existsSync(result.registryRoot)).toBe(true)
    // Runtime 主题不再强制提供 Nuxt Layer；有 layer 时才软链，否则 themeLayer 为空。
    if (result.themeLayer) {
      expect(fs.existsSync(result.themeLayer)).toBe(true)
      expect(fs.lstatSync(result.themeLayer).isSymbolicLink()).toBe(true)
    }

    const metadata = fs.readFileSync(path.join(result.registryRoot, 'metadata.ts'), 'utf8')
    expect(metadata).toContain(`export const releaseId = "${DEV_COMPOSE_RELEASE_ID}"`)
    expect(metadata).toContain('theme-settings-page')
    expect(metadata).toContain('smtp-settings-page')
    // 主题 locale 字面量键必须保留点号，供 host.t / 最长前缀解析
    expect(metadata).toContain('home.notice.zh-CN')
    expect(metadata).toContain('首页提示条（中文）')

    const registry = fs.readFileSync(path.join(result.registryRoot, 'registry.client.ts'), 'utf8')
    expect(registry).toContain('sforum.default-theme:theme-settings-page')
    expect(registry).toContain('../extensions/sforum.default-theme/frontend/admin/components/ThemeSettingsPage.vue')

    const guard = JSON.parse(fs.readFileSync(path.join(outDir, 'guard-policy.json'), 'utf8'))
    expect(guard.hostPeers).toEqual([...DEV_HOST_PEERS].sort())
    expect(guard.roots.some((item: { root: string }) => item.root.includes('sforum.default-theme'))).toBe(true)

    const adminLink = path.join(outDir, 'extensions/sforum.default-theme/frontend/admin')
    expect(fs.lstatSync(adminLink).isSymbolicLink()).toBe(true)
    const realAdmin = fs.realpathSync(adminLink)
    expect(realAdmin).toContain(`${path.sep}extensions${path.sep}builtin${path.sep}themes${path.sep}sforum-default${path.sep}frontend${path.sep}admin`)

    // 宿主 peer 由 Nuxt alias 解析；源码 admin 根不得再有 node_modules。
    expect(fs.existsSync(path.join(realAdmin, 'node_modules'))).toBe(false)
    const smtpAdmin = path.join(
      repoRoot,
      'extensions/builtin/plugins/sforum-smtp/frontend/admin',
    )
    expect(fs.existsSync(path.join(smtpAdmin, 'node_modules'))).toBe(false)
  })

  test('locale edits change compositionHash; pure recompose with same inputs is stable', () => {
    const realRepo = path.resolve(import.meta.dir, '../../..')
    const fixture = tempRoot()
    const builtin = path.join(fixture, 'extensions/builtin')
    // 最小可 compose 主题夹具
    const themeRoot = path.join(builtin, 'themes/demo')
    fs.mkdirSync(path.join(themeRoot, 'frontend/admin/components'), { recursive: true })
    fs.mkdirSync(path.join(themeRoot, 'frontend/admin/locales'), { recursive: true })
    fs.mkdirSync(path.join(themeRoot, 'layer'), { recursive: true })
    fs.writeFileSync(path.join(themeRoot, 'sforum.extension.json'), JSON.stringify({
      id: 'demo.theme',
      version: '1.0.0',
      type: 'theme',
      includes: {
        frontend: 'manifest/frontend.json',
        contributions: 'manifest/contributions.json',
      },
    }))
    fs.mkdirSync(path.join(themeRoot, 'manifest'), { recursive: true })
    fs.writeFileSync(path.join(themeRoot, 'manifest/frontend.json'), JSON.stringify({
      layer: 'layer',
      admin: {
        root: 'frontend/admin',
        components: { page: 'components/Page.vue' },
        locales: { 'zh-CN': 'locales/zh-CN.json' },
      },
    }))
    fs.writeFileSync(path.join(themeRoot, 'manifest/contributions.json'), JSON.stringify([{
      point: 'admin.extension.settings.page',
      id: 'page',
      order: 1,
      payload: { component: 'page' },
    }]))
    fs.writeFileSync(path.join(themeRoot, 'frontend/admin/components/Page.vue'), '<template><div/></template>\n')
    const localePath = path.join(themeRoot, 'frontend/admin/locales/zh-CN.json')
    fs.writeFileSync(localePath, JSON.stringify({ title: '一' }))

    const outDir = path.join(fixture, 'out')
    // peer 链接必须指向真实 apps/web（夹具里没有 node_modules）
    const webRoot = path.join(realRepo, 'apps/web')
    const first = composeDevAdmin({ repoRoot: fixture, outDir, builtinRoot: builtin, webRoot })
    const second = composeDevAdmin({ repoRoot: fixture, outDir, builtinRoot: builtin, webRoot })
    expect(second.compositionHash).toBe(first.compositionHash)

    fs.writeFileSync(localePath, JSON.stringify({ title: '二' }))
    const third = composeDevAdmin({ repoRoot: fixture, outDir, builtinRoot: builtin, webRoot })
    expect(third.compositionHash).not.toBe(first.compositionHash)
  })

  test('ignores node_modules and go plugin binaries in watch paths', () => {
    expect(shouldIgnoreWatchPath('themes/x/frontend/admin/node_modules/vue/index.js')).toBe(true)
    expect(shouldIgnoreWatchPath('plugins/x/backend/plugin')).toBe(true)
    expect(shouldIgnoreWatchPath('themes/x/frontend/admin/locales/zh-CN.json')).toBe(false)
    expect(shouldIgnoreWatchPath('themes/x/frontend/admin/components/Page.vue')).toBe(false)
  })

  test('prunes legacy host-peer node_modules under source admin roots', () => {
    const realRepo = path.resolve(import.meta.dir, '../../..')
    const fixture = tempRoot()
    const builtin = path.join(fixture, 'extensions/builtin')
    const themeRoot = path.join(builtin, 'themes/demo')
    const adminRoot = path.join(themeRoot, 'frontend/admin')
    fs.mkdirSync(path.join(adminRoot, 'components'), { recursive: true })
    fs.mkdirSync(path.join(adminRoot, 'locales'), { recursive: true })
    fs.writeFileSync(path.join(themeRoot, 'sforum.extension.json'), JSON.stringify({
      id: 'demo.theme',
      version: '1.0.0',
      type: 'theme',
      includes: {
        frontend: 'manifest/frontend.json',
        contributions: 'manifest/contributions.json',
      },
    }))
    fs.mkdirSync(path.join(themeRoot, 'manifest'), { recursive: true })
    fs.writeFileSync(path.join(themeRoot, 'manifest/frontend.json'), JSON.stringify({
      admin: {
        root: 'frontend/admin',
        components: { page: 'components/Page.vue' },
        locales: { 'zh-CN': 'locales/zh-CN.json' },
      },
    }))
    fs.writeFileSync(path.join(themeRoot, 'manifest/contributions.json'), JSON.stringify([{
      point: 'admin.extension.settings.page',
      id: 'page',
      order: 1,
      payload: { component: 'page' },
    }]))
    fs.writeFileSync(path.join(adminRoot, 'components/Page.vue'), '<template><div/></template>\n')
    fs.writeFileSync(path.join(adminRoot, 'locales/zh-CN.json'), JSON.stringify({ title: '一' }))

    // 模拟旧 compose 写进源码的 peer 软链
    const peerDir = path.join(adminRoot, 'node_modules/vue')
    fs.mkdirSync(path.dirname(peerDir), { recursive: true })
    fs.symlinkSync(path.join(realRepo, 'apps/web/node_modules/vue'), peerDir)

    composeDevAdmin({
      repoRoot: fixture,
      outDir: path.join(fixture, 'out'),
      builtinRoot: builtin,
      webRoot: path.join(realRepo, 'apps/web'),
    })
    expect(fs.existsSync(path.join(adminRoot, 'node_modules'))).toBe(false)
  })
})
