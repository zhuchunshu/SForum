<script setup lang="ts">
import {
  forumAuthorName,
  forumCategoryPath,
  forumTagPath,
  forumTopicExtensionActionLabel,
  forumTopicPath,
  forumUserProfilePath,
  parseTopicPath,
  topicPathLookupCandidates,
  FORUM_TOPIC_ACTIONS,
  type ForumComment,
  type ForumCommentList,
  type ForumTopicDetail,
  type ForumTopicExtensionAction,
  type TopicPathLookup
} from '~/utils/forumTaxonomy'
import { buildTopicActionMenuItems } from '~/utils/forumTopicPresentation'

definePageMeta({
  // 主题详情对所有人可见（公开读限定 active/locked）。
  public: true
})

const route = useRoute()
const { t, locale } = useI18n()
const localePath = useLocalePath()
const { seoSettings } = useWebOptions()
// 当前帖子 URL 形态：决定 catch-all 解析方式与规范化目标。
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()
const { can, canEditTopic, canDeleteTopic } = usePermissions()
const toast = useToast()

function showSuccessToast(title: string) {
  toast.add({
    color: 'success',
    icon: 'i-lucide-check',
    title,
    duration: 10000
  })
}

// 顶级回复编辑器状态。
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
const topicLookupKey = computed(() => topicLookups.value.map((item) => item.kind === 'id' ? `id:${item.topicId}` : `slug:${item.slug}`).join('|'))
const topicID = computed(() => parsedPath.value?.topicId ?? topicLookups.value.find((item) => item.kind === 'id')?.topicId ?? 0)

// 按 URL 候选顺序加载主题：当前 mode 的规范形态优先，旧的 id/id+slug/slug 链接作为回退。
const { data: topic, error: topicError } = await useAsyncData(
  () => `forum-topic-${topicUrlMode.value}-${topicLookupKey.value}`,
  () => loadTopicFromCandidates(topicLookups.value),
  {
    // 后端对 hidden/deleted 主题返回 404，这里正常抛错由 error 页处理。
    default: () => null as ForumTopicDetail | null
  }
)

