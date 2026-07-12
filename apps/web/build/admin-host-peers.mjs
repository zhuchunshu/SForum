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
 *
 * 重要：必须使用 **bundler/browser** 条件，不能用 Node 的 `import.meta.resolve` /
 * `createRequire` 默认结果。后者对 `vue` 会解析到 `exports.import.node` →
 * `index.mjs`，浏览器侧没有 `effectScope` 等具名导出，导致：
 *   The requested module '.../vue/index.mjs' does not provide an export named 'effectScope'
 *
 * @param {string} webRoot
 * @param {string} id bare import，如 `nuxt/app`、`@nuxt/ui`、`vue`
 * @returns {string | null} 绝对文件路径
 */
export function resolveHostPeerId(webRoot, id) {
  const absoluteWeb = defaultWebRoot(webRoot)
  const parsed = splitBarePackageId(id)
  if (!parsed) return null

  const pkgDir = resolveHostPeerDirectory(absoluteWeb, parsed.name)
  if (!pkgDir) {
    // 回退：仅作诊断路径；bundler 条件失败时再试 Node（可能仍是 node 条件）
    return resolveHostPeerIdNodeFallback(absoluteWeb, id)
  }

  const fromExports = resolvePackageExportFile(pkgDir, parsed.subpath)
  if (fromExports) return fromExports

  return resolveHostPeerIdNodeFallback(absoluteWeb, id)
}

/**
 * Vite pre 插件用：仅当 importer 在宿主 apps/web 树之外（扩展源码 / 外置 compose）
 * 时强制解析到宿主 peer。宿主自身代码交给 Vite 默认解析，避免误伤 vue 条件导出。
 *
 * @param {string} webRoot
 * @param {string | undefined} importer
 * @returns {boolean}
 */
export function shouldForceHostPeerResolve(webRoot, importer) {
  if (!importer) return false
  const absoluteWeb = path.resolve(defaultWebRoot(webRoot))
  const clean = importer.split('?')[0].split('#')[0]
  if (!clean || clean.startsWith('\0')) return false
  // virtual / dep optimized ids
  if (clean.includes('node_modules/.vite') || clean.includes('\0')) return false
  let abs
  try {
    abs = path.resolve(clean)
  } catch {
    return false
  }
  // 宿主 app 内：不拦截（含 packages/、app/、.nuxt 等）
  if (isPathInside(abs, absoluteWeb)) {
    // 例外：dev-compose / release 把扩展挂到 web 树外的情况已由 !isPathInside 覆盖
    return false
  }
  return true
}

/**
 * 按 package.json exports 解析 subpath，优先 browser/bundler 条件，显式跳过 node。
 * @param {string} packageDir
 * @param {string} subpath  '' | 'app' | 'dist/...'（不含包名）
 * @returns {string | null}
 */
