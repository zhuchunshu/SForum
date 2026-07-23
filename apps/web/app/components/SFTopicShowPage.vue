<script setup lang="ts">
/**
 * 宿主 body 岛：forum.topic.show。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

import {
  commentFloorLabel,
  forumAuthorName,
  forumCategoryPath,
  forumContentFromEditorPayload,
  forumTagPath,
  advancedReplyDraftStorageKey,
  forumTopicAdvancedReplyPath,
  forumTopicExtensionActionLabel,
  forumTopicPath,
  forumUserProfilePath,
  parseTopicPath,
  topicPathLookupCandidates,
  FORUM_TOPIC_ACTIONS,
  type ForumCategoryGroup,
  type ForumComment,
  type ForumCommentExtensionAction,
  type ForumCommentList,
  type ForumTopicDetail,
  type ForumTopicExtensionAction,
  type TopicPathLookup
} from '~/utils/forumTaxonomy'
import { buildCommentActionMenuItems, buildTopicActionMenuItems } from '~/utils/forumTopicPresentation'


const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const localePath = useLocalePath()
const { seoSettings, webOption } = useWebOptions()
// 当前帖子 URL 形态：决定 catch-all 解析方式与规范化目标。
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const { format: formatSiteDateTime } = useSiteDateTime()
const forumApi = useForumApi()
const { can, canEditTopic, canDeleteTopic } = usePermissions()
const { user: reportUser } = useAuthSession()
const toast = useToast()
const replyActorName = computed(() => reportUser.value?.displayName || reportUser.value?.username || '')

function showSuccessToast(title: string) {
  toast.add({ color: 'success', icon: 'i-lucide-check', title, duration: 10000 })
}

// 顶级回复编辑器状态（始终展开，无折叠态）。
const replyMarkdown = ref('')
const replySubmitting = ref(false)
const replyError = ref('')
const showReplyError = ref(false)
const showReplyEditor = computed(() => Boolean(topic.value && topic.value.status !== 'locked' && can(FORUM_PERMISSIONS.postCreate)))

// 评论编辑/删除状态：同一时刻只允许一个内联编辑器或回复目标。
const editingCommentId = ref<number | null>(null)
const editingMarkdown = ref('')
const editingSubmitting = ref(false)
const editingError = ref('')
const deletingCommentId = ref<number | null>(null)

// catch-all 路由参数解析：按当前 mode 把 /t/<...> 段解析为定位键。
// id_slug → { topicId, slug }；id → { topicId }；slug → { slug }。
// [...path] 恒为数组，但 vue-router 类型宽松，统一归一为 string[]。
const pathSegments = computed<string[]>(() => {
  const raw = route.params.path
  if (Array.isArray(raw)) {
    return raw
  }
  return raw ? [String(raw)] : []
})
const parsedPath = computed(() => parseTopicPath(pathSegments.value, topicUrlMode.value))
const topicLookups = computed(() => topicPathLookupCandidates(pathSegments.value, topicUrlMode.value))
const topicLookupKey = computed(() => topicLookups.value.map((item) => {
  if (item.kind === 'id') {
    return `id:${item.topicId}`
  }
  return `slug:${item.slug}`
}).join('|'))
const topicID = computed(() => {
  const fromPath = parsedPath.value?.topicId
  if (fromPath && fromPath > 0) {
    return fromPath
  }
  const idLookup = topicLookups.value.find((item): item is Extract<TopicPathLookup, { kind: 'id' }> => item.kind === 'id')
  return idLookup?.topicId ?? 0
})
// URL 已带数字 id 时（id / id_slug 模式）可与评论并行拉取；纯 slug 需等详情 resolve。
const urlTopicID = computed(() => {
  const fromPath = topicLookups.value.find((item): item is Extract<TopicPathLookup, { kind: 'id' }> => (
    item.kind === 'id' && item.topicId > 0
  ))
  return fromPath?.topicId ?? 0
})

// 编辑模式：通过 ?edit=1 query 进入（避免 catch-all 嵌套子路由问题）。
// 需登录；未登录时全局 auth 中间件会重定向到登录页。
const isEditing = computed(() => route.query.edit !== undefined && route.query.edit !== null)
const commentPage = computed({
  get: () => parsePublicPage(route.query.page),
  set: (page: number) => {
    const query: Record<string, string> = isEditing.value ? { edit: '1' } : {}
    void router.replace(publicPageLocation(route.path, page, query))
  }
})

// 默认主题只提供连续时间流；回复关系由引用块表达，不再暴露树/平铺切换。
const commentView = ref<'flat'>('flat')
const commentQuery = computed(() => ({
  view: commentView.value,
  page: commentPage.value
}))
function emptyCommentList(): ForumCommentList {
  return { items: [], total: 0, page: 1, perPage: 20, view: commentView.value }
}

async function loadTopicFromCandidates(candidates: TopicPathLookup[]) {
  let lastNotFound: unknown
  for (const candidate of candidates) {
    try {
      // 成功即停：不要在 id 命中后再打 by-slug（否则会重复计浏览）。
      return candidate.kind === 'id'
        ? await forumApi.getTopic(candidate.topicId)
        : await forumApi.getTopicBySlug(candidate.slug)
    } catch (error) {
      if (!isTopicLookupNotFound(error)) {
        throw error
      }
      lastNotFound = error
    }
  }
  if (lastNotFound) {
    throw lastNotFound
  }
  return null
}

function isTopicLookupNotFound(error: unknown) {
  const candidate = error as { statusCode?: unknown, status?: unknown, response?: { status?: unknown } }
  return candidate.statusCode === 404 || candidate.status === 404 || candidate.response?.status === 404
}

// 按 URL 候选顺序加载主题：当前 mode 的规范形态优先，旧的 id/id+slug/slug 链接作为回退。
// D3：一次导航只应成功打一次公开详情 GET（浏览计数副作用在 GET 上；无 POST /view）。
// useAsyncData 在 SSR 与客户端 hydration 间复用 payload，避免 onMounted 二次拉取双计。
// M4：URL 含 id 时 topic+comments Promise.all 并行；纯 slug 时评论等详情 id。
const topicAsync = useAsyncData(
  () => `forum-topic-${topicUrlMode.value}-${topicLookupKey.value}`,
  () => loadTopicFromCandidates(topicLookups.value),
  {
    // 后端对 hidden/deleted 主题返回 404，这里正常抛错由 error 页处理。
    default: () => null as ForumTopicDetail | null
  }
)

// 评论 key：优先 URL id（并行稳定）；slug 模式用 lookup key，resolve 后 watch 刷新。
const commentsKeyTopic = computed(() => (urlTopicID.value > 0 ? String(urlTopicID.value) : `lookup:${topicLookupKey.value}`))
const commentsAsync = useAsyncData(
  () => `forum-topic-comments-${commentsKeyTopic.value}-${commentView.value}-${commentPage.value}`,
  async () => {
    if (urlTopicID.value > 0) {
      return forumApi.listTopicComments(urlTopicID.value, commentQuery.value)
    }
    // 纯 slug：复用同页 topicAsync，避免二次详情 GET（D3）。
    const topicResult = await topicAsync
    const id = topicResult.data.value?.id ?? 0
    if (id <= 0) {
      return emptyCommentList()
    }
    return forumApi.listTopicComments(id, commentQuery.value)
  },
  {
    default: () => emptyCommentList(),
    // SSR 仍等待完整评论；客户端导航先提交正文，再由现有骨架承接评论加载。
    lazy: true,
    // 翻页/视图切换；slug 路径下 topic id 从 0→N 时也会触发。
    watch: [() => topicAsync.data.value?.id ?? 0, commentQuery]
  }
)

// 左栏分类导航（route 模式）：与首页共用 SFHomeNavigation，仅展示 API 目录数据。
const categoryGroupsAsync = useAsyncData(
  'forum-topic-show-category-groups',
  () => forumApi.listCategoryGroups(),
  {
    default: () => [] as ForumCategoryGroup[],
    lazy: true
  }
)

// 初始 SSR 保持正文、评论和导航完整；客户端切页只让正文阻塞导航提交。
const topicResult = await topicAsync
if (import.meta.server) {
  await Promise.all([commentsAsync, categoryGroupsAsync])
}
const { data: topic, error: topicError } = topicResult
const { data: commentData, pending: commentsPending, error: commentsError, refresh: refreshComments } = commentsAsync
const { data: categoryGroups, pending: categoriesPending } = categoryGroupsAsync
const navCategories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const navTotalTopics = computed(() => navCategories.value.reduce((sum, category) => sum + category.topicCount, 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const showTopicSide = computed(() => Boolean(topic.value && !isEditing.value))
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)
const canNormalizeTopicURL = import.meta.client
  || (useRequestHeaders(['accept']).accept || '').includes('text/html')

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

// 规范化：URL 形态/slug 与当前 mode 下的规范路径不符时，301（SSR）/ replace（客户端）。
// 触发场景：模式切换后的旧 URL、slug 变更后的旧 slug、id 模式下多余的 slug 段。
// 编辑态（?edit=1）时保留 query，避免规范化时丢失编辑意图。
watchEffect(() => {
  if (!topic.value || !canNormalizeTopicURL) {
    return
  }
  const targetPath = localePath(forumTopicPath(topic.value, topicUrlMode.value))
  if (targetPath !== route.path) {
    const query: Record<string, string> = isEditing.value ? { edit: '1' } : {}
    const target = publicPageLocation(targetPath, commentPage.value, query)
    if (import.meta.server) {
      navigateTo(target, { redirectCode: 301 })
    } else {
      navigateTo(target, { replace: true })
    }
  }
})

// canonical 用当前 mode 的规范路径（与上面的规范化目标一致）。
const canonicalTopicPath = computed(() => topic.value ? forumTopicPath(topic.value, topicUrlMode.value) : route.path)
const canonicalPath = computed(() => publicPagePath(canonicalTopicPath.value, commentPage.value))

useSForumSeo(computed(() => ({
  type: 'topic',
  path: canonicalPath.value,
  title: topic.value?.title || '',
  excerpt: topic.value?.content.excerpt || t('topicDetail.metaDescription'),
  public: Boolean(topic.value && (topic.value.status === 'active' || topic.value.status === 'locked')),
  published: Boolean(topic.value && !isEditing.value),
  noindex: !topic.value || isEditing.value,
  variables: {
    topicTitle: topic.value?.title,
    categoryName: topic.value?.categoryName,
    authorName: topic.value ? forumAuthorName(topic.value.author, topic.value.authorUserId) : undefined
  },
  datePublished: topic.value?.createdAt,
  dateModified: topic.value?.updatedAt,
  authorName: topic.value ? forumAuthorName(topic.value.author, topic.value.authorUserId) : undefined,
  breadcrumbs: topic.value ? [
    { name: seoSettings.value.seoSiteName, path: '/' },
    { name: topic.value.categoryName, path: forumCategoryPath(topic.value.categorySlug) },
    { name: topic.value.title, path: canonicalPath.value }
  ] : []
})))

// 编辑保存成功后跳回规范详情路径（用新 slug，规范化兜底 -2 后缀）。
async function onTopicSaved(updated: ForumTopicDetail) {
  topic.value = updated
  showSuccessToast(t('topicDetail.topicUpdated'))
  await navigateTo(localePath(forumTopicPath(updated, topicUrlMode.value)))
}

function cancelEditing() {
  // 去掉 ?edit 后回到详情视图。
  navigateTo({ path: route.path, query: { ...route.query, edit: undefined } })
}

function commentPageTo(page: number) { return publicPageLocation(localePath(canonicalTopicPath.value), page) }

// 已加载主题 id（动作/回复路径）；列表拉取优先 urlTopicID 以支持并行。
const loadedTopicID = computed(() => topic.value?.id ?? topicID.value)

const comments = computed(() => commentData.value.items)
const commentTotal = computed(() => commentData.value.total)
const commentTotalPages = computed(() => Math.ceil(commentTotal.value / Math.max(commentData.value.perPage, 1)) || 1)
const commentFloorLabelsById = computed(() => new Map<number, string>(
  comments.value.map((comment, index) => [comment.id, commentFloorLabel(index, commentData.value)])
))
const replyTarget = computed(() => replyingTo.value ? {
  author: commentAuthorName(replyingTo.value),
  href: `#comment-${replyingTo.value.id}`,
  floorLabel: commentFloor(replyingTo.value)
} : null)

// 高级回复：完整编辑器独立页；带上当前回复目标与草稿交接。
const advancedReplyTo = computed(() => {
  if (!topic.value) {
    return ''
  }
  return localePath(forumTopicAdvancedReplyPath(topic.value.id, replyingTo.value?.id))
})

function prepareAdvancedReply() {
  if (!import.meta.client || !topic.value) {
    return
  }
  try {
    sessionStorage.setItem(advancedReplyDraftStorageKey(topic.value.id), replyMarkdown.value)
  } catch {
    // sessionStorage 不可用时仍允许跳转
  }
}
// E2.2：列表级评论扩展动作；requiresAuth 仅 UX 过滤，鉴权在扩展路由代理。
const commentExtensionActions = computed(() => commentData.value?.extensionActions || [])
const commentExtensionActionRunning = ref('')
/** D2：正在通过 ListCommentReplies 补全子孙的评论 id */
const loadingMoreCommentId = ref<number | null>(null)