async function loadTopicFromCandidates(candidates: TopicPathLookup[]) {
  let lastNotFound: unknown
  for (const candidate of candidates) {
    try {
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

// 编辑模式：通过 ?edit=1 query 进入（避免 catch-all 嵌套子路由问题）。
// 需登录；未登录时全局 auth 中间件会重定向到登录页。
const isEditing = computed(() => route.query.edit !== undefined && route.query.edit !== null)

// 规范化：URL 形态/slug 与当前 mode 下的规范路径不符时，301（SSR）/ replace（客户端）。
// 触发场景：模式切换后的旧 URL、slug 变更后的旧 slug、id 模式下多余的 slug 段。
// 编辑态（?edit=1）时保留 query，避免规范化时丢失编辑意图。
watchEffect(() => {
  if (!topic.value) {
    return
  }
  const targetPath = localePath(forumTopicPath(topic.value, topicUrlMode.value))
  if (targetPath !== route.path) {
    const target = isEditing.value ? { path: targetPath, query: { edit: '1' } } : targetPath
    if (import.meta.server) {
      navigateTo(target, { redirectCode: 301 })
    } else {
      navigateTo(target, { replace: true })
    }
  }
})

// canonical 用当前 mode 的规范路径（与上面的规范化目标一致）。
const canonicalPath = computed(() => topic.value ? forumTopicPath(topic.value, topicUrlMode.value) : (route.path))

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
  authorName: topic.value ? forumAuthorName(topic.value.author, topic.value.authorUserId) : undefined
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

// 评论数据：默认 tree 视图。
const commentPage = ref(1)
const commentView = ref<'tree' | 'flat'>('tree')
const commentQuery = computed(() => ({
  view: commentView.value,
  page: commentPage.value
}))
watch(commentView, () => {
  commentPage.value = 1
}, { flush: 'sync' })

// 评论查询基于已加载主题的真实 id（slug 模式下 topicID 可能为 0，必须用 topic.value.id）。
const loadedTopicID = computed(() => topic.value?.id ?? topicID.value)
const { data: commentData, pending: commentsPending, error: commentsError, refresh: refreshComments } = await useAsyncData(
  () => `forum-topic-comments-${loadedTopicID.value}-${commentView.value}-${commentPage.value}`,
  () => forumApi.listTopicComments(loadedTopicID.value, commentQuery.value),
  {
    default: () => ({ items: [], total: 0, page: 1, perPage: 20, view: commentView.value }) as ForumCommentList,
    // topic 加载完成（id 变化）或翻页/视图切换时重新拉取。
    watch: [() => loadedTopicID.value, commentQuery]
  }
)

const comments = computed(() => commentData.value.items)
const commentTotal = computed(() => commentData.value.total)
const commentTotalPages = computed(() => Math.ceil(commentTotal.value / Math.max(commentData.value.perPage, 1)) || 1)

const authorName = computed(() => topic.value ? forumAuthorName(topic.value.author, topic.value.authorUserId) : '')
const authorPath = computed(() => {
  if (!topic.value?.author?.username) {
    return ''
  }
  return localePath(forumUserProfilePath(topic.value.author.username))
})

function tagPath(slug: string) {
  return localePath(forumTagPath(slug))
}

function categoryPath(slug: string) {
  return localePath(forumCategoryPath(slug))
}

const headingTags = computed(() => (topic.value?.tags || []).map(tag => ({
  id: tag.id,
  name: tag.name,
  to: tagPath(tag.slug)
})))

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  return new Intl.DateTimeFormat(String(locale.value || 'zh-CN'), {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'UTC'
  }).format(date)
}

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
  const updated = new Date(comment.updatedAt).getTime()
  const created = new Date(comment.createdAt).getTime()
  const suffix = updated > created ? ` · ${t('topicDetail.edited')}` : ''
  return `${formatDate(comment.createdAt)}${suffix}`
}

// 主题生命周期动作。前端仅做 UX 提示，后端 policy 是权威。
type ActionState = 'idle' | 'pending' | 'error'
const actionState = ref<ActionState>('idle')
const actionError = ref('')
const showActionError = ref(false)
const canLock = computed(() => can(FORUM_PERMISSIONS.topicLock))
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
  // 操作按钮内聚进 SFComment 的 actions，替代之前硬编码在模板里的按钮。
  // 颜色由 .sf-comment__action 统一用 --sf-* token 控制（之前硬编码 teal-300/slate-500 违反主题规范）。
  const actions: { label: string; value: string; icon?: string }[] = []
  if (canReplyToComments.value) {
    actions.push({ label: t('topicDetail.reply'), value: 'reply', icon: 'i-lucide-reply' })
  }
  if (isCommentEditable(comment)) {
    actions.push({ label: t('topicDetail.edit'), value: 'edit', icon: 'i-lucide-pencil' })
  }
  if (isCommentDeletable(comment)) {
    actions.push({
      label: deletingCommentId.value === comment.id ? t('topicDetail.deleting') : t('topicDetail.delete'),
      value: 'delete',
      icon: 'i-lucide-trash-2'
    })
  }
  if (canReportComment()) {
    actions.push({ label: t('topicDetail.report'), value: 'report', icon: 'i-lucide-flag' })
  }
  return actions
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
  } else if (value === 'edit') {
    startEditComment(comment)
  } else if (value === 'delete') {
    deleteComment(comment)
  } else if (value === 'report') {
    openReportDialog({ type: 'comment', id: comment.id })
  }
}

// 回复：仅在主题未锁定且当前用户有 post.create 时允许。
const canReplyToComments = computed(() => Boolean(topic.value && topic.value.status !== 'locked' && can(FORUM_PERMISSIONS.postCreate)))

async function handleCommentAction(_value: string) {
  // 回复入口由 Task 3 的内联编辑器处理；这里先预留。
}

