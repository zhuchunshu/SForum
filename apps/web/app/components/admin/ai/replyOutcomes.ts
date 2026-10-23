/**
 * 回复结果的展示口径。
 *
 * 这些判定必须只有一处：控制台的表格、将来可能的公共读面、以及测试都依赖
 * 「什么算成功」「链接指向哪里」。散落在模板里会让三处慢慢不一致。
 *
 * 边界：
 * - `posted` 是唯一算成功的结果。其余五种都是「没回」的不同原因，颜色区分
 *   可执行方向（查模型 / 查权限 / 查网关），而不是严重程度。
 * - 链接永远指向主题；评论锚点只有发布成功时才有。`commentId` 为 0 或缺失时
 *   不得拼出 `#comment-0`——那会指向一个不存在的楼层。
 */

export type ReplyOutcomeCode =
  | 'posted'
  | 'empty_reply'
  | 'tool_loop_stuck'
  | 'unavailable'
  | 'publish_failed'
  | 'failed'

export type ReplyOutcomeBadgeColor = 'success' | 'warning' | 'error' | 'neutral'

export function replyOutcomeColor(outcome: string): ReplyOutcomeBadgeColor {
  switch (outcome) {
    case 'posted':
      return 'success'
    case 'empty_reply':
    case 'tool_loop_stuck':
      return 'warning'
    case 'unavailable':
    case 'publish_failed':
    case 'failed':
      return 'error'
    default:
      return 'neutral'
  }
}

export type ReplyOutcomeTarget = {
  topicId: number | string
  commentId?: number | string | null
}

/**
 * 回复结果的落点：`/t/{topicId}` 加可选的评论锚点。
 *
 * 与引用渲染共用同一套地址（见 AIReply 的 Citation 注释）：主题永远可点，
 * 评论锚点只在 `commentId > 0` 时出现。没有主题 id 时返回空串，调用方据此
 * 渲染纯文本而不是一个指向 `/t/0` 的死链。
 */
export function replyOutcomeHref(item: ReplyOutcomeTarget): string {
  const topicID = Number(item.topicId)
  if (!Number.isFinite(topicID) || topicID <= 0) return ''
  const commentID = Number(item.commentId ?? 0)
  const anchor = Number.isFinite(commentID) && commentID > 0 ? `#comment-${commentID}` : ''
  return `/t/${topicID}${anchor}`
}
