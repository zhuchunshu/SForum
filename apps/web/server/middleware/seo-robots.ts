const staticAssetPattern = /\.[a-z0-9]{2,8}$/i

export default defineEventHandler(async (event) => {
  if (event.method !== 'GET') {
    return
  }

  const url = getRequestURL(event)
  if (staticAssetPattern.test(url.pathname)) {
    return
  }

  const settings = await loadServerSEOSettings()
  const protectedPath = isProtectedSEOPath(url.pathname)
  const rule = protectedPath || !serverSEOIndexable(settings)
    ? 'noindex, nofollow'
    : 'index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1'

  setHeader(event, 'X-Robots-Tag', rule)
})

function isProtectedSEOPath(path: string) {
  const adminPrefix = (process.env.NUXT_PUBLIC_ADMIN_ROUTE_PREFIX || process.env.ADMIN_ROUTE_PREFIX || '/control-panel').replace(/\/+$/, '')
  return path.startsWith('/api/') ||
    path === '/login' ||
    path.startsWith('/login/') ||
    path === '/register' ||
    path.startsWith('/register/') ||
    path === '/components' ||
    path.startsWith('/components/') ||
    path === adminPrefix ||
    path.startsWith(`${adminPrefix}/`)
}
