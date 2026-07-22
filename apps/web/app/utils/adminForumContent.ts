import type { ForumRenderedContent, ForumTopicTagSummary, ForumUserSummary } from '~/utils/forumTaxonomy'

export type AdminForumContentKind = 'topics' | 'comments'
export type AdminForumContentTargetType = 'topic' | 'comment'

export type AdminForumContentFilters = {
  status?: string
  authorUserID?: number
  authorUsername?: string
  updatedFrom?: string
  updatedTo?: string
  topicID?: number
  titlePrefix?: string
  categorySlug?: string
  perPage?: number
}

export type AdminForumContentRow = {
  targetType: AdminForumContentTargetType
  id: number
  topicId?: number
  topicTitle?: string
  categorySlug?: string
  authorUserId: number
  author?: ForumUserSummary
  status: string
  title?: string
  excerpt: string
  currentRevision: number
  createdAt: string
  updatedAt: string
  tags?: ForumTopicTagSummary[]
}

export type AdminForumContentList = {
  items: AdminForumContentRow[]
  perPage: number
  hasMore: boolean
  nextCursor?: string
}

export type AdminForumTopicDetail = AdminForumContentRow & {
  targetType: 'topic'
  title: string
  categorySlug: string
  tags: ForumTopicTagSummary[]
  content: ForumRenderedContent
  slug: string
}

export type AdminForumCommentDetail = AdminForumContentRow & {
  targetType: 'comment'
  topicId: number
  content: ForumRenderedContent
  rootCommentId: number
  pathKey: string
  depth: number
  replyCount: number
}

export type AdminForumContentDetail = AdminForumTopicDetail | AdminForumCommentDetail

export type ForumRevisionList = {
  items: unknown[]
  hasMore: boolean
  nextCursor?: string
}

export function buildAdminForumContentQuery(filters: AdminForumContentFilters, after?: string) {
  const query: Record<string, string> = {}
  const values: Record<string, string | number | undefined> = {
    after,
    status: filters.status,
    authorUserID: filters.authorUserID,
    authorUsername: filters.authorUsername,
    updatedFrom: filters.updatedFrom,
    updatedTo: filters.updatedTo,
    topicID: filters.topicID,
    titlePrefix: filters.titlePrefix,
    categorySlug: filters.categorySlug,
    perPage: filters.perPage
  }

  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && `${value}`.trim() !== '') query[key] = `${value}`
  }

  return query
}

export function adminForumContentPath(kind: AdminForumContentKind, filters: AdminForumContentFilters, after?: string) {
  const query = new URLSearchParams(buildAdminForumContentQuery(filters, after)).toString()
  const path = `/admin/forum/content/${kind}`
  return query ? `${path}?${query}` : path
}
