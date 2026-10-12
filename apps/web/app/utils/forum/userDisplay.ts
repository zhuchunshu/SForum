import type { ForumUserSummary } from '~/utils/forum/forumTaxonomy'

/**
 * 全站唯一的展示名解析：displayName -> username -> fallback。
 *
 * 接受任何带 username/displayName 的对象（UserSummary、PublicProfile 等），
 * 回退终点由调用方通过 fallback 表达（评论区用 `#userId`，通知用 i18n unknownAuthor），
 * 本函数不写死任何场景语义，也不做网络兜底——资料补齐是预览卡片的职责。
 */
export type ForumUserRef = {
  username?: string | null
  displayName?: string | null
  avatar?: ForumUserSummary['avatar'] | null
} | undefined | null

export function userDisplayName(user: ForumUserRef, fallback = ''): string {
  const display = user?.displayName?.trim()
  if (display) {
    return display
  }
  const username = user?.username?.trim()
  if (username) {
    return username
  }
  return fallback
}
