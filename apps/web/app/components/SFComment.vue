<script lang="ts">
// provide/inject key 与 renderer 类型：父页面 provide 一个"按 comment 渲染内联编辑器"的函数，
// 评论列表通过 inject 在评论原位渲染编辑器/回复框。
// 用字符串 key 方便父页面直接 provide，无需 import Symbol。详情页本地定义相同字符串即可。
export type CommentEditorRenderer = (comment: import('~/utils/forumTaxonomy').ForumComment | null) => unknown
export const COMMENT_EDITOR_RENDERER_KEY = 'sforum-comment-editor-renderer'
</script>

<script setup lang="ts">
import {
  forumAuthorName,
  type ForumComment
} from '~/utils/forumTaxonomy'
import {
  commentBranchPresentation,
  type CommentPresentationMode
} from '~/utils/forumCommentPresentation'
import type { AvatarView } from '~/composables/useProfileApi'

type CommentAction = {
  label: string
  value: string
  icon?: string
}

const props = withDefaults(defineProps<{
  // 当前评论节点同时承载内联编辑器匹配和 tree 模式的递归数据。
  comment?: ForumComment
  presentation?: CommentPresentationMode
  depth?: number
  collapseFromDepth?: number
  author: string
  // content 为纯文本（组件预览/历史用法）；htmlContent 优先，渲染后端已 sanitize 的 HTML。
  content?: string
  htmlContent?: string
  authorLink?: string
  meta?: string
  avatar?: AvatarView | null
  // 被回复的评论引用：E3 方案用左侧 accent 竖条引用块展示，人名 + 内容预览分两行。
  replyTo?: { author?: string; excerpt?: string }
  actions?: CommentAction[]
  commentMetaBuilder?: (comment: ForumComment) => string
  commentAuthorLinkBuilder?: (comment: ForumComment) => string
  commentActionsBuilder?: (comment: ForumComment) => CommentAction[]
  /** 正在加载更多回复的评论 id（详情页控制） */
  loadingMoreCommentId?: number | null
}>(), {
  comment: undefined,
  presentation: 'flat',
  depth: 0,
  collapseFromDepth: 2,
  content: '',
  htmlContent: undefined,
  authorLink: undefined,
  meta: undefined,
  avatar: undefined,
  replyTo: undefined,
  actions: () => [
    { label: '回复', value: 'reply', icon: 'i-lucide-reply' }
  ],
  commentMetaBuilder: undefined,
  commentAuthorLinkBuilder: undefined,
  commentActionsBuilder: undefined,
  loadingMoreCommentId: null
})

const emit = defineEmits<{
  action: [value: string]
  actionComment: [comment: ForumComment, value: string]
  loadMoreReplies: [comment: ForumComment]
}>()

const { t } = useI18n()

// 后端已用 bluemonday sanitize，前端可直接 v-html 渲染。
const showHtml = computed(() => Boolean(props.htmlContent))

// 当前评论节点：用于内联编辑器/回复编辑器匹配（inject renderer 据此判断原位渲染）。
const commentNode = computed(() => props.comment ?? null)
const childComments = computed(() => props.comment?.children || [])
const branchPresentation = computed(() => commentBranchPresentation(
  props.presentation,
  props.depth,
  childComments.value,
  props.collapseFromDepth
))
const branchExpanded = ref(false)
const branchId = `sf-comment-branch-${useId()}`
const branchVisible = computed(() => !branchPresentation.value.collapsible || branchExpanded.value)

function toggleBranch() {
  branchExpanded.value = !branchExpanded.value
}

// 操作按钮点击。
function onAction(actionItem: CommentAction) {
  emit('action', actionItem.value)
  if (props.comment) {
    emit('actionComment', props.comment, actionItem.value)
  }
}

function childAuthorName(comment: ForumComment) {
  return forumAuthorName(comment.author, comment.authorUserId)
}

function childMeta(comment: ForumComment) {
  return props.commentMetaBuilder?.(comment) || ''
}

function childAuthorLink(comment: ForumComment) {
  return props.commentAuthorLinkBuilder?.(comment) || ''
}

function childActions(comment: ForumComment) {
  return props.commentActionsBuilder?.(comment) || props.actions
}

function childReplyTo(comment: ForumComment) {
  if (!comment.replyTo) {
    return undefined
  }
  return {
    author: forumAuthorName(comment.replyTo.author, comment.replyTo.id),
    excerpt: comment.replyTo.excerpt
  }
}

function forwardChildAction(comment: ForumComment, value: string) {
  emit('actionComment', comment, value)
}

function forwardLoadMoreReplies(comment: ForumComment) {
  emit('loadMoreReplies', comment)
}

function onLoadMoreReplies() {
  if (props.comment) {
    emit('loadMoreReplies', props.comment)
  }
}

const showLoadMoreReplies = computed(() =>
  props.presentation === 'tree'
  && Boolean(props.comment?.hasMoreChildren)
)
const isLoadingMoreReplies = computed(() =>
  props.comment != null && props.loadingMoreCommentId === props.comment.id
)

