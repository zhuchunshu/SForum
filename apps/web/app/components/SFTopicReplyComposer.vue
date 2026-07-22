<script setup lang="ts">
type AvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

defineProps<{
  modelValue: string
  open: boolean
  actorName?: string
  avatar?: AvatarView | null
  replyTarget?: { author: string, href?: string, floorLabel?: string } | null
  submitting?: boolean
  error?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'open': []
  'cancel': []
  'submit': [payload: { markdown: string, native?: unknown, text?: string }]
  'dismiss-error': []
}>()

const { t } = useI18n()
</script>

<template>
  <section id="topic-reply-editor" class="sforum-topic-comments__reply">
    <header class="sforum-topic-comments__reply-head">
      <SFAvatar v-if="actorName" :name="actorName" :avatar="avatar" size="sm" />
      <span>
        <strong>{{ t('topicDetail.replyTitle') }}</strong>
        <small v-if="actorName">{{ t('topicDetail.replyAs', { name: actorName }) }}</small>
      </span>
    </header>

    <button
      v-if="!open"
      type="button"
      class="sforum-topic-page__action-btn sforum-topic-page__action-btn--primary"
      @click="emit('open')"
    >
      <UIcon name="i-lucide-reply" class="size-4" aria-hidden="true" />
      {{ t('topicDetail.replyTopic') }}
    </button>

    <div v-else-if="replyTarget" class="sforum-topic-comments__reply-target">
      <span>
        <UIcon name="i-lucide-corner-up-left" class="size-4" aria-hidden="true" />
        {{ t('topicDetail.replyingTo') }}
        <strong>@{{ replyTarget.author }}</strong>
        <a v-if="replyTarget.href && replyTarget.floorLabel" :href="replyTarget.href">{{ replyTarget.floorLabel }}</a>
      </span>
      <button type="button" :aria-label="t('topicDetail.cancel')" @click="emit('cancel')">
        <UIcon name="i-lucide-x" class="size-4" aria-hidden="true" />
      </button>
    </div>

    <LazySFEditor
      v-if="open"
      :model-value="modelValue"
      compact
      :rows="5"
      :placeholder="t('topicDetail.replyPlaceholder')"
      :submit-label="submitting ? t('topicDetail.submitting') : t('topicDetail.submitReply')"
      :cancel-label="t('topicDetail.cancel')"
      :support-label="t('topicDetail.markdownSupported')"
      :disabled="submitting"
      @update:model-value="emit('update:modelValue', $event)"
      @cancel="emit('cancel')"
      @submit="emit('submit', $event)"
    />

    <SFAlert
      v-if="error"
      variant="danger"
      :title="error"
      closable
      class="mt-3"
      @close="emit('dismiss-error')"
    />
  </section>
</template>
