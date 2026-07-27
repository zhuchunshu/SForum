export type TopicActionTone = 'default' | 'danger'

export type TopicActionLabels = {
  edit: string
  delete: string
  lock: string
  unlock: string
  pin: string
  unpin: string
  hide: string
  restore: string
  report: string
}

export type TopicExtensionActionDescriptor = {
  extensionId: string
  id: string
  label: string
  icon?: string
  confirm?: boolean
}

export type TopicActionMenuItem = {
  id: string
  label: string
  icon: string
  tone?: TopicActionTone
  requiresConfirm?: boolean
  extension?: {
    extensionId: string
    actionId: string
  }
}

export type CommentActionMenuItem = {
  label: string
  value: string
  icon?: string
}

export type CommentActionMenuInput = {
  canReply: boolean
  canEdit: boolean
  canDelete: boolean
  canReport: boolean
  labels: {
    reply: string
    link: string
    edit: string
    delete: string
    report: string
  }
  extensions: CommentActionMenuItem[]
}

export type TopicActionMenuInput = {
  canEdit: boolean
  canDelete: boolean
  canLock: boolean
  canPin: boolean
  canModerate: boolean
  canReport: boolean
  locked: boolean
  pinned: boolean
  hidden: boolean
  labels: TopicActionLabels
  extensions: TopicExtensionActionDescriptor[]
}

/**
 * 只把路由已经判定过的权限和状态转换成展示描述，不自行推断权限。
 */
export function buildTopicActionMenuItems(input: TopicActionMenuInput): TopicActionMenuItem[] {
  const items: TopicActionMenuItem[] = []

  if (input.canEdit) {
    items.push({ id: 'edit', label: input.labels.edit, icon: 'i-lucide-pencil' })
  }
  if (input.canLock) {
    items.push(input.locked
      ? { id: 'unlock', label: input.labels.unlock, icon: 'i-lucide-lock-open' }
      : { id: 'lock', label: input.labels.lock, icon: 'i-lucide-lock' })
  }
  if (input.canPin) {
    items.push(input.pinned
      ? { id: 'unpin', label: input.labels.unpin, icon: 'i-lucide-pin-off' }
      : { id: 'pin', label: input.labels.pin, icon: 'i-lucide-pin' })
  }
  if (input.canModerate) {
    items.push(input.hidden
      ? { id: 'restore', label: input.labels.restore, icon: 'i-lucide-eye' }
      : { id: 'hide', label: input.labels.hide, icon: 'i-lucide-eye-off' })
  }
  if (input.canDelete) {
    items.push({
      id: 'delete',
      label: input.labels.delete,
      icon: 'i-lucide-trash-2',
      tone: 'danger',
      requiresConfirm: true
    })
  }
  if (input.canReport) {
    items.push({ id: 'report', label: input.labels.report, icon: 'i-lucide-flag' })
  }

  for (const action of input.extensions) {
    items.push({
      id: `extension:${action.extensionId}:${action.id}`,
      label: action.label,
      icon: action.icon || 'i-lucide-plug',
      requiresConfirm: action.confirm,
      extension: {
        extensionId: action.extensionId,
        actionId: action.id
      }
    })
  }

  return items
}

/**
 * 权限仍由路由和 API 决定；这里仅统一评论操作的展示顺序。
 */
export function buildCommentActionMenuItems(input: CommentActionMenuInput): CommentActionMenuItem[] {
  const items: CommentActionMenuItem[] = []
  if (input.canReply) {
    // 评论区只保留回复；引用块由 replyTo 关系在展示层表达，不再提供「引用」入口。
    items.push({ label: input.labels.reply, value: 'reply', icon: 'i-lucide-reply' })
  }
  items.push({ label: input.labels.link, value: 'link', icon: 'i-lucide-link' })
  if (input.canEdit) items.push({ label: input.labels.edit, value: 'edit', icon: 'i-lucide-pencil' })
  if (input.canDelete) items.push({ label: input.labels.delete, value: 'delete', icon: 'i-lucide-trash-2' })
  if (input.canReport) items.push({ label: input.labels.report, value: 'report', icon: 'i-lucide-flag' })
  return items.concat(input.extensions)
}