const authorName = computed(() => topic.value ? forumAuthorName(topic.value.author, topic.value.authorUserId) : '')
const authorPath = computed(() => {
  if (!topic.value?.author?.username) {
    return ''
  }
  return localePath(forumUserProfilePath(topic.value.author.username))
})

function tagPath(slug: string) { return localePath(forumTagPath(slug)) }

function commentFloor(comment: ForumComment) { return commentFloorLabelsById.value.get(comment.id) || '' }

function categoryPath(slug: string) { return localePath(forumCategoryPath(slug)) }

const headingTags = computed(() => (topic.value?.tags || []).map(tag => ({
  id: tag.id,
  name: tag.name,
  to: tagPath(tag.slug)
})))

// 按站点时区与日期时间格式展示（不再硬编码 UTC）。
function formatDate(value: string) { return formatSiteDateTime(value) }

function commentAuthorName(comment: ForumComment) {
  return forumAuthorName(comment.author, comment.authorUserId)
}

function commentAuthorPath(comment: ForumComment) {
  if (!comment.author?.username) {
    return ''
  }
  return localePath(forumUserProfilePath(comment.author.username))
}

function commentMeta(comment: ForumComment) {
  const suffix = comment.edited ? ` · ${t('topicDetail.edited')}` : ''
  return `${formatDate(comment.createdAt)}${suffix}`
}

