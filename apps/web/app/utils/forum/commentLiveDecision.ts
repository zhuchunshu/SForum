// 评论实时提示的纯决策逻辑：输入服务端修订事实 + 本地阅读状态，输出页面动作。
// 抽成纯函数是为了让「什么时候自动追加 / 什么时候只出胶囊 / 什么时候静默对齐」
// 这一矩阵可以在没有浏览器的情况下单测（见 tests/forum/commentLiveDecision.test.ts）。

/** `GET /topics/:topicID/comments/revision` 的响应体（公开评论修订事实）。 */
export type ForumCommentRevisionState = {
  /** 单调递增修订号；只表示「公开可见状态变了」。 */
  revision: number
  /** 公开（active）评论总数，与列表 total 同口径。 */
  commentCount: number
  /** 当前末尾评论 id；主题没有 active 评论时缺省。 */
  lastCommentId?: number
  /** 当前末尾评论的发表时间；与 lastCommentId 同时缺省。 */
  lastCommentCreatedAt?: string
}

export type CommentRevisionFact = {
  revision: number
  commentCount: number
}

export type CommentLocalFact = {
  /** 本地已渲染列表的 total（服务端口径：公开 active 评论数）。 */
  total: number
  /** 本地列表最后一条评论 id；空列表为 0。 */
  lastItemId: number
  /** 本地已知修订号。 */
  revision: number
}

export type CommentLiveContext = {
  /** 当前是否停在最后一页（只有最后一页才可能自动追加）。 */
  isLastPage: boolean
  /** 视口是否贴近列表底部。 */
  nearBottom: boolean
  /** 编辑器/回复抽屉是否打开（打开时绝不改动 DOM）。 */
  composerOpen: boolean
  /** 自己刚发布，正在等待本地刷新对齐（避免把自己的评论提示成「新评论」）。 */
  selfJustPosted: boolean
  /** 单次自动补数的上限；超过则只提示跳页，避免一次拉取过多。 */
  autoAppendLimit: number
}

export type CommentLiveDecision =
  | { kind: 'ignore' }
  | { kind: 'append', newCount: number }
  | { kind: 'pill-new', newCount: number, jumpToLastPage: boolean }
  | { kind: 'reconcile' }

/**
 * 决策矩阵（与决策记录 2026-10-03-topic-comment-live-stream.md 一致）：
 * - 修订号未前进 → ignore；
 * - 有新评论且读者在最后一页底部、编辑器关闭 → append（静默追加）；
 * - 有新评论但读者已上滑 / 不在最后一页 / 新评论数超上限 → pill-new；
 * - 修订号前进但数量不变（编辑、隐藏、回复数变化）→ reconcile（静默重取当前页并 diff）。
 */
export function decideCommentLive(
  remote: CommentRevisionFact,
  local: CommentLocalFact,
  context: CommentLiveContext
): CommentLiveDecision {
  if (!Number.isFinite(remote.revision) || remote.revision <= local.revision) {
    return { kind: 'ignore' }
  }
  if (context.selfJustPosted) {
    return { kind: 'ignore' }
  }
  const newCount = Math.max(0, Math.trunc(remote.commentCount) - Math.max(0, Math.trunc(local.total)))
  if (newCount === 0) {
    return { kind: 'reconcile' }
  }
  if (context.composerOpen) {
    return { kind: 'pill-new', newCount, jumpToLastPage: false }
  }
  if (!context.isLastPage) {
    return { kind: 'pill-new', newCount, jumpToLastPage: true }
  }
  if (newCount > Math.max(1, context.autoAppendLimit)) {
    return { kind: 'pill-new', newCount, jumpToLastPage: true }
  }
  if (!context.nearBottom) {
    return { kind: 'pill-new', newCount, jumpToLastPage: false }
  }
  return { kind: 'append', newCount }
}

/** 站点级默认：单次自动补数上限等于一页（超过一页时改为跳页，避免拉取风暴）。 */
export function commentLiveAppendLimit(perPage: number): number {
  const value = Number.isFinite(perPage) ? Math.trunc(perPage) : 0
  // perPage 非法（0/负数/NaN）时回落到后端同款默认页大小，而不是 1。
  return value > 0 ? value : 20
}

type CommentComparable = {
  id: number
  updatedAt?: string
  status?: string
  replyCount?: number
}

function commentSignature(comment: CommentComparable): string {
  return [
    comment.id,
    comment.updatedAt || '',
    comment.status || '',
    comment.replyCount ?? 0
  ].join('|')
}

/**
 * 比较重取前后的列表是否真的变了：修订号前进不等于读者可见内容变化
 * （例如别人回复了另一页的评论、或插入后的定位写入）。
 */
export function commentListChanged(previous: CommentComparable[], next: CommentComparable[]): boolean {
  if (previous.length !== next.length) {
    return true
  }
  for (let index = 0; index < previous.length; index += 1) {
    if (commentSignature(previous[index]!) !== commentSignature(next[index]!)) {
      return true
    }
  }
  return false
}
