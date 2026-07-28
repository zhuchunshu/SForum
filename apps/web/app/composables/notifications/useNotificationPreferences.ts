import { useApiClient } from '~/composables/useApiClient'

export type NotificationPreferenceState = 'inherit' | 'enabled' | 'disabled'

export type NotificationPreferenceItem = {
  type: string
  category: string
  channel: string
  active: boolean
  enabled: boolean
  recommendedEnabled: boolean
  userConfigurable: boolean
  required: boolean
  state: NotificationPreferenceState
  effective: boolean
  /** V2 plugin descriptors may expose ownership metadata after lifecycle publication. */
  ownerExtensionId?: string
  ownerLabel?: string
  /** External providers can report an unavailable transport without hiding inbox policy. */
  channelAvailable?: boolean
}

export type NotificationPreferenceCatalog = {
  revision: number
  items: NotificationPreferenceItem[]
}

export type NotificationPreferenceUpdate = {
  type: string
  channel: string
  state: NotificationPreferenceState
}

export function notificationPreferenceKey(item: Pick<NotificationPreferenceItem, 'type' | 'channel'>) {
  return `${item.type}:${item.channel}`
}

export function canOverrideNotificationPreference(item: NotificationPreferenceItem) {
  return item.active
    && item.enabled
    && item.channelAvailable !== false
    && item.userConfigurable
    && !item.required
}

export function preferenceUpdateItems(items: NotificationPreferenceItem[], states: Record<string, NotificationPreferenceState>): NotificationPreferenceUpdate[] {
  return items
    .filter(canOverrideNotificationPreference)
    .map(item => ({
      type: item.type,
      channel: item.channel,
      state: states[notificationPreferenceKey(item)] || 'inherit'
    }))
}

export function useNotificationPreferences() {
  const { request } = useApiClient()

  function list() {
    return request<NotificationPreferenceCatalog>('/notification-preferences')
  }

  function replace(revision: number, items: NotificationPreferenceUpdate[]) {
    return request<NotificationPreferenceCatalog>('/notification-preferences', {
      method: 'PUT',
      body: { revision, items }
    })
  }

  function restore(revision: number) {
    return request<NotificationPreferenceCatalog>('/notification-preferences/restore', {
      method: 'POST',
      body: { revision }
    })
  }

  return { list, replace, restore }
}
