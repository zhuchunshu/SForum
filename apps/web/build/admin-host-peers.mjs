// 可信 admin 扩展的宿主 peer 解析：
// - 作者包 / extensions 源码树不得出现 node_modules
// - 开发态由 Nuxt/Vite alias 指向 apps/web 依赖
// - 生产 Web Release 仍在隔离 workspace 内 link peers（见 Go linkPluginHostPeers）

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

/** 与 apps/api WebReleaseRuntime.HostPeers 对齐的宿主 peer 包名。 */
export const ADMIN_HOST_PEER_NAMES = [
  '@nuxt/ui',
  '@sforum/admin-sdk',
  'nuxt',
  'vue',
  'vue-router',
]

/** 兼容旧 compose 导出名。 */
export const DEV_HOST_PEERS = ADMIN_HOST_PEER_NAMES

/**
 * 默认 apps/web 根（本文件位于 apps/web/build/）。
 * @param {string} [webRoot]
 */
export function defaultWebRoot(webRoot) {
  if (webRoot) return path.resolve(webRoot)
  return path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
}

/**
 * 解析宿主上 peer 包的目录（兼容 bun 的 node_modules/.bun 布局）。
 * @returns {string | null}
 */
export function resolveHostPeerDirectory(webRoot, packageName) {
  const absoluteWeb = path.resolve(webRoot)
  if (packageName === '@sforum/admin-sdk') {
    const sdk = path.join(absoluteWeb, 'packages/admin-sdk')
    return fs.existsSync(sdk) ? sdk : null
  }

  const direct = path.join(absoluteWeb, 'node_modules', ...packageName.split('/'))
  if (isPackageDirectory(direct, packageName)) {
    return direct
  }

  try {
    if (typeof Bun !== 'undefined' && typeof Bun.resolveSync === 'function') {
      const resolvedFile = Bun.resolveSync(packageName, absoluteWeb)
      const fromFile = packageRootFromResolvedFile(resolvedFile, packageName)
      if (fromFile) return fromFile
    }
  } catch {
    // continue
  }

  const bunStore = path.join(absoluteWeb, 'node_modules/.bun')
  if (fs.existsSync(bunStore)) {
    const encoded = packageName.startsWith('@')
      ? packageName.replace('/', '+')
      : packageName
    let best = ''
    for (const entry of fs.readdirSync(bunStore)) {
      if (!entry.startsWith(`${encoded}@`)) continue
      const candidate = path.join(bunStore, entry, 'node_modules', ...packageName.split('/'))
      if (isPackageDirectory(candidate, packageName) && candidate.length > best.length) {
        best = candidate
      }
    }
    if (best) return best
  }

  return null
}

/**
 * 供 Nuxt/Vite resolve.alias 使用的绝对路径映射。
 * admin-sdk 指向源码入口，与 nuxt.config 历史 alias 一致。
 * @param {string} [webRoot]
 * @returns {Record<string, string>}
 */
export function resolveAdminHostPeerAliases(webRoot) {
  const absoluteWeb = defaultWebRoot(webRoot)
  /** @type {Record<string, string>} */
  const aliases = {}

  for (const name of ADMIN_HOST_PEER_NAMES) {
    if (name === '@sforum/admin-sdk') {
      const index = path.join(absoluteWeb, 'packages/admin-sdk/src/index.ts')
      const internal = path.join(absoluteWeb, 'packages/admin-sdk/src/internal.ts')
      if (!fs.existsSync(index)) {
        throw new Error(`host peer unavailable: @sforum/admin-sdk (${index})`)
      }
      aliases['@sforum/admin-sdk'] = index
      aliases['@sforum/admin-sdk/internal'] = internal
      continue
    }
    const dir = resolveHostPeerDirectory(absoluteWeb, name)
    if (!dir) {
      throw new Error(`host peer unavailable for ${name} under ${absoluteWeb}`)
    }
    aliases[name] = dir
  }

  return aliases
}

/**
 * 清理扩展 admin 源码根下由旧 dev-compose 写入的 host peer node_modules。
 * 仅删除「只含已知 host peers」的树；若出现未知包则拒绝，避免误删真实依赖。
 * @param {string} adminRoot
 * @returns {{ pruned: boolean, path: string }}
 */
export function pruneHostPeerNodeModules(adminRoot) {
  const absoluteAdmin = path.resolve(adminRoot)
  const nodeModules = path.join(absoluteAdmin, 'node_modules')
  if (!fs.existsSync(nodeModules)) {
    return { pruned: false, path: nodeModules }
  }

  const found = listInstalledPackageNames(nodeModules)
  const allowed = new Set(ADMIN_HOST_PEER_NAMES)
  for (const name of found) {
    if (!allowed.has(name)) {
      throw new Error(
        `refusing to prune ${nodeModules}: unexpected package ${name} `
        + `(only host peers ${ADMIN_HOST_PEER_NAMES.join(', ')} may be removed)`,
      )
    }
  }

  fs.rmSync(nodeModules, { recursive: true, force: true })
  return { pruned: true, path: nodeModules }
}

/**
 * 列出 node_modules 顶层（含 @scope/name）包名。
 * @param {string} nodeModulesDir
 * @returns {string[]}
 */
export function listInstalledPackageNames(nodeModulesDir) {
  if (!fs.existsSync(nodeModulesDir)) return []
  const names = []
  for (const entry of fs.readdirSync(nodeModulesDir, { withFileTypes: true })) {
    if (entry.name.startsWith('.')) continue
    if (entry.name.startsWith('@')) {
      const scopeDir = path.join(nodeModulesDir, entry.name)
      if (!entry.isDirectory() && !entry.isSymbolicLink()) continue
      // 跟随软链后的目录
      let scopedEntries = []
      try {
        scopedEntries = fs.readdirSync(scopeDir, { withFileTypes: true })
      } catch {
        continue
      }
      for (const child of scopedEntries) {
        if (child.name.startsWith('.')) continue
        names.push(`${entry.name}/${child.name}`)
      }
      continue
    }
    names.push(entry.name)
  }
  return names.sort()
}

function isPackageDirectory(dir, packageName) {
  if (!fs.existsSync(dir)) return false
  try {
    const pkg = JSON.parse(fs.readFileSync(path.join(dir, 'package.json'), 'utf8'))
    return pkg?.name === packageName
  } catch {
    return false
  }
}

function packageRootFromResolvedFile(filePath, packageName) {
  let current = path.dirname(path.resolve(filePath))
  for (let i = 0; i < 12; i += 1) {
    if (isPackageDirectory(current, packageName)) return current
    const parent = path.dirname(current)
    if (parent === current) break
    current = parent
  }
  return null
}
