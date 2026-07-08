<script lang="ts">
// provide/inject key 与 renderer 类型：父页面 provide 一个"按 comment 渲染内联编辑器"的函数，
// 递归子评论通过 inject 在原位渲染编辑器/回复框，避免递归 slot 透传导致的 TS 类型循环。
// 用字符串 key 方便父页面直接 provide，无需 import Symbol。详情页本地定义相同字符串即可。
export type CommentEditorRenderer = (comment: import('~/utils/forumTaxonomy').ForumComment | null) => unknown
export const COMMENT_EDITOR_RENDERER_KEY = 'sforum-comment-editor-renderer'
</script>

<script setup lang="ts">
import type { ForumComment } from '~/utils/forumTaxonomy'
import { countCommentDescendants, shouldCollapseByDefault } from '~/utils/forumTaxonomy'

type CommentAction = {
  label: string
  value: string
  icon?: string
}

const props = withDefaults(defineProps<{
  // 直接传整条评论：启用递归树渲染 + 折叠。优先于此组基础展示 props。
  comment?: ForumComment
  // 纯展示 props（向后兼容 components.vue 预览页与历史用法）。
  author: string
  // content 为纯文本（组件预览/历史用法）；htmlContent 优先，渲染后端已 sanitize 的 HTML。
  content?: string
  htmlContent?: string
  authorLink?: string
  meta?: string
  avatar?: string
  depth?: number
  replyTo?: { author?: string; excerpt?: string }
  actions?: CommentAction[]
  // 子评论 ≥ 此阈值时默认折叠（任意层级）。默认 4。
  collapsedThreshold?: number
  // 递归渲染子评论时，用这两个 builder 格式化子评论的 meta（时间等）和作者链接。
  // 父页面传入，避免组件内部硬编码时间/路由逻辑。
  commentMetaBuilder?: (comment: ForumComment) => string
  commentAuthorLinkBuilder?: (comment: ForumComment) => string
}>(), {
  comment: undefined,
  content: '',
  htmlContent: undefined,
  authorLink: undefined,
  meta: undefined,
  avatar: undefined,
  depth: 0,
  replyTo: undefined,
  actions: () => [
    { label: '回复', value: 'reply', icon: 'i-lucide-reply' }
  ],
  collapsedThreshold: 4,
  commentMetaBuilder: undefined,
  commentAuthorLinkBuilder: undefined
})

const emit = defineEmits<{
  action: [value: string]
  // 带 comment 引用的事件，方便父级在递归树中定位具体评论。
  'action-comment': [comment: ForumComment, value: string]
  reply: [comment: ForumComment]
  edit: [comment: ForumComment]
  delete: [comment: ForumComment]
  report: [comment: ForumComment]
  toggle: [comment: ForumComment, collapsed: boolean]
}>()

// 最大视觉缩进层级：超过此层级的评论不再继续缩进，改为平铺在同级，
// 用"回复 @父作者"引用块表达回复对象（类似 Reddit Mobile / Discord）。
// 避免深层讨论把正文挤压成窄条、竖线拉得又细又长。
const MAX_INDENT_DEPTH = 1

// 缩进：封顶 MAX_INDENT_DEPTH 层。
const commentIndent = computed(() => ({
  marginLeft: `${Math.min(Math.max(props.depth, 0), MAX_INDENT_DEPTH) * 1.25}rem`
}))

// 是否已到达缩进封顶层：本层及更深的评论用引用块表达回复对象。
const isAtMaxIndent = computed(() => props.depth >= MAX_INDENT_DEPTH)

// 后端已用 bluemonday sanitize，前端可直接 v-html 渲染。
const showHtml = computed(() => Boolean(props.htmlContent))

// —— 递归树渲染：从 comment prop 派生展示数据 ——
const commentNode = computed(() => props.comment ?? null)
const children = computed<ForumComment[]>(() => commentNode.value?.children ?? [])
const hasChildren = computed(() => children.value.length > 0)

// 扁平化后代：到达缩进封顶层后，不再一层层嵌套竖线容器，
// 而是把整条后代链（子、孙、曾孙…）拍平成一个列表，全部平铺在同一容器里，
// 靠各自的"回复 @某某"引用块表达回复对象。这样视觉上只有一层缩进，不再有深窄竖线。
const flattenedDescendants = computed<ForumComment[]>(() => {
  if (!isAtMaxIndent.value || !commentNode.value) return []
  const result: ForumComment[] = []
  const walk = (list: ForumComment[]) => {
    for (const c of list) {
      result.push(c)
      if (c.children && c.children.length > 0) {
        walk(c.children)
      }
    }
  }
  walk(children.value)
  return result
})

