<script setup lang="ts">
import { useTopicCommentSubmission } from '~/composables/forum/useTopicCommentSubmission'
import type { ForumComment, ForumTopicDetail } from '~/utils/forum/forumTaxonomy'

type AvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

const props = defineProps<{
  topic: ForumTopicDetail
  refreshComments: () => Promise<unknown>
  actorName?: string
  avatar?: AvatarView | null
}>()

const emit = defineEmits<{
  open: [draft: string]
}>()

const { t } = useI18n()
const replyingTo = ref<ForumComment | null>(null)
const {
  replyMarkdown,
  replySubmitting,
  replyError,
  showReplyError,
  commentCooldownActive,
  replyDisplayError,
  submitReply
} = useTopicCommentSubmission({
  topic: toRef(props, 'topic'),
  replyingTo,
  refreshComments: props.refreshComments
})

function cancelReply() {
  replyMarkdown.value = ''
  replyError.value = ''
  showReplyError.value = false
}
</script>

<template>
  <section id="topic-reply-editor" class="sforum-topic-comments__reply">
    <header class="sforum-topic-comments__reply-head">
      <SFAvatar v-if="actorName" :name="actorName" :avatar="avatar" size="sm" />
      <span class="sforum-topic-comments__reply-head-copy">
        <strong>{{ t('topicDetail.replyTitle') }}</strong>
        <small v-if="actorName">{{ t('topicDetail.replyAs', { name: actorName }) }}</small>
      </span>
      <button type="button" class="sforum-topic-comments__reply-advanced" @click="emit('open', replyMarkdown)">
        {{ t('topicDetail.advancedReply') }}
      </button>
    </header>

    <LazySFEditor
      v-model="replyMarkdown"
      compact
      :rows="5"
      :placeholder="t('topicDetail.replyPlaceholder')"
      :submit-label="replySubmitting ? t('topicDetail.submitting') : t('topicDetail.submitReply')"
      :cancel-label="t('topicDetail.cancel')"
      :support-label="t('topicDetail.markdownSupported')"
      :disabled="replySubmitting"
      :submit-disabled="commentCooldownActive"
      @cancel="cancelReply"
      @submit="submitReply"
    />

    <SFAlert
      v-if="showReplyError"
      variant="danger"
      :title="replyDisplayError"
      :closable="!commentCooldownActive"
      class="mx-3 mb-3"
      @close="showReplyError = false"
    />
  </section>
</template>
