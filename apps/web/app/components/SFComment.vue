<script lang="ts">
// provide/inject key 与 renderer 类型：父页面 provide 一个"按 comment 渲染内联编辑器"的函数，
// 评论列表通过 inject 在评论原位渲染编辑器/回复框。
// 用字符串 key 方便父页面直接 provide，无需 import Symbol。详情页本地定义相同字符串即可。
export type CommentEditorRenderer = (comment: import('~/utils/forumTaxonomy').ForumComment | null) => unknown
export const COMMENT_EDITOR_RENDERER_KEY = 'sforum-comment-editor-renderer'
</script>

<script setup lang="ts">
import type { ForumComment } from '~/utils/forumTaxonomy'

type CommentAction = {
  label: string
  value: string
  icon?: string
}

const props = withDefaults(defineProps<{
  // 当前评论节点：仅用于内联编辑器/回复编辑器匹配（扁平布局不再用它做树渲染）。
  // 父页面扁平化后逐条传入，编辑器 renderer 据此判断是否在原位渲染编辑框。
  comment?: ForumComment
  // 纯展示 props。扁平列表布局，不再递归渲染子评论。
  author: string
  // content 为纯文本（组件预览/历史用法）；htmlContent 优先，渲染后端已 sanitize 的 HTML。
  content?: string
  htmlContent?: string
  authorLink?: string
  meta?: string
  avatar?: string
  // 被回复的评论引用：E3 方案用左侧 accent 竖条引用块展示，人名 + 内容预览分两行。
  replyTo?: { author?: string; excerpt?: string }
  actions?: CommentAction[]
}>(), {
  comment: undefined,
  content: '',
  htmlContent: undefined,
  authorLink: undefined,
  meta: undefined,
  avatar: undefined,
  replyTo: undefined,
  actions: () => [
    { label: '回复', value: 'reply', icon: 'i-lucide-reply' }
  ]
})

const emit = defineEmits<{
  action: [value: string]
}>()

// 后端已用 bluemonday sanitize，前端可直接 v-html 渲染。
const showHtml = computed(() => Boolean(props.htmlContent))

// 当前评论节点：用于内联编辑器/回复编辑器匹配（inject renderer 据此判断原位渲染）。
const commentNode = computed(() => props.comment ?? null)

// 操作按钮点击。
function onAction(actionItem: CommentAction) {
  emit('action', actionItem.value)
}

// 内联编辑器渲染：优先用父级 provide 的 renderer（评论列表原位编辑/回复），
// 否则用本组件的 #editor slot（components.vue 预览页场景）。
const injectedEditorRenderer = inject<CommentEditorRenderer | null>(COMMENT_EDITOR_RENDERER_KEY, null)
const hasEditorSlot = computed(() => Boolean(useSlots().editor))
const InlineEditorHost = () => {
  const node = commentNode.value
  if (!node || !injectedEditorRenderer) return null
  const rendered = injectedEditorRenderer(node)
  if (rendered == null) return null
  return h('div', { class: 'sf-comment__inline-editor' }, [rendered as never])
}
</script>

<template>
  <!-- 扁平列表布局：所有评论平铺，无缩进无树。
       回复对象用左侧 accent 竖条引用块表达（E3 方案）。 -->
  <article class="sf-comment sf-comment--flat">
    <component
      :is="authorLink ? 'NuxtLink' : 'div'"
      v-if="authorLink"
      :to="authorLink"
      class="sf-comment__avatar-link"
    >
      <SFAvatar :name="author" :src="avatar" size="sm" />
    </component>
    <SFAvatar v-else :name="author" :src="avatar" size="sm" />
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

      <!-- E3：左侧 accent 竖条引用块。人名加粗 accent 色，内容预览灰色，最多两行省略。 -->
      <blockquote v-if="replyTo" class="sf-comment__reply-to">
        <UIcon name="i-lucide-corner-up-left" class="sf-comment__reply-to-icon size-3.5 shrink-0" />
        <span v-if="replyTo.author" class="sf-comment__reply-to-author">{{ replyTo.author }}</span>
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
          <UIcon v-if="actionItem.icon" :name="actionItem.icon" class="size-3.5" />
          <span>{{ actionItem.label }}</span>
        </button>
      </div>
    </div>
  </article>
</template>
