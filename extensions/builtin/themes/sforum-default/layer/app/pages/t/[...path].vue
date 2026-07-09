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

definePageMeta({
  // 主题详情对所有人可见（公开读限定 active/locked）。
  public: true
})

const route = useRoute()
const { t, locale } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
// 当前帖子 URL 形态：决定 catch-all 解析方式与规范化目标。
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()
const { can, canEditTopic, canDeleteTopic } = usePermissions()

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

useSForumSeo({
  title: () => {
    if (!topic.value) {
      return siteName.value
    }
    return isEditing.value ? `${t('composer.editTitle')} - ${siteName.value}` : `${topic.value.title} - ${siteName.value}`
  },
  description: () => topic.value?.content.excerpt || t('topicDetail.metaDescription'),
  type: 'article',
  path: () => canonicalPath.value,
  // 编辑态不索引；主题未加载也不索引。
  noindex: () => !topic.value || isEditing.value,
  schema: () => (topic.value && !isEditing.value) ? {
    type: 'DiscussionForumPosting',
    datePublished: topic.value.createdAt,
    dateModified: topic.value.updatedAt,
    authorName: forumAuthorName(topic.value.author, topic.value.authorUserId)
  } : undefined
})

// 编辑保存成功后跳回规范详情路径（用新 slug，规范化兜底 -2 后缀）。
async function onTopicSaved(updated: ForumTopicDetail) {
  topic.value = updated
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
  page: commentPage.value,
  perPage: 20
}))

// 评论查询基于已加载主题的真实 id（slug 模式下 topicID 可能为 0，必须用 topic.value.id）。
const loadedTopicID = computed(() => topic.value?.id ?? topicID.value)
const { data: commentData, pending: commentsPending, refresh: refreshComments } = await useAsyncData(
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

// 主题状态标签。
type TopicBadge = { label: string; variant: 'neutral' | 'primary' | 'info' | 'success' | 'warning' | 'danger' }
function topicBadges(): TopicBadge[] {
  if (!topic.value) {
    return []
  }
  const badges: TopicBadge[] = []
  if (topic.value.isPinned) {
    badges.push({ label: t('topicDetail.badge.pinned'), variant: 'danger' })
  }
  badges.push({ label: topic.value.categoryName, variant: 'primary' })
  if (topic.value.status === 'locked') {
    badges.push({ label: t('topicDetail.badge.locked'), variant: 'warning' })
  }
  return badges
}

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

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  return date.toLocaleString()
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
  try {
    await forumApi.deleteTopic(topic.value.id)
    await navigateTo(localePath('/'))
  } catch (error) {
    actionState.value = 'error'
    actionError.value = apiErrorMessage(error) || t('topicDetail.actionFailed')
    showActionError.value = true
  }
}

