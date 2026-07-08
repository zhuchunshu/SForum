export type ForumTagCreationMode = 'controlled' | 'review' | 'open'
export type ForumTagStatus = 'active' | 'pending' | 'disabled'
export type ForumVisibility = 'public' | 'hidden'
export type ForumDefaultSort = 'latest' | 'hot'
export type ForumTopicStatus = 'active' | 'locked' | 'hidden' | 'deleted'

export type ForumSettings = {
  defaultCategorySlug: string
  tagCreationMode: ForumTagCreationMode
  tagPublicPages: boolean
  tagMaxPerTopic: number
}

export type ForumCategory = {
  id: number
  groupId: number
  groupSlug: string
  groupName: string
  slug: string
  name: string
  description: string
  visibility: ForumVisibility
  position: number
  defaultSort: ForumDefaultSort
  topicCount: number
  commentCount: number
  createdAt: string
  updatedAt: string
}

export type ForumCategoryGroup = {
  id: number
  slug: string
  name: string
  description: string
  visibility: ForumVisibility
  position: number
  categories: ForumCategory[]
  createdAt: string
  updatedAt: string
}

export type ForumTag = {
  id: number
  slug: string
  name: string
  description: string
  status: ForumTagStatus
  topicCount: number
  createdAt: string
  updatedAt: string
}

export type ForumUserSummary = {
  id: number
  username: string
  displayName: string
}

export type ForumTopicTagSummary = {
  id: number
  slug: string
  name: string
  status: ForumTagStatus
}

export type ForumTopicSummary = {
  id: number
  categoryId: number
  categorySlug: string
  categoryName: string
  authorUserId: number
  author?: ForumUserSummary
  title: string
  slug: string
  status: ForumTopicStatus
  isPinned: boolean
  commentCount: number
  viewCount: number
  tags?: ForumTopicTagSummary[]
  excerpt: string
  createdAt: string
  updatedAt: string
  lastActivityAt: string
}

export type ForumRenderedContent = {
  id: number
  rawContent: string
  htmlContent: string
  plainText: string
  excerpt: string
  sourceFormat: 'markdown' | 'html' | 'json'
  editorType: string
  editorVersion?: string
  renderVersion: string
  contentHash: string
}

export type ForumTopicDetail = ForumTopicSummary & {
  content: ForumRenderedContent
}

export type ForumCommentStatus = 'active' | 'hidden' | 'deleted'

export type ForumReplyReference = {
  id: number
  author?: ForumUserSummary
  excerpt: string
  depth: number
}

// 评论节点。tree 视图下 children 为嵌套回复；flat 视图下 children 为空。
export type ForumComment = {
  id: number
  topicId: number
  authorUserId: number
  author?: ForumUserSummary
  parentId?: number | null
  rootCommentId: number
  pathKey: string
  depth: number
  replyCount: number
  status: ForumCommentStatus
  content: ForumRenderedContent
  replyTo?: ForumReplyReference
  children?: ForumComment[]
  createdAt: string
  updatedAt: string
}

export type ForumCommentList = {
  items: ForumComment[]
  total: number
  page: number
  perPage: number
  view: 'tree' | 'flat'
}

// 主题生命周期动作结果。
export type ForumTopicAction = {
  topicId: number
  status: ForumTopicStatus
  isPinned: boolean
}

// 编辑器提交给后端的正文输入。rawContent 为 markdown 源文本。
export type ForumContentInput = {
  rawContent: string
  sourceFormat?: 'markdown' | 'html'
  editorType?: string
  editorVersion?: string
}

// 更新主题输入，所有字段可选；未提供即不修改。
export type ForumTopicUpdateInput = {
  categorySlug?: string
  title?: string
  tagSlugs?: string[]
  content?: ForumContentInput
}

export type ForumCommentListView = 'tree' | 'flat'

export type ForumCommentListQuery = {
  view?: ForumCommentListView
  page?: number
  perPage?: number
}

// 主题生命周期动作枚举，与后端 TopicAction 常量保持一致。
export const FORUM_TOPIC_ACTIONS = {
  hide: 'hide',
  restore: 'restore',
  lock: 'lock',
  unlock: 'unlock',
  pin: 'pin',
  unpin: 'unpin'
} as const

export type ForumTopicActionKey = keyof typeof FORUM_TOPIC_ACTIONS

export type ForumTopicList = {
  items: ForumTopicSummary[]
  total: number
  page: number
  perPage: number
}

export type ForumTopicFilters = {
  categorySlug?: string
  tagSlug?: string
  query?: string
  page?: number
  perPage?: number
}

// 搜索输入：query 为关键词，必填；其余为可选过滤与分页。
export type ForumTopicSearchFilters = {
  query: string
  categorySlug?: string
  tagSlug?: string
  page?: number
  perPage?: number
}

