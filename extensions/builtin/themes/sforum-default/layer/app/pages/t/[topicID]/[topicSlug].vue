<script setup lang="ts">
import {
  forumAuthorName,
  forumCategoryPath,
  forumTagPath,
  forumTopicPath,
  forumUserProfilePath,
  FORUM_TOPIC_ACTIONS,
  type ForumComment,
  type ForumCommentList,
  type ForumTopicDetail
} from '~/utils/forumTaxonomy'

definePageMeta({
  // 主题详情对所有人可见（公开读限定 active/locked）。
  public: true
})

const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const { siteName } = useWebOptions()
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

const topicID = computed(() => Number(route.params.topicID))
const topicSlug = computed(() => String(route.params.topicSlug ?? ''))

// canonical slug 校验：若 URL slug 与实际 topic.slug 不同则 301 到规范路径。
const canonicalPath = computed(() => forumTopicPath({ id: topicID.value, slug: topicSlug.value }))

const { data: topic, error: topicError } = await useAsyncData(
  () => `forum-topic-${topicID.value}`,
  () => forumApi.getTopic(topicID.value),
  {
    // 后端对 hidden/deleted 主题返回 404，这里正常抛错由 error 页处理。
    default: () => null as ForumTopicDetail | null
  }
)

// 拿到真实 slug 后做 canonical 重定向。
watchEffect(() => {
  if (!topic.value) {
    return
  }
  const expected = topic.value.slug
  if (topicSlug.value && expected && topicSlug.value !== expected) {
    const target = localePath(forumTopicPath({ id: topicID.value, slug: expected }))
    if (import.meta.server) {
      navigateTo(target, { redirectCode: 301 })
    } else {
      navigateTo(target, { replace: true })
    }
  }
})

useSForumSeo({
  title: () => topic.value ? `${topic.value.title} - ${siteName.value}` : siteName.value,
  description: () => topic.value?.content.excerpt || t('topicDetail.metaDescription'),
  type: 'article',
  path: () => canonicalPath.value,
  noindex: () => !topic.value,
  schema: () => topic.value ? {
    type: 'DiscussionForumPosting',
    datePublished: topic.value.createdAt,
    dateModified: topic.value.updatedAt,
    authorName: forumAuthorName(topic.value.author, topic.value.authorUserId)
  } : undefined
})

// 评论数据：默认 tree 视图。
const commentPage = ref(1)
const commentView = ref<'tree' | 'flat'>('tree')
const commentQuery = computed(() => ({
  view: commentView.value,
  page: commentPage.value,
  perPage: 20
}))