// 自动关闭非错误 toast 10s。错误不自动关闭。
watch(showActionError, (visible) => {
  if (!visible) {
    return
  }
  setTimeout(() => {
    showActionError.value = false
  }, 10000)
})

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
    await forumApi.createTopicComment(topic.value.id, {
      rawContent: markdown,
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      editorVersion: 'sf-editor-v1'
    })
    replyMarkdown.value = ''
    await refreshComments()
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
    await forumApi.createTopicComment(topic.value.id, {
      rawContent: markdown,
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      editorVersion: 'sf-editor-v1'
    }, comment.id)
    cancelReply()
    await refreshComments()
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
  <main class="min-h-screen py-8" style="background-color: var(--sf-surface)">
    <div class="max-w-[1376px] mx-auto px-4 sm:px-6">
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
        <!-- 面包屑 -->
        <nav class="text-sm text-slate-400 dark:text-zinc-500 mb-4 flex items-center gap-1.5">
          <NuxtLink :to="localePath('/')" class="hover:text-[color:var(--sf-accent)]">
            {{ t('topicDetail.breadcrumbHome') }}
          </NuxtLink>
          <UIcon name="i-lucide-chevron-right" class="size-3" />
          <NuxtLink
            v-if="topic.categorySlug"
            :to="categoryPath(topic.categorySlug)"
            class="hover:text-[color:var(--sf-accent)]"
          >
            {{ topic.categoryName }}
          </NuxtLink>
          <span v-else>{{ topic.categoryName }}</span>
        </nav>

        <!-- 双栏:正文 + 侧边栏 -->
        <div class="grid grid-cols-1 lg:grid-cols-[1fr_300px] gap-8 items-start">

        <!-- 主栏:主题头 + 评论区 -->
        <div class="min-w-0">
        <!-- 主题头部 -->
        <SFCard class="p-6 sm:p-8 mb-4">
          <div class="flex flex-wrap items-center gap-2 mb-3">
            <NuxtLink :to="categoryPath(topic.categorySlug)">
              <SFBadge variant="primary">{{ topic.categoryName }}</SFBadge>
            </NuxtLink>
            <SFBadge v-if="topic.isPinned" variant="danger">
              <UIcon name="i-lucide-pin" class="size-3.5" />
              {{ t('topicDetail.badge.pinned') }}
            </SFBadge>
            <SFBadge v-if="isLocked" variant="warning">
              <UIcon name="i-lucide-lock" class="size-3.5" />
              {{ t('topicDetail.badge.locked') }}
            </SFBadge>
          </div>

          <h1 class="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-zinc-50 tracking-tight mb-4 leading-tight">
            {{ topic.title }}
          </h1>

          <div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-slate-500 dark:text-zinc-400 pb-4 mb-5 border-b border-slate-100 dark:border-zinc-800">
            <component
              :is="authorPath ? 'NuxtLink' : 'span'"
              :to="authorPath"
              class="inline-flex items-center gap-2 font-medium text-slate-700 hover:text-[#0F766E] dark:text-zinc-300 dark:hover:text-teal-300"
            >
              <SFAvatar :name="authorName" size="sm" />
              <span>{{ authorName }}</span>
            </component>
            <span>{{ formatDate(topic.createdAt) }}</span>
            <span class="inline-flex items-center gap-1">
              <UIcon name="i-lucide-message-circle" class="size-3.5" />
              {{ topic.commentCount }}
            </span>
            <span class="inline-flex items-center gap-1">
              <UIcon name="i-lucide-eye" class="size-3.5" />
              {{ topic.viewCount }}
            </span>
          </div>

          <!-- 正文（后端已 sanitize）:sf-prose 由 @tailwindcss/typography 提供；v-highlight 负责代码块语法高亮 -->
          <div class="sf-prose" v-highlight v-html="sanitizeHtml(topic.content.htmlContent)" />

          <!-- 标签 -->
          <div v-if="topic.tags && topic.tags.length" class="flex flex-wrap gap-1.5 mt-6 pt-4 border-t border-slate-100 dark:border-zinc-800">
            <NuxtLink v-for="tag in topic.tags" :key="tag.id" :to="tagPath(tag.slug)">
              <SFBadge variant="neutral">#{{ tag.name }}</SFBadge>
            </NuxtLink>
          </div>

          <!-- 版主/作者动作区 -->
          <div
            v-if="canEditTopic(topic) || canDeleteTopic(topic) || canLock || canPin || canModerate || extensionActions.length"
            class="flex flex-wrap items-center gap-2 mt-6 pt-4 border-t border-slate-100 dark:border-zinc-800"
          >
            <SFButton
              v-if="canEditTopic(topic)"
              variant="ghost"
              size="sm"
              :to="`${localePath(forumTopicPath(topic, topicUrlMode))}?edit=1`"
            >
              <UIcon name="i-lucide-pencil" class="size-4" />
              <span>{{ t('topicDetail.edit') }}</span>
            </SFButton>
            <SFButton
              v-if="canDeleteTopic(topic)"
              variant="ghost"
              size="sm"
              @click="deleteTopic"
            >
              <UIcon name="i-lucide-trash-2" class="size-4" />
              <span>{{ t('topicDetail.delete') }}</span>
            </SFButton>
            <SFButton
              v-if="canLock"
              variant="ghost"
              size="sm"
              :disabled="actionState === 'pending'"
              @click="runTopicAction(isLocked ? 'unlock' : 'lock', 'topicDetail.lockToggled')"
            >
              <UIcon :name="isLocked ? 'i-lucide-lock-open' : 'i-lucide-lock'" class="size-4" />
              <span>{{ isLocked ? t('topicDetail.unlock') : t('topicDetail.lock') }}</span>
            </SFButton>
            <SFButton
              v-if="canPin"
              variant="ghost"
              size="sm"
              :disabled="actionState === 'pending'"
              @click="runTopicAction(isPinned ? 'unpin' : 'pin', 'topicDetail.pinToggled')"
            >
              <UIcon :name="isPinned ? 'i-lucide-pin-off' : 'i-lucide-pin'" class="size-4" />
              <span>{{ isPinned ? t('topicDetail.unpin') : t('topicDetail.pin') }}</span>
            </SFButton>
            <SFButton
              v-if="canModerate && !isLocked"
              variant="ghost"
              size="sm"
              :disabled="actionState === 'pending'"
              @click="runTopicAction('hide', 'topicDetail.hidden')"
            >
              <UIcon name="i-lucide-eye-off" class="size-4" />
              <span>{{ t('topicDetail.hide') }}</span>
            </SFButton>
            <SFButton
              v-if="reportUser"
              variant="ghost"
              size="sm"
              @click="openReportDialog({ type: 'topic', id: topic.id })"
            >
              <UIcon name="i-lucide-flag" class="size-4" />
              <span>{{ t('topicDetail.report') }}</span>
            </SFButton>
            <SFButton
              v-for="action in extensionActions"
              :key="topicExtensionActionKey(action)"
              variant="ghost"
              size="sm"
              :disabled="actionState === 'pending'"
              @click="runTopicExtensionAction(action)"
            >
              <UIcon
                :name="action.icon || 'i-lucide-plug'"
                class="size-4"
                :class="{ 'animate-spin': extensionActionRunning === topicExtensionActionKey(action) }"
              />
              <span>{{ topicExtensionActionLabel(action) }}</span>
            </SFButton>
          </div>

          <!-- 动作错误（不自动消失） -->
          <SFAlert
            v-if="showActionError"
            variant="danger"
            :title="actionError"
            closable
            class="mt-3"
            @close="showActionError = false"
          />
        </SFCard>

        <!-- 评论区域 -->
        <section class="space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-bold text-slate-800 dark:text-zinc-100">
              {{ t('topicDetail.commentsTitle', { count: commentTotal }) }}
            </h2>
            <SFTabs
              v-model="commentView"
              :items="[
                { label: t('topicDetail.viewTree'), value: 'tree' },
                { label: t('topicDetail.viewFlat'), value: 'flat' }
              ]"
              aria-label="评论视图切换"
            />
          </div>

          <!-- 评论加载骨架 -->
          <template v-if="commentsPending">
            <SFCard v-for="i in 3" :key="i" class="p-4">
              <SFSkeleton width="20%" height="1rem" class="mb-2" />
              <SFSkeleton width="90%" class="mb-1" />
              <SFSkeleton width="70%" />
            </SFCard>
          </template>

          <!-- 评论列表 -->
          <template v-else-if="comments.length">
            <SFCard class="sf-comment-list p-5">
              <!-- 递归评论树：SFComment 内部自递归渲染 children（含任意深度 + 折叠）。
                   操作按钮（回复/编辑/删除/举报）通过 commentActions 动态生成，颜色走 --sf-* token。
                   内联编辑器/回复编辑器由本页 provide 的 commentEditorRenderer 在评论原位渲染（任意层级）。 -->
              <SFComment
                v-for="comment in comments"
                :key="comment.id"
                :comment="comment"
                :author="commentAuthorName(comment)"
                :author-link="commentAuthorPath(comment)"
                :html-content="editingCommentId === comment.id ? undefined : comment.content.htmlContent"
                :content="editingCommentId === comment.id ? '' : undefined"
                :meta="commentMeta(comment)"
                :depth="0"
                :reply-to="comment.replyTo ? { author: forumAuthorName(comment.replyTo.author, comment.replyTo.id), excerpt: comment.replyTo.excerpt } : undefined"
                :actions="commentActions(comment)"
                :comment-meta-builder="commentMeta"
                :comment-author-link-builder="commentAuthorPath"
                @action-comment="(c: ForumComment, value: string) => handleCommentClick(c, value)"
              />
            </SFCard>

            <!-- 分页 -->
            <div v-if="commentTotalPages > 1" class="flex justify-center pt-2">
              <SFPagination v-model:page="commentPage" :total-pages="commentTotalPages" />
            </div>
          </template>

          <!-- 空评论 -->
          <SFCard v-else class="p-10">
            <SFEmptyState
              :title="t('topicDetail.emptyComments.title')"
              :description="t('topicDetail.emptyComments.description')"
            />
          </SFCard>

          <!-- 顶级回复编辑器 -->
          <SFCard v-if="showReplyEditor" class="p-5">
            <h3 class="text-sm font-semibold text-slate-700 mb-3 dark:text-zinc-300">
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
          </SFCard>

          <!-- 锁定提示 -->
          <SFAlert
            v-if="isLocked"
            variant="warning"
            :title="t('topicDetail.lockedNotice')"
            closable
          />
        </section>
        </div><!-- /主栏 -->

        <!-- 侧边栏:sticky 跟随,lg 以上显示 -->
        <aside class="hidden lg:block lg:sticky lg:top-20 space-y-4">
          <!-- 作者卡 -->
          <SFCard class="p-4">
            <div class="flex items-center gap-3">
              <SFAvatar :name="authorName" size="md" />
              <div class="min-w-0">
                <component
                  :is="authorPath ? 'NuxtLink' : 'span'"
                  :to="authorPath"
                  class="block font-semibold text-sm truncate hover:text-[color:var(--sf-accent)]"
                >
                  {{ authorName }}
                </component>
                <p class="text-xs text-slate-400 dark:text-zinc-500 mt-0.5">{{ t('topicDetail.authorLabel') }}</p>
              </div>
            </div>
          </SFCard>

          <!-- 话题统计 -->
          <SFCard class="p-4">
            <h3 class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wide mb-3">
              {{ t('topicDetail.statsTitle') }}
            </h3>
            <dl class="space-y-2.5 text-sm">
              <div class="flex items-center justify-between">
                <dt class="text-slate-400 dark:text-zinc-500 flex items-center gap-1.5">
                  <UIcon name="i-lucide-eye" class="size-3.5" />{{ t('topicDetail.statsViews') }}
                </dt>
                <dd class="font-medium text-slate-700 dark:text-zinc-200">{{ topic.viewCount }}</dd>
              </div>
              <div class="flex items-center justify-between">
                <dt class="text-slate-400 dark:text-zinc-500 flex items-center gap-1.5">
                  <UIcon name="i-lucide-message-circle" class="size-3.5" />{{ t('topicDetail.statsComments') }}
                </dt>
                <dd class="font-medium text-slate-700 dark:text-zinc-200">{{ topic.commentCount }}</dd>
              </div>
              <div class="flex items-center justify-between">
                <dt class="text-slate-400 dark:text-zinc-500 flex items-center gap-1.5">
                  <UIcon name="i-lucide-calendar" class="size-3.5" />{{ t('topicDetail.statsCreated') }}
                </dt>
                <dd class="font-medium text-slate-700 dark:text-zinc-200 text-xs">{{ formatDate(topic.createdAt) }}</dd>
              </div>
            </dl>
          </SFCard>
        </aside>

        </div><!-- /双栏 grid -->
      </template>
    </div>

    <!-- 举报对话框 -->
    <Teleport to="body">
      <div v-if="reportingTarget" class="sf-modal-overlay" @click.self="closeReportDialog">
        <div class="sf-modal" role="dialog" aria-modal="true">
          <div class="sf-modal__header">
            <h2 class="text-lg font-bold text-slate-900 dark:text-zinc-50">
              {{ t('moderation.reportTitle') }}
            </h2>
            <button type="button" class="sf-modal__close" :aria-label="t('moderation.close')" @click="closeReportDialog">
              <UIcon name="i-lucide-x" class="size-5" />
            </button>
          </div>
          <div v-if="reportSuccess" class="sf-modal__body">
            <SFAlert variant="success" :title="t('moderation.reportSubmitted')" />
          </div>
          <div v-else class="sf-modal__body space-y-4">
            <SFAlert v-if="reportError" variant="danger" :title="reportError" closable @close="reportError = ''" />
            <div>
              <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
                {{ t('moderation.reasonLabel') }}
              </label>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="opt in reportReasonOptions"
                  :key="opt.value"
                  type="button"
                  class="sf-modal__reason"
                  :class="{ 'sf-modal__reason--active': reportReason === opt.value }"
                  @click="reportReason = opt.value"
                >
                  {{ opt.label }}
                </button>
              </div>
            </div>
            <div>
              <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
                {{ t('moderation.bodyLabel') }}
              </label>
              <textarea
                v-model="reportBody"
                rows="3"
                maxlength="2000"
                class="sf-modal__textarea"
                :placeholder="t('moderation.bodyPlaceholder')"
              />
            </div>
          </div>
          <div v-if="!reportSuccess" class="sf-modal__footer">
            <SFButton variant="ghost" size="sm" :disabled="reportSubmitting" @click="closeReportDialog">
              {{ t('moderation.cancel') }}
            </SFButton>
            <SFButton variant="primary" size="sm" :disabled="!reportReason || reportSubmitting" @click="submitReport">
              {{ reportSubmitting ? t('moderation.submitting') : t('moderation.submit') }}
            </SFButton>
          </div>
        </div>
      </div>
    </Teleport>
  </main>
