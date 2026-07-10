import { dirname, isAbsolute, relative, resolve, sep } from 'node:path'

export type AdminExtensionGuardPolicy = {
  roots: Array<{ root: string, dependencies: string[] }>
  hostPeers: string[]
}

export function validateAdminExtensionImport(source: string, importer: string | undefined, policy: AdminExtensionGuardPolicy) {
  if (!importer) return
  const cleanImporter = importer.split('?', 1)[0] || importer
  const owner = policy.roots.find(candidate => inside(candidate.root, cleanImporter))
  if (!owner) return

  if (source.startsWith('~/') || source.startsWith('@/') || source === '#build' || source.startsWith('#build/')) {
    throw new Error(`trusted admin extension import is outside the supported host API: ${source}`)
  }
  if (source === '@sforum/admin-sdk/internal' || source.startsWith('@sforum/admin-sdk/internal/')) {
    throw new Error('trusted admin extensions cannot import the host-only Admin SDK entrypoint')
  }
  if (source.startsWith('.') || isAbsolute(source)) {
    if (isAbsolute(source) || !inside(owner.root, resolve(dirname(cleanImporter), source))) {
      throw new Error(`trusted admin extension import escapes its frontend root: ${source}`)
    }
    return
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

function inside(root: string, target: string) {
  const path = relative(resolve(root), resolve(target))
  return path === '' || (path !== '..' && !path.startsWith(`..${sep}`) && !isAbsolute(path))
}

function barePackageName(source: string) {
  const segments = source.split('/')
  return source.startsWith('@') ? segments.slice(0, 2).join('/') : segments[0] || source
}
