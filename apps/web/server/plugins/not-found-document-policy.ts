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
    // Nuxt 4.6 起渲染头直接落在 event 上，render:response 的 response.headers 会缺省；
    // 归一化容器并从 event 兜底读取，Nitro 会在钩子后把 response.headers 回写到 event。
    response.headers = response.headers || {}
    const headers = response.headers as Record<string, string | undefined>
    const contentType = headers['content-type'] || headers['Content-Type'] || String(getResponseHeader(event, 'content-type') || '')
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
