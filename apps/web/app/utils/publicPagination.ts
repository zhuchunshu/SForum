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
