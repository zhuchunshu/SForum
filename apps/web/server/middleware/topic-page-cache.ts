/**
 * 主题详情包含实时评论、会话权限与请求级 protected shortcode 组合。
 * 中间件在 API 读取前无法判断某个主题是否含 protected block，因此所有主题
 * HTML/payload 都必须禁止浏览器和共享缓存存储。
 */
export default defineEventHandler((event) => {
  const url = getRequestURL(event)
  const path = url.pathname
  // no_prefix：主题路径仅 /t/**（旧 /en/t/** 由 locale-prefix-compat 301 剥离）。
  const isTopicPath = path === '/t' || path.startsWith('/t/')
  if (!isTopicPath) {
    return
  }

  setHeader(event, 'cache-control', 'private, no-store')
  appendHeader(event, 'vary', 'Cookie')
  appendHeader(event, 'vary', 'Authorization')
  appendHeader(event, 'vary', 'Accept-Language')
})
