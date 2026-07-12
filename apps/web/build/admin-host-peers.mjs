// 可信 admin 扩展的宿主 peer 解析：
// - 作者包 / extensions 源码树不得出现 node_modules
// - 开发态由 Vite resolve 插件从 apps/web 解析 npm peers；admin-sdk 走文件 alias
// - 生产 Web Release 仍在隔离 workspace 内 link peers（见 Go linkPluginHostPeers）
//
// 不要把 vue/nuxt/@nuxt/ui 等包目录写进 Nuxt/Vite 的 string alias：
// 1) Nuxt kit loadNuxtModuleInstance 会对 modules 做 resolveAlias，目录路径会
//    ERR_UNSUPPORTED_DIR_IMPORT → “Could not load … Is it installed?”
// 2) Vite/rollup alias 会把 `nuxt/app` 拼成 `<dir>/app`，破坏 package exports

import fs from 'node:fs'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath, pathToFileURL } from 'node:url'

/** 与 apps/api WebReleaseRuntime.HostPeers 对齐的宿主 peer 包名。 */
export const ADMIN_HOST_PEER_NAMES = [
  '@nuxt/ui',
  '@sforum/admin-sdk',
  'nuxt',
  'vue',
  'vue-router',
]

/** 走 Node package exports 解析的 npm peer（不含 workspace 源码包）。 */
export const ADMIN_HOST_NPM_PEER_NAMES = ADMIN_HOST_PEER_NAMES.filter(
  (name) => name !== '@sforum/admin-sdk',
)

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
 * 从 apps/web 解析 peer 包及其 subpath（尊重 package.json exports）。
 * @param {string} webRoot
 * @param {string} id bare import，如 `nuxt/app`、`@nuxt/ui`
 * @returns {string | null} 绝对文件路径
 */
export function resolveHostPeerId(webRoot, id) {
  const absoluteWeb = defaultWebRoot(webRoot)
  const parentURL = pathToFileURL(path.join(absoluteWeb, 'package.json')).href

  try {
    return fileURLToPath(import.meta.resolve(id, parentURL))
  } catch {
    // 部分包对 CJS require 有 main、对 ESM 无；或反过来
  }

  try {
    return createRequire(path.join(absoluteWeb, 'package.json')).resolve(id)
  } catch {
    // continue
  }

  try {
    if (typeof Bun !== 'undefined' && typeof Bun.resolveSync === 'function') {
      return Bun.resolveSync(id, absoluteWeb)
    }
  } catch {
    // continue
  }

  return null
}

/**
 * Nuxt top-level / Vite 可用的**文件级** alias。
 * 仅含 @sforum/admin-sdk 源码入口；npm peers 由 Vite 插件解析，避免目录 alias。
 *
 * @param {string} [webRoot]
 * @returns {Record<string, string>}
 */
export function resolveAdminHostPeerAliases(webRoot) {
  const absoluteWeb = defaultWebRoot(webRoot)
  const index = path.join(absoluteWeb, 'packages/admin-sdk/src/index.ts')
  const internal = path.join(absoluteWeb, 'packages/admin-sdk/src/internal.ts')
  if (!fs.existsSync(index)) {
    throw new Error(`host peer unavailable: @sforum/admin-sdk (${index})`)
  }
  // 仍校验 npm peers 在宿主上可解析，尽早暴露缺依赖
  for (const name of ADMIN_HOST_NPM_PEER_NAMES) {
    if (!resolveHostPeerDirectory(absoluteWeb, name)) {
      throw new Error(`host peer unavailable for ${name} under ${absoluteWeb}`)
    }
  }
  // 更长的 subpath 必须先于包根 alias，否则 Vite/Nuxt 会把
  // `@sforum/admin-sdk/internal` 解析成 `index.ts/internal`。
  return {
    '@sforum/admin-sdk/internal': internal,
    '@sforum/admin-sdk': index,
  }
}

/**
 * Vite pre 插件：把 host npm peers（含 subpath）固定解析到 apps/web 依赖树。
 * 扩展 SFC 的 importer 可能在 extensions/**，默认 Node 解析会失败。
 *
 * @param {string} [webRoot]
 * @returns {import('vite').Plugin}
 */
export function createAdminHostPeerResolvePlugin(webRoot) {
  const absoluteWeb = defaultWebRoot(webRoot)
  const peers = [...ADMIN_HOST_NPM_PEER_NAMES].sort((a, b) => b.length - a.length)

  function matchPeerImport(source) {
    if (!source || source.startsWith('\0') || source.startsWith('.') || source.startsWith('/')) {
      return false
    }
    // Windows 绝对路径 / 虚拟模块
    if (path.isAbsolute(source) || source.includes(':')) return false
    for (const name of peers) {
      if (source === name || source.startsWith(`${name}/`)) return true
    }
    return false
  }

  return {
    name: 'sforum-admin-host-peer-resolve',
    enforce: 'pre',
    resolveId(source) {
      if (!matchPeerImport(source)) return null
      const resolved = resolveHostPeerId(absoluteWeb, source)
      return resolved || null
    },
  }
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
