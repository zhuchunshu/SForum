/**
 * 兼容旧 i18n prefix 策略（/en、/en/*）：301 剥前缀到无语言路径。
 * 新策略 no_prefix 下不再生成这些路由，但书签与外链仍可能访问。
 */
export default defineEventHandler((event) => {
  const url = getRequestURL(event)
  const path = url.pathname
  if (path !== '/en' && !path.startsWith('/en/')) {
    return
  }

  const stripped = path === '/en' ? '/' : (path.slice('/en'.length) || '/')
  const target = `${stripped}${url.search}${url.hash}`
  return sendRedirect(event, target, 301)
})
