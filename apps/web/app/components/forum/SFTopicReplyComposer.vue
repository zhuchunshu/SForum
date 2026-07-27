<script setup lang="ts">
type AvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

defineProps<{
  modelValue: string
  actorName?: string
  avatar?: AvatarView | null
  replyTarget?: { author: string, href?: string, floorLabel?: string } | null
  submitting?: boolean
  error?: string
  /** 高级回复独立页路径（含 query）；空则不渲染入口 */
  advancedTo?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'cancel': []
  'submit': [payload: { markdown: string, native?: unknown, text?: string }]
  'dismiss-error': []
  /** 跳转高级回复前：父级可写入草稿交接 */
  'advanced': []
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
      <NuxtLink
        v-if="advancedTo"
        :to="advancedTo"
        class="sforum-topic-comments__reply-advanced"
        @click="emit('advanced')"
      >
        {{ t('topicDetail.advancedReply') }}
      </NuxtLink>
    </header>

    <div v-if="replyTarget" class="sforum-topic-comments__reply-target">
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

    <!-- 内容页评论输入始终展开，不提供折叠态 -->
    <LazySFEditor
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
