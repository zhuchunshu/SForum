/**
 * 主题详情包含实时评论与会话权限，不进入 Nitro 整页缓存。
 * 匿名响应允许浏览器/CDN保存，但每次必须向源站重验证；会话和编辑态完全禁止存储。
 */
export default defineEventHandler((event) => {
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
    setHeader(event, 'cache-control', 'private, no-store')
    return
  }

  setHeader(event, 'cache-control', 'public, no-cache')
})
