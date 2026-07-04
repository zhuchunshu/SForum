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
  const normalizedChild = childPath.trim()

  if (!normalizedChild || normalizedChild === '/') {
    return normalizedPrefix
  }

  return `${normalizedPrefix}/${normalizedChild.replace(/^\/+|\/+$/g, '')}`
}
