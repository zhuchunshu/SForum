<script setup lang="ts">
type AvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

defineProps<{
  actorName?: string
  avatar?: AvatarView | null
}>()

const emit = defineEmits<{
  open: []
}>()

const { t } = useI18n()
</script>

<template>
  <section id="topic-reply-editor" class="sforum-topic-comments__reply">
    <header class="sforum-topic-comments__reply-head">
      <SFAvatar v-if="actorName" :name="actorName" :avatar="avatar" size="sm" />
      <span class="sforum-topic-comments__reply-head-copy">
        <strong>{{ t('topicDetail.replyTitle') }}</strong>
        <small v-if="actorName">{{ t('topicDetail.replyAs', { name: actorName }) }}</small>
      </span>
      <button type="button" class="sforum-topic-comments__reply-advanced" @click="emit('open')">
        {{ t('topicDetail.advancedReply') }}
      </button>
    </header>

    <button type="button" class="sforum-topic-comments__reply-launcher" @click="emit('open')">
      <span>{{ t('topicDetail.replyPlaceholder') }}</span>
      <UIcon name="i-lucide-expand" class="size-4" aria-hidden="true" />
    </button>
  </section>
</template>
