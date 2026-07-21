type MutableRouteRulesContext = {
  _nitro?: {
    routeRules?: {
      cache?: false | Record<string, unknown>
      swr?: false | number
    }
  }
}

/**
 * 登录用户的公共页面会在 SSR 阶段恢复会话，因此绝不能进入共享页面缓存。
 * 这里只处理 HTML 与 Nuxt payload；API、构建资源和普通静态资源保留原缓存策略。
 */
export default defineEventHandler((event) => {
  const cookie = getHeader(event, 'cookie') || ''
  if (!/(?:^|;\s*)sforum_session=/.test(cookie)) {
    return
  }

  const path = getRequestURL(event).pathname
  if (path.startsWith('/api/') || path.startsWith('/_nuxt/')) {
    return
  }

  const accept = getHeader(event, 'accept') || ''
  const isPageRequest = accept.includes('text/html')
    || path.endsWith('/_payload.json')
    || path.endsWith('/_payload.js')
  if (!isPageRequest) {
    return
  }

  setHeader(event, 'cache-control', 'no-store')
  const routeRules = (event.context as MutableRouteRulesContext)._nitro?.routeRules
  if (!routeRules) {
    return
  }

  routeRules.cache = false
  routeRules.swr = false
})