</template>

<style scoped>
.sf-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: 1rem;
}
.sf-modal {
  background: #ffffff;
  border-radius: 0.75rem;
  width: 100%;
  max-width: 28rem;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}
:global(.dark) .sf-modal {
  background: #18181b;
}
.sf-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid #f3f4f6;
}
:global(.dark) .sf-modal__header {
  border-bottom-color: #27272a;
}
.sf-modal__close {
  background: transparent;
  border: none;
  cursor: pointer;
  color: #6b7280;
  padding: 0.25rem;
}
.sf-modal__body {
  padding: 1.25rem;
}
.sf-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid #f3f4f6;
}
:global(.dark) .sf-modal__footer {
  border-top-color: #27272a;
}
.sf-modal__reason {
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  padding: 0.4rem 0.75rem;
  font-size: 0.85rem;
  background: #ffffff;
  color: #374151;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.sf-modal__reason--active {
  border-color: #0f766e;
  background: #e6f4f1;
  color: #0f766e;
}
:global(.dark) .sf-modal__reason {
  background: #18181b;
  border-color: #3f3f46;
  color: #d4d4d8;
}
:global(.dark) .sf-modal__reason--active {
  border-color: #14b8a6;
  background: rgba(20, 184, 166, 0.15);
  color: #5eead4;
}
.sf-modal__textarea {
  width: 100%;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.9rem;
  background: #ffffff;
  color: #111827;
  outline: none;
  resize: vertical;
}
.sf-modal__textarea:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}
:global(.dark) .sf-modal__textarea {
  background: #18181b;
  border-color: #3f3f46;
  color: #f4f4f5;
}
</style>