// 主题生命周期动作。前端仅做 UX 提示，后端 policy 是权威。
type ActionState = 'idle' | 'pending' | 'error'
const actionState = ref<ActionState>('idle')
const actionError = ref('')
const showActionError = ref(false)
const allowAuthorCloseReplies = computed(() => normalizeEnabledOption(
  webOption('forum.topics.allow_author_close_replies', 'enabled'),
  true
))
const canLock = computed(() => Boolean(
  can(FORUM_PERMISSIONS.topicLock) || (
    allowAuthorCloseReplies.value &&
    topic.value?.authorUserId === reportUser.value?.id &&
    can(FORUM_PERMISSIONS.topicEditOwn)
  )
))
const canPin = computed(() => can(FORUM_PERMISSIONS.topicPin))
const canModerate = computed(() => can(FORUM_PERMISSIONS.topicDeleteAny))
const isLocked = computed(() => topic.value?.status === 'locked')
const isPinned = computed(() => Boolean(topic.value?.isPinned))
const extensionActions = computed(() => topic.value?.extensionActions || [])
const extensionActionRunning = ref('')

async function runTopicAction(action: keyof typeof FORUM_TOPIC_ACTIONS, successMessageKey: string) {
  if (!topic.value) {
    return
  }
  actionState.value = 'pending'
  actionError.value = ''
  showActionError.value = false
  try {
    await forumApi.applyTopicAction(topic.value.id, action)
    // 刷新主题以拿到最新状态。
    topic.value = await forumApi.getTopic(topic.value.id)
    showSuccessToast(t(successMessageKey))
  } catch (error) {
    actionState.value = 'error'
    actionError.value = apiErrorMessage(error) || t('topicDetail.actionFailed')
    showActionError.value = true
    return
  }
  actionState.value = 'idle'
}

