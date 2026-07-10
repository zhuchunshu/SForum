const AUTH_RETURN_BASE = 'http://sforum.local'
const AUTH_PAGE_PATTERN = /\/(?:login|register)\/?$/

export function normalizeAuthReturnPath(candidate: unknown): string | null {
  if (typeof candidate !== 'string' || !candidate.startsWith('/') || candidate.startsWith('//')) {
    return null
  }

  try {
    const url = new URL(candidate, AUTH_RETURN_BASE)
    if (url.origin !== AUTH_RETURN_BASE) {
      return null
    }

    const pathname = decodeURI(url.pathname)
    if (AUTH_PAGE_PATTERN.test(pathname)) {
      return null
    }

    return `${url.pathname}${url.search}${url.hash}`
  } catch {
    return null
  }
}

export function resolveAuthReturnPath(
  explicit: unknown,
  referrer: unknown,
  fallback: string
): string {
  return normalizeAuthReturnPath(explicit)
    ?? normalizeAuthReturnPath(referrer)
    ?? normalizeAuthReturnPath(fallback)
    ?? '/'
}
