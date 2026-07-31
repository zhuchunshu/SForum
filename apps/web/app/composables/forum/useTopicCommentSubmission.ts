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
  /** 兼容旧高级回复链接：目标评论不在当前页时仍保留服务端 parentId。 */
  replyParentId?: Ref<number | null>
  refreshComments: () => Promise<unknown>
}

export type TopicCommentEditorContent = {
  markdown?: string
  native?: unknown
  text?: string
  attachmentIds?: number[]
  pendingUploadCount?: number
}

export function topicCommentEditorContentIsMeaningful(
  payload?: TopicCommentEditorContent,
  fallbackMarkdown = ''
) {
  if ([payload?.text, payload?.markdown, fallbackMarkdown]
    .some(value => typeof value === 'string' && value.trim() !== '')) {
    return true
  }

  function containsImage(node: unknown): boolean {
    if (!node || typeof node !== 'object') return false
    const candidate = node as { type?: unknown, content?: unknown }
    if (candidate.type === 'image') return true
    return Array.isArray(candidate.content) && candidate.content.some(containsImage)
  }

  return containsImage(payload?.native)
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

  async function submitReply(payload?: TopicCommentEditorContent) {
    if (
      !options.topic.value
      || replySubmitting.value
      || commentCooldownActive.value
      || (payload?.pendingUploadCount || 0) > 0
    ) return
    const markdown = payload?.markdown ?? replyMarkdown.value
    if (!topicCommentEditorContentIsMeaningful(payload, markdown)) return

    replySubmitting.value = true
    replyError.value = ''
    showReplyError.value = false
    try {
      const created = await forumApi.createTopicComment(
        options.topic.value.id,
        forumContentFromEditorPayload({
          markdown,
          native: payload?.native,
          text: payload?.text,
          attachmentIds: payload?.attachmentIds
        }),
        options.replyParentId?.value ?? options.replyingTo.value?.id
      )
      replyMarkdown.value = ''
      options.replyingTo.value = null
      if (options.replyParentId) {
        options.replyParentId.value = null
      }
      if (created.status === 'pending') {
        toast.add({ color: 'primary', icon: 'i-lucide-clock-3', title: t('topicDetail.replySubmittedForReview'), duration: 10000 })
      } else {
        await options.refreshComments()
        toast.add({ color: 'success', icon: 'i-lucide-check', title: t('topicDetail.replyPosted'), duration: 10000 })
      }
      return created
    } catch (error) {
      replyError.value = captureCommentCooldown(error) ? '' : apiErrorMessage(error) || t('topicDetail.replyFailed')
      showReplyError.value = true
      return null
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