function topicExtensionActionKey(action: ForumTopicExtensionAction) {
  return `${action.extensionId}:${action.id}`
}

function topicExtensionActionLabel(action: ForumTopicExtensionAction) {
  return forumTopicExtensionActionLabel(action, String(locale.value || 'zh-CN'))
}

async function runTopicExtensionAction(action: ForumTopicExtensionAction) {
  if (!topic.value) {
    return
  }
  const label = topicExtensionActionLabel(action)
  if (action.confirm && !window.confirm(t('topicDetail.confirmExtensionAction', { action: label }))) {
    return
  }
  actionState.value = 'pending'
  actionError.value = ''
  showActionError.value = false
  extensionActionRunning.value = topicExtensionActionKey(action)
  try {
    await forumApi.applyTopicExtensionAction(topic.value.id, action)
    topic.value = await forumApi.getTopic(topic.value.id)
    showSuccessToast(t('topicDetail.extensionActionCompleted'))
  } catch (error) {
    actionState.value = 'error'
    actionError.value = apiErrorMessage(error) || t('topicDetail.extensionActionFailed')
    showActionError.value = true
    return
  } finally {
    extensionActionRunning.value = ''
  }
  actionState.value = 'idle'
}

async function deleteTopic() {
  if (!topic.value) {
    return
  }
  if (!window.confirm(t('topicDetail.confirmDelete'))) {
    return
  }
  actionState.value = 'pending'
  actionError.value = ''
  showActionError.value = false
  try {
    await forumApi.deleteTopic(topic.value.id)
    showSuccessToast(t('topicDetail.topicDeleted'))
    await navigateTo(localePath('/'))
  } catch (error) {
    actionState.value = 'error'
    actionError.value = apiErrorMessage(error) || t('topicDetail.actionFailed')
    showActionError.value = true
  }
}

function commentActions(comment: ForumComment) {
  return buildCommentActionMenuItems({
    canReply: canReplyToComments.value,
    canEdit: isCommentEditable(comment),
    canDelete: isCommentDeletable(comment),
    canReport: canReportComment(),
    labels: {
      reply: t('topicDetail.reply'),
      link: t('topicDetail.commentLink'),
      edit: t('topicDetail.edit'),
      delete: deletingCommentId.value === comment.id ? t('topicDetail.deleting') : t('topicDetail.delete'),
      report: t('topicDetail.report')
    },
    extensions: visibleCommentExtensionActions.value.map(action => ({
      label: forumTopicExtensionActionLabel(action, String(locale.value || 'zh-CN')),
      value: `extension:${action.extensionId}:${action.id}`,
      icon: action.icon
    }))
  })
}

const visibleCommentExtensionActions = computed(() => {
  const loggedIn = Boolean(reportUser.value)
  return commentExtensionActions.value.filter(action => !action.requiresAuth || loggedIn)
})

function commentExtensionActionKey(action: ForumCommentExtensionAction) {
  return `${action.extensionId}:${action.id}`
}

async function runCommentExtensionAction(comment: ForumComment, action: ForumCommentExtensionAction) {
  if (!topic.value) {
    return
  }
  const label = forumTopicExtensionActionLabel(action, String(locale.value || 'zh-CN'))
  if (action.confirm && !window.confirm(t('topicDetail.confirmExtensionAction', { action: label }))) {
    return
  }
  commentExtensionActionRunning.value = commentExtensionActionKey(action)
  try {
    await forumApi.applyCommentExtensionAction(topic.value.id, comment.id, action)
    showSuccessToast(t('topicDetail.extensionActionCompleted'))
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-circle',
      title: apiErrorMessage(error) || t('topicDetail.extensionActionFailed'),
      duration: 0
    })
  } finally {
    commentExtensionActionRunning.value = ''
  }
}

// 评论行内的"举报"按钮（独立于 actions，避免占用回复入口）。
function canReportComment() {
  return Boolean(reportUser.value)
}

// 评论是否可被当前用户编辑/删除。
const { canEditComment, canDeleteComment } = usePermissions()
function isCommentEditable(comment: ForumComment) {
  return canEditComment(comment)
}
function isCommentDeletable(comment: ForumComment) {
  return canDeleteComment(comment)
}

function handleCommentClick(comment: ForumComment, value: string) {
  // 评论操作分发：由 SFComment 的 actions 触发，替代之前硬编码在模板里的按钮。
  if (value === 'reply') {
    startReply(comment)
  } else if (value === 'link') {
    void copyCommentLink(comment)
  } else if (value === 'edit') {
    startEditComment(comment)
  } else if (value === 'delete') {
    deleteComment(comment)
  } else if (value === 'report') {
    openReportDialog({ type: 'comment', id: comment.id })
  } else if (value.startsWith('extension:')) {
    const action = visibleCommentExtensionActions.value.find(
      candidate => value === `extension:${candidate.extensionId}:${candidate.id}`
    )
    if (action) {
      void runCommentExtensionAction(comment, action)
    }
  }
}

