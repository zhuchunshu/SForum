import type { ForumCategory, ForumCategoryGroup } from './forumTaxonomy'

export type CategoryDirectorySort = 'default' | 'active' | 'name'

export type CategoryDirectoryGroup = ForumCategoryGroup & {
  categories: ForumCategory[]
}

export type CategoryDirectoryStats = {
  groupCount: number
  categoryCount: number
  topicCount: number
  commentCount: number
}

export function visibleCategoryDirectoryGroups(groups: ForumCategoryGroup[] | null | undefined): CategoryDirectoryGroup[] {
  return (groups || [])
    .filter((group) => group.visibility !== 'hidden')
    .map((group) => ({
      ...group,
      categories: (group.categories || []).filter((category) => category.visibility !== 'hidden')
    }))
}

export function categoryDirectoryGroupKey(group: Pick<ForumCategoryGroup, 'id'>) {
  return String(group.id)
}

export function findCategoryDirectoryGroup(groups: CategoryDirectoryGroup[], groupKey: string) {
  return groups.find((group) => categoryDirectoryGroupKey(group) === groupKey)
}

export function sortCategoryDirectoryCategories(
  categories: ForumCategory[],
  mode: CategoryDirectorySort,
  locale?: string
) {
  const list = categories.slice()
  switch (mode) {
    case 'active':
      // 目录页沿用既有“活跃”定义：当前公开 DTO 的 topicCount 降序。
      return list.sort((left, right) =>
        numericDesc(left.topicCount, right.topicCount)
        || byDefaultPosition(left, right)
        || byStableID(left, right)
      )
    case 'name':
      return list.sort((left, right) =>
        left.name.localeCompare(right.name, locale, { sensitivity: 'base' })
        || byDefaultPosition(left, right)
        || byStableID(left, right)
      )
    default:
      return list.sort((left, right) => byDefaultPosition(left, right) || byStableID(left, right))
  }
}

export function buildCategoryDirectoryDisplayGroups(
  groups: CategoryDirectoryGroup[],
  options: {
    sort: CategoryDirectorySort
    locale?: string
    focusedGroupKey?: string
    query?: string
  }
) {
  const focused = options.focusedGroupKey
    ? findCategoryDirectoryGroup(groups, options.focusedGroupKey)
    : undefined
  const source = focused ? [focused] : groups
  const query = normalizeDirectoryQuery(options.query)

  return source
    .map((group) => {
      const filtered = query
        ? group.categories.filter((category) => categoryMatchesDirectoryQuery(category, query))
        : group.categories
      return {
        ...group,
        categories: sortCategoryDirectoryCategories(filtered, options.sort, options.locale)
      }
    })
    .filter((group) => !query || group.categories.length > 0)
}

export function categoryMatchesDirectoryQuery(category: ForumCategory, normalizedQuery: string) {
  const haystack = [
    category.name,
    category.description,
    category.slug
  ].join(' ').toLocaleLowerCase()
  return haystack.includes(normalizedQuery)
}

export function normalizeDirectoryQuery(query: string | null | undefined) {
  return (query || '').trim().toLocaleLowerCase()
}

export function summarizeCategoryDirectory(groups: CategoryDirectoryGroup[]): CategoryDirectoryStats {
  const categories = groups.flatMap((group) => group.categories)
  return summarizeCategoryList(groups.length, categories)
}

export function summarizeCategoryDirectoryDisplay(groups: CategoryDirectoryGroup[]): CategoryDirectoryStats {
  return summarizeCategoryList(groups.length, groups.flatMap((group) => group.categories))
}

export function activeCategoryDirectoryCategories(
  groups: CategoryDirectoryGroup[],
  limit: number,
  locale?: string
) {
  if (!Number.isFinite(limit) || limit <= 0) {
    return []
  }
  return sortCategoryDirectoryCategories(
    groups.flatMap((group) => group.categories),
    'active',
    locale
  ).slice(0, Math.trunc(limit))
}

export function categoryDirectoryDistribution(groups: CategoryDirectoryGroup[]) {
  const max = Math.max(1, ...groups.map((group) => group.categories.length))
  return groups.map((group) => ({
    group,
    count: group.categories.length,
    percent: Math.round((group.categories.length / max) * 100)
  }))
}

function summarizeCategoryList(groupCount: number, categories: ForumCategory[]): CategoryDirectoryStats {
  return {
    groupCount,
    categoryCount: categories.length,
    topicCount: categories.reduce((sum, category) => sum + safeCount(category.topicCount), 0),
    commentCount: categories.reduce((sum, category) => sum + safeCount(category.commentCount), 0)
  }
}

function safeCount(value: number) {
  return Number.isFinite(value) ? Math.max(0, Math.trunc(value)) : 0
}

function numericDesc(left: number, right: number) {
  return safeCount(right) - safeCount(left)
}

function byDefaultPosition(left: Pick<ForumCategory, 'position'>, right: Pick<ForumCategory, 'position'>) {
  return safeCount(left.position) - safeCount(right.position)
}

function byStableID(left: Pick<ForumCategory, 'id'>, right: Pick<ForumCategory, 'id'>) {
  return safeCount(left.id) - safeCount(right.id)
}
