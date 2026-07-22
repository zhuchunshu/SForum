import {
  loadPublicSurfaceRevision,
  PUBLIC_SURFACE_REVISION_HEADER
} from '../utils/publicSurfaceRevision'

type MutableRouteRulesContext = {
  _nitro?: {
    routeRules?: {
      cache?: false | Record<string, unknown>
      swr?: false | number
    }
  }
}

/**
 * 主题详情 HTML 缓存策略（M4 + public_surface_revision）：
 * - 匿名 + 非编辑态：允许 routeRules 短 SWR（见 nuxt.config `/t/**`）
 *   并将 site.public_surface_revision 写入 varies 请求头，使扩展公开贡献变更后
 *   新请求不再命中旧 HTML（无需重新激活主题）。
 * - 登录会话（sforum_session）或 ?edit=：禁缓存（个性化动作菜单 / 编辑壳）
 *
 * Nitro 按 URL + varies 头键缓存渲染结果，不能按用户区分；故仅匿名响应可进入 SWR。
 */
export default defineEventHandler(async (event) => {
  const url = getRequestURL(event)
  const path = url.pathname
  // no_prefix：主题路径仅 /t/**（旧 /en/t/** 由 locale-prefix-compat 301 剥离）。
  const isTopicPath = path === '/t' || path.startsWith('/t/')
  if (!isTopicPath) {
    return
  }

  const hasEdit = url.searchParams.has('edit')
  const cookie = getHeader(event, 'cookie') || ''
  const hasSession = /(?:^|;\s*)sforum_session=/.test(cookie)
  if (hasEdit || hasSession) {
    setHeader(event, 'cache-control', 'no-store')
    const routeRules = (event.context as MutableRouteRulesContext)._nitro?.routeRules
    if (!routeRules) {
      return
    }
    routeRules.cache = false
    routeRules.swr = false
    return
  }

  // 匿名：把 revision 写入请求头，供 Nitro SWR getKey 的 varies 参与缓存键。
  const revision = await loadPublicSurfaceRevision()
  const headers = event.node.req.headers
  headers[PUBLIC_SURFACE_REVISION_HEADER] = revision
})
