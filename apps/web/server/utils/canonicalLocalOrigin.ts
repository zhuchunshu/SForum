const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '[::1]'])
const EXCLUDED_PATH_PREFIXES = ['/api', '/_nuxt', '/_sforum', '/health'] as const

export type CanonicalLocalOrigin = Readonly<{
  origin: string
  protocol: 'http:' | 'https:'
  hostname: string
  port: number
}>

export type CanonicalLocalRequest = Readonly<{
  development: boolean
  method: string
  accept?: string
  host?: string
  protocol: string
  pathname: string
  search: string
}>

type LocalRequestHost = Readonly<{
  hostname: string
  explicitPort?: number
}>

function effectivePort(protocol: string, explicitPort?: number) {
  return explicitPort || (protocol === 'https:' ? 443 : 80)
}

export function parseCanonicalLocalOrigin(raw?: string): CanonicalLocalOrigin | null {
  const value = raw?.trim()
  if (!value) return null

  try {
    const url = new URL(value)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return null
    if (!LOOPBACK_HOSTS.has(url.hostname.toLowerCase())) return null
    if (url.username || url.password || url.pathname !== '/' || url.search || url.hash) return null

    return Object.freeze({
      origin: url.origin,
      protocol: url.protocol,
      hostname: url.hostname.toLowerCase(),
      port: effectivePort(url.protocol, url.port ? Number(url.port) : undefined)
    })
  } catch {
    return null
  }
}

export function parseLocalRequestHost(raw?: string): LocalRequestHost | null {
  const value = raw?.trim()
  if (!value || /[\s,@/\\]/.test(value)) return null

  const match = /^(localhost|127\.0\.0\.1|\[::1\])(?::([0-9]{1,5}))?$/i.exec(value)
  if (!match?.[1]) return null

  const explicitPort = match[2] ? Number(match[2]) : undefined
  if (explicitPort !== undefined && (explicitPort < 1 || explicitPort > 65535)) return null

  return {
    hostname: match[1].toLowerCase(),
    explicitPort
  }
}

function isExcludedPath(pathname: string) {
  return EXCLUDED_PATH_PREFIXES.some(prefix =>
    pathname === prefix || pathname.startsWith(`${prefix}/`)
  )
}

export function resolveCanonicalLocalRedirect(
  canonical: CanonicalLocalOrigin | null,
  request: CanonicalLocalRequest
): string | null {
  if (!canonical || !request.development) return null

  const method = request.method.toUpperCase()
  if (method !== 'GET' && method !== 'HEAD') return null
  if (!/\btext\/html\b/i.test(request.accept || '')) return null

  const requestHost = parseLocalRequestHost(request.host)
  const protocol = request.protocol.endsWith(':') ? request.protocol.toLowerCase() : `${request.protocol.toLowerCase()}:`
  if (!requestHost || (protocol !== 'http:' && protocol !== 'https:')) return null
  if (effectivePort(protocol, requestHost.explicitPort) !== canonical.port) return null

  const { pathname, search } = request
  if (
    !pathname.startsWith('/')
    || pathname.startsWith('//')
    || /[\r\n]/.test(pathname)
    || (search && (!search.startsWith('?') || /[\r\n]/.test(search)))
    || isExcludedPath(pathname)
  ) {
    return null
  }

  if (requestHost.hostname === canonical.hostname && protocol === canonical.protocol) {
    return null
  }

  return `${canonical.origin}${pathname}${search}`
}
