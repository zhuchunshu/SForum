import {
  parseCanonicalLocalOrigin,
  parseLocalRequestHost,
  resolveCanonicalLocalRedirect
} from '../utils/canonicalLocalOrigin'

const canonicalLocalOrigin = parseCanonicalLocalOrigin(process.env.APP_URL)

export default defineEventHandler((event) => {
  const host = getRequestHeader(event, 'host')
  // 先严格校验原始 Host，再让 H3 解析 path/query；不启用 proxy Host fallback。
  if (!parseLocalRequestHost(host)) return

  const url = getRequestURL(event, { xForwardedProto: false })
  const target = resolveCanonicalLocalRedirect(canonicalLocalOrigin, {
    development: import.meta.dev,
    method: event.method,
    accept: getRequestHeader(event, 'accept'),
    host,
    protocol: getRequestProtocol(event, { xForwardedProto: false }),
    pathname: url.pathname,
    search: url.search
  })
  if (!target) return

  setHeader(event, 'cache-control', 'no-store')
  return sendRedirect(event, target, 307)
})
