<script setup lang="ts">
type CommentAction = {
  label: string
  value: string
}

const props = withDefaults(defineProps<{
  author: string
  content: string
  meta?: string
  avatar?: string
  depth?: number
  actions?: CommentAction[]
}>(), {
  meta: undefined,
  avatar: undefined,
  depth: 0,
  actions: () => [
    { label: '回复', value: 'reply' },
    { label: '赞同', value: 'like' }
  ]
})

const emit = defineEmits<{
  action: [value: string]
}>()

const commentIndent = computed(() => ({
  marginLeft: `${Math.min(Math.max(props.depth, 0), 4) * 1.25}rem`
}))
</script>

<template>
  <article class="sf-comment" :style="commentIndent">
    <SFAvatar :name="author" :src="avatar" size="sm" />
    <div class="sf-comment__body">
      <header class="sf-comment__header">
        <span class="sf-comment__author">{{ author }}</span>
        <span v-if="meta" class="sf-comment__meta">{{ meta }}</span>
      </header>
      <p class="sf-comment__content">
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
          {{ actionItem.label }}
        </button>
      </div>
    </div>
  </article>
</template>
