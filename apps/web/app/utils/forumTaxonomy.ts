import type { AvatarView } from '~/composables/useProfileApi'

export type ForumTagCreationMode = 'controlled' | 'review' | 'open'
export type ForumTagStatus = 'active' | 'pending' | 'disabled'
export type ForumVisibility = 'public' | 'hidden'
export type ForumDefaultSort = 'latest' | 'hot'
export type ForumTopicStatus = 'active' | 'locked' | 'hidden' | 'deleted' | 'pending' | 'rejected'

export type ForumGuestRead = 'public' | 'login_required'
export type ForumListSort = 'latest' | 'active' | 'hot'
export type ForumDuplicateTitlePolicy = 'off' | 'warn' | 'block'
export type ForumSoftDeleteVisibility = 'author_and_staff' | 'staff_only' | 'hidden'

export type ForumSettings = {
  defaultCategorySlug: string
  tagCreationMode: ForumTagCreationMode
  tagPublicPages: boolean
  tagMinPerTopic: number
  tagMaxPerTopic: number
  topicsPerPage: number
  commentsPerPage: number
  topicTitleMinRunes: number
  topicTitleMaxRunes: number
  topicContentMinRunes: number
  topicContentMaxRunes: number
  topicEditWindowMinutes: number
  topicCooldownSeconds: number
  dailyTopicLimit: number
  commentMinRunes: number
  commentMaxRunes: number
  commentMaxNestingDepth: number
  commentEditWindowMinutes: number
  commentCooldownSeconds: number
  dailyCommentLimit: number
  excerptRuneLimit: number
  guestRead: ForumGuestRead
  listDefaultSort: ForumListSort
  listHotWindowDays: number
  allowAuthorCloseReplies: boolean
  allowAuthorDelete: boolean
  autoLockIdleDays: number
  showTopicEditMark: boolean
  duplicateTitlePolicy: ForumDuplicateTitlePolicy
  showCommentEditMark: boolean
  softDeleteVisibility: ForumSoftDeleteVisibility
  mentionsEnabled: boolean
  mentionsMaxPerPost: number
}

export type ForumCategory = {
  id: number
  groupId: number
  groupSlug: string
  groupName: string
  slug: string
  name: string
  description: string
  icon: string
  iconColor: string
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
  icon: string
  iconColor: string
  status: ForumTagStatus
  topicCount: number
  createdAt: string
  updatedAt: string
}

export type ForumUserSummary = {
  id: number
  username: string
  displayName: string
  avatar: AvatarView
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
  extensionActions?: ForumTopicExtensionAction[]
  /** forum.topic.sidebar 宿主描述符（E2.1） */
  extensionSidebar?: ForumTopicExtensionSidebarItem[]
  /** forum.topic.badges 宿主描述符（E2.1） */
  extensionBadges?: ForumTopicExtensionBadge[]
}

export type ForumTopicExtensionActionMethod = 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export type ForumTopicExtensionAction = {
  extensionId: string
  id: string
  label?: Record<string, string>
  icon?: string
  method: ForumTopicExtensionActionMethod
  url: string
  confirm?: boolean
}

/** forum.comment.actions 宿主描述符（E2.2）；挂在 CommentList 上 */
export type ForumCommentExtensionAction = ForumTopicExtensionAction & {
  /** UX 提示：为 true 时游客隐藏；权威鉴权仍在扩展路由代理 */
  requiresAuth?: boolean
}

/** forum.topic.sidebar 宿主描述符（E2.1） */
export type ForumTopicExtensionSidebarItem = {
  extensionId: string
  id: string
  order: number
  label?: Record<string, string>
  icon?: string
  kind: 'extensionRoute' | 'hostLink'
  method?: string
  url: string
}

/** forum.topic.badges 宿主描述符（E2.1） */
export type ForumTopicExtensionBadgeTone = 'neutral' | 'info' | 'success' | 'warning' | 'danger'

export type ForumTopicExtensionBadge = {
  extensionId: string
  id: string
  order: number
  label?: Record<string, string>
  tone: ForumTopicExtensionBadgeTone
  href?: string
}

/** forum.composer.toolbar 宿主描述符（F4.3） */
export type ForumComposerToolbarAction = {
  extensionId: string
  id: string
  order: number
  label?: Record<string, string>
  icon?: string
  method: ForumTopicExtensionActionMethod
  url: string
  confirm?: boolean
}

export type ForumCommentStatus = 'active' | 'hidden' | 'deleted' | 'pending' | 'rejected'

export type ForumAuthorReviewItem = {
  targetType: 'topic' | 'comment'
  targetId: number
  topicId?: number
  title: string
  excerpt: string
  status: 'pending' | 'rejected'
  reviewNote: string
  createdAt: string
}

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
  /** forum.comment.actions（E2.2）；列表级一次返回，前端挂到每行菜单 */
  extensionActions?: ForumCommentExtensionAction[]
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

