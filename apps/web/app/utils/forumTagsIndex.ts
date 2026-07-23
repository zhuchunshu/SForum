import {
  isCreatedWithinDays,
  tagHotThreshold,
  type ForumTag
} from './forumTaxonomy'

export type TagIndexFilter = 'all' | 'hot' | 'week' | 'az'

export type TagIndexFilterOptions = {
  filter: TagIndexFilter
  query?: string
  locale?: string
  nowMs?: number
}

export type TagHeatEntry = {
  tag: ForumTag
  widthPercent: number
}

export type TagIndexOverview = {
  totalTags: number
  totalTopicReferences: number
  weekNewTags: number
  hotThreshold: number
}

export function activeForumTags(tags: ForumTag[]) {
  return tags.filter((tag) => tag.status === 'active')
}

export function tagDisplayDescription(tag: ForumTag, fallback: string) {
  const description = tag.description.trim()
  return description || fallback
}

export function compareTagsByName(left: ForumTag, right: ForumTag, locale?: string) {
  return left.name.localeCompare(right.name, locale, { sensitivity: 'base' })
    || left.slug.localeCompare(right.slug, locale, { sensitivity: 'base' })
}

export function compareTagsByHeat(left: ForumTag, right: ForumTag, locale?: string) {
  return (right.topicCount || 0) - (left.topicCount || 0)
    || compareTagsByName(left, right, locale)
}

export function tagIndexOverview(tags: ForumTag[], nowMs = Date.now()): TagIndexOverview {
  const activeTags = activeForumTags(tags)
  return {
    totalTags: activeTags.length,
    totalTopicReferences: activeTags.reduce((sum, tag) => sum + (tag.topicCount || 0), 0),
    weekNewTags: activeTags.filter((tag) => isCreatedWithinDays(tag.createdAt, 7, nowMs)).length,
    hotThreshold: tagHotThreshold(activeTags.map((tag) => tag.topicCount || 0))
  }
}

export function filterTagIndexTags(tags: ForumTag[], options: TagIndexFilterOptions) {
  const activeTags = activeForumTags(tags)
  const query = (options.query || '').trim().toLocaleLowerCase()
  const threshold = tagHotThreshold(activeTags.map((tag) => tag.topicCount || 0))
  let list = activeTags.slice()

  if (query) {
    list = list.filter((tag) => {
      const haystack = `${tag.name}\n${tag.slug}\n${tag.description}`.toLocaleLowerCase()
      return haystack.includes(query)
    })
  }

  switch (options.filter) {
    case 'hot':
      list = list.filter((tag) => (tag.topicCount || 0) >= threshold)
      break
    case 'week':
      // 现有公开标签接口没有周活跃定义，本周筛选按真实 createdAt 表达“近 7 天新增”。
      list = list.filter((tag) => isCreatedWithinDays(tag.createdAt, 7, options.nowMs))
      break
    case 'az':
      list = list.sort((left, right) => compareTagsByName(left, right, options.locale))
      break
    default:
      break
  }

  return list
}

export function tagHeatEntries(tags: ForumTag[], limit = 6, locale?: string): TagHeatEntry[] {
  const sorted = activeForumTags(tags)
    .sort((left, right) => compareTagsByHeat(left, right, locale))
    .slice(0, Math.max(0, limit))
  const max = sorted.reduce((current, tag) => Math.max(current, tag.topicCount || 0), 0)

  return sorted.map((tag) => ({
    tag,
    widthPercent: max > 0
      ? Math.max(8, Math.round(((tag.topicCount || 0) / max) * 100))
      : 8
  }))
}

export function recentTagIndexTags(tags: ForumTag[], limit = 4, nowMs = Date.now(), locale?: string) {
  return activeForumTags(tags)
    .filter((tag) => isCreatedWithinDays(tag.createdAt, 7, nowMs))
    .sort((left, right) => {
      const createdDiff = Date.parse(right.createdAt) - Date.parse(left.createdAt)
      return createdDiff || compareTagsByName(left, right, locale)
    })
    .slice(0, Math.max(0, limit))
}