const { data: commentData, pending: commentsPending, refresh: refreshComments } = await useAsyncData(
  () => `forum-topic-comments-${topicID.value}-${commentView.value}-${commentPage.value}`,
  () => forumApi.listTopicComments(topicID.value, commentQuery.value),
  {
    default: () => ({ items: [], total: 0, page: 1, perPage: 20, view: commentView.value }) as ForumCommentList,
    watch: [commentQuery]
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
  const actions: { label: string; value: string; icon?: string }[] = []
  if (canReplyToComments.value) {
    actions.push({ label: t('topicDetail.reply'), value: 'reply', icon: 'i-lucide-reply' })
  }
  return actions
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
  if (value === 'reply') {
    startReply(comment)
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

function onCommentEditSubmit(comment: ForumComment) {
  return (payload: { markdown: string }) => saveCommentEdit(comment, { markdown: payload.markdown })
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

function onNestedReplySubmit(comment: ForumComment) {
  return (payload: { markdown: string }) => submitNestedReply(comment, { markdown: payload.markdown })
}
</script>

<template>
  <main class="min-h-screen py-8" style="background-color: var(--sf-surface)">
    <div class="max-w-4xl mx-auto px-4 sm:px-6">
      <!-- 错误 / 未找到 -->
      <SFCard v-if="topicError && !topic" class="p-10">
        <SFEmptyState
          :title="t('topicDetail.notFound.title')"
          :description="t('topicDetail.notFound.description')"
        />
      </SFCard>

      <template v-else-if="topic">
        <!-- 主题头部 -->
        <SFCard class="p-6 mb-4">
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

          <h1 class="text-2xl font-bold text-slate-900 dark:text-zinc-50 mb-3">
            {{ topic.title }}
          </h1>

          <div class="flex flex-wrap items-center gap-3 text-sm text-slate-500 dark:text-zinc-400 mb-4">
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

          <!-- 正文（后端已 sanitize） -->
          <div class="sf-prose text-slate-800 dark:text-zinc-200" v-html="topic.content.htmlContent" />

          <!-- 标签 -->
          <div v-if="topic.tags && topic.tags.length" class="flex flex-wrap gap-2 mt-4">
            <NuxtLink v-for="tag in topic.tags" :key="tag.id" :to="tagPath(tag.slug)">
              <SFBadge variant="neutral">#{{ tag.name }}</SFBadge>
            </NuxtLink>
          </div>

          <!-- 版主/作者动作区 -->
          <div
            v-if="canEditTopic(topic) || canDeleteTopic(topic) || canLock || canPin || canModerate"
            class="flex flex-wrap items-center gap-2 mt-6 pt-4 border-t border-slate-100 dark:border-zinc-800"
          >
            <SFButton
              v-if="canEditTopic(topic)"
              variant="ghost"
              size="sm"
              :to="localePath(`/t/${topic.id}/${topic.slug}/edit`)"
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
            <SFCard class="p-5 space-y-5">
              <div v-for="comment in comments" :key="comment.id" class="space-y-2">
                <SFComment
                  :author="commentAuthorName(comment)"
                  :author-link="commentAuthorPath(comment)"
                  :html-content="editingCommentId === comment.id ? undefined : comment.content.htmlContent"
                  :content="editingCommentId === comment.id ? '' : undefined"
                  :meta="commentMeta(comment)"
                  :depth="0"
                  :reply-to="comment.replyTo ? { author: forumAuthorName(comment.replyTo.author, comment.replyTo.id), excerpt: comment.replyTo.excerpt } : undefined"
                  :actions="commentActions(comment)"
                  @action="(value: string) => handleCommentClick(comment, value)"
                >
                  <!-- 内联编辑器（替换正文） -->
                  <template v-if="editingCommentId === comment.id">
                    <SFEditor
                      v-model="editingMarkdown"
                      :placeholder="t('topicDetail.editPlaceholder')"
                      :submit-label="t('topicDetail.saveEdit')"
                      :disabled="editingSubmitting"
                      :error="editingError"
                      @submit="onCommentEditSubmit(comment)"
                    />
                    <div class="flex gap-2 mt-2">
                      <SFButton variant="ghost" size="sm" :disabled="editingSubmitting" @click="cancelEditComment">
                        {{ t('topicDetail.cancel') }}
                      </SFButton>
                    </div>
                  </template>
                  <!-- 评论操作按钮 -->
                  <template v-else>
                    <div v-if="isCommentEditable(comment) || isCommentDeletable(comment)" class="flex gap-2 mt-2">
                      <button
                        v-if="isCommentEditable(comment)"
                        type="button"
                        class="inline-flex items-center gap-1 text-xs text-slate-500 hover:text-[#0F766E] dark:text-zinc-400 dark:hover:text-teal-300"
                        :aria-label="t('topicDetail.edit')"
                        @click="startEditComment(comment)"
                      >
                        <UIcon name="i-lucide-pencil" class="size-3.5" />
                        <span>{{ t('topicDetail.edit') }}</span>
                      </button>
                      <button
                        v-if="isCommentDeletable(comment)"
                        type="button"
                        class="inline-flex items-center gap-1 text-xs text-slate-500 hover:text-red-600 dark:text-zinc-400 dark:hover:text-red-400"
                        :aria-label="t('topicDetail.delete')"
                        :disabled="deletingCommentId === comment.id"
                        @click="deleteComment(comment)"
                      >
                        <UIcon name="i-lucide-trash-2" class="size-3.5" />
                        <span>{{ deletingCommentId === comment.id ? t('topicDetail.deleting') : t('topicDetail.delete') }}</span>
                      </button>
                    </div>
                  </template>

                  <!-- 嵌套回复编辑器 -->
                  <template v-if="replyingTo && replyingTo.id === comment.id">
                    <SFEditor
                      v-model="nestedReplyMarkdown"
                      :placeholder="t('topicDetail.replyPlaceholder')"
                      :submit-label="t('topicDetail.submitReply')"
                      :disabled="nestedReplySubmitting"
                      @submit="onNestedReplySubmit(comment)"
                    />
                    <div class="flex gap-2 mt-2">
                      <SFButton variant="ghost" size="sm" :disabled="nestedReplySubmitting" @click="cancelReply">
                        {{ t('topicDetail.cancel') }}
                      </SFButton>
                    </div>
                  </template>

                  <!-- 嵌套子评论（tree 视图） -->
                  <template v-if="comment.children && comment.children.length && editingCommentId !== comment.id">
                    <SFComment
                      v-for="child in comment.children"
                      :key="child.id"
                      :author="commentAuthorName(child)"
                      :author-link="commentAuthorPath(child)"
                      :html-content="editingCommentId === child.id ? undefined : child.content.htmlContent"
                      :meta="commentMeta(child)"
                      :depth="1"
                      :reply-to="child.replyTo ? { author: forumAuthorName(child.replyTo.author, child.replyTo.id), excerpt: child.replyTo.excerpt } : undefined"
                      :actions="commentActions(child)"
                      @action="(value: string) => handleCommentClick(child, value)"
                    />
                  </template>
                </SFComment>
              </div>
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
            <SFEditor
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
      </template>
    </div>
  </main>
</template>