export type ForumTopicCreateInput = ForumContentInput & {
  title: string
  categorySlug?: string
  tagSlugs?: string[]
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
  /** forum.topic.list.badges（E2.4）；列表级一次返回，前端挂到每行标题旁 */
  extensionListBadges?: ForumTopicExtensionBadge[]
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
  tagMinPerTopic: 0,
  tagMaxPerTopic: 5,
  topicsPerPage: 20,
  commentsPerPage: 20,
  topicTitleMinRunes: 2,
  topicTitleMaxRunes: 100,
  topicContentMinRunes: 0,
  topicContentMaxRunes: 50000,
  topicEditWindowMinutes: 0,
  topicCooldownSeconds: 0,
  dailyTopicLimit: 0,
  commentMinRunes: 1,
  commentMaxRunes: 10000,
  commentMaxNestingDepth: 5,
  commentEditWindowMinutes: 0,
  commentCooldownSeconds: 0,
  dailyCommentLimit: 0,
  excerptRuneLimit: 180,
  guestRead: 'public',
  listDefaultSort: 'latest',
  listHotWindowDays: 7,
  allowAuthorCloseReplies: true,
  allowAuthorDelete: true,
  autoLockIdleDays: 0,
  showTopicEditMark: true,
  duplicateTitlePolicy: 'warn',
  showCommentEditMark: true,
  softDeleteVisibility: 'author_and_staff',
  mentionsEnabled: true,
  mentionsMaxPerPost: 10
}

export function normalizeForumPageSize(value: number | string | undefined, fallback = 20) {
  const normalized = typeof value === 'number' ? value : Number(value)
  return Number.isInteger(normalized) && normalized >= 1 && normalized <= 100 ? normalized : fallback
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
  return normalizeForumBoundedInt(value, 0, 10, recommendedForumSettings.tagMaxPerTopic)
}

export function normalizeForumBoundedInt(
  value: string | number | undefined,
  min: number,
  max: number,
  fallback: number
) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    return fallback
  }
  const normalized = Math.trunc(parsed)
  return normalized >= min && normalized <= max ? normalized : fallback
}