// 后代总数（递归统计所有层级），用于折叠按钮文案"展开 N 条回复"。
const descendantCount = computed(() => countCommentDescendants(commentNode.value))

// 折叠状态：直接子评论数 ≥ 阈值时默认折叠（任意层级）。
const collapsed = ref(false)
// 仅在 comment 首次可用时按阈值初始化一次，之后完全交给用户手动控制。
// 避免评论数据刷新（refreshComments）时重置用户的展开/折叠选择。
let collapseInitialized = false
watch(
  () => commentNode.value,
  (node) => {
    if (!collapseInitialized && node && node.children && node.children.length > 0) {
      collapsed.value = shouldCollapseByDefault(node, props.collapsedThreshold)
      collapseInitialized = true
    }
  },
  { immediate: true }
)

const isCollapsed = computed(() => commentNode.value ? collapsed.value : false)

function toggleCollapse() {
  if (!commentNode.value) return
  collapsed.value = !collapsed.value
  emit('toggle', commentNode.value, collapsed.value)
}

// nuxt-i18n 的 useI18n 在所有组件可用。折叠文案走 i18n key。
const { t } = useI18n()
function toggleLabel() {
  if (!commentNode.value) return ''
  // 折叠态显示后代总数（不只是直接子评论数），让用户知道折掉了多少内容。
  return collapsed.value
    ? t('topicDetail.expand', { n: descendantCount.value })
    : t('topicDetail.collapse')
}

// 操作按钮点击：同时触发 action（兼容旧用法）与语义化事件（带 comment）。
function onAction(actionItem: CommentAction) {
  emit('action', actionItem.value)
  if (commentNode.value) {
    emit('action-comment', commentNode.value, actionItem.value)
    switch (actionItem.value) {
      case 'reply': emit('reply', commentNode.value); break
      case 'edit': emit('edit', commentNode.value); break
      case 'delete': emit('delete', commentNode.value); break
      case 'report': emit('report', commentNode.value); break
    }
  }
}

// 递归子评论的展示数据：用 builder 格式化，回退到空字符串。
function childAuthor(c: ForumComment): string {
  if (c.author) {
    return c.author.displayName || c.author.username || `#${c.authorUserId}`
  }
  return `#${c.authorUserId}`
}
function childMeta(c: ForumComment): string {
  return props.commentMetaBuilder ? props.commentMetaBuilder(c) : ''
}
function childAuthorLink(c: ForumComment): string {
  return props.commentAuthorLinkBuilder ? props.commentAuthorLinkBuilder(c) : ''
}
function childReplyTo(c: ForumComment): { author?: string; excerpt?: string } | undefined {
  // 优先用后端提供的 replyTo 引用（flat 视图或后端已填充）。
  if (c.replyTo) {
    const name = c.replyTo.author?.displayName || c.replyTo.author?.username
    return { author: name, excerpt: c.replyTo.excerpt }
  }
  // 子评论深度达到缩进封顶层：即使后端没给 replyTo，也用父评论（当前节点）构造引用块，
  // 让深层回复明确"回复的是谁"，替代缩进层级表达。
  // 判断依据是子评论 c 的 depth（不是当前组件 depth）。
  if (c.depth >= MAX_INDENT_DEPTH && commentNode.value) {
    const parentName = commentNode.value.author?.displayName
      || commentNode.value.author?.username
      || `#${commentNode.value.authorUserId}`
    return { author: parentName, excerpt: commentNode.value.content.excerpt }
  }
  return undefined
}

// 透传事件给上层，保持递归树的事件冒泡。
function bubbleActionComment(c: ForumComment, v: string) { emit('action-comment', c, v) }
function bubbleReply(c: ForumComment) { emit('reply', c) }
function bubbleEdit(c: ForumComment) { emit('edit', c) }
function bubbleDelete(c: ForumComment) { emit('delete', c) }
function bubbleReport(c: ForumComment) { emit('report', c) }
function bubbleToggle(c: ForumComment, col: boolean) { emit('toggle', c, col) }

