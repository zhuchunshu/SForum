export type NotificationPolicyItem = {
  type: string
  category: string
  channel: string
  active: boolean
  enabled: boolean
  recommendedEnabled: boolean
  userConfigurable: boolean
  required: boolean
  ownerExtensionId?: string
  ownerLabel?: string
  channelAvailable?: boolean
}

export type NotificationPolicyCatalog = {
  revision: number
  items: NotificationPolicyItem[]
}

export type NotificationPolicyUpdate = Pick<NotificationPolicyItem, 'type' | 'channel' | 'enabled' | 'recommendedEnabled' | 'userConfigurable'>

export function notificationPolicyKey(item: Pick<NotificationPolicyItem, 'type' | 'channel'>) {
  return `${item.type}:${item.channel}`
}

export function canEditNotificationPolicy(item: NotificationPolicyItem) {
  return item.active && !item.required
}

export function notificationPolicyUpdates(items: NotificationPolicyItem[]): NotificationPolicyUpdate[] {
  return items
    .filter(canEditNotificationPolicy)
    .map(item => ({
      type: item.type,
      channel: item.channel,
      enabled: item.enabled,
      recommendedEnabled: item.recommendedEnabled,
      userConfigurable: item.userConfigurable
    }))
}