// 回复：仅在主题未锁定且当前用户有 post.create 时允许。
const canReplyToComments = computed(() => Boolean(topic.value && topic.value.status !== 'locked' && can(FORUM_PERMISSIONS.postCreate)))

async function handleCommentAction(_value: string) {
  // 回复入口由 Task 3 的内联编辑器处理；这里先预留。
}

// 提交顶级回复。
async function submitReply(payload?: { markdown?: string; native?: unknown; text?: string }) {
  if (!topic.value || replySubmitting.value) {
    return
  }
  const markdown = payload?.markdown ?? replyMarkdown.value
  if (!(payload?.text || markdown).trim()) {
    return
  }
  const content = forumContentFromEditorPayload({
    markdown,
    native: payload?.native,
    text: payload?.text
  })
  replySubmitting.value = true
  replyError.value = ''
  showReplyError.value = false
  try {
    const created = await forumApi.createTopicComment(topic.value.id, content, replyingTo.value?.id)
    replyMarkdown.value = ''
    replyingTo.value = null
    if (created.status === 'pending') {
      toast.add({ color: 'primary', icon: 'i-lucide-clock-3', title: t('topicDetail.replySubmittedForReview'), duration: 10000 })
    } else {
      await refreshComments()
      showSuccessToast(t('topicDetail.replyPosted'))
    }
  } catch (error) {
    replyError.value = apiErrorMessage(error) || t('topicDetail.replyFailed')
    showReplyError.value = true
  } finally {
    replySubmitting.value = false
  }
}

function onReplyEditorSubmit(payload: { markdown: string; native?: unknown; text?: string }) {
  submitReply(payload)
}

// 评论编辑。
function startEditComment(comment: ForumComment) {
  editingCommentId.value = comment.id
  editingMarkdown.value = comment.content.rawContent
  editingError.value = ''
}

function cancelEditComment() {
  editingCommentId.value = null
  editingMarkdown.value = ''
  editingError.value = ''
}

async function saveCommentEdit(comment: ForumComment, payload?: { markdown?: string }) {
  const markdown = payload?.markdown ?? editingMarkdown.value
  if (!markdown.trim() || editingSubmitting.value) {
    return
  }
  editingSubmitting.value = true
  editingError.value = ''
  try {
    await forumApi.updateComment(comment.id, {
      rawContent: markdown,
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      editorVersion: 'sf-editor-v1'
    }, comment.currentRevision)
    cancelEditComment()
    await refreshComments()
    showSuccessToast(t('topicDetail.commentUpdated'))
  } catch (error) {
    editingError.value = apiErrorMessage(error) || t('topicDetail.editFailed')
  } finally {
    editingSubmitting.value = false
  }
}

// 评论删除（软删）。
async function deleteComment(comment: ForumComment) {
  if (deletingCommentId.value) {
    return
  }
  if (!window.confirm(t('topicDetail.confirmCommentDelete'))) {
    return
  }
  deletingCommentId.value = comment.id
  try {
    await forumApi.deleteComment(comment.id)
    await refreshComments()
    showSuccessToast(t('topicDetail.commentDeleted'))
  } catch (error) {
    replyError.value = apiErrorMessage(error) || t('topicDetail.deleteFailed')
    showReplyError.value = true
  } finally {
    deletingCommentId.value = null
  }
}

// 评论回复统一汇入评论流末尾的主编辑器，避免多个内联编辑器打断阅读。
const replyingTo = ref<ForumComment | null>(null)

function startReply(comment: ForumComment) {
  cancelEditComment()
  replyingTo.value = comment
  nextTick(() => scrollToElement('topic-reply-editor'))
}

function cancelReply() {
  replyingTo.value = null
  replyMarkdown.value = ''
  replyError.value = ''
  showReplyError.value = false
}

// D2：树视图截断后，用 ListCommentReplies 合并直系回复到本地树。
function mergeCommentReplies(items: ForumComment[], parentId: number, replies: ForumComment[]): ForumComment[] {
  return items.map((item) => {
    if (item.id === parentId) {
      const existing = new Map((item.children || []).map(child => [child.id, child]))
      const merged: ForumComment[] = []
      for (const reply of replies) {
        const prev = existing.get(reply.id)
        merged.push(prev ? { ...reply, children: prev.children, hasMoreChildren: prev.hasMoreChildren } : reply)
        existing.delete(reply.id)
      }
      // 保留 API 未返回、但本地已有的深层节点（path 序可能被 cap 截断过）。
      for (const leftover of existing.values()) {
        merged.push(leftover)
      }
      return { ...item, children: merged, hasMoreChildren: false }
    }
    if (item.children?.length) {
      return { ...item, children: mergeCommentReplies(item.children, parentId, replies) }
    }
    return item
  })
}

async function loadMoreCommentReplies(comment: ForumComment) {
  if (loadingMoreCommentId.value === comment.id) {
    return
  }
  loadingMoreCommentId.value = comment.id
  try {
    const replies = await forumApi.listCommentReplies(comment.id)
    if (!commentData.value) {
      return
    }
    commentData.value = {
      ...commentData.value,
      items: mergeCommentReplies(commentData.value.items, comment.id, replies)
    }
    showSuccessToast(t('topicDetail.loadMoreReplies'))
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('topicDetail.loadMoreRepliesFailed')
    })
  } finally {
    loadingMoreCommentId.value = null
  }
}