// 内联编辑器渲染：优先用父级 provide 的 renderer（递归子评论走这条路径），
// 否则用本组件的 #editor slot（顶层评论/预览页场景）。
const injectedEditorRenderer = inject<CommentEditorRenderer | null>(COMMENT_EDITOR_RENDERER_KEY, null)
const hasEditorSlot = computed(() => Boolean(useSlots().editor))
// 函数式包装组件：在原位调用 renderer 渲染编辑器 vnode。
// 模板里用 <InlineEditorHost /> 渲染；无 renderer 或无内容时返回 null（不占位）。
const InlineEditorHost = () => {
  const node = commentNode.value
  if (!node || !injectedEditorRenderer) return null
  const rendered = injectedEditorRenderer(node)
  if (rendered == null) return null
  // renderer 返回 vnode / vnode 数组，统一包进一个 div。
  return h('div', { class: 'sf-comment__inline-editor' }, [rendered as never])
}
</script>

<template>
  <article
    class="sf-comment"
    :class="{ 'sf-comment--collapsed': isCollapsed, 'sf-comment--deep': depth > 0 }"
    :style="commentIndent"
  >
    <component
      :is="authorLink ? 'NuxtLink' : 'div'"
      v-if="authorLink"
      :to="authorLink"
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
      <blockquote v-if="replyTo" class="sf-comment__reply-to">
        <UIcon name="i-lucide-corner-up-left" class="size-3.5 shrink-0" />
        <span v-if="replyTo.author" class="sf-comment__reply-to-author">{{ replyTo.author }}</span>
        <span class="sf-comment__reply-to-excerpt">{{ replyTo.excerpt }}</span>
      </blockquote>
      <div v-if="showHtml" class="sf-comment__content sf-prose" v-html="htmlContent" />
      <p v-else class="sf-comment__content">
        {{ content }}
      </p>

      <!-- 内联编辑器/回复编辑器：
           优先用 inject 的 renderer（详情页 provide，整棵递归树一致）；
           回退到 #editor slot（components.vue 预览页等无 renderer 场景）。 -->
      <InlineEditorHost v-if="injectedEditorRenderer" />
      <slot v-else name="editor" :comment="commentNode" />

      <div v-if="actions.length || hasChildren" class="sf-comment__actions">
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
        <!-- 折叠按钮：仅递归树模式 + 有子评论时出现 -->
        <button
          v-if="hasChildren"
          type="button"
          class="sf-comment__toggle"
          :aria-expanded="!isCollapsed"
          @click="toggleCollapse"
        >
          <UIcon name="i-lucide-chevron-down" class="size-3.5 sf-comment__toggle-icon" />
          <span>{{ toggleLabel() }}</span>
        </button>
      </div>

      <!-- 嵌套子评论：
           - 未到缩进封顶层：递归渲染竖线树（带 border-left 容器）。
           - 到达封顶层：把所有后代扁平化平铺，不再嵌套竖线容器，
             靠"回复 @某某"引用块表达回复对象，避免深窄竖线。 -->
      <div v-if="hasChildren && !isCollapsed" class="sf-comment__children">
        <!-- 封顶层：扁平化平铺所有后代 -->
        <template v-if="isAtMaxIndent">
          <SFComment
            v-for="descendant in flattenedDescendants"
            :key="descendant.id"
            :comment="descendant"
            :author="childAuthor(descendant)"
            :author-link="childAuthorLink(descendant)"
            :html-content="descendant.content.htmlContent"
            :meta="childMeta(descendant)"
            :depth="depth"
            :reply-to="childReplyTo(descendant)"
            :actions="actions"
            :collapsed-threshold="collapsedThreshold"
            :comment-meta-builder="commentMetaBuilder"
            :comment-author-link-builder="commentAuthorLinkBuilder"
            @action-comment="bubbleActionComment"
            @reply="bubbleReply"
            @edit="bubbleEdit"
            @delete="bubbleDelete"
            @report="bubbleReport"
            @toggle="bubbleToggle"
          />
        </template>
        <!-- 非封顶层：递归竖线树 -->
        <template v-else>
          <SFComment
            v-for="child in children"
            :key="child.id"
            :comment="child"
            :author="childAuthor(child)"
            :author-link="childAuthorLink(child)"
            :html-content="child.content.htmlContent"
            :meta="childMeta(child)"
            :depth="depth + 1"
            :reply-to="childReplyTo(child)"
            :actions="actions"
            :collapsed-threshold="collapsedThreshold"
            :comment-meta-builder="commentMetaBuilder"
            :comment-author-link-builder="commentAuthorLinkBuilder"
            @action-comment="bubbleActionComment"
            @reply="bubbleReply"
            @edit="bubbleEdit"
            @delete="bubbleDelete"
            @report="bubbleReport"
            @toggle="bubbleToggle"
          />
        </template>
      </div>
    </div>
  </article>
</template>
