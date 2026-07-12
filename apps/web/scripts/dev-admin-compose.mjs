// 本地轻量 Web 开发 compose（P1）：
// 从 extensions/builtin 源码生成 registry / guard-policy / 主题 layer 入口，
// 组件与 layer 用软链指向源码，Vue 改动能走 Vite HMR，无需完整 Web Release。
//
// 完整 Web Release（隔离 workspace、production build、digest）仍是生产路径；
// 本脚本只服务 `bun run dev:compose`（plain `bun run dev` 不经过此处）。
//
// 宿主 peer（vue/nuxt/…）由 Nuxt alias 解析，禁止写入扩展源码树的 node_modules。

import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

import {
  ADMIN_HOST_PEER_NAMES,
  DEV_HOST_PEERS,
  pruneHostPeerNodeModules,
  resolveHostPeerDirectory,
} from '../build/admin-host-peers.mjs'

export { ADMIN_HOST_PEER_NAMES, DEV_HOST_PEERS, pruneHostPeerNodeModules, resolveHostPeerDirectory }

export const DEV_COMPOSE_RELEASE_ID = 'dev-local'
export const DEV_COMPOSE_DIRNAME = 'dev-compose'

/**
 * @param {object} options
 * @param {string} options.repoRoot 仓库根目录
 * @param {string} [options.outDir] 输出目录，默认 storage/theme-releases/dev-compose
 * @param {string} [options.builtinRoot] builtin 扩展根
 * @returns {{ outDir: string, registryRoot: string, themeLayer: string, releaseId: string, extensions: string[], compositionHash: string }}
 */
