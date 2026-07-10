export type ForumHomeFilters = {
  query: string
  categorySlug: string
  tagSlug: string
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
