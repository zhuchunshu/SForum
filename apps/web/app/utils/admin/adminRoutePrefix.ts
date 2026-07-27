export const DEFAULT_ADMIN_ROUTE_PREFIX = '/control-panel'
export const LEGACY_ADMIN_ROUTE_PREFIX = '/admin'

export function normalizeAdminRoutePrefix(prefix?: string | null) {
  const value = `${prefix || ''}`.trim()

  if (!value || value === '/') {
    return DEFAULT_ADMIN_ROUTE_PREFIX
  }

  const withLeadingSlash = value.startsWith('/') ? value : `/${value}`
  return withLeadingSlash.replace(/\/+$/, '') || DEFAULT_ADMIN_ROUTE_PREFIX
}

export function joinAdminRoutePath(prefix: string, childPath = '') {
  const normalizedPrefix = normalizeAdminRoutePrefix(prefix)
  const normalizedChild = normalizeAdminChildPath(childPath)

  if (normalizedChild === '/') {
    return normalizedPrefix
  }

  return `${normalizedPrefix}${normalizedChild}`
}

export function normalizeAdminChildPath(childPath = '') {
  const normalizedChild = childPath.trim()

  if (!normalizedChild || normalizedChild === '/') {
    return '/'
  }

  return `/${normalizedChild.replace(/^\/+|\/+$/g, '')}`
}

export function resolveAdminRouteChildPath(prefix: string, routePath: string) {
  const normalizedPrefix = normalizeAdminRoutePrefix(prefix)
  const cleanPath = `${routePath || ''}`.split(/[?#]/, 1)[0]?.replace(/\/+$/, '') || '/'
  // 后台源路由会被 Nuxt i18n 加上可选语言前缀，这里只还原后台内部相对路径。
  const pathWithoutLocale = cleanPath.replace(/^\/[a-zA-Z]{2}(?:-[a-zA-Z]{2})?(?=\/)/, '') || '/'

  if (pathWithoutLocale === normalizedPrefix) {
    return '/'
  }

  if (pathWithoutLocale.startsWith(`${normalizedPrefix}/`)) {
    return normalizeAdminChildPath(pathWithoutLocale.slice(normalizedPrefix.length))
  }

  return null
}
