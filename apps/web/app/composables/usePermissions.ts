import type { CurrentUser } from './useAuthSession'

// 论坛权限键，与后端 seeds.go 常量保持一致。
export const FORUM_PERMISSIONS = {
  topicCreate: 'topic.create',
  topicEditAny: 'topic.edit_any',
  topicDeleteAny: 'topic.delete_any',
  topicLock: 'topic.lock',
  topicPin: 'topic.pin',
  postCreate: 'post.create',
  postEditOwn: 'post.edit_own',
  postEditAny: 'post.edit_any',
  postDeleteOwn: 'post.delete_own',
  postDeleteAny: 'post.delete_any',
  settingsManage: 'settings.manage',
  moderationManage: 'moderation.manage',
  moderationReview: 'moderation.review'
} as const

/**
 * 权限辅助 composable。基于当前登录用户的 permissions 数组判定。
 * 注意：前端权限判定只是 UX 辅助，后端 policy 检查才是权威。
 */
export function usePermissions() {
  const { user } = useAuthSession()

  function can(permission: string): boolean {
    return Boolean(user.value && user.value.permissions?.includes(permission))
  }

  function canAny(permissions: string[]): boolean {
    return permissions.some((permission) => can(permission))
  }

  // 是否可编辑指定主题：版主任意编辑，或作者凭 post.edit_own。
  function canEditTopic(topic: { authorUserId: number }): boolean {
    if (can(FORUM_PERMISSIONS.topicEditAny)) {
      return true
    }
    return Boolean(user.value?.id === topic.authorUserId && can(FORUM_PERMISSIONS.postEditOwn))
  }

  // 是否可删除指定主题：版主凭 topic.delete_any，或作者凭 post.delete_own。
  function canDeleteTopic(topic: { authorUserId: number }): boolean {
    if (can(FORUM_PERMISSIONS.topicDeleteAny)) {
      return true
    }
    return Boolean(user.value?.id === topic.authorUserId && can(FORUM_PERMISSIONS.postDeleteOwn))
  }

  // 是否可编辑/删除评论：版主或作者。
  function canEditComment(comment: { authorUserId: number }): boolean {
    if (can(FORUM_PERMISSIONS.postEditAny)) {
      return true
    }
    return Boolean(user.value?.id === comment.authorUserId && can(FORUM_PERMISSIONS.postEditOwn))
  }

  function canDeleteComment(comment: { authorUserId: number }): boolean {
    if (can(FORUM_PERMISSIONS.postDeleteAny)) {
      return true
    }
    return Boolean(user.value?.id === comment.authorUserId && can(FORUM_PERMISSIONS.postDeleteOwn))
  }

  return {
    user,
    can,
    canAny,
    canEditTopic,
    canDeleteTopic,
    canEditComment,
    canDeleteComment
  }
}

export type PermissionChecker = ReturnType<typeof usePermissions>

export function hasPermission(user: CurrentUser | null, permission: string): boolean {
  return Boolean(user && user.permissions?.includes(permission))
}
