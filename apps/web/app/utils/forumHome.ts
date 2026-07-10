export type ForumHomeFilters = {
  query: string
  categorySlug: string
  tagSlug: string
}

export type ForumHomeRequestToken = {
  generation: number
  feedKey: string
}

export type ForumHomePageProgress = {
  requestedPage: number
  responsePage: number
  responseItemCount: number
  newItemCount: number
  loadedCount: number
  total: number
  perPage: number
}

type RouteQuery = Record<string, unknown>

const scalar = (value: unknown) => typeof value === 'string' ? value.trim() : ''

export const parseForumHomeQuery = (query: RouteQuery): ForumHomeFilters => ({
  query: scalar(query.q),
  categorySlug: scalar(query.category),
  tagSlug: scalar(query.tag)
})

export const buildForumHomeQuery = (filters: ForumHomeFilters) => Object.fromEntries(
  [
    ['q', filters.query.trim()],
    ['category', filters.categorySlug.trim()],
    ['tag', filters.tagSlug.trim()]
  ].filter((entry): entry is [string, string] => Boolean(entry[1]))
)

export const forumHomeFeedKey = (filters: ForumHomeFilters) => JSON.stringify([
  filters.query.trim(), filters.categorySlug.trim(), filters.tagSlug.trim()
])

export function isForumHomeRequestCurrent(
  request: ForumHomeRequestToken,
  currentGeneration: number,
  activeFeedKey: string
) {
  return request.generation === currentGeneration && request.feedKey === activeFeedKey
}

export function hasReachedForumHomeEnd(progress: ForumHomePageProgress) {
  return progress.responsePage < progress.requestedPage
    || progress.responseItemCount === 0
    || progress.responseItemCount < progress.perPage
    || progress.newItemCount === 0
    || progress.loadedCount >= progress.total
}