export function composeDevAdmin({
  repoRoot,
  outDir = path.join(repoRoot, 'storage/theme-releases', DEV_COMPOSE_DIRNAME),
  builtinRoot = path.join(repoRoot, 'extensions/builtin'),
  webRoot = path.join(repoRoot, 'apps/web'),
} = {}) {
  if (!repoRoot) {
    throw new Error('composeDevAdmin requires repoRoot')
  }
  const absoluteOut = path.resolve(outDir)
  const absoluteBuiltin = path.resolve(builtinRoot)
  // webRoot 保留在参数中供调用方与历史 API 兼容；peer 解析改由 Nuxt alias 负责。
  void webRoot

  const packages = discoverBuiltinAdminPackages(absoluteBuiltin)
  if (!packages.length) {
    throw new Error(`no builtin packages with frontend.admin under ${absoluteBuiltin}`)
  }

  // 增量更新：不要每次 rm 整个 outDir，否则会打断 Vite 对软链目标的 HMR 监听。
  fs.mkdirSync(absoluteOut, { recursive: true })

  const extensionsRoot = path.join(absoluteOut, 'extensions')
  fs.mkdirSync(extensionsRoot, { recursive: true })

  const contributions = []
  const localeMessages = {}
  const guardRoots = []
  const keepPackageDirs = new Set()

  for (const pkg of packages) {
    const packageTarget = path.join(extensionsRoot, pkg.id)
    keepPackageDirs.add(pkg.id)
    const adminTarget = path.join(packageTarget, 'frontend', 'admin')
    fs.mkdirSync(path.dirname(adminTarget), { recursive: true })
    ensureSymlink(pkg.adminRoot, adminTarget)
    // 清理旧 compose 写进源码树的 peer node_modules；解析改由宿主 Vite alias 负责。
    pruneHostPeerNodeModules(pkg.adminRoot)

    localeMessages[pkg.id] = loadLocales(pkg.adminRoot, pkg.locales)
    for (const contribution of pkg.contributions) {
      const componentRelative = pkg.components[contribution.componentId]
      if (!componentRelative) {
        throw new Error(`${pkg.id}: contribution ${contribution.id} component ${contribution.componentId} missing from frontend.admin.components`)
      }
      const modulePath = path.join(adminTarget, componentRelative)
      if (!fs.existsSync(modulePath)) {
        throw new Error(`${pkg.id}: component file missing: ${modulePath}`)
      }
      contributions.push({
        point: contribution.point,
        extensionId: pkg.id,
        contributionId: contribution.id,
        componentId: contribution.componentId,
        order: contribution.order,
        label: contribution.label,
        options: contribution.options,
        modulePath,
      })
    }
    guardRoots.push({
      root: adminTarget,
      dependencies: [],
    })
  }

  // 清理已删除的扩展目录。
  for (const entry of fs.readdirSync(extensionsRoot, { withFileTypes: true })) {
    if (entry.isDirectory() && !keepPackageDirs.has(entry.name)) {
      fs.rmSync(path.join(extensionsRoot, entry.name), { recursive: true, force: true })
    }
  }

  contributions.sort((left, right) => (
    left.order - right.order
    || left.extensionId.localeCompare(right.extensionId)
    || left.contributionId.localeCompare(right.contributionId)
  ))

  const theme = packages.find(item => item.type === 'theme') || packages[0]
  const themeDir = path.join(absoluteOut, 'theme')
  fs.mkdirSync(themeDir, { recursive: true })
  let themeLayer = ''
  if (theme?.layerRoot && fs.existsSync(theme.layerRoot)) {
    themeLayer = path.join(themeDir, 'layer')
    ensureSymlink(theme.layerRoot, themeLayer)
  }

  const registryRoot = path.join(absoluteOut, 'registry')
  fs.mkdirSync(registryRoot, { recursive: true })
  writeMetadata(path.join(registryRoot, 'metadata.ts'), contributions, localeMessages)
  writeRegistry(path.join(registryRoot, 'registry.client.ts'), registryRoot, contributions)
  writeJSON(path.join(absoluteOut, 'guard-policy.json'), {
    roots: guardRoots,
    hostPeers: [...ADMIN_HOST_PEER_NAMES].sort(),
  })

  // hash 含 locales / contributions，改文案或插槽映射会变；不含 .vue 文件内容
  // （组件经软链由 Vite HMR，避免每次保存都重启 Nuxt）。
  const compositionHash = crypto
    .createHash('sha256')
    .update(JSON.stringify({
      extensions: packages.map(item => ({
        id: item.id,
        version: item.version,
        admin: item.adminRoot,
        layer: item.layerRoot || '',
        components: item.components,
        contributions: item.contributions,
        locales: localeMessages[item.id],
      })),
      peers: ADMIN_HOST_PEER_NAMES,
    }))
    .digest('hex')

  writeJSON(path.join(absoluteOut, 'compose.json'), {
    schemaVersion: 1,
    releaseId: DEV_COMPOSE_RELEASE_ID,
    compositionHash,
    themeId: theme?.id || '',
    themeVersion: theme?.version || '',
    themeLayer,
    registryRoot,
    extensions: packages.map(item => item.id),
    composedAt: new Date().toISOString(),
  })

  return {
    outDir: absoluteOut,
    registryRoot,
    themeLayer,
    releaseId: DEV_COMPOSE_RELEASE_ID,
    extensions: packages.map(item => item.id),
    compositionHash,
    themeId: theme?.id || '',
    themeVersion: theme?.version || '',
  }
}

/**
 * 监听 builtin 中会影响 compose 的路径（manifest / locales / 目录增删）。
 * .vue 组件本身经软链由 Vite HMR，不必每次 recompose；但 locales 内联进 metadata，需 watch。
 */
