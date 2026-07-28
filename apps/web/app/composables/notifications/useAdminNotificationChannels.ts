import type { ProviderSlotItem, ProviderSlotSelection } from '~/composables/admin/useAdminProviderSlots'
import { useApiClient } from '~/composables/useApiClient'

export type NotificationChannelCatalog = {
  revision: number
  items: ProviderSlotItem[]
}

export type NotificationChannelDelivery = {
  id: number
  notificationId?: number
  type: string
  channel: 'web_push'
  providerExtensionId?: string
  providerArtifactDigest?: string
  payloadVersion: number
  status: 'queued' | 'sending' | 'sent' | 'failed' | 'skipped'
  attemptCount: number
  reason?: string
  errorSummary?: string
  createdAt: string
  updatedAt: string
  completedAt?: string
}

export function useAdminNotificationChannels() {
  const { request } = useApiClient()
  const base = '/admin/notifications'

  const inspect = () => request<NotificationChannelCatalog>(`${base}/channels`)
  const select = (channel: string, candidateId: string, expectedRevision: number) => request<ProviderSlotSelection>(`${base}/channels/${channel}`, {
    method: 'PUT', body: { candidateId, expectedRevision }
  })
  const reset = (channel: string, revision: number) => request<{ reset: true }>(`${base}/channels/${channel}/reset`, {
    method: 'POST', body: { revision }
  })
  const test = (channel: string) => request<NotificationChannelDelivery>(`${base}/channels/${channel}/test`, { method: 'POST' })
  const deliveries = (limit = 50) => request<{ items: NotificationChannelDelivery[] }>(`${base}/deliveries?limit=${encodeURIComponent(limit)}`)

  return { inspect, select, reset, test, deliveries }
}
