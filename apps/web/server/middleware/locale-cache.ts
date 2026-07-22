type MutableRouteRulesContext = {
  _nitro?: {
    routeRules?: {
      cache?: false | Record<string, unknown>
      swr?: false | number
    }
  }
}

const LOCALE_COOKIE = 'sforum_locale'
const DEFAULT_LOCALE = 'zh-CN'

/**
 * no_prefix 下同一 URL 服务多语言；Nitro SWR 只按路径键缓存。
 * 非默认语 cookie 请求必须绕过共享缓存，否则会把 zh 首屏串给 en 用户（或反过来）。
 */
export default defineEventHandler((event) => {
  const cookie = getHeader(event, 'cookie') || ''
  const match = cookie.match(new RegExp(`(?:^|;\\s*)${LOCALE_COOKIE}=([^;]+)`))
  const raw = match?.[1] ? decodeURIComponent(match[1].trim()) : ''
  if (!raw || raw === DEFAULT_LOCALE) {
    return
  }

  setHeader(event, 'cache-control', 'private, no-store')
  const routeRules = (event.context as MutableRouteRulesContext)._nitro?.routeRules
  if (!routeRules) {
    return
  }
  routeRules.cache = false
  routeRules.swr = false
})