// 评论内联编辑器渲染器：provide 给 SFComment 递归树，让任意层级的评论都能在原位
// 渲染编辑/回复编辑器。用 h() 构造 vnode，替代递归 slot 透传（避免 Volar 类型循环）。
// 顶层评论的编辑器也走这条路径，保证整棵树行为一致。
// key 与 SFComment.vue 内 inject 的字符串保持一致。
const COMMENT_EDITOR_RENDERER_KEY = 'sforum-comment-editor-renderer'
const SFEditorComponent = resolveComponent('LazySFEditor')
const SFButtonComponent = resolveComponent('SFButton')
const commentEditorRenderer = (comment: ForumComment | null) => {
  if (!comment) return null
  const nodes: unknown[] = []
  // 编辑态
  if (editingCommentId.value === comment.id) {
    nodes.push(
      h(SFEditorComponent, {
        modelValue: editingMarkdown.value,
        'onUpdate:modelValue': (v: string) => { editingMarkdown.value = v },
        placeholder: t('topicDetail.editPlaceholder'),
        submitLabel: t('topicDetail.saveEdit'),
        disabled: editingSubmitting.value,
        error: editingError.value,
        onSubmit: () => saveCommentEdit(comment)
      }),
      h('div', { class: 'flex gap-2 mt-2' }, [
        h(SFButtonComponent, {
          variant: 'ghost', size: 'sm', disabled: editingSubmitting.value,
          onClick: cancelEditComment
        }, () => t('topicDetail.cancel'))
      ])
    )
  }
  return nodes.length ? nodes : null
}
provide(COMMENT_EDITOR_RENDERER_KEY, commentEditorRenderer)

// 举报对话框：支持举报主题或评论。同一时刻只展开一个。
const reportingTarget = ref<{ type: 'topic' | 'comment'; id: number } | null>(null)
const reportReason = ref<string>('')
const reportBody = ref('')
const reportSubmitting = ref(false)
const reportError = ref('')
const reportSuccess = ref(false)
const moderationApi = useModerationApi()

const reportReasonOptions = [
  { label: t('moderation.reason.spam'), value: 'spam' },
  { label: t('moderation.reason.abuse'), value: 'abuse' },
  { label: t('moderation.reason.illegal'), value: 'illegal' },
  { label: t('moderation.reason.off_topic'), value: 'off_topic' },
  { label: t('moderation.reason.other'), value: 'other' }
]

const topicActionItems = computed(() => {
  if (!topic.value) {
    return []
  }
  return buildTopicActionMenuItems({
    canEdit: canEditTopic(topic.value),
    canDelete: canDeleteTopic(topic.value),
    canLock: canLock.value,
    canPin: canPin.value,
    canModerate: canModerate.value,
    canReport: Boolean(reportUser.value),
    locked: isLocked.value,
    pinned: isPinned.value,
    hidden: topic.value.status === 'hidden',
    labels: {
      edit: t('topicDetail.edit'),
      delete: t('topicDetail.delete'),
      lock: t('topicDetail.lock'),
      unlock: t('topicDetail.unlock'),
      pin: t('topicDetail.pin'),
      unpin: t('topicDetail.unpin'),
      hide: t('topicDetail.hide'),
      restore: t('topicDetail.restore'),
      report: t('topicDetail.report')
    },
    extensions: extensionActions.value.map(action => ({
      extensionId: action.extensionId,
      id: action.id,
      label: topicExtensionActionLabel(action),
      icon: action.icon,
      confirm: action.confirm
    }))
  })
})

async function handleTopicActionSelect(id: string) {
  if (!topic.value) {
    return
  }
  if (id === 'edit') {
    await navigateTo({ path: localePath(forumTopicPath(topic.value, topicUrlMode.value)), query: { edit: '1' } })
    return
  }
  if (id === 'delete') {
    await deleteTopic()
    return
  }
  if (id === 'report') {
    openReportDialog({ type: 'topic', id: topic.value.id })
    return
  }
  if (id.startsWith('extension:')) {
    const action = extensionActions.value.find(candidate => id === `extension:${candidate.extensionId}:${candidate.id}`)
    if (action) {
      await runTopicExtensionAction(action)
    }
    return
  }
  if (id in FORUM_TOPIC_ACTIONS) {
    const successKey = id === 'lock' || id === 'unlock'
      ? 'topicDetail.lockToggled'
      : id === 'pin' || id === 'unpin'
        ? 'topicDetail.pinToggled'
        : 'topicDetail.hidden'
    await runTopicAction(id as keyof typeof FORUM_TOPIC_ACTIONS, successKey)
  }
}

function scrollToElement(id: string) {
  if (!import.meta.client) {
    return
  }
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function startTopLevelReply() {
  if (!showReplyEditor.value) {
    return
  }
  replyingTo.value = null
  await nextTick()
  scrollToElement('topic-reply-editor')
}

async function copyCommentLink(comment: ForumComment) {
  if (!import.meta.client) {
    return
  }
  const url = `${window.location.origin}${window.location.pathname}${window.location.search}#comment-${comment.id}`
  try {
    await navigator.clipboard.writeText(url)
    showSuccessToast(t('topicDetail.commentLinkCopied'))
  } catch {
    window.prompt(t('topicDetail.copyLinkHint'), url)
  }
}

async function shareTopic() {
  if (!import.meta.client || !topic.value) {
    return
  }
  const url = window.location.href
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
      showSuccessToast(t('topicDetail.linkCopied'))
      return
    }
  } catch {
    // 剪贴板失败时退到 prompt，避免静默失败。
  }
  window.prompt(t('topicDetail.copyLinkHint'), url)
}

