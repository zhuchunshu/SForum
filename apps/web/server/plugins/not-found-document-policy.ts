const SYSTEM_ERROR_STATUSES = new Set([403, 404, 429, 500, 502, 503, 504])

/** 收口 Nuxt 渲染出的系统错误 document/payload；API 响应保持既有契约。 */
export default defineNitroPlugin((nitroApp) => {
  nitroApp.hooks.hook('render:response', (response, { event }) => {
    if (!SYSTEM_ERROR_STATUSES.has(response.statusCode || 0)) {
      return
    }

    const path = getRequestURL(event).pathname
    if (path.startsWith('/api/')) {
      return
    }
    const accept = getRequestHeader(event, 'accept') || ''
    const headers = response.headers as Record<string, string | undefined>
    const contentType = headers['content-type'] || headers['Content-Type'] || ''
    const isHTML = accept.includes('text/html') || contentType.includes('text/html')
    const isPayload = contentType.includes('application/json') || path.endsWith('/_payload.json')
    if (!isHTML && !isPayload) {
      return
    }

    headers['cache-control'] = 'no-store'
    if (isHTML) {
      headers['x-robots-tag'] = 'noindex,nofollow'
    }
    const routeRules = (event.context as { _nitro?: { routeRules?: { cache?: boolean, swr?: boolean } } })._nitro?.routeRules
    if (routeRules) {
      routeRules.cache = false
      routeRules.swr = false
    }
  })
})