// 内联编辑器渲染：优先用父级 provide 的 renderer（评论列表原位编辑/回复），
// 否则用本组件的 #editor slot（components.vue 预览页场景）。
const injectedEditorRenderer = inject<CommentEditorRenderer | null>(COMMENT_EDITOR_RENDERER_KEY, null)
const InlineEditorHost = () => {
  const node = commentNode.value
  if (!node || !injectedEditorRenderer) return null
  const rendered = injectedEditorRenderer(node)
  if (rendered == null) return null
  return h('div', { class: 'sf-comment__inline-editor' }, [rendered as never])
}
</script>

<template>
  <div
    class="sf-comment"
    :class="[
      `sf-comment--${presentation}`,
      { 'sf-comment--indented': branchPresentation.indentation === 1 }
    ]"
    :data-comment-depth="depth"
    :data-visual-depth="branchPresentation.indentation"
  >
    <article class="sf-comment__entry">
      <component
        :is="authorLink ? 'NuxtLink' : 'div'"
        v-if="authorLink"
        :to="authorLink"
        class="sf-comment__avatar-link"
      >
        <SFAvatar :name="author" :avatar="avatar" size="sm" />
      </component>
      <SFAvatar v-else :name="author" :avatar="avatar" size="sm" />
      <div class="sf-comment__body">
        <header class="sf-comment__header">
          <component
            :is="authorLink ? 'NuxtLink' : 'span'"
            :to="authorLink"
            class="sf-comment__author"
          >
            {{ author }}
          </component>
          <span v-if="meta" class="sf-comment__meta">{{ meta }}</span>
        </header>

        <blockquote v-if="replyTo" class="sf-comment__reply-to">
          <span class="sf-comment__reply-to-label">
            <UIcon name="i-lucide-corner-up-left" class="sf-comment__reply-to-icon size-3.5 shrink-0" aria-hidden="true" />
            <span class="sf-comment__reply-to-author">
              {{ t('topicDetail.reply') }}<template v-if="replyTo.author"> @{{ replyTo.author }}</template>
            </span>
          </span>
          <span v-if="replyTo.excerpt" class="sf-comment__reply-to-excerpt">{{ replyTo.excerpt }}</span>
        </blockquote>

        <div v-if="showHtml" class="sf-comment__content sf-prose" v-highlight v-html="sanitizeHtml(htmlContent)" />
        <p v-else class="sf-comment__content">
          {{ content }}
        </p>

        <!-- 内联编辑器/回复编辑器 -->
        <InlineEditorHost v-if="injectedEditorRenderer" />
        <slot v-else name="editor" :comment="commentNode" />

        <div v-if="actions.length" class="sf-comment__actions">
          <button
            v-for="actionItem in actions"
            :key="actionItem.value"
            type="button"
            class="sf-comment__action"
            @click="onAction(actionItem)"
          >
            <UIcon v-if="actionItem.icon" :name="actionItem.icon" class="size-3.5" aria-hidden="true" />
            <span>{{ actionItem.label }}</span>
          </button>
        </div>
      </div>
    </article>

    <button
      v-if="branchPresentation.collapsible"
      type="button"
      class="sf-comment__disclosure"
      :aria-expanded="branchExpanded"
      :aria-controls="branchId"
      @click="toggleBranch"
    >
      <UIcon
        :name="branchExpanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
        class="size-4"
        aria-hidden="true"
      />
      <span>
        {{ branchExpanded
          ? t('topicDetail.collapse')
          : t('topicDetail.expand', { n: branchPresentation.followUpCount }) }}
      </span>
    </button>

    <div
      v-if="presentation === 'tree' && childComments.length"
      :id="branchId"
      class="sf-comment__branch"
      :class="{ 'sf-comment__branch--connected': branchPresentation.connectionRail }"
      :hidden="!branchVisible"
    >
      <SFComment
        v-for="child in childComments"
        :key="child.id"
        :comment="child"
        :presentation="presentation"
        :depth="depth + 1"
        :collapse-from-depth="collapseFromDepth"
        :author="childAuthorName(child)"
        :avatar="child.author?.avatar"
        :author-link="childAuthorLink(child)"
        :html-content="child.content.htmlContent"
        :meta="childMeta(child)"
        :reply-to="childReplyTo(child)"
        :actions="childActions(child)"
        :comment-meta-builder="commentMetaBuilder"
        :comment-author-link-builder="commentAuthorLinkBuilder"
        :comment-actions-builder="commentActionsBuilder"
        :loading-more-comment-id="loadingMoreCommentId"
        @load-more-replies="forwardLoadMoreReplies"
        @action-comment="forwardChildAction"
      />
    </div>

    <!-- D2：树子孙截断后通过 ListCommentReplies 加载更多 -->
    <button
      v-if="showLoadMoreReplies"
      type="button"
      class="sf-comment__load-more"
      :disabled="isLoadingMoreReplies"
      @click="onLoadMoreReplies"
    >
      <UIcon
        :name="isLoadingMoreReplies ? 'i-lucide-loader-2' : 'i-lucide-chevrons-down'"
        class="size-4"
        :class="{ 'animate-spin': isLoadingMoreReplies }"
        aria-hidden="true"
      />
      <span>{{ t('topicDetail.loadMoreReplies') }}</span>
    </button>
  </div>
</template>
