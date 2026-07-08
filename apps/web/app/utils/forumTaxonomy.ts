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

// 帖子 URL 形态枚举。与 useWebOptions 的 TopicUrlMode 保持一致；
// 此处独立声明以保持 forumTaxonomy 为纯工具模块（不依赖 Nuxt auto-import）。
export type TopicUrlMode = 'id_slug' | 'id' | 'slug'

// previewTopicSlug 在前端预览标题对应的 slug（与后端 slugify 主逻辑对齐：
// 转小写、非 [a-z0-9-] 字符替换为 -、首尾去 -）。仅用于编辑保存后跳转的预期路径；
// 若后端因全局唯一追加了 -2/-3 后缀，详情页 canonical 会 301 到真实 slug。
export function previewTopicSlug(title: string): string {
  const slug = title.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
  return slug || 'topic'
}

export function forumTopicPath(
  topic: Pick<ForumTopicSummary, 'id' | 'slug'>,
  mode: TopicUrlMode = 'id_slug'
) {
  switch (mode) {
    case 'id':
      return `/t/${topic.id}`
    case 'slug':
      return `/t/${encodeURIComponent(topic.slug)}`
    default:
      return `/t/${topic.id}/${encodeURIComponent(topic.slug)}`
  }
}

// parseTopicPath 解析 catch-all 详情页路由参数为帖子定位键。
// 返回值：id 模式下为 { topicId }；slug 模式下为 { slug }；
// id_slug 模式下为 { topicId, slug }。无法识别时返回 null（调用方应 404）。
export function parseTopicPath(
  segments: string[] | readonly string[] | undefined,
  mode: TopicUrlMode = 'id_slug'
): { topicId?: number, slug?: string } | null {
  const parts = (segments ? [...segments] : []).filter((s) => s !== '')
  if (parts.length === 0) {
    return null
  }
  const first = parts[0]!
  if (mode === 'id') {
    const topicId = Number(first)
    return Number.isInteger(topicId) && topicId > 0 ? { topicId } : null
  }
  if (mode === 'slug') {
    return { slug: decodeURIComponent(first) }
  }
  // id_slug：期望 [id, slug]
  const topicId = Number(first)
  if (!Number.isInteger(topicId) || topicId <= 0) {
    return null
  }
  const rest = parts.slice(1).join('/')
  return { topicId, slug: rest ? decodeURIComponent(rest) : '' }
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

// 扁平化评论树：把 tree 视图返回的嵌套 children 拍平成一维列表。
// 扁平列表布局（无缩进无树）：每条评论平铺，回复对象靠各自的 replyTo 引用块表达。
//
// 规则：
// - 根评论（depth=0）照常保留；非根评论如果后端没给 replyTo，
//   用其父评论补一个 replyTo（人名 + 摘要），保证每条回复都能看出"回复的是谁"。
// - 拍平顺序按 children 原始顺序深度优先（保留讨论时间线）。
export function flattenCommentTree(roots: ForumComment[]): ForumComment[] {
  const result: ForumComment[] = []
  const walk = (list: ForumComment[], parent: ForumComment | null) => {
    for (const c of list) {
      // 非根评论且后端未填充 replyTo：用父评论补上引用信息。
      if (parent && !c.replyTo) {
        const parentName = parent.author?.displayName
          || parent.author?.username
          || `#${parent.authorUserId}`
        result.push({
          ...c,
          replyTo: {
            id: parent.id,
            author: parent.author,
            excerpt: parent.content.excerpt,
            depth: parent.depth
          }
        })
      } else {
        result.push(c)
      }
      if (c.children && c.children.length > 0) {
        walk(c.children, c)
      }
    }
  }
  walk(roots, null)
  return result
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
