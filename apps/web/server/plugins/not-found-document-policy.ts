/** 仅收口 Nuxt 渲染出的 404 document/payload；API 响应保持既有契约。 */
export default defineNitroPlugin((nitroApp) => {
  nitroApp.hooks.hook('render:response', (response, { event }) => {
    if (response.statusCode !== 404) {
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
