import type { NotificationItem, NotificationListFilters } from '~/composables/notifications/useNotifications'

export const NOTIFICATION_PREVIEW_LIMIT = 3

export const notificationPreviewTabs = ['all', 'reply', 'mention'] as const

export type NotificationPreviewTab = typeof notificationPreviewTabs[number]

export function notificationPreviewFilters(tab: NotificationPreviewTab): NotificationListFilters {
  return tab === 'all' ? {} : { type: tab }
}

export function limitNotificationPreviewItems<T extends NotificationItem>(items: T[]): T[] {
  return items.slice(0, NOTIFICATION_PREVIEW_LIMIT)
}
