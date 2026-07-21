type MutableRouteRulesContext = {
  _nitro?: {
    routeRules?: {
      cache?: false | Record<string, unknown>
      swr?: false | number
    }
  }
}

/**
 * 主题详情 HTML 缓存策略（M4）：
 * - 匿名 + 非编辑态：允许 routeRules 短 SWR（见 nuxt.config `/t/**`）
 * - 登录会话（sforum_session）或 ?edit=：禁缓存（个性化动作菜单 / 编辑壳）
 *
 * Nitro 按 URL 键缓存渲染结果，不能按用户区分；故仅匿名响应可进入 SWR。
 */
export default defineEventHandler((event) => {
  const url = getRequestURL(event)
  const path = url.pathname
  const isTopicPath = path === '/t' || path.startsWith('/t/') || path === '/en/t' || path.startsWith('/en/t/')
  if (!isTopicPath) {
    return
  }

  const hasEdit = url.searchParams.has('edit')
  const cookie = getHeader(event, 'cookie') || ''
  const hasSession = /(?:^|;\s*)sforum_session=/.test(cookie)
  if (!hasEdit && !hasSession) {
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