export const recommendedForumSettings: ForumSettings = {
  defaultCategorySlug: 'general',
  tagCreationMode: 'controlled',
  tagPublicPages: true,
  tagMaxPerTopic: 5
}

export function parseForumTagPublicPagesOption(value: string | undefined, fallback = true) {
  switch (value?.trim().toLowerCase()) {
    case 'enabled':
    case 'true':
    case '1':
    case 'yes':
    case 'on':
      return true
    case 'disabled':
    case 'false':
    case '0':
    case 'no':
    case 'off':
      return false
    default:
      return fallback
  }
}

export function normalizeForumTagCreationMode(value: string | undefined): ForumTagCreationMode {
  const normalized = value?.trim().toLowerCase()
  return normalized === 'review' || normalized === 'open' ? normalized : recommendedForumSettings.tagCreationMode
}

export function normalizeForumTagMaxPerTopic(value: string | number | undefined) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    return recommendedForumSettings.tagMaxPerTopic
  }

  const normalized = Math.trunc(parsed)
  return normalized >= 0 && normalized <= 10 ? normalized : recommendedForumSettings.tagMaxPerTopic
}

export function buildForumTopicQuery(filters: ForumTopicFilters = {}) {
  const query: Record<string, string> = {}
  addStringQuery(query, 'categorySlug', filters.categorySlug)
  addStringQuery(query, 'tagSlug', filters.tagSlug)
  addStringQuery(query, 'query', filters.query)
  addPositiveNumberQuery(query, 'page', filters.page)
  addPositiveNumberQuery(query, 'perPage', filters.perPage)
  return query
}

// 构造搜索端点（GET /search）的 query。不复用 buildForumTopicQuery，
// 因 search 要求 query 必填，且语义独立于列表。
export function buildForumSearchQuery(filters: ForumTopicSearchFilters) {
  const query: Record<string, string> = {}
  addStringQuery(query, 'query', filters.query)
  addStringQuery(query, 'categorySlug', filters.categorySlug)
  addStringQuery(query, 'tagSlug', filters.tagSlug)
  addPositiveNumberQuery(query, 'page', filters.page)
  addPositiveNumberQuery(query, 'perPage', filters.perPage)
  return query
}

export function forumCategoryPath(slug: string) {
  return `/c/${encodeURIComponent(slug)}`
}

export function forumTagPath(slug: string) {
  return `/tags/${encodeURIComponent(slug)}`
}

export function forumTopicPath(topic: Pick<ForumTopicSummary, 'id' | 'slug'>) {
  return `/t/${topic.id}/${encodeURIComponent(topic.slug)}`
}

export function forumUserProfilePath(username: string) {
  return `/u/${encodeURIComponent(username)}`
}

// 构建评论列表查询参数，仅写入非空字段。
export function buildForumCommentQuery(query: ForumCommentListQuery = {}) {
  const params: Record<string, string> = {}
  if (query.view === 'tree' || query.view === 'flat') {
    params.view = query.view
  }
  addPositiveNumberQuery(params, 'page', query.page)
  addPositiveNumberQuery(params, 'perPage', query.perPage)
  return params
}

// 统一取作者展示名：优先 displayName，其次 username，最后回退用户 ID。
export function forumAuthorName(user: ForumUserSummary | undefined, fallbackUserId: number) {
  if (user) {
    return user.displayName || user.username || `#${fallbackUserId}`
  }
  return `#${fallbackUserId}`
}

// 递归统计一条评论的所有后代总数（含各层级），用于折叠按钮文案"展开 N 条回复"。
// 不含评论自身。无 children 或空 children 返回 0。
export function countCommentDescendants(comment: ForumComment | null | undefined): number {
  if (!comment || !comment.children || comment.children.length === 0) {
    return 0
  }
  let total = 0
  for (const child of comment.children) {
    total += 1 + countCommentDescendants(child)
  }
  return total
}

// 判断一条评论是否应默认折叠其子评论。
// 规则：直接子评论数 ≥ threshold 时折叠（任意层级）。threshold 默认 4。
// 用于评论树渲染时的初始折叠状态，避免热门分支刷屏。
export function shouldCollapseByDefault(
  comment: ForumComment | null | undefined,
  threshold = 4
): boolean {
  if (!comment || !comment.children) {
    return false
  }
  return comment.children.length >= threshold
}

function addStringQuery(query: Record<string, string>, key: string, value: string | undefined) {
  const normalized = value?.trim()
  if (normalized) {
    query[key] = normalized
  }
}

function addPositiveNumberQuery(query: Record<string, string>, key: string, value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return
  }

  const normalized = Math.trunc(value)
  if (normalized > 0) {
    query[key] = String(normalized)
  }
}