export function normalizeForumTagMinPerTopic(value: string | number | undefined) {
  return normalizeForumBoundedInt(value, 0, 10, recommendedForumSettings.tagMinPerTopic)
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

// 公开「全部分类」列表页路径。
export function forumCategoriesIndexPath() {
  return '/categories'
}

export function forumTagPath(slug: string) {
  return `/tags/${encodeURIComponent(slug)}`
}

// 公开「全部标签」列表页路径（受 forum.tags.public_pages 控制）。
export function forumTagsIndexPath() {
  return '/tags'
}

// 标签云字号分桶：1（最小）…6（最大）。按 topicCount 在当前集合 min/max 上
// 做 log 归一化，避免写死 demo 字号，也避免单点极值把云挤扁。
export function tagCloudSizeBucket(
  topicCount: number,
  minCount: number,
  maxCount: number
): 1 | 2 | 3 | 4 | 5 | 6 {
  const count = Math.max(0, Number.isFinite(topicCount) ? topicCount : 0)
  const min = Math.max(0, Number.isFinite(minCount) ? minCount : 0)
  const max = Math.max(min, Number.isFinite(maxCount) ? maxCount : min)

  if (max <= min) {
    return 3
  }

  const logMin = Math.log1p(min)
  const logMax = Math.log1p(max)
  const span = logMax - logMin
  if (span <= 0) {
    return 3
  }

  const ratio = (Math.log1p(count) - logMin) / span
  const clamped = Math.min(1, Math.max(0, ratio))
  // 映射到 1..6：均匀 6 档
  const bucket = Math.min(6, Math.max(1, Math.floor(clamped * 6) + 1))
  return bucket as 1 | 2 | 3 | 4 | 5 | 6
}

// 热门阈值：取 topicCount 降序后约前 25% 的下界，至少为 1。
// 用于「热门」chip 与统计格，客户端从当前 active 标签集合计算。
export function tagHotThreshold(topicCounts: number[]): number {
  const counts = topicCounts
    .map((n) => (Number.isFinite(n) ? Math.max(0, Math.trunc(n)) : 0))
    .filter((n) => n > 0)
    .sort((a, b) => b - a)

  if (counts.length === 0) {
    return 1
  }
  if (counts.length === 1) {
    return Math.max(1, counts[0]!)
  }

  const index = Math.min(counts.length - 1, Math.max(0, Math.ceil(counts.length * 0.25) - 1))
  return Math.max(1, counts[index]!)
}

// 近 7 天内创建（本周新增 / 「本周」chip）。无「本周活跃」API 时的合理降级。
export function isCreatedWithinDays(isoDate: string | undefined, days: number, nowMs = Date.now()) {
  if (!isoDate || !Number.isFinite(days) || days <= 0) {
    return false
  }
  const created = Date.parse(isoDate)
  if (Number.isNaN(created)) {
    return false
  }
  return created >= nowMs - days * 24 * 60 * 60 * 1000
}

const forumTagSlugPattern = /^[\p{L}\p{N}]+(?:-[\p{L}\p{N}]+)*$/u

export function normalizeForumTagSlugInput(value: string) {
  return value.trim().toLowerCase()
}

export function isForumTagSlug(value: string) {
  const normalized = normalizeForumTagSlugInput(value)
  return normalized !== '' && forumTagSlugPattern.test(normalized)
}

// 帖子 URL 形态枚举。与 useWebOptions 的 TopicUrlMode 保持一致；
// 此处独立声明以保持 forumTaxonomy 为纯工具模块（不依赖 Nuxt auto-import）。
export type TopicUrlMode = 'id_slug' | 'id' | 'slug'
export type TopicPathLookup =
  | { kind: 'id', topicId: number }
  | { kind: 'slug', slug: string }

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

// 为详情页生成有序查询候选。当前 URL 模式的规范形态优先，同时兼容切换模式前
// 留下的 id、id+slug、slug 三种旧链接，再由详情页统一跳转到当前 canonical。
export function topicPathLookupCandidates(
  segments: string[] | readonly string[] | undefined,
  mode: TopicUrlMode = 'id_slug'
): TopicPathLookup[] {
  const parts = (segments ? [...segments] : []).filter((s) => s !== '')
  if (parts.length === 0) {
    return []
  }

  const candidates: TopicPathLookup[] = []
  const first = parts[0]!
  const topicId = parsePositiveInteger(first)
  const addID = () => {
    if (topicId && !candidates.some((item) => item.kind === 'id' && item.topicId === topicId)) {
      candidates.push({ kind: 'id', topicId })
    }
  }
  const addSlug = (raw: string) => {
    const slug = decodeURIComponent(raw)
    if (slug && !candidates.some((item) => item.kind === 'slug' && item.slug === slug)) {
      candidates.push({ kind: 'slug', slug })
    }
  }

  if (parts.length >= 2) {
    addID()
    return candidates
  }

  if (mode === 'slug') {
    addSlug(first)
    addID()
    return candidates
  }

  addID()
  addSlug(first)
  return candidates
}

function parsePositiveInteger(value: string) {
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 0
}

export function forumUserProfilePath(username: string) {
  return `/u/${encodeURIComponent(username)}`
}

export function forumTopicExtensionLabel(
  item: { label?: Record<string, string> },
  locale = 'zh-CN'
) {
  const labels = item.label || {}
  return labels[locale] || labels['zh-CN'] || labels['en-US'] || Object.values(labels)[0] || ''
}

export function forumTopicExtensionActionLabel(action: ForumTopicExtensionAction, locale = 'zh-CN') {
  return forumTopicExtensionLabel(action, locale) || action.id
}

export function forumTopicExtensionActionRequestPath(action: Pick<ForumTopicExtensionAction, 'url'>) {
  const url = `${action.url || ''}`.trim()
  if (!url.startsWith('/extensions/') || url.includes('://') || url.includes('..') || url.startsWith('/api')) {
    return ''
  }
  return url
}

export function forumTopicExtensionActionRequest(action: ForumTopicExtensionAction, topicId: number) {
  const path = forumTopicExtensionActionRequestPath(action)
  if (!path || !Number.isInteger(topicId) || topicId <= 0 || !isTopicExtensionActionMethod(action.method)) {
    return null
  }
  return {
    path,
    method: action.method,
    body: { topicId }
  }
}

/** 评论行扩展动作：body 带 topicId + commentId，路径校验与主题动作相同 */
export function forumCommentExtensionActionRequest(
  action: ForumCommentExtensionAction,
  topicId: number,
  commentId: number
) {
  const path = forumTopicExtensionActionRequestPath(action)
  if (
    !path
    || !Number.isInteger(topicId)
    || topicId <= 0
    || !Number.isInteger(commentId)
    || commentId <= 0
    || !isTopicExtensionActionMethod(action.method)
  ) {
    return null
  }
  return {
    path,
    method: action.method,
    body: { topicId, commentId }
  }
}

function isTopicExtensionActionMethod(value: string): value is ForumTopicExtensionActionMethod {
  return value === 'POST' || value === 'PUT' || value === 'PATCH' || value === 'DELETE'
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
