<script setup lang="ts">
type CommentAction = {
  label: string
  value: string
  icon?: string
}

const props = withDefaults(defineProps<{
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
}>(), {
  content: '',
  htmlContent: undefined,
  authorLink: undefined,
  meta: undefined,
  avatar: undefined,
  depth: 0,
  replyTo: undefined,
  actions: () => [
    { label: '回复', value: 'reply', icon: 'i-lucide-reply' }
  ]
})

const emit = defineEmits<{
  action: [value: string]
}>()

const commentIndent = computed(() => ({
  marginLeft: `${Math.min(Math.max(props.depth, 0), 4) * 1.25}rem`
}))

// 后端已用 bluemonday sanitize，前端可直接 v-html 渲染。
const showHtml = computed(() => Boolean(props.htmlContent))
</script>

<template>
  <article class="sf-comment" :style="commentIndent">
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
      <div v-if="actions.length" class="sf-comment__actions">
        <button
          v-for="actionItem in actions"
          :key="actionItem.value"
          type="button"
          class="sf-comment__action"
          @click="emit('action', actionItem.value)"
        >
          <UIcon v-if="actionItem.icon" :name="actionItem.icon" class="size-3.5" />
          <span>{{ actionItem.label }}</span>
        </button>
      </div>
      <!-- 嵌套子评论插槽：tree 视图下父评论承载子评论 -->
      <div v-if="$slots.default" class="sf-comment__children mt-3 space-y-3">
        <slot />
      </div>
    </div>
  </article>
</template>
