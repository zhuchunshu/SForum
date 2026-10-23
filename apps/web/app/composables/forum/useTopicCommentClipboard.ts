import { sforumCommentShortcodeText } from '~/utils/editor/shortcodes'
import type { ForumComment } from '~/utils/forum/forumTaxonomy'

/**
 * 主题详情页评论剪贴板操作：复制评论深链、复制「引用此评论」的 shortcode 文本。
 * 成功走 Toast，剪贴板不可用时回退 window.prompt 兜底。
 */
export function useTopicCommentClipboard() {
  const { t } = useI18n()
  const toast = useToast()

  function showCopiedToast(key: string) {
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t(key), duration: 5000 })
  }

  async function writeClipboard(text: string, successKey: string, promptKey: string) {
    if (!import.meta.client) {
      return
    }
    try {
      await navigator.clipboard.writeText(text)
      showCopiedToast(successKey)
    } catch {
      window.prompt(t(promptKey), text)
    }
  }

  async function copyCommentLink(comment: ForumComment) {
    if (!import.meta.client) {
      return
    }
    const url = `${window.location.origin}${window.location.pathname}${window.location.search}#comment-${comment.id}`
    await writeClipboard(url, 'topicDetail.commentLinkCopied', 'topicDetail.copyLinkHint')
  }

  // 复制 shortcode 文本，粘贴进编辑器后渲染为评论引用块。
  async function copyCommentShortcode(comment: ForumComment) {
    await writeClipboard(sforumCommentShortcodeText(comment.id), 'topicDetail.shortcodeCopied', 'topicDetail.copyShortcodeHint')
  }

  return { copyCommentLink, copyCommentShortcode }
}
