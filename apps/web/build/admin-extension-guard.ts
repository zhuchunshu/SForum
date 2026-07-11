import { dirname, isAbsolute, relative, resolve, sep } from 'node:path'

export type AdminExtensionGuardPolicy = {
  roots: Array<{ root: string, dependencies: string[] }>
  hostPeers: string[]
}

export function validateAdminExtensionImport(source: string, importer: string | undefined, policy: AdminExtensionGuardPolicy) {
  if (!importer) return
  // Vite/Vue SFC 会在路径后附带 ?vue&type=script 等 query，校验时需剥离。
  const cleanImporter = stripQuery(importer)
  const owner = policy.roots.find(candidate => inside(candidate.root, cleanImporter))
  if (!owner) return

  if (source.startsWith('~/') || source.startsWith('@/') || source === '#build' || source.startsWith('#build/')) {
    throw new Error(`trusted admin extension import is outside the supported host API: ${source}`)
  }
  if (source === '@sforum/admin-sdk/internal' || source.startsWith('@sforum/admin-sdk/internal/')) {
    throw new Error('trusted admin extensions cannot import the host-only Admin SDK entrypoint')
  }
  if (source.startsWith('.') || isAbsolute(source)) {
    // 绝对路径 / 相对路径：允许扩展 root 内文件，以及解析到 host peer / 私有依赖包的路径。
    // Vite 在编译 SFC 时会把 UButton 等解析成 apps/web/node_modules/@nuxt/ui/... 绝对路径；
    // 旧逻辑只允许 root 内路径，会把合法 host peer 当成越界。
    const cleanSource = stripQuery(source)
    const resolved = isAbsolute(cleanSource)
      ? resolve(cleanSource)
      : resolve(dirname(cleanImporter), cleanSource)
    if (inside(owner.root, resolved)) {
      return
    }
    if (isAllowedExternalPackagePath(resolved, owner.dependencies, policy.hostPeers)) {
      return
    }
    throw new Error(`trusted admin extension import escapes its frontend root: ${source}`)
  }
  if (source === '#app' || source.startsWith('#app/') || source === '#imports') return

  const packageName = barePackageName(source)
  if (!policy.hostPeers.includes(packageName) && !owner.dependencies.includes(packageName)) {
    throw new Error(`trusted admin extension imports undeclared package: ${packageName}`)
  }
}

export function adminExtensionGuard(policy: AdminExtensionGuardPolicy) {
  return {
    name: 'sforum-admin-extension-guard',
    enforce: 'pre' as const,
    resolveId(source: string, importer?: string) {
      validateAdminExtensionImport(source, importer, policy)
      return null
    }
  }
}

function stripQuery(value: string) {
  return value.split('?', 1)[0] || value
}

function inside(root: string, target: string) {
  const path = relative(resolve(root), resolve(target))
  return path === '' || (path !== '..' && !path.startsWith(`..${sep}`) && !isAbsolute(path))
}

function barePackageName(source: string) {
  const segments = source.split('/')
  return source.startsWith('@') ? segments.slice(0, 2).join('/') : segments[0] || source
}

/** 从绝对路径中识别 node_modules 下的包名（取最后一段 node_modules 之后）。 */
export function packageNameFromNodeModulesPath(absolutePath: string) {
  const normalized = resolve(absolutePath)
  const marker = `${sep}node_modules${sep}`
  const index = normalized.lastIndexOf(marker)
  if (index < 0) {
    return ''
  }
  const after = normalized.slice(index + marker.length)
  const segments = after.split(sep).filter(Boolean)
  const first = segments[0]
  if (!first) {
    return ''
  }
  if (first.startsWith('@')) {
    const scope = segments[1]
    if (!scope) {
      return ''
    }
    return `${first}/${scope}`
  }
  return first
}

function isAllowedExternalPackagePath(resolved: string, dependencies: string[], hostPeers: string[]) {
  const allowed = new Set([...hostPeers, ...dependencies])
  const fromNodeModules = packageNameFromNodeModulesPath(resolved)
  if (fromNodeModules && allowed.has(fromNodeModules)) {
    return true
  }
  // @sforum/admin-sdk 在发布工作区里是 packages/admin-sdk 符号链接，不一定在 node_modules 下。
  if (allowed.has('@sforum/admin-sdk') && isWorkspaceAdminSdkPath(resolved)) {
    return true
  }
  return false
}

function isWorkspaceAdminSdkPath(absolutePath: string) {
  const normalized = `${resolve(absolutePath)}${sep}`
  return normalized.includes(`${sep}packages${sep}admin-sdk${sep}`)
}
