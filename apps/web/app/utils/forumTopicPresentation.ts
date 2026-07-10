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
