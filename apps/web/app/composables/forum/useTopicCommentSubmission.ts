import type { Ref } from 'vue'
import { apiErrorMessage } from '~/composables/useApiClient'
import { useForumApi } from '~/composables/forum/useForumApi'
import { useForumCooldownError } from '~/composables/forum/useForumCooldownError'
import {
  forumContentFromEditorPayload,
  type ForumComment,
  type ForumTopicDetail
} from '~/utils/forum/forumTaxonomy'

type TopicCommentSubmissionOptions = {
  topic: Ref<ForumTopicDetail | null | undefined>
  replyingTo: Ref<ForumComment | null>
  refreshComments: () => Promise<unknown>
}

export function useTopicCommentSubmission(options: TopicCommentSubmissionOptions) {
  const { t } = useI18n()
  const forumApi = useForumApi()
  const toast = useToast()
  const replyMarkdown = ref('')
  const replySubmitting = ref(false)
  const replyError = ref('')
  const showReplyError = ref(false)
  const {
    active: commentCooldownActive,
    message: commentCooldownMessage,
    capture: captureCommentCooldown
  } = useForumCooldownError('comment')
  const replyDisplayError = computed(() => commentCooldownMessage.value || replyError.value)

  async function submitReply(payload?: { markdown?: string; native?: unknown; text?: string }) {
    if (!options.topic.value || replySubmitting.value || commentCooldownActive.value) return
    const markdown = payload?.markdown ?? replyMarkdown.value
    if (!(payload?.text || markdown).trim()) return

    replySubmitting.value = true
    replyError.value = ''
    showReplyError.value = false
    try {
      const created = await forumApi.createTopicComment(
        options.topic.value.id,
        forumContentFromEditorPayload({ markdown, native: payload?.native, text: payload?.text }),
        options.replyingTo.value?.id
      )
      replyMarkdown.value = ''
      options.replyingTo.value = null
      if (created.status === 'pending') {
        toast.add({ color: 'primary', icon: 'i-lucide-clock-3', title: t('topicDetail.replySubmittedForReview'), duration: 10000 })
      } else {
        await options.refreshComments()
        toast.add({ color: 'success', icon: 'i-lucide-check', title: t('topicDetail.replyPosted'), duration: 10000 })
      }
    } catch (error) {
      replyError.value = captureCommentCooldown(error) ? '' : apiErrorMessage(error) || t('topicDetail.replyFailed')
      showReplyError.value = true
    } finally {
      replySubmitting.value = false
    }
  }

  return {
    replyMarkdown,
    replySubmitting,
    replyError,
    showReplyError,
    commentCooldownActive,
    replyDisplayError,
    submitReply
  }
}
