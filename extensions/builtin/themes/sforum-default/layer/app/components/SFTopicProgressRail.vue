<script setup lang="ts">
withDefaults(defineProps<{
  currentPage: number
  totalPages: number
  totalPosts: number
  firstLabel: string
  latestLabel: string
  canReply: boolean
  locked: boolean
  pending?: boolean
}>(), {
  pending: false
})

const emit = defineEmits<{
  reply: []
  first: []
  latest: []
}>()

const { t } = useI18n()
</script>

<template>
  <aside class="sf-topic-progress" :aria-label="t('topicDetail.progress.label')">
    <button
      v-if="canReply"
      type="button"
      class="sf-topic-progress__reply"
      :disabled="pending"
      @click="emit('reply')"
    >
      <UIcon name="i-lucide-reply" class="size-4" aria-hidden="true" />
      <span>{{ t('topicDetail.reply') }}</span>
    </button>
    <div v-else class="sf-topic-progress__locked" role="status">
      <UIcon name="i-lucide-lock" class="size-4" aria-hidden="true" />
      <span>{{ locked ? t('topicDetail.progress.replyLocked') : t('topicDetail.progress.replyUnavailable') }}</span>
    </div>

    <div class="sf-topic-progress__position">
      <strong>{{ t('topicDetail.progress.position', { current: currentPage, total: totalPages }) }}</strong>
      <span>{{ t('topicDetail.progress.totalPosts', { count: totalPosts }) }}</span>
    </div>

    <div class="sf-topic-progress__track" aria-hidden="true">
      <span :style="{ height: `${Math.min(100, Math.max(0, currentPage / Math.max(totalPages, 1) * 100))}%` }" />
    </div>

    <div class="sf-topic-progress__anchors">
      <button type="button" @click="emit('first')">
        <UIcon name="i-lucide-arrow-up-to-line" class="size-4" aria-hidden="true" />
        <span>
          <strong>{{ t('topicDetail.progress.first') }}</strong>
          <small>{{ firstLabel }}</small>
        </span>
      </button>
      <button type="button" @click="emit('latest')">
        <UIcon name="i-lucide-arrow-down-to-line" class="size-4" aria-hidden="true" />
        <span>
          <strong>{{ t('topicDetail.progress.latest') }}</strong>
          <small>{{ latestLabel }}</small>
        </span>
      </button>
    </div>
  </aside>
</template>
