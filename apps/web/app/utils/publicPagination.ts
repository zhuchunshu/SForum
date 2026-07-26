export type PublicPageLocation = {
  path: string
  query?: Record<string, string>
}

export function parsePublicPage(value: unknown) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) {
    return 1
  }
  const page = Number(value)
  return Number.isSafeInteger(page) ? page : 1
}

// 严格版：仅合法正整数字符串视为"显式指定了页码"，其余返回 0。
// 与 parsePublicPage 的区别：非法值不回落到 1——回落会被调用方误判为
// "用户显式要第 1 页"，进而抑制锚点反查等仅在未显式指定时才启用的逻辑。
export function parseExplicitPublicPage(value: unknown) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) {
    return 0
  }
  const page = Number(value)
  return Number.isSafeInteger(page) ? page : 0
}

export function publicPageLocation(
  path: string,
  page: number,
  query: Record<string, string> = {}
): PublicPageLocation {
  const normalizedPage = Number.isSafeInteger(page) && page > 1 ? page : 1
  const nextQuery = { ...query }
  if (normalizedPage > 1) {
    nextQuery.page = String(normalizedPage)
  } else {
    delete nextQuery.page
  }
  return Object.keys(nextQuery).length > 0 ? { path, query: nextQuery } : { path }
}

export function publicPagePath(path: string, page: number, query: Record<string, string> = {}) {
  const location = publicPageLocation(path, page, query)
  if (!location.query) {
    return location.path
  }
  return `${location.path}?${new URLSearchParams(location.query).toString()}`
}
