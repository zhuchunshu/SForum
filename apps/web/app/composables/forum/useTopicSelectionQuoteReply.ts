import type { ComputedRef } from 'vue'
import { flattenCommentTree, type ForumComment } from '~/utils/forum/forumTaxonomy'
import type { SelectionQuoteRequest } from '~/utils/forum/forumSelectionQuote'

type TopicSelectionQuoteReplyOptions = {
  comments: ComputedRef<ForumComment[]>
  canReply: ComputedRef<boolean>
  startReply: (comment: ForumComment, initialDraft?: string) => void
  openTopicReply: (initialDraft?: string) => void
}

export function useTopicSelectionQuoteReply(options: TopicSelectionQuoteReplyOptions) {
  return (request: SelectionQuoteRequest) => {
    if (!options.canReply.value) return

    if (request.target.kind === 'topic') {
      options.openTopicReply(request.markdown)
      return
    }

    const commentId = request.target.commentId
    const comment = flattenCommentTree(options.comments.value).find(candidate => candidate.id === commentId)
    if (comment) options.startReply(comment, request.markdown)
  }
}
