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

  // Vite 虚拟模块（如 \0plugin-vue:export-helper）由编译器注入，不是扩展可声明的 npm 依赖。
  // \0 是 Rollup/Vite 约定的 virtual id 前缀；virtual: 是常见插件虚拟模块前缀。
  if (isViteVirtualModuleId(source)) {
    return
  }

  if (source.startsWith('~/') || source.startsWith('@/') || source === '#build' || source.startsWith('#build/')) {
    throw new Error(`trusted admin extension import is outside the supported host API: ${source}`)
  }
  if (source === '@sforum/admin-sdk/internal' || source.startsWith('@sforum/admin-sdk/internal/')) {
    throw new Error('trusted admin extensions cannot import the host-only Admin SDK entrypoint')
  }
  if (source.startsWith('.') || isAbsolute(source)) {
    // 绝对 / 相对路径：允许扩展 root 内文件，以及解析到宿主 node_modules 的路径。
    // Vite/Nuxt 会把 UButton 写成绝对路径，也会把 dev tracer 写成相对路径
    // （如 ../../../../apps/web/node_modules/vite-plugin-vue-tracer/...）。
    // bare import 仍走下面的 hostPeers / dependencies 白名单，防止声明未授权包名。
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

/** Rollup/Vite 虚拟模块 id：\0 前缀或 virtual: 前缀，不参与扩展依赖白名单。 */
export function isViteVirtualModuleId(source: string) {
  return source.startsWith('\0') || source.startsWith('virtual:')
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
  // 任意 node_modules 路径都视为宿主工具链/依赖图产物（含 Nuxt dev tracer、
  // 传递依赖），不要求包名落在 hostPeers 里。真正的依赖边界由 bare import 校验。
  if (packageNameFromNodeModulesPath(resolved)) {
    return true
  }
  const allowed = new Set([...hostPeers, ...dependencies])
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