// 提交顶级回复。
async function submitReply(payload?: { markdown?: string }) {
  if (!topic.value || replySubmitting.value) {
    return
  }
  const markdown = payload?.markdown ?? replyMarkdown.value
  if (!markdown.trim()) {
    return
  }
  replySubmitting.value = true
  replyError.value = ''
  showReplyError.value = false
  try {
    const created = await forumApi.createTopicComment(topic.value.id, {
      rawContent: markdown,
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      editorVersion: 'sf-editor-v1'
    })
    replyMarkdown.value = ''
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

function onReplyEditorSubmit(payload: { markdown: string }) {
  submitReply({ markdown: payload.markdown })
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
    })
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

// 内联回复目标：点击评论的"回复"后展开一个编辑器，提交时带 parentId。
const replyingTo = ref<ForumComment | null>(null)
const nestedReplyMarkdown = ref('')
const nestedReplySubmitting = ref(false)

function startReply(comment: ForumComment) {
  // 同一时刻只展开一个回复编辑器。
  cancelEditComment()
  replyingTo.value = comment
  nestedReplyMarkdown.value = ''
}

function cancelReply() {
  replyingTo.value = null
  nestedReplyMarkdown.value = ''
}

async function submitNestedReply(comment: ForumComment, payload?: { markdown?: string }) {
  if (!topic.value || nestedReplySubmitting.value) {
    return
  }
  const markdown = payload?.markdown ?? nestedReplyMarkdown.value
  if (!markdown.trim()) {
    return
  }
  nestedReplySubmitting.value = true
  try {
    const created = await forumApi.createTopicComment(topic.value.id, {
      rawContent: markdown,
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      editorVersion: 'sf-editor-v1'
    }, comment.id)
    cancelReply()
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
    nestedReplySubmitting.value = false
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
  // 回复态
  if (replyingTo.value && replyingTo.value.id === comment.id) {
    nodes.push(
      h(SFEditorComponent, {
        modelValue: nestedReplyMarkdown.value,
        'onUpdate:modelValue': (v: string) => { nestedReplyMarkdown.value = v },
        placeholder: t('topicDetail.replyPlaceholder'),
        submitLabel: t('topicDetail.submitReply'),
        disabled: nestedReplySubmitting.value,
        onSubmit: () => submitNestedReply(comment)
      }),
      h('div', { class: 'flex gap-2 mt-2' }, [
        h(SFButtonComponent, {
          variant: 'ghost', size: 'sm', disabled: nestedReplySubmitting.value,
          onClick: cancelReply
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
const { user: reportUser } = useAuthSession()

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

function startTopLevelReply() {
  if (!showReplyEditor.value) {
    return
  }
  scrollToElement('topic-reply-editor')
}

async function jumpToLatest() {
  commentPage.value = commentTotalPages.value
  await nextTick()
  scrollToElement('topic-latest')
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
  <main class="sforum-topic-page">
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
            />
            <SFTopicActionMenu
              :items="topicActionItems"
              :pending="actionState === 'pending'"
              :running-id="extensionActionRunning ? `extension:${extensionActionRunning}` : ''"
              @select="handleTopicActionSelect"
            />
          </div>

          <!-- 正文（后端已 sanitize）:sf-prose 由 @tailwindcss/typography 提供；v-highlight 负责代码块语法高亮 -->
          <div class="sforum-topic-page__prose sf-prose" v-highlight v-html="sanitizeHtml(topic.content.htmlContent)" />

          <!-- 动作错误（不自动消失） -->
          <SFAlert
            v-if="showActionError"
            variant="danger"
            :title="actionError"
            closable
            class="mt-3"
            @close="showActionError = false"
          />
        </article>

        <button
          v-if="showReplyEditor"
          type="button"
          class="sforum-topic-page__mobile-reply"
          @click="startTopLevelReply"
        >
          <UIcon name="i-lucide-reply" class="size-4" aria-hidden="true" />
          {{ t('topicDetail.reply') }}
        </button>

        <!-- 评论区域 -->
        <section id="topic-latest" class="sforum-topic-comments">
          <SFCommentStreamControls v-model="commentView" :count="topic.commentCount" />

          <div v-if="commentsError" class="sforum-topic-comments__error">
            <SFAlert variant="danger" :title="t('topicDetail.commentsLoadFailed')" />
            <SFButton variant="ghost" size="sm" @click="refreshComments">
              <UIcon name="i-lucide-refresh-cw" class="size-4" aria-hidden="true" />
              {{ t('topicDetail.retryComments') }}
            </SFButton>
          </div>

          <!-- 评论加载骨架 -->
          <template v-if="commentsPending && !comments.length">
            <div v-for="i in 3" :key="i" class="sforum-topic-comments__skeleton">
              <SFSkeleton width="20%" height="1rem" class="mb-2" />
              <SFSkeleton width="90%" class="mb-1" />
              <SFSkeleton width="70%" />
            </div>
          </template>

          <!-- 评论列表 -->
          <template v-else-if="comments.length">
            <div class="sforum-topic-comments__stream sf-comment-list">
              <!-- 递归评论树：SFComment 内部自递归渲染 children（含任意深度 + 折叠）。
                   操作按钮（回复/编辑/删除/举报）通过 commentActions 动态生成，颜色走 --sf-* token。
                   内联编辑器/回复编辑器由本页 provide 的 commentEditorRenderer 在评论原位渲染（任意层级）。 -->
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
                :presentation="commentView"
                :depth="0"
                :collapse-from-depth="2"
                :reply-to="comment.replyTo ? { author: forumAuthorName(comment.replyTo.author, comment.replyTo.id), excerpt: comment.replyTo.excerpt } : undefined"
                :actions="commentActions(comment)"
                :comment-meta-builder="commentMeta"
                :comment-author-link-builder="commentAuthorPath"
                :comment-actions-builder="commentActions"
                @action-comment="(c: ForumComment, value: string) => handleCommentClick(c, value)"
              />
            </div>

            <!-- 分页 -->
            <div v-if="commentTotalPages > 1" class="flex justify-center pt-2">
              <SFPagination v-model:page="commentPage" :total-pages="commentTotalPages" />
            </div>
          </template>

          <!-- 空评论 -->
          <div v-else-if="!commentsError" class="sforum-topic-comments__empty">
            <SFEmptyState
              :title="t('topicDetail.emptyComments.title')"
              :description="t('topicDetail.emptyComments.description')"
            />
          </div>

          <!-- 顶级回复编辑器 -->
          <section v-if="showReplyEditor" id="topic-reply-editor" class="sforum-topic-comments__reply">
            <h3>
              {{ t('topicDetail.replyTitle') }}
            </h3>
            <LazySFEditor
              v-model="replyMarkdown"
              :placeholder="t('topicDetail.replyPlaceholder')"
              :submit-label="replySubmitting ? t('topicDetail.submitting') : t('topicDetail.submitReply')"
              :disabled="replySubmitting"
              @submit="onReplyEditorSubmit"
            />
            <SFAlert
              v-if="showReplyError"
              variant="danger"
              :title="replyError"
              closable
              class="mt-3"
              @close="showReplyError = false"
            />
          </section>

          <!-- 锁定提示 -->
          <SFAlert
            v-if="isLocked"
            variant="warning"
            :title="t('topicDetail.lockedNotice')"
            closable
          />
        </section>
          </div>

          <SFTopicProgressRail
            :current-page="commentPage"
            :total-pages="commentTotalPages"
            :total-posts="topic.commentCount + 1"
            :first-label="formatDate(topic.createdAt)"
            :latest-label="formatDate(topic.lastActivityAt || topic.updatedAt)"
            :can-reply="showReplyEditor"
            :locked="isLocked"
            :pending="replySubmitting"
            @reply="startTopLevelReply"
            @first="scrollToElement('topic-start')"
            @latest="jumpToLatest"
          />
        </div>
      </template>
    </div>

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