export function watchDevAdminCompose({
  repoRoot,
  outDir,
  builtinRoot,
  debounceMs = 400,
  onComposed,
  onError = (error) => console.error('[sforum-dev-compose]', error.message),
  log = (message) => console.log(`[sforum-dev-compose] ${message}`),
} = {}) {
  const absoluteBuiltin = path.resolve(builtinRoot || path.join(repoRoot, 'extensions/builtin'))
  let timer = null
  let closed = false
  const watchers = []

  const run = (reason) => {
    if (closed) return
    clearTimeout(timer)
    timer = setTimeout(() => {
      try {
        const result = composeDevAdmin({ repoRoot, outDir, builtinRoot: absoluteBuiltin })
        log(`composed ${result.extensions.join(', ')} (${reason})`)
        onComposed?.(result, reason)
      } catch (error) {
        onError(error)
      }
    }, debounceMs)
  }

  const watchPath = (target) => {
    if (!fs.existsSync(target)) return
    try {
      const watcher = fs.watch(target, { recursive: true }, (_event, filename) => {
        const name = filename ? filename.toString() : ''
        if (shouldIgnoreWatchPath(name)) return
        run(name || path.basename(target))
      })
      watchers.push(watcher)
    } catch (error) {
      onError(error)
    }
  }

  // 首次立即 compose（不 debounce）
  const initial = composeDevAdmin({ repoRoot, outDir, builtinRoot: absoluteBuiltin })
  log(`composed ${initial.extensions.join(', ')} (startup)`)
  onComposed?.(initial, 'startup')

  watchPath(path.join(absoluteBuiltin, 'themes'))
  watchPath(path.join(absoluteBuiltin, 'plugins'))

  return {
    initial,
    close() {
      closed = true
      clearTimeout(timer)
      for (const watcher of watchers) {
        try {
          watcher.close()
        } catch {
          // ignore
        }
      }
    },
  }
}

export function shouldIgnoreWatchPath(relativePath) {
  if (!relativePath) return false
  const normalized = relativePath.replace(/\\/g, '/')
  return (
    normalized.includes('node_modules/')
    || normalized.includes('/.git/')
    || normalized.endsWith('.DS_Store')
    || normalized.endsWith('bun.lock')
    || /(?:^|\/)plugin$/.test(normalized) // 编译出的 go plugin 二进制
    || normalized.endsWith('.test.go')
    || normalized.endsWith('_test.go')
  )
}