export function resolvePackageExportFile(packageDir, subpath = '') {
  let pkg
  try {
    pkg = JSON.parse(fs.readFileSync(path.join(packageDir, 'package.json'), 'utf8'))
  } catch {
    return null
  }

  const relRequest = subpath ? `./${subpath.replace(/^\.\//, '')}` : '.'
  let target = null

  if (pkg.exports != null) {
    target = matchExportsField(pkg.exports, relRequest)
  }

  if (!target && (relRequest === '.' || relRequest === './')) {
    // 无 exports 时的经典字段（bundler 优先 module）
    target = pkg.module || pkg.browser || pkg.main || null
    if (target && typeof target === 'object') {
      target = pickExportTarget(target)
    }
  }

  if (typeof target !== 'string' || !target) return null

  const abs = path.resolve(packageDir, target)
  if (!fs.existsSync(abs)) return null
  return abs
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
 * Vite pre 插件：把 **扩展树外** importer 的 host npm peers 解析到 apps/web 依赖树。
 * 使用 bundler 条件导出（避免 vue → index.mjs）。宿主 app 内 import 不拦截。
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
    resolveId(source, importer) {
      if (!matchPeerImport(source)) return null
      // 宿主自身代码必须走 Vite 默认条件解析；强制 Node resolve 会弄坏 vue。
      if (!shouldForceHostPeerResolve(absoluteWeb, importer)) {
        return null
      }
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

function isPathInside(child, parent) {
  const rel = path.relative(path.resolve(parent), path.resolve(child))
  return rel === '' || (!rel.startsWith('..') && !path.isAbsolute(rel))
}

/**
 * @param {string} id
 * @returns {{ name: string, subpath: string } | null}
 */
function splitBarePackageId(id) {
  if (!id || id.startsWith('.') || id.startsWith('/') || id.includes(':')) return null
  if (id.startsWith('@')) {
    const parts = id.split('/')
    if (parts.length < 2) return null
    return {
      name: `${parts[0]}/${parts[1]}`,
      subpath: parts.slice(2).join('/'),
    }
  }
  const slash = id.indexOf('/')
  if (slash === -1) return { name: id, subpath: '' }
  return { name: id.slice(0, slash), subpath: id.slice(slash + 1) }
}

/** 运行时文件解析时跳过的 exports 条件（Node / 类型）。 */
const EXPORT_SKIP_CONDITIONS = new Set([
  'types',
  'typings',
  'node',
  'node-addons',
  'deno',
  'electron',
])

/**
 * 在 exports 条件对象中挑选 bundler/browser 目标，永不选 node/types。
 * @param {unknown} entry
 * @returns {string | null}
 */
export function pickExportTarget(entry) {
  if (typeof entry === 'string') return entry
  if (Array.isArray(entry)) {
    for (const item of entry) {
      const picked = pickExportTarget(item)
      if (picked) return picked
    }
    return null
  }
  if (!entry || typeof entry !== 'object') return null

  // 优先顺序：browser → import → module → default → require
  for (const key of ['browser', 'import', 'module', 'default', 'require']) {
    if (Object.prototype.hasOwnProperty.call(entry, key) && !EXPORT_SKIP_CONDITIONS.has(key)) {
      const picked = pickExportTarget(/** @type {any} */ (entry)[key])
      if (picked) return picked
    }
  }
  for (const [key, value] of Object.entries(entry)) {
    if (EXPORT_SKIP_CONDITIONS.has(key)) continue
    if (['browser', 'import', 'module', 'default', 'require'].includes(key)) continue
    const picked = pickExportTarget(value)
    if (picked) return picked
  }
  return null
}

/**
 * @param {unknown} exportsField
 * @param {string} relRequest '.' | './app'
 * @returns {string | null}
 */
function matchExportsField(exportsField, relRequest) {
  const request = relRequest === './' ? '.' : relRequest

  if (typeof exportsField === 'string' || Array.isArray(exportsField)) {
    return request === '.' ? pickExportTarget(exportsField) : null
  }
  if (!exportsField || typeof exportsField !== 'object') return null

  // 条件式根 exports（无 "." 键）
  const keys = Object.keys(exportsField)
  const hasSubpathKeys = keys.some((k) => k.startsWith('.') || k === '.')
  if (!hasSubpathKeys) {
    return request === '.' ? pickExportTarget(exportsField) : null
  }

  // 精确匹配
  if (Object.prototype.hasOwnProperty.call(exportsField, request)) {
    return pickExportTarget(/** @type {any} */ (exportsField)[request])
  }
  // 兼容 "./" vs "."
  if (request === '.' && Object.prototype.hasOwnProperty.call(exportsField, './')) {
    return pickExportTarget(/** @type {any} */ (exportsField)['./'])
  }

  // 简单 * 通配（如 ./dist/*）
  for (const [pattern, value] of Object.entries(exportsField)) {
    if (!pattern.includes('*')) continue
    const star = pattern.indexOf('*')
    const prefix = pattern.slice(0, star)
    const suffix = pattern.slice(star + 1)
    if (!request.startsWith(prefix) || !request.endsWith(suffix)) continue
    const mid = request.slice(prefix.length, request.length - suffix.length)
    const mapped = pickExportTarget(value)
    if (typeof mapped !== 'string') continue
    return mapped.replace('*', mid)
  }
  return null
}

function resolveHostPeerIdNodeFallback(absoluteWeb, id) {
  const parentURL = pathToFileURL(path.join(absoluteWeb, 'package.json')).href
  try {
    const resolved = fileURLToPath(import.meta.resolve(id, parentURL))
    // 若误落到 vue/index.mjs，尝试纠正为 bundler 构建
    const corrected = correctKnownNodeOnlyEntries(resolved)
    return corrected || resolved
  } catch {
    // continue
  }
  try {
    const resolved = createRequire(path.join(absoluteWeb, 'package.json')).resolve(id)
    const corrected = correctKnownNodeOnlyEntries(resolved)
    return corrected || resolved
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

/** Node 条件落到 vue/index.mjs 时改写到 runtime esm-bundler。 */
function correctKnownNodeOnlyEntries(resolvedPath) {
  if (!resolvedPath) return null
  const normalized = resolvedPath.replace(/\\/g, '/')
  if (normalized.endsWith('/vue/index.mjs') || normalized.endsWith('/vue/index.js')) {
    const bundler = path.join(path.dirname(resolvedPath), 'dist/vue.runtime.esm-bundler.js')
    if (fs.existsSync(bundler)) return bundler
  }
  return null
}