function openReportDialog(target: { type: 'topic' | 'comment'; id: number }) {
  if (!reportUser.value) {
    return
  }
  reportingTarget.value = target
  reportReason.value = ''
  reportBody.value = ''
  reportError.value = ''
  reportSuccess.value = false
}

function closeReportDialog() {
  reportingTarget.value = null
}

async function submitReport() {
  if (!reportingTarget.value || !reportReason.value || reportSubmitting.value) {
    return
  }
  reportSubmitting.value = true
  reportError.value = ''
  try {
    await moderationApi.createReport({
      targetType: reportingTarget.value.type,
      targetId: reportingTarget.value.id,
      reasonCode: reportReason.value as 'spam' | 'abuse' | 'illegal' | 'off_topic' | 'other',
      body: reportBody.value
    })
    reportSuccess.value = true
    setTimeout(() => closeReportDialog(), 2000)
  } catch (error) {
    reportError.value = apiErrorMessage(error) || t('moderation.reportFailed')
  } finally {
    reportSubmitting.value = false
  }
}
</script>

<template>
  <main class="sforum-topic-page" data-layout="fullwidth-3col">
    <div
      class="sforum-topic-page__layout"
      :class="{ 'sforum-topic-page__layout--with-side': showTopicSide }"
    >
      <div class="sforum-topic-page__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="navCategories"
          :selected-category-slug="topic?.categorySlug || ''"
          :total-topics="navTotalTopics"
          :pending="categoriesPending"
          :can-create-topic="canCreateTopic"
        />
      </div>

      <div class="sforum-topic-page__main">
        <div class="sforum-topic-page__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="navCategories"
            :selected-category-slug="topic?.categorySlug || ''"
            :total-topics="navTotalTopics"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
          />
        </div>

        <div class="sforum-topic-page__inner">
          <!-- 错误 / 未找到 -->
          <SFCard v-if="topicError && !topic" class="p-10">
            <SFEmptyState
              :title="t('topicDetail.notFound.title')"
              :description="t('topicDetail.notFound.description')"
            />
          </SFCard>

          <!-- 编辑模式：通过 ?edit=1 切入，渲染独立编辑器组件。 -->
          <div v-else-if="topic && isEditing" class="max-w-3xl">
            <h1 class="text-2xl font-bold text-slate-900 mb-6 dark:text-zinc-50">
              {{ t('composer.editTitle') }}
            </h1>
            <SFTopicEditor
              :topic="topic"
              @saved="onTopicSaved"
              @cancel="cancelEditing"
            />
          </div>

          <template v-else-if="topic">
            <div class="sforum-topic-page__shell">
              <div class="sforum-topic-page__reading">
                <article id="topic-start" class="sforum-topic-page__article">
                  <div class="sforum-topic-page__heading-row">
                    <SFTopicHeading
                      :topic="topic"
                      :author-name="authorName"
                      :author-to="authorPath"
                      :category-to="categoryPath(topic.categorySlug)"
                      :tags="headingTags"
                      :published-label="formatDate(topic.createdAt)"
                      :extension-badges="topic.extensionBadges || []"
                    />
                    <SFTopicActionMenu
                      :items="topicActionItems"
                      :pending="actionState === 'pending'"
                      :running-id="extensionActionRunning ? `extension:${extensionActionRunning}` : ''"
                      @select="handleTopicActionSelect"
                    />
                  </div>

                  <div class="sforum-topic-page__post-card">
                    <!-- 正文（后端已 sanitize）；v-highlight 负责代码块语法高亮 -->
                    <div class="sforum-topic-page__prose sf-prose" v-highlight v-html="sanitizeHtml(topic.content.htmlContent)" />

                    <div class="sforum-topic-page__actions">
                      <button type="button" class="sforum-topic-page__action-btn" @click="shareTopic">
                        <UIcon name="i-lucide-share-2" class="size-4" aria-hidden="true" />
                        {{ t('topicDetail.share') }}
                      </button>
                      <button
                        v-if="showReplyEditor"
                        type="button"
                        class="sforum-topic-page__action-btn sforum-topic-page__action-btn--primary"
                        @click="startTopLevelReply"
                      >
                        <UIcon name="i-lucide-reply" class="size-4" aria-hidden="true" />
                        {{ t('topicDetail.replyTopic') }}
                      </button>
                    </div>

                    <SFAlert
                      v-if="showActionError"
                      variant="danger"
                      :title="actionError"
                      closable
                      class="mt-3"
                      @close="showActionError = false"
                    />
                  </div>
                </article>

                <section id="topic-latest" class="sforum-topic-comments">
                  <header class="sf-comment-stream-controls">
                    <div>
                      <span>{{ topic.commentCount }} REPLIES</span>
                      <h2>{{ t('topicDetail.discussionContinues') }}</h2>
                    </div>
                    <a
                      v-if="comments.length"
                      class="sf-comment-stream-controls__latest"
                      :href="`#comment-${comments[comments.length - 1]?.id}`"
                    >
                      <UIcon name="i-lucide-arrow-down" class="size-4" aria-hidden="true" />
                      {{ t('topicDetail.progress.latest') }}
                    </a>
                  </header>

                  <div v-if="commentsError" class="sforum-topic-comments__error">
                    <SFAlert variant="danger" :title="t('topicDetail.commentsLoadFailed')" />
                    <SFButton variant="ghost" size="sm" @click="() => { void refreshComments() }">
                      <UIcon name="i-lucide-refresh-cw" class="size-4" aria-hidden="true" />
                      {{ t('topicDetail.retryComments') }}
                    </SFButton>
                  </div>

                  <template v-if="commentsPending && !comments.length">
                    <div v-for="i in 3" :key="i" class="sforum-topic-comments__skeleton">
                      <SFSkeleton width="20%" height="1rem" class="mb-2" />
                      <SFSkeleton width="90%" class="mb-1" />
                      <SFSkeleton width="70%" />
                    </div>
                  </template>

                  <template v-else-if="comments.length">
                    <div class="sforum-topic-comments__stream sf-comment-list">
                      <SFComment
                        v-for="comment in comments"
                        :key="comment.id"
                        :comment="comment"
                        :author="commentAuthorName(comment)"
                        :avatar="comment.author?.avatar"
                        :author-link="commentAuthorPath(comment)"
                        :html-content="editingCommentId === comment.id ? undefined : comment.content.htmlContent"
                        :content="editingCommentId === comment.id ? '' : undefined"
                        :meta="commentMeta(comment)"
                        :floor-label="commentFloor(comment)"
                        :presentation="commentView"
                        :depth="0"
                        :collapse-from-depth="2"
                        :reply-to="comment.replyTo ? { id: comment.replyTo.id, author: forumAuthorName(comment.replyTo.author, comment.replyTo.id), excerpt: comment.replyTo.excerpt } : undefined"
                        :actions="commentActions(comment)"
                        :comment-meta-builder="commentMeta"
                        :comment-author-link-builder="commentAuthorPath"
                        :comment-actions-builder="commentActions"
                        :loading-more-comment-id="loadingMoreCommentId"
                        @action-comment="(c: ForumComment, value: string) => handleCommentClick(c, value)"
                        @load-more-replies="(c: ForumComment) => { void loadMoreCommentReplies(c) }"
                      />
                    </div>

                    <div v-if="commentTotalPages > 1" class="flex justify-center pt-2">
                      <SFPagination
                        :page="commentPage"
                        :total-pages="commentTotalPages"
                        :page-to="commentPageTo"
                      />
                    </div>
                  </template>

                  <div v-else-if="!commentsError" class="sforum-topic-comments__empty">
                    <SFEmptyState
                      :title="t('topicDetail.emptyComments.title')"
                      :description="t('topicDetail.emptyComments.description')"
                    />
                  </div>

                  <SFTopicReplyComposer
                    v-if="showReplyEditor"
                    v-model="replyMarkdown"
                    :actor-name="replyActorName"
                    :avatar="reportUser?.avatar"
                    :reply-target="replyTarget"
                    :submitting="replySubmitting"
                    :error="showReplyError ? replyError : ''"
                    :advanced-to="advancedReplyTo"
                    @cancel="cancelReply"
                    @submit="onReplyEditorSubmit"
                    @dismiss-error="showReplyError = false"
                    @advanced="prepareAdvancedReply"
                  />

                  <SFAlert
                    v-if="isLocked"
                    variant="warning"
                    :title="t('topicDetail.lockedNotice')"
                    closable
                  />
                </section>
              </div>
            </div>
          </template>
        </div>
      </div>

      <SFTopicSideCard
        v-if="showTopicSide && topic"
        :topic="topic"
        :author-name="authorName"
        :author-to="authorPath"
        :tags="headingTags"
        :category-to="categoryPath(topic.categorySlug)"
        :first-comment-id="comments[0]?.id"
        :latest-comment-id="comments[comments.length - 1]?.id"
        :extension-sidebar="topic.extensionSidebar || []"
      />
    </div>

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('topicDetail.cancel')"
      @click="closeMobileDrawers"
    />

    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.navTitle') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        desktop-only
        navigation-mode="route"
        :categories="navCategories"
        :selected-category-slug="topic?.categorySlug || ''"
        :total-topics="navTotalTopics"
        :pending="categoriesPending"
        :can-create-topic="canCreateTopic"
      />
    </aside>

    <aside v-if="mobileInfoOpen && topic" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('topicDetail.side.title') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFTopicSideCard
        :topic="topic"
        :author-name="authorName"
        :author-to="authorPath"
        :tags="headingTags"
        :category-to="categoryPath(topic.categorySlug)"
        :first-comment-id="comments[0]?.id"
        :latest-comment-id="comments[comments.length - 1]?.id"
        :extension-sidebar="topic.extensionSidebar || []"
      />
    </aside>

    <SFReportDialog
      :open="Boolean(reportingTarget)"
      :reasons="reportReasonOptions"
      :reason="reportReason"
      :body="reportBody"
      :submitting="reportSubmitting"
      :error="reportError"
      :success="reportSuccess"
      @update:reason="reportReason = $event"
      @update:body="reportBody = $event"
      @dismiss-error="reportError = ''"
      @close="closeReportDialog"
      @submit="submitReport"
    />
  </main>
</template>
