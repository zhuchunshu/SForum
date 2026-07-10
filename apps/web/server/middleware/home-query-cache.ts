type MutableRouteRulesContext = {
  _nitro?: {
    routeRules?: {
      cache?: false | Record<string, unknown>
      swr?: false | number
    }
  }
}

export default defineEventHandler((event) => {
  const url = getRequestURL(event)
  if ((url.pathname !== '/' && url.pathname !== '/en') || !url.search) {
    return
  }

  // Nitro 的根路由 payload 文件键无法安全承载 query；筛选页保持 SSR，但绕过页面缓存。
  setHeader(event, 'cache-control', 'no-store')
  const routeRules = (event.context as MutableRouteRulesContext)._nitro?.routeRules
  if (!routeRules) {
    return
  }

  routeRules.cache = false
  routeRules.swr = false
})