export function discoverBuiltinAdminPackages(builtinRoot) {
  const results = []
  for (const kind of ['themes', 'plugins']) {
    const kindRoot = path.join(builtinRoot, kind)
    if (!fs.existsSync(kindRoot)) continue
    for (const entry of fs.readdirSync(kindRoot, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue
      const packageRoot = path.join(kindRoot, entry.name)
      const discovered = readBuiltinAdminPackage(packageRoot)
      if (discovered) results.push(discovered)
    }
  }
  results.sort((left, right) => left.id.localeCompare(right.id))
  return results
}

function readBuiltinAdminPackage(packageRoot) {
  const manifestPath = path.join(packageRoot, 'sforum.extension.json')
  if (!fs.existsSync(manifestPath)) return null
  const rootManifest = readJSON(manifestPath)
  const id = String(rootManifest.id || '').trim()
  if (!id) return null

  const frontendRel = rootManifest.includes?.frontend
  if (!frontendRel) return null
  const frontend = readJSON(path.join(packageRoot, frontendRel))
  const admin = frontend.admin
  if (!admin?.root || !admin.components || typeof admin.components !== 'object') {
    return null
  }

  const adminRoot = path.resolve(packageRoot, admin.root)
  if (!fs.existsSync(adminRoot)) {
    return null
  }

  const contributionsRel = rootManifest.includes?.contributions
  const contributionsRaw = contributionsRel
    ? readJSON(path.join(packageRoot, contributionsRel))
    : []
  if (!Array.isArray(contributionsRaw) || !contributionsRaw.length) {
    return null
  }

  const contributions = []
  for (const entry of contributionsRaw) {
    const point = String(entry.point || '').trim()
    // 主题/插件 admin 前端目前只消费设置页相关插槽。
    if (!point.startsWith('admin.extension.settings.')) continue
    const contributionId = String(entry.id || '').trim()
    const componentId = String(entry.payload?.component || '').trim()
    if (!contributionId || !componentId) continue
    const options = { ...(entry.payload || {}) }
    delete options.component
    contributions.push({
      point,
      id: contributionId,
      order: Number(entry.order) || 0,
      label: normalizeLabel(entry.label),
      componentId,
      options,
    })
  }
  if (!contributions.length) return null

  let layerRoot = ''
  if (frontend.layer) {
    layerRoot = path.resolve(packageRoot, frontend.layer)
  }

  return {
    id,
    version: String(rootManifest.version || '0.0.0'),
    type: String(rootManifest.type || (path.basename(path.dirname(packageRoot)) === 'themes' ? 'theme' : 'plugin')),
    packageRoot,
    adminRoot,
    layerRoot,
    components: admin.components,
    locales: admin.locales && typeof admin.locales === 'object' ? admin.locales : {},
    contributions,
  }
}

function loadLocales(adminRoot, localeMap) {
  const out = {}
  for (const [locale, relative] of Object.entries(localeMap || {})) {
    const file = path.join(adminRoot, relative)
    if (!fs.existsSync(file)) {
      throw new Error(`locale file missing: ${file}`)
    }
    out[locale] = readJSON(file)
  }
  return out
}

function writeMetadata(target, contributions, localeMessages) {
  const publicItems = contributions.map(item => ({
    point: item.point,
    extensionId: item.extensionId,
    contributionId: item.contributionId,
    componentId: item.componentId,
    order: item.order,
    label: item.label || {},
    options: item.options || {},
  }))
  const body = [
    "import type { AdminComponentMetadata, AdminExtensionLocaleMessages } from '~/runtime/admin-extensions/types'",
    '',
    `export const releaseId = ${JSON.stringify(DEV_COMPOSE_RELEASE_ID)}`,
    'export const reloadMode = "prompt"',
    `export const contributions: readonly AdminComponentMetadata[] = ${JSON.stringify(publicItems)}`,
    `export const locales: AdminExtensionLocaleMessages = ${JSON.stringify(localeMessages)}`,
    '',
  ].join('\n')
  fs.writeFileSync(target, body, 'utf8')
}

function writeRegistry(target, registryRoot, contributions) {
  const lines = [
    "import type { AdminComponentRegistry } from '~/runtime/admin-extensions/types'",
    '',
    'export const registry: AdminComponentRegistry = {',
  ]
  for (const item of contributions) {
    let relative = path.relative(registryRoot, item.modulePath)
    relative = relative.split(path.sep).join('/')
    if (!relative.startsWith('.')) relative = `./${relative}`
    const key = `${item.extensionId}:${item.contributionId}`
    lines.push(`  ${JSON.stringify(key)}: () => import(${JSON.stringify(relative)}),`)
  }
  lines.push('}', '')
  fs.writeFileSync(target, lines.join('\n'), 'utf8')
}

function ensureSymlink(target, linkPath) {
  const absoluteTarget = path.resolve(target)
  if (!fs.existsSync(absoluteTarget)) {
    throw new Error(`symlink target missing: ${absoluteTarget}`)
  }
  fs.mkdirSync(path.dirname(linkPath), { recursive: true })
  try {
    const stat = fs.lstatSync(linkPath)
    if (stat.isSymbolicLink()) {
      const current = fs.readlinkSync(linkPath)
      const resolved = path.isAbsolute(current) ? current : path.resolve(path.dirname(linkPath), current)
      if (path.resolve(resolved) === absoluteTarget) {
        return
      }
      fs.unlinkSync(linkPath)
    } else {
      fs.rmSync(linkPath, { recursive: true, force: true })
    }
  } catch (error) {
    if (error.code !== 'ENOENT') throw error
  }
  // 目录目标用 'dir'；Windows 上 junction 更稳，mac/linux 均可。
  fs.symlinkSync(absoluteTarget, linkPath, 'dir')
}

function normalizeLabel(label) {
  if (!label || typeof label !== 'object' || Array.isArray(label)) return {}
  const out = {}
  for (const [key, value] of Object.entries(label)) {
    if (typeof value === 'string' && value.trim()) out[key] = value.trim()
  }
  return out
}

function readJSON(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'))
}

function writeJSON(file, value) {
  fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`, 'utf8')
}
